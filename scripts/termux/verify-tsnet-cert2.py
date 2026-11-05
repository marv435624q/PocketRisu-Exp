#!/usr/bin/env python3
"""Verify the packaged candidate offline; HTTPS verification is explicit opt-in."""
import argparse
import hashlib
import http.client
import json
import os
from pathlib import Path
import re
import selectors
import signal
import subprocess
import sys
import tempfile
import time

ROOT = Path(__file__).resolve().parents[2]
PACKAGE = ROOT / "prebuilt/termux/arm64/tsnet-v1.104.5-cert2"
BINARY = PACKAGE / "libpocketrisu_tsnet_exec.so"
HEX = re.compile(r"^[0-9a-f]{64}$")
STATUSES = {"off", "connecting", "reconnecting", "needs-login", "needs-approval", "running", "https-error", "error"}
BACKENDS = {"NoState", "InUseOtherUser", "NeedsLogin", "NeedsMachineAuth", "Stopped", "Starting", "Running"}
HTTPS_ERRORS = {"request", "certificate", "deadline", "https-request", "http-response"}
HTTPS_STAGES = {"prerequisites", "certificate", "post-certificate", "listener"}
NETWORK_CLASSES = {"dns", "tcp", "tls", "timeout", "other"}
PROGRESS_STAGES = {"account", "order", "dns", "issuance-wait", "certificate"}
PROGRESS_STATES = {"start", "done", "error"}


def child_environment():
    # Do not pass any Tailscale credentials or control overrides to the probe.
    return {key: value for key, value in os.environ.items()
            if not key.upper().startswith("TS_") and key.upper() != "TSNET_FORCE_LOGIN"}


def classify_error(value):
    text = value.lower() if isinstance(value, str) else ""
    if "existing tailscale registration required" in text:
        return "registration-required"
    if "zero" in text and "noise" in text and "key" in text:
        return "zero-noise-key"
    if "deadline" in text or "timeout" in text or "timed out" in text:
        return "deadline"
    return "other"



def sanitized_status(obj):
    status, backend, registered = obj.get("status"), obj.get("backendState"), obj.get("registered")
    if not isinstance(status, str) or status not in STATUSES:
        return None
    row = {"status": status, "backendState": backend if isinstance(backend, str) and backend in BACKENDS else None,
           "registered": registered if type(registered) is bool else None}
    stage = obj.get("httpsStage")
    network = obj.get("networkClass")
    if isinstance(stage, str) and stage in HTTPS_STAGES:
        row["httpsStage"] = stage
    if isinstance(network, str) and network in NETWORK_CLASSES:
        row["networkClass"] = network
    if obj.get("error"):
        row["errorCode"] = classify_error(obj["error"])
    return row


def sanitized_progress(obj):
    stage, state, elapsed = obj.get("stage"), obj.get("state"), obj.get("elapsedMs")
    if not isinstance(stage, str) or stage not in PROGRESS_STAGES:
        return None
    if not isinstance(state, str) or state not in PROGRESS_STATES:
        return None
    if type(elapsed) is not int or not 0 <= elapsed <= 300000:
        return None
    return {"stage": stage, "state": state, "elapsedMs": elapsed}


def sanitized_https(obj):
    status = obj.get("httpStatus")
    valid_status = type(status) is int and 100 <= status <= 599
    trusted = obj.get("tlsVerified") is True
    good = obj.get("success") is True and trusted and valid_status and 200 <= status < 500
    code = obj.get("errorCode")
    if good:
        code = None
    elif not isinstance(code, str) or code not in HTTPS_ERRORS:
        code = "http-response" if trusted and valid_status and status >= 500 else "other"
    row = {"success": bool(good), "tlsVerified": trusted,
           "httpStatus": status if valid_status else None, "errorCode": code}
    network = obj.get("networkClass")
    if isinstance(network, str) and network in NETWORK_CLASSES:
        row["networkClass"] = network
    return row


