#!/usr/bin/env bash
# Offline parser probes only. Never invoke the normal Tailscale start mode.
set -u
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DIR="$ROOT/prebuilt/termux/arm64/tsnet-v1.104.5-argv1"
BIN="$DIR/libpocketrisu_tsnet_exec.so"
[[ -x "$BIN" ]] || { echo 'Candidate is missing or not executable'; exit 2; }
read -r EXPECTED _ < "$DIR/SHA256SUMS"
ACTUAL="$(sha256sum < "$BIN")"
ACTUAL="${ACTUAL%% *}"
[[ "$EXPECTED" =~ ^[0-9a-f]{64}$ && "$ACTUAL" == "$EXPECTED" ]] || { echo 'Candidate hash mismatch'; exit 2; }
umask 077
BASE="${XDG_STATE_HOME:-$HOME/.local/state}/pocketrisu-exp-tsnet-argv-probe"
mkdir -p "$BASE" || exit 2
OUT="$(mktemp -d "$BASE/run.XXXXXXXX")" || exit 2
STATE="$OUT/never-created/tailscale-state"
FAILED=0
probe() {
  local name="$1" expected="$2" code
  shift 2
  "$BIN" "$@" > "$OUT/$name.stdout.txt" 2> "$OUT/$name.stderr.txt"
  code=$?
  printf '%s exit=%s expected=%s\n' "$name" "$code" "$expected" | tee -a "$OUT/results.txt"
  if [[ "$code" != "$expected" ]]; then FAILED=1; fi
}
probe diagnostic 0 --diagnose-argv
probe help 0 --help
probe unknown-flag 2 --definitely-invalid-tsnet-flag
probe invalid-state 1 --state-dir relative/tailscale-state
probe special-safe-rejection 2 --exec-cloudflared /dev/null
probe parse-only 0 --check-args --state-dir "$STATE" --hostname pocketrisu-exp-termux-v1045
if ! grep -q '"type":"args-check"' "$OUT/parse-only.stdout.txt"; then FAILED=1; fi
if [[ -e "$STATE" ]]; then
  printf 'Unexpected state creation; stop here and retain the evidence\n' | tee -a "$OUT/results.txt"
  FAILED=1
fi
printf '%s\n' "$OUT" > "$BASE/LAST_RUN"
printf 'Offline probes completed. Logs are in the LAST_RUN directory under pocketrisu-exp-tsnet-argv-probe.\n'
printf 'No login or normal start command was issued. Keep the logs; do not delete existing state.\n'
exit "$FAILED"
