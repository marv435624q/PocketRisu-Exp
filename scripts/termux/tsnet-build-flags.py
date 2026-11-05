#!/usr/bin/env python3
"""Identify candidate build inputs and print the Tailscale linker flags."""
import argparse
import hashlib
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SOURCE = ROOT / "native/tsnet"
VENDOR = SOURCE / "third_party/tailscale"


def build_flags():
    candidate = json.loads((SOURCE / "CANDIDATE.json").read_text(encoding="utf-8"))
    if (candidate.get("candidate"), candidate.get("tailscaleVersion"), candidate.get("goVersion")) != (
        "tsnet-v1.104.5-cert1", "1.104.5", "1.27.3"
    ):
        raise ValueError("unexpected candidate configuration")
    if not VENDOR.is_dir() or VENDOR.is_symlink() or VENDOR.parent.is_symlink():
        raise ValueError("patched Tailscale source is missing; prepare it first")
    if (VENDOR / "VERSION.txt").read_text(encoding="utf-8").strip() != candidate["tailscaleVersion"]:
        raise ValueError("Tailscale source version does not match candidate")
    files = [p for p in SOURCE.glob("*.go") if not p.name.endswith("_test.go")]
    if SOURCE / "main.go" not in files:
        raise ValueError("candidate main source is missing")
    files += [SOURCE / "go.mod", SOURCE / "go.sum", SOURCE / "CANDIDATE.json",
              ROOT / "scripts/termux/tsnet-build-flags.py", ROOT / "scripts/termux/build-tsnet-candidate.sh"]
    for path in VENDOR.rglob("*"):
        if path.is_symlink():
            raise ValueError("source links are not supported")
        if path.is_file():
            files.append(path)
    inputs = {}
    for path in files:
        if path.is_symlink() or not path.is_file():
            raise ValueError("required source file missing or linked: " + path.relative_to(ROOT).as_posix())
        inputs[path.relative_to(ROOT).as_posix()] = hashlib.sha256(path.read_bytes()).hexdigest()
    # Canonical UTF-8 JSON object: sorted relative paths, no extra spaces or newline.
    encoded = json.dumps(inputs, sort_keys=True, separators=(",", ":"), ensure_ascii=True).encode("utf-8")
    source_id = hashlib.sha256(encoded).hexdigest()
    long_version = "1.104.5-pocketrisu-exp-cert1-s" + source_id[:12]
    flags = ("-s -w -X tailscale.com/version.shortStamp=1.104.5 "
             "-X tailscale.com/version.longStamp=" + long_version + " -X main.buildSourceID=" + source_id)
    return {"identity": {"candidate": candidate["candidate"], "baseVersion": candidate["tailscaleVersion"],
                         "sourceId": source_id, "longVersion": long_version},
            "inputs": dict(sorted(inputs.items())), "ldflags": flags}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()
    result = build_flags()
    print(json.dumps(result, indent=2, ensure_ascii=True) if args.json else result["ldflags"])


if __name__ == "__main__":
    main()