class SafeArgumentParser(argparse.ArgumentParser):
    def error(self, message):
        safe_messages = {"timeout must be between 30 and 300 seconds", "--https requires --state-dir and --hostname",
                         "existing absolute tailscale-state directory and tailscaled.state required",
                         "--state-dir and --hostname require --https"}
        super().error(message if message in safe_messages else "invalid command options; use --help")


def identity_and_binary():
    metadata = json.loads((PACKAGE / "provenance.json").read_text(encoding="utf-8"))
    binary = metadata["binary"]
    identity = metadata["identity"]
    digest = binary["sha256"]
    source_id = identity["sourceId"]
    if not isinstance(digest, str) or not HEX.fullmatch(digest) or not isinstance(source_id, str) or not HEX.fullmatch(source_id):
        raise ValueError("invalid provenance")
    if identity.get("candidate") != "tsnet-v1.104.5-cert2" or identity.get("baseVersion") != "1.104.5":
        raise ValueError("invalid candidate identity")
    if identity.get("longVersion") != "1.104.5-pocketrisu-exp-cert2-s" + source_id[:12]:
        raise ValueError("invalid long version")
    if type(binary["bytes"]) is not int or BINARY.stat().st_size != binary["bytes"]:
        raise ValueError("binary size mismatch")
    with BINARY.open("rb") as stream:
        digest_builder = hashlib.sha256()
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest_builder.update(chunk)
        actual = digest_builder.hexdigest()
    if actual != digest:
        raise ValueError("binary hash mismatch")
    return identity


def single_probe(args, expected, json_type=None):
    try:
        result = subprocess.run([str(BINARY), *args], stdin=subprocess.DEVNULL, stdout=subprocess.PIPE,
                                stderr=subprocess.DEVNULL, timeout=15, check=False, env=child_environment())
        good = result.returncode == expected
        if json_type:
            lines = result.stdout.decode("utf-8").splitlines()
            good = good and any(isinstance(x, dict) and x.get("type") == json_type
                                for x in (json.loads(line) for line in lines))
        return {"ok": bool(good), "exitCode": result.returncode}
    except (OSError, ValueError, subprocess.TimeoutExpired):
        return {"ok": False, "errorCode": "probe-failed"}


def build_info(identity):
    try:
        result = subprocess.run([str(BINARY), "--build-info"], stdin=subprocess.DEVNULL, stdout=subprocess.PIPE,
                                stderr=subprocess.DEVNULL, timeout=15, check=False, env=child_environment())
        info = json.loads(result.stdout)
        source_id = identity["sourceId"]
        expected = {"type": "build-info", "candidate": identity["candidate"], "baseVersion": "1.104.5",
                    "shortVersion": "1.104.5", "longVersion": identity["longVersion"], "sourceId": source_id,
                    "custom": True, "goVersion": "go1.27.3", "goos": "android", "goarch": "arm64"}
        if result.returncode != 0 or not isinstance(info, dict) or any(info.get(k) != v for k, v in expected.items()):
            return {"ok": False, "errorCode": "build-info-mismatch"}
        return {"ok": True, **expected}
    except (OSError, ValueError, subprocess.TimeoutExpired):
        return {"ok": False, "errorCode": "build-info-unavailable"}


def preflight():
    conn = http.client.HTTPConnection("127.0.0.1", 6001, timeout=3)
    try:
        conn.request("GET", "/")
        response = conn.getresponse()
        return 100 <= response.status <= 599
    except (OSError, http.client.HTTPException):
        return False
    finally:
        conn.close()


