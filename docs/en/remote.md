<p align="center">
  <strong>English</strong> | <a href="../ko/remote.md">한국어</a> | <a href="../de/remote.md">Deutsch</a> | <a href="../cn/remote.md">简体中文</a> | <a href="../es/remote.md">Español</a> | <a href="../vi/remote.md">Tiếng Việt</a> | <a href="../zh-Hant/remote.md">繁體中文</a>
</p>

# Remote Access Guide

There are two ways to access a running PocketRisu server from another device (smartphone, tablet, another PC). Quick Tunnel also supports a server running on Android through Termux.

- [1. Quick Tunnel](#1-quick-tunnel) — Built into PocketRisu. No additional app required. URL changes on server restart.
- [2. Tailscale](#2-tailscale) — Based on a private network (VPN). Persistent URL.


## 1. Quick Tunnel

A built-in feature of PocketRisu that issues a temporary remote access URL.

1. On the device running PocketRisu, select **Settings → Remote Access → Open Remote Access**.
2. Scan the displayed QR code or open the URL from another device's browser.
3. Select **Close Remote Access** to end the session.

For Android/Termux, install `cloudflared` first using `pkg install -y cloudflared`; see the [Termux Quick Tunnel guide](termux.md). You do not need a separate manual tunnel session.

> Quick Tunnel provides a public address, so take care when sharing it. The URL can change when the tunnel or server restarts. For a persistent URL, consider [Tailscale](#2-tailscale).


---

## 2. Tailscale

Builds a private network (VPN) accessible only from devices logged in with the same account. The URL persists across server restarts.

### Step 1: Install Tailscale

- PC: [tailscale.com](https://tailscale.com/)
- Smartphone: Search for "Tailscale" in App Store / Google Play
- Other PC: Same — [tailscale.com](https://tailscale.com/)

### Step 2: Sign in with the same account

Sign into the Tailscale app on the PC and on every device you'll access from, using the same account (Google, Microsoft, etc.).

### Step 3: Enable HTTPS sharing on the PC

In the terminal on the PC running PocketRisu, run once:

```bash
tailscale serve --bg http://localhost:6001
```

### Step 4: Access from another device

Open in a browser using this URL format:

```
https://your-pc-name.tail-something.ts.net
```

Find the exact address from your PC's entry in the Tailscale app's device list. Bookmark it in the browser, and from then on you only need to start the server on the PC to access from any device.


---

← [Back to README](../../README.md)
