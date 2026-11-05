# Termux tsnet cert2 diagnostic source patch — 2026-11-05

Base: `1aac88da628f8da7f54d69fb071fa3904f88abb6` (`fix/termux-tsnet-cert-version-20261105`).

The cert1 real Termux run reached `Running` with `registered:true`, then failed HTTPS preparation with `deadline`. The verifier had `verification:null`, so the failure happened before the self-HTTPS request, but cert1 collapsed certificate acquisition and listener preparation into one status error.

This patch adds a privacy-safe `httpsStage` enum to the existing status stream. It never records the certificate domain, Tailnet name, URL, key material, state contents, or raw error text in verifier results.

Allowed stages:
- `prerequisites`: MagicDNS/certificate-domain prerequisites were not available.
- `certificate`: `LocalClient.CertPair` failed, including its existing 30-second timeout.
- `post-certificate`: the parent context was canceled after the certificate call succeeded but before `ListenTLS`.
- `listener`: `ListenTLS("tcp", ":443")` failed.

The normal success path is unchanged and emits no stage. The existing verifier still stores only allowlisted status fields and coarse error classes.

This commit is source/test instrumentation only. It does not contain a rebuilt ARM64 candidate, does not replace cert1, does not touch device state, does not run GitHub Actions, and does not claim real HTTPS success.