def stop_process(proc):
    if proc.poll() is None:
        try:
            proc.stdin.write(b'{"action":"stop"}\n')
            proc.stdin.flush()
        except (OSError, ValueError):
            pass
        try:
            proc.wait(timeout=10)
        except subprocess.TimeoutExpired:
            pass
    # Even after the leader exits, its process group may contain descendants.
    # Probe and signal only the group created by Popen(start_new_session=True).
    def group_exists():
        try:
            os.killpg(proc.pid, 0)
            return True
        except ProcessLookupError:
            return False

    try:
        if not group_exists():
            return True
        os.killpg(proc.pid, signal.SIGTERM)
        deadline = time.monotonic() + 5
        while group_exists() and time.monotonic() < deadline:
            proc.poll()
            time.sleep(0.1)
        if group_exists():
            os.killpg(proc.pid, signal.SIGKILL)
            deadline = time.monotonic() + 5
            while group_exists() and time.monotonic() < deadline:
                proc.poll()
                time.sleep(0.1)
        return not group_exists()
    except ProcessLookupError:
        return True
    except OSError:
        return False


class CleanupFailure(Exception):
    def __init__(self, statuses, progress):
        self.statuses = statuses
        self.progress = progress


def online(state, hostname, timeout):
    try:
        return _online(state, hostname, timeout)
    except CleanupFailure as err:
        return {"ok": False, "errorCode": "cleanup-failed", "statuses": err.statuses, "progress": err.progress}


def consume_child_json(raw, statuses, progress):
    try:
        obj = json.loads(raw)
    except (ValueError, UnicodeError):
        return None, None
    if not isinstance(obj, dict):
        return None, None
    if obj.get("type") == "https-verification":
        return sanitized_https(obj), None
    if obj.get("type") == "https-progress":
        row = sanitized_progress(obj)
        if row is not None and len(progress) < 256 and (not progress or row != progress[-1]):
            progress.append(row)
        return None, None
    if obj.get("type") == "status":
        row = sanitized_status(obj)
        if row is not None:
            if len(statuses) < 256 and (not statuses or row != statuses[-1]):
                statuses.append(row)
            if row.get("errorCode") in {"registration-required", "zero-noise-key"} or row["status"] in {"needs-login", "needs-approval"}:
                return None, row.get("errorCode", "registration-required")
    return None, None


def successful_https_exit(returncode, final):
    return bool(returncode == 0 and final is not None and final["success"] and final["tlsVerified"]
                and final["httpStatus"] is not None and final["errorCode"] is None)


def _online(state, hostname, timeout):
    if not preflight():
        return {"ok": False, "errorCode": "preflight-failed", "statuses": [], "progress": []}
    statuses, progress = [], []
    final = None
    proc = None
    selector = None
    try:
        proc = subprocess.Popen([str(BINARY), "--require-registered", "--verify-https", "--state-dir", str(state),
                                 "--hostname", hostname], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                stderr=subprocess.DEVNULL, start_new_session=True, env=child_environment())
        selector = selectors.DefaultSelector()
        selector.register(proc.stdout, selectors.EVENT_READ)
        deadline = time.monotonic() + timeout
        pending = b""
        while time.monotonic() < deadline:
            if not selector.get_map():
                try:
                    proc.wait(timeout=min(1, max(0.001, deadline - time.monotonic())))
                except subprocess.TimeoutExpired:
                    continue
                break
            events = selector.select(min(1, max(0, deadline - time.monotonic())))
            if not events and proc.poll() is not None:
                break
            for key, _ in events:
                chunk = os.read(key.fd, 4096)
                if not chunk:
                    selector.unregister(proc.stdout)
                    break
                pending += chunk
                if len(pending) > 65536:
                    return {"ok": False, "errorCode": "output-limit", "statuses": statuses, "progress": progress}
                while b"\n" in pending:
                    line, pending = pending.split(b"\n", 1)
                    parsed, terminal = consume_child_json(line, statuses, progress)
                    if parsed is not None:
                        final = parsed
                    if terminal is not None:
                        return {"ok": False, "errorCode": terminal, "statuses": statuses, "progress": progress,
                                "verification": final}
            if proc.poll() is not None and not events:
                break

        if pending.strip():
            parsed, terminal = consume_child_json(pending, statuses, progress)
            if parsed is not None:
                final = parsed
            if terminal is not None:
                return {"ok": False, "errorCode": terminal, "statuses": statuses, "progress": progress,
                        "verification": final}

        if proc.poll() is None:
            return {"ok": False, "errorCode": "deadline", "statuses": statuses, "progress": progress,
                    "verification": final}
        ok = successful_https_exit(proc.returncode, final)
        return {"ok": ok, "exitCode": proc.returncode, "statuses": statuses, "progress": progress,
                "verification": final}
    except (OSError, ValueError, TypeError):
        return {"ok": False, "errorCode": "process-failed", "statuses": statuses, "progress": progress}
    finally:
        if selector is not None:
            selector.close()
        if proc is not None:
            cleaned = stop_process(proc)
            for stream in (proc.stdin, proc.stdout):
                if stream is not None:
                    stream.close()
            if not cleaned:
                raise CleanupFailure(statuses, progress)

