# PocketRisu tsnet sidecar — third-party notices

This is the Android ARM64 sidecar from PocketRisu-Android commit `a67a75217f94c0d756f4066197e9b0a458bdebe6`, built with Go 1.27.3 and based on Tailscale v1.104.4. The bundled Tailscale source contains local changes to two localapi build constraints for the Android certificate API. See the accompanying source and the original license files.

The PocketRisu-Android repository provides GPL version 3 in its top-level `LICENSE`. The sidecar source files do not contain separate file-level license declarations. The distribution includes that license and the corresponding source alongside this executable.

Tailscale's BSD-3-Clause copyright notice, conditions and disclaimer are provided in the original Tailscale `LICENSE`. Go's `LICENSE` and `PATENTS`, the license and patent notices of the Go standard library's vendored packages, and the original license and patent notice files from the modules recorded in the executable's Go build information are included in the accompanying license materials. Some modules contain additional nested licenses, preserved at their original relative paths. Module versions and original file paths are recorded in the accompanying index.

The exact module and Go toolchain versions are build metadata, not proof of a reproducible rebuild. The Tailscale Android client's upstream dependency list is not the dependency list of this sidecar.
