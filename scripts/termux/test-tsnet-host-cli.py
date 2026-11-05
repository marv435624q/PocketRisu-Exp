from pathlib import Path
import hashlib
import json
import os
import stat
import subprocess

import argparse
parser = argparse.ArgumentParser(description='Linux offline CLI regression checks; use an external-network-free namespace.')
parser.add_argument('--binary', type=Path, required=True)
parser.add_argument('--output', type=Path, required=True)
parser.add_argument('--identity', type=Path, required=True, help='build-flags.json from the matching host build')
options = parser.parse_args()
HOST = options.binary.resolve()
OUT = options.output.resolve()
OUT.mkdir(parents=True, exist_ok=False)
state = OUT / 'never-created/tailscale-state'
existing = OUT / 'preserved/tailscale-state'
existing.mkdir(parents=True)
(existing / 'sentinel.txt').write_text('synthetic data that must remain unchanged\n')
before = hashlib.sha256((existing / 'sentinel.txt').read_bytes()).hexdigest()
before_mode = stat.S_IMODE(existing.stat().st_mode)
results = []
def run(name, args, expected, kind=None, env=None, private=False):
    result = subprocess.run([str(HOST), *args], env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=10)
    (OUT / (name + '.stdout.txt')).write_bytes(result.stdout)
    (OUT / (name + '.stderr.txt')).write_bytes(result.stderr)
    assert result.returncode == expected, (name, result.returncode)
    assert b'"status":"starting"' not in result.stdout
    if private:
        assert b'private-value-sentinel' not in result.stdout + result.stderr
    parsed = json.loads(result.stdout) if kind else None
    if kind:
        assert parsed['type'] == kind, name
    assert not state.exists(), 'Offline check created state'
    results.append({'case': name, 'exitCode': result.returncode, 'expectedExitCode': expected,
                    'outputType': kind, 'startingSeen': False, 'stateCreated': False})
    return parsed

run('help', ['--help'], 0)
assert b'state-dir' in (OUT / 'help.stderr.txt').read_bytes()
run('unknown-flag', ['--definitely-invalid-tsnet-flag'], 2)
run('relative-state-dir', ['--state-dir', 'relative/tailscale-state'], 1, 'status')
run('wrong-final-name', ['--state-dir', str(OUT / 'wrong-final-name')], 1, 'status')
run('special-exec-safe-rejection', ['--exec-cloudflared', '/dev/null'], 2)
parsed = run('check-valid-state', ['--check-args', '--state-dir', str(state), '--hostname', 'private-value-sentinel'], 0, 'args-check', private=True)
assert parsed['parsed'] and parsed['stateDirValid'] and not parsed['clientMode'] and not parsed['logoutRequested']
parsed = run('check-client-logout-preserves-state', ['--check-args', '--state-dir', str(existing), '--client-target', 'example.invalid:443', '--logout-only'], 0, 'args-check')
assert parsed['clientMode'] and parsed['logoutRequested']
assert existing.exists() and hashlib.sha256((existing / 'sentinel.txt').read_bytes()).hexdigest() == before
assert stat.S_IMODE(existing.stat().st_mode) == before_mode
run('private-malformed-bool', ['--check-args', '--logout-only=private-value-sentinel'], 2, private=True)
run('private-unknown-flag', ['--private-value-sentinel'], 2, private=True)
run('private-positional', ['private-value-sentinel'], 2, private=True)
run('help-after-private-value', ['--hostname', 'private-value-sentinel', '--help'], 0, private=True)
diag = run('diagnose-direct', ['--diagnose-argv'], 0, 'argv-diagnostic')
assert not diag['android'] and not diag['removedExecutableArg']
env = os.environ.copy()
env['TERMUX_EXEC__PROC_SELF_EXE'] = str(HOST)
diag = run('diagnose-injected-shape-host', [str(HOST), '--diagnose-argv'], 0, 'argv-diagnostic', env=env)
assert diag['rawArgc'] == 3 and diag['firstArgMatchesMarker'] and not diag['removedExecutableArg']
assert str(HOST).encode() not in (OUT / 'diagnose-injected-shape-host.stdout.txt').read_bytes()
run('diagnostic-word-is-hostname-value', ['--state-dir', str(state), '--hostname', '--diagnose-argv', '--check-args'], 0, 'args-check')
identity = json.loads(options.identity.read_text())['identity']
info = run('stamped-build-info', ['--build-info'], 0, 'build-info')
assert info['baseVersion'] == info['shortVersion'] == '1.104.5'
assert info['sourceId'] == identity['sourceId'] and info['longVersion'] == identity['longVersion']
assert info['custom'] and info['goVersion'] == 'go1.27.3'
assert info['goos'] == 'linux' and info['goarch'] == 'amd64'
run('build-info-word-is-hostname-value', ['--state-dir', str(state), '--hostname', '--build-info', '--check-args'], 0, 'args-check')
run('guarded-https-parse-only', ['--state-dir', str(state), '--require-registered', '--verify-https', '--check-args'], 0, 'args-check')
run('unguarded-https-refused', ['--state-dir', str(state), '--verify-https'], 2)
run('guarded-https-missing-state', ['--state-dir', str(state), '--require-registered', '--verify-https'], 1, 'status')
assert len(results) == 19
record = {'cases': results, 'stateCreated': False, 'existingSyntheticStateUntouched': True,
          'scope': 'Linux host program in an external-network-free namespace; shared parsing and dispatch only. Not Android/Termux execution.'}
(OUT / 'results.json').write_text(json.dumps(record, indent=2) + '\n')
print(json.dumps(record, indent=2))