def main():
    parser = SafeArgumentParser(description=__doc__)
    parser.add_argument("--https", action="store_true")
    parser.add_argument("--state-dir")
    parser.add_argument("--hostname")
    parser.add_argument("--timeout", type=int, default=180)
    args = parser.parse_args()
    if not 30 <= args.timeout <= 300:
        parser.error("timeout must be between 30 and 300 seconds")
    if args.https:
        if not args.state_dir or not args.hostname:
            parser.error("--https requires --state-dir and --hostname")
        state = Path(args.state_dir)
        if not state.is_absolute() or state.name != "tailscale-state" or not (state / "tailscaled.state").is_file():
            parser.error("existing absolute tailscale-state directory and tailscaled.state required")
    elif args.state_dir or args.hostname:
        parser.error("--state-dir and --hostname require --https")
    os.umask(0o077)
    base = Path.home() / ".local/state/pocketrisu-exp-tsnet-cert2-probe"
    base.mkdir(parents=True, exist_ok=True)
    folder = Path(tempfile.mkdtemp(prefix="run-", dir=base))
    results = {"mode": "https" if args.https else "offline", "ok": False}
    try:
        identity = identity_and_binary()
        results["binaryVerified"] = True
        never = folder / "never-created/tailscale-state"
        cases = [("diagnostic", ["--diagnose-argv"], 0, "argv-diagnostic"),
                 ("help", ["--help"], 0, None), ("unknown-flag", ["--definitely-invalid-tsnet-flag"], 2, None),
                 ("invalid-state", ["--state-dir", "relative/tailscale-state"], 1, None),
                 ("special-safe-rejection", ["--exec-cloudflared", "/dev/null"], 2, None),
                 ("parse-only", ["--check-args", "--state-dir", str(never), "--hostname", "pocketrisu-exp-termux-v1045"], 0, "args-check")]
        results["probes"] = {name: single_probe(command, code, kind) for name, command, code, kind in cases}
        results["probes"]["build-info"] = build_info(identity)
        results["stateNotCreated"] = not never.exists()
        offline_ok = results["stateNotCreated"] and all(row["ok"] for row in results["probes"].values())
        if args.https and offline_ok:
            results["https"] = online(state, args.hostname, args.timeout)
        results["ok"] = bool(offline_ok and (not args.https or results.get("https", {}).get("ok")))
    except (OSError, ValueError, KeyError, TypeError):
        results["errorCode"] = "provenance-or-binary-invalid"
    (folder / "results.json").write_text(json.dumps(results, indent=2) + "\n", encoding="utf-8")
    (base / "LAST_RUN").write_text(folder.name + "\n", encoding="utf-8")
    print(json.dumps(results, indent=2))
    print("Verification passed." if results["ok"] else "Verification failed; see results.json in the private probe folder.")
    return 0 if results["ok"] else 1


if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        print("Verification interrupted; owned candidate stopped.", file=sys.stderr)
        sys.exit(130)
    except OSError:
        print("Verification failed: local file or process error.", file=sys.stderr)
        sys.exit(1)
