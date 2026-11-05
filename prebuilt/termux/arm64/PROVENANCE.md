# Existing APK tsnet binary: provenance and notices

This is an unmodified copy of the existing PocketRisu-Android executable,
prepared for an independent Termux execution check. It is not a Termux success
report, a new build, or a Tailscale version upgrade.

- Android source snapshot: `a67a75217f94c0d756f4066197e9b0a458bdebe6`.
- Source path: `prebuilt/jniLibs/arm64-v8a/libpocketrisu_tsnet_exec.so`.
- Binary bytes: 21,758,246.
- SHA-256: `c9b5c75f4c9792b79b1f840986582289d2ff1ad822e88fc2206b1a551fe5a71b`.
- Binary's embedded build information: Go 1.27.3, Tailscale 1.104.4 replaced by
  `./third_party/tailscale`, Android/ARM64, CGO disabled, trimpath enabled.
- The original source pins and binary hash agree with the Android build
  manifest. The source contains the Android ACME certificate API patch.
- These metadata and hash checks do not prove bit-for-bit rebuild reproducibility.

## Publicly supplied source and license material

Download `tsnet-v1.104.4-source.tar.gz` next to this file for the accompanying source.
The private Android repository is not required to retrieve this source archive.

- Archive bytes: 81,151,418.
- Archive SHA-256: `f5a17e6896c1ba42f21f860a09614c29a21b48a9e2878eeee6fd42ccd0bdece7`.
- Includes the sidecar source/tests, exact local Tailscale replacement source,
  patch and build records, and exact sources of the 27 externally versioned Go
  modules listed by the executable's embedded build information.
- Includes original copyright/license/notice/patent files in their source
  directories. The archive's `SOURCE_FILES.json` records sizes and SHA-256.
- Its README records the original standalone Go build command and its limits.
  No rebuild was performed for this distribution.

The PocketRisu-Android application code is distributed under the repository's
GPLv3 terms; its original license is preserved in
`LICENSES/PocketRisu-Android-GPL-3.0.txt`. Tailscale and its dependencies retain
their original licenses. Tailscale's BSD-3-Clause text is preserved in
`LICENSES/Tailscale-BSD-3-Clause.txt`. `LICENSES/go-modules/` preserves the
module-specific license/notice/patent files, and `LICENSES/go1.27.3/` preserves
the Go toolchain and its vendored standard-library notices. No license is being
replaced with the surrounding repository's license.

Android source changes relevant to this executable include enabling the local
ACME certificate API for Android and selecting the bundled patched Tailscale
source. The sidecar also implements the local-client proxy path used in the
APK. The included source snapshot preserves those changes; this test branch
does not change them.

## Static review and excluded data

The original and copied binary bytes are identical. ELF inspection reports
ELF64 little-endian AArch64, ET_DYN/PIE, `/system/bin/linker64`, no `DT_NEEDED`
entries, and 65,536-byte load alignment. `BUILDINFO.txt` comes from the binary's
embedded metadata, without executing the binary.

The static review found no matches for the checked private-key/token/device-key
or private-host-path patterns in the executable. Pattern checks cannot prove
the absence of every possible secret. Source and license files were selected
from tracked component sources and exact public module-cache sources; no APK
application data, Tailscale state directory, authentication output, account
database, signing key or runtime log was copied. Public upstream test fixtures
inside dependency source retain their original provenance and are not APK
device credentials.

See [the Termux smoke guide](../../../docs/ko/tsnet-termux-smoke.md)
for the intended commands. Only local static
inspection was performed: no Android/Termux executable invocation, login,
device registration, Tailnet connection, HTTPS or streaming test was run.
