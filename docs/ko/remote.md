<p align="center">
  <a href="../en/remote.md">English</a> | <strong>한국어</strong> | <a href="../de/remote.md">Deutsch</a> | <a href="../cn/remote.md">简体中文</a> | <a href="../es/remote.md">Español</a> | <a href="../vi/remote.md">Tiếng Việt</a> | <a href="../zh-Hant/remote.md">繁體中文</a>
</p>

# 원격 접속 가이드

PocketRisu 서버에 다른 기기(스마트폰, 태블릿, 다른 PC 등)에서 접속하는 방법은 두 가지입니다. Quick Tunnel은 PC 서버뿐 아니라 Termux에서 실행하는 Android 서버에서도 사용할 수 있도록 구성되어 있습니다.

- [1. Quick Tunnel](#1-quick-tunnel) — PocketRisu 내장 기능. 별도 앱 불필요. 서버 재시작 시 URL 변경.
- [2. Tailscale](#2-tailscale) — 사설 네트워크(VPN) 기반. URL 고정.


## 1. Quick Tunnel

PocketRisu가 임시 원격 접속 URL을 발급하는 내장 기능입니다.

1. PocketRisu 서버가 실행 중인 기기에서 **설정 → 원격 접속 → 원격 접속 열기** 클릭
2. 표시되는 QR 코드를 카메라로 스캔하거나, URL을 다른 기기 브라우저에 직접 입력
3. 공유를 끝낼 때 **원격 접속 닫기** 클릭

Android/Termux에서는 먼저 `pkg install -y cloudflared`를 실행해야 합니다. [Termux 설치 및 Quick Tunnel 가이드](termux.md)를 참고하세요. 별도 Termux 세션에서 터널을 수동 실행하는 방식이 아닙니다.

> Quick Tunnel URL은 외부에서 접근할 수 있는 공개 주소이므로 공유에 주의하세요. 터널을 새로 열거나 서버를 재시작하면 URL이 변경될 수 있습니다. 고정 URL이 필요하다면 [Tailscale](#2-tailscale)을 검토하세요.


---

## 2. Tailscale

같은 계정으로 묶인 기기끼리만 접근 가능한 사설 네트워크(VPN)를 구성해 접속합니다. 서버 재시작 후에도 URL이 유지됩니다.

### 1단계: Tailscale 설치

- PC: [tailscale.com](https://tailscale.com/)
- 스마트폰: App Store / Google Play 에서 "Tailscale" 검색
- 다른 PC: 동일하게 [tailscale.com](https://tailscale.com/)

### 2단계: 같은 계정으로 로그인

PC와 접속할 모든 기기에서 동일한 계정(Google, Microsoft 등)으로 Tailscale 앱에 로그인합니다.

### 3단계: PC에서 HTTPS 공유 활성화

PocketRisu가 실행 중인 PC의 터미널에서 한 번만 실행:

```bash
tailscale serve --bg http://localhost:6001
```

### 4단계: 다른 기기에서 접속

브라우저에서 아래 형식의 주소로 접속:

```
https://내PC이름.tail어쩌구.ts.net
```

정확한 주소는 Tailscale 앱의 기기 목록에서 PC 이름으로 확인할 수 있습니다. 브라우저 즐겨찾기에 등록해두면 다음부터 PC에서 서버만 실행하면 바로 접속 가능합니다.


---

← [README로 돌아가기](../../i18n/README.ko.md)
