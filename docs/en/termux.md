<p align="center">
  <strong>English</strong> | <a href="../ko/termux.md">한국어</a> | <a href="../de/termux.md">Deutsch</a> | <a href="../cn/termux.md">简体中文</a> | <a href="../es/termux.md">Español</a> | <a href="../vi/termux.md">Tiếng Việt</a> | <a href="../zh-Hant/termux.md">繁體中文</a>
</p>

# Termux Installation Guide

This guide explains how to build and run PocketRisu directly on an Android phone using Termux. Open `http://localhost:6001` on the phone or use Quick Tunnel to connect from devices on other networks.

- [1. Prerequisites](#1-prerequisites) — Termux and system requirements
- [2. Install and Build](#2-install-and-build) — single-command setup
- [3. Run and Connect](#3-run-and-connect) — open from the phone's browser
- [4. Quick Tunnel Remote Access](#4-quick-tunnel-remote-access) — open and close from the existing UI
- [5. Keep It Running](#5-keep-it-running) — survive screen-off
- [6. Update](#6-update) — pull the latest version
- [7. Limitations](#7-limitations) — considerations for Termux


## System Requirements

| Item              | Minimum         | Recommended                |
| ----------------- | --------------- | -------------------------- |
| **Android**       | 7.0 (API 24)    | 10 or newer                |
| **CPU**           | ARM64           | ARM64                      |
| **RAM**           | 2 GB            | 4 GB or more               |
| **Free storage** | 2 GB            | 4 GB or more (with build)  |

PocketRisu does not ship pre-compiled binaries for Termux, so the phone builds everything itself. Including native modules such as `better-sqlite3`, the build takes roughly **10 to 40 minutes** depending on phone performance.


---

## 1. Prerequisites

### Install Termux

Install Termux from one of:

- **F-Droid (recommended)**: https://f-droid.org/packages/com.termux/
- **GitHub Releases**: https://github.com/termux/termux-app/releases
- **Google Play**: https://play.google.com/store/apps/details?id=com.termux&hl=en&pli=1


---

## 2. Install and Build

Open Termux and run this single command:

```bash
pkg install -y git && \
  git clone https://github.com/marv435624q/PocketRisu-Exp.git && \
  cd PocketRisu-Exp && \
  bash scripts/termux/build.sh
```

The command takes care of:

1. Installing `git`
2. Cloning the private PocketRisu-Exp repository (GitHub authentication required)
3. Installing build dependencies and `cloudflared` (`nodejs-lts`, `python`, `make`, `clang`, `pnpm`, etc.)
4. `pnpm install` — JavaScript dependencies and native module compilation
5. `pnpm build` — frontend bundling

When the build finishes, you'll see:

```
Build OK. Start the server with:
  node server/node/server.cjs

Then open this address in the phone's own browser:
  http://localhost:6001
```


---

## 3. Run and Connect

From the same directory, start the server:

```bash
node server/node/server.cjs
```

Once the server boot log appears, open the phone's browser (Chrome, Firefox, etc.) and navigate to:

```
http://localhost:6001
```

The PocketRisu UI should load. `localhost` is automatically treated as a secure context by the browser, so clipboard, `crypto.subtle`, and other secure-context APIs work normally.

Stop the server with `Ctrl + C`.


---

## 4. Quick Tunnel Remote Access

Keep the PocketRisu server running and use the existing UI to start and stop a Cloudflare Quick Tunnel. **Do not launch a separate `cloudflared tunnel` session manually.**

1. Open `http://localhost:6001` in the phone's browser.
2. Select **Settings → Remote Access → Open Remote Access**.
3. Once started, open the displayed `https://…trycloudflare.com` URL from another PC, tablet, or phone, or scan its QR code.
4. To stop sharing, select **Close Remote Access** on the same screen.

The build script installs Termux's packaged `cloudflared`. For an existing installation, run `pkg install -y cloudflared` once if it is missing, then restart the PocketRisu server. The backend uses `$PREFIX/bin/cloudflared`, never an old Linux binary in the project's `bin/` folder. If the package is missing, it displays an installation error instead of downloading a Linux binary.

> Quick Tunnel provides a temporary public HTTPS URL. Anyone with that address can attempt to connect; do not share the URL or authentication credentials publicly. A new URL may be issued when the server or tunnel restarts.

---

## 5. Keep It Running

To keep the server alive when the screen turns off or you switch to other apps, enable Termux's wake lock:

```bash
termux-wake-lock
```

The Termux notification will show the wake-lock indicator, and the server continues to run even with the screen off.

Release with: `termux-wake-unlock`

> Android may still kill Termux under memory pressure. For long-running use, exclude Termux from your phone's battery optimization settings.


---

## 6. Update

Stop the running server with `Ctrl + C`, then run:

```bash
cd ~/PocketRisu-Exp && \
  git pull --ff-only && \
  GYP_DEFINES="android_ndk_path=''" pnpm install --frozen-lockfile && \
  NODE_OPTIONS="--max-old-space-size=2048" pnpm build
```

Each step runs only if the previous one succeeded, so the command stops at the first error instead of rebuilding the old version without updated sources. When the build completes, start the server again with `node server/node/server.cjs`.

- `pnpm install` applies dependency changes in the new version. Run it on every update; it finishes right away when nothing changed.
- `GYP_DEFINES` is the same Termux setting the build script uses. It is needed when native modules such as `better-sqlite3` are recompiled.

### If `git pull` fails

If `git pull` stops with `Your local changes to the following files would be overwritten by merge` or `Not possible to fast-forward`, reset the source to the latest version with the following command, then run the update command above again:

```bash
cd ~/PocketRisu-Exp && git fetch origin && git reset --hard origin/main
```

The `save/` and `backups/` folders are not tracked by git, so chats, characters, and settings stay as they are. Any edits you made to source files (a changed port, dependencies changed with `pnpm add`, and so on) are discarded.


---

## 7. Limitations

Quick Tunnel requires the Termux `cloudflared` package. Android background restrictions and changing networks may interrupt an active connection. The standalone Termux package was tested separately; the full PocketRisu UI flow still needs device verification.


---

← [Back to README](../../README.md)
