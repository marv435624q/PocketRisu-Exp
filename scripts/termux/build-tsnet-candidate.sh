#!/usr/bin/env bash
# Developer build only: output is isolated and never replaces a prebuilt or device state.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SOURCE="$ROOT/native/tsnet"
GO="${POCKETRISU_GO_BIN:-$(command -v go || true)}"
[[ -x "$GO" ]] || { echo 'Go 1.27.3 is required'; exit 2; }
[[ "$("$GO" version)" == 'go version go1.27.3 '* ]] || { echo 'Pinned compiler is Go 1.27.3'; exit 2; }
python3 "$ROOT/scripts/termux/prepare-tsnet-source.py"
OUTPUT="$SOURCE/output/tsnet-v1.104.5-cert5-$(date -u +%Y%m%dT%H%M%SZ)-$RANDOM"
mkdir -p "$SOURCE/output"
mkdir "$OUTPUT"
mkdir "$OUTPUT/go-tmp"
export GOTOOLCHAIN=local GOWORK=off GOFLAGS='' GOTMPDIR="$OUTPUT/go-tmp"
python3 "$ROOT/scripts/termux/tsnet-build-flags.py" --json > "$OUTPUT/build-flags.json"
LDFLAGS="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1], encoding="utf-8"))["ldflags"])' "$OUTPUT/build-flags.json")"
cd "$SOURCE"
CGO_ENABLED=0 GOOS=android GOARCH=arm64 "$GO" build \
  -buildvcs=false -mod=readonly -trimpath -ldflags="$LDFLAGS" \
  -o "$OUTPUT/libpocketrisu_tsnet_exec.so" . > "$OUTPUT/build.log" 2>&1
python3 "$ROOT/scripts/termux/tsnet-build-flags.py" --json > "$OUTPUT/build-flags-after.json"
cmp -s "$OUTPUT/build-flags.json" "$OUTPUT/build-flags-after.json" || { echo 'Build inputs changed during build' >&2; exit 1; }
"$GO" version -m "$OUTPUT/libpocketrisu_tsnet_exec.so" > "$OUTPUT/buildinfo.txt"
sha256sum "$OUTPUT/libpocketrisu_tsnet_exec.so" > "$OUTPUT/SHA256SUMS"
printf 'Built without executing the candidate: %s\n' "$OUTPUT"
cat "$OUTPUT/SHA256SUMS"
