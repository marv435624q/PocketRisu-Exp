<p align="center">
  <a href="../en/termux.md">English</a> | <strong>한국어</strong> | <a href="../de/termux.md">Deutsch</a> | <a href="../cn/termux.md">简体中文</a> | <a href="../es/termux.md">Español</a> | <a href="../vi/termux.md">Tiếng Việt</a> | <a href="../zh-Hant/termux.md">繁體中文</a>
</p>

# Termux 설치 가이드

안드로이드 폰에서 Termux를 통해 PocketRisu를 직접 빌드해 사용하는 방법을 안내합니다. 같은 폰의 브라우저에서 `http://localhost:6001`로 접속하거나, Quick Tunnel을 열어 다른 네트워크의 기기에서도 접속할 수 있습니다.

- [1. 사전 준비](#1-사전-준비) — Termux + 시스템 요구사항
- [2. 설치 및 빌드](#2-설치-및-빌드) — 한 줄 명령으로 자동 진행
- [3. 실행 및 접속](#3-실행-및-접속) — 폰 브라우저로 사용
- [4. Quick Tunnel 원격 접속](#4-quick-tunnel-원격-접속) — 기존 UI에서 열기·닫기
- [5. 백그라운드 유지](#5-백그라운드-유지) — 화면 꺼진 상태에서도 서버 유지
- [6. 업데이트](#6-업데이트) — 최신 버전 받기
- [7. 제한사항](#7-제한사항) — 사용 시 주의할 점


## 시스템 요구사항

| 항목         | 최소           | 권장                          |
| ------------ | -------------- | ----------------------------- |
| **Android**  | 7.0 (API 24)   | 10 이상                       |
| **CPU 아키텍처** | ARM64          | ARM64                         |
| **RAM**      | 2GB            | 4GB 이상                      |
| **여유 공간**  | 2GB            | 4GB 이상 (빌드 결과물 포함)   |

PocketRisu는 Termux에서 미리 컴파일된 바이너리를 제공하지 않으며, 폰에서 직접 빌드합니다. `better-sqlite3` 같은 네이티브 모듈을 포함해 빌드 시간은 폰 성능에 따라 **약 10~40분** 소요됩니다.


---

## 1. 사전 준비

### Termux 설치

다음 중 한 곳에서 Termux를 받습니다.

- **F-Droid (권장)**: https://f-droid.org/packages/com.termux/
- **GitHub Releases**: https://github.com/termux/termux-app/releases
- **Google Play**: https://play.google.com/store/apps/details?id=com.termux&hl=ko&pli=1


---

## 2. 설치 및 빌드

Termux 앱을 연 후 다음 한 줄을 실행합니다.

```bash
pkg install -y git && \
  git clone https://github.com/marv435624q/PocketRisu-Exp.git && \
  cd PocketRisu-Exp && \
  bash scripts/termux/build.sh
```

이 명령이 자동으로 처리합니다.

1. `git` 설치
2. PocketRisu-Exp 소스 다운로드 (비공개 저장소이므로 GitHub 인증 필요)
3. 빌드 의존성 및 `cloudflared` 설치 (`nodejs-lts`, `python`, `make`, `clang`, `pnpm` 등)
4. `pnpm install` — JavaScript 의존성 다운로드 및 네이티브 모듈 컴파일
5. `pnpm build` — 프론트엔드 번들링

빌드가 완료되면 다음 메시지가 표시됩니다.

```
Build OK. Start the server with:
  node server/node/server.cjs

Then open this address in the phone's own browser:
  http://localhost:6001
```


---

## 3. 실행 및 접속

빌드가 끝난 디렉터리에서 서버를 실행합니다.

```bash
node server/node/server.cjs
```

서버 부팅 로그가 표시되면 폰의 브라우저(Chrome, Firefox 등)에서 다음 주소를 엽니다.

```
http://localhost:6001
```

PocketRisu UI가 표시되면 정상입니다. `localhost`는 브라우저가 자동으로 secure context로 인정하므로 클립보드, crypto.subtle 등 모든 기능이 동작합니다.

서버 종료: 터미널에서 `Ctrl + C`


---

## 4. Quick Tunnel 원격 접속

서버가 실행 중인 상태에서 PocketRisu의 기존 버튼으로 Cloudflare Quick Tunnel을 열고 닫을 수 있습니다. **별도의 Termux 세션에서 `cloudflared tunnel` 명령을 직접 실행할 필요는 없습니다.**

1. 폰의 브라우저에서 `http://localhost:6001`을 엽니다.
2. **설정 → 원격 접속 → 원격 접속 열기**를 누릅니다.
3. 시작이 완료되면 표시되는 `https://…trycloudflare.com` 주소를 다른 PC·태블릿·폰의 브라우저에 입력하거나 QR 코드를 스캔합니다.
4. 공유를 종료할 때는 같은 화면에서 **원격 접속 닫기**를 누릅니다.

빌드 스크립트는 Termux 패키지 `cloudflared`를 설치합니다. 기존 설치에서 패키지가 없다면 Termux에서 `pkg install -y cloudflared`를 실행한 뒤 **원격 접속 버튼을 다시 누르세요. 서버 재시작은 필요하지 않습니다.** 서버는 `$PREFIX/bin/cloudflared`를 사용하며 프로젝트의 `bin/cloudflared`에 남은 Linux 바이너리는 실행하지 않습니다. 패키지를 찾지 못하면 설치 안내 오류를 표시하고 자동 다운로드하지 않습니다.

> Quick Tunnel 주소는 임시 공개 HTTPS 주소입니다. 주소를 아는 사람은 서버에 접속을 시도할 수 있으므로 주소 및 인증 정보를 외부에 공개하지 마세요. 서버를 재시작하거나 터널을 다시 열면 주소가 변경될 수 있습니다.

---

## 5. 백그라운드 유지

화면이 꺼지거나 다른 앱을 사용하는 동안에도 서버를 유지하려면 슬립 방지를 활성화합니다.

```bash
termux-wake-lock
```

Termux 알림 영역에 wake lock 상태가 표시되며, 폰 화면이 꺼져도 서버 프로세스가 종료되지 않습니다.

해제: `termux-wake-unlock`

> Android가 메모리 부족 상황에서 Termux를 강제 종료할 수 있습니다. 안정적으로 항상 켜두려면 폰의 배터리 최적화 설정에서 Termux를 제외해 두는 것을 권장합니다.


---

## 6. 업데이트

실행 중인 서버를 `Ctrl + C`로 종료한 뒤 다음을 실행합니다.

```bash
cd ~/PocketRisu-Exp && \
  git pull --ff-only && \
  GYP_DEFINES="android_ndk_path=''" pnpm install --frozen-lockfile && \
  NODE_OPTIONS="--max-old-space-size=2048" pnpm build
```

각 단계는 앞 단계가 성공했을 때만 실행됩니다. 오류가 나면 그 자리에서 멈추므로, 소스가 갱신되지 않은 채 이전 버전을 다시 빌드하는 일이 없습니다. 빌드가 끝나면 `node server/node/server.cjs`로 서버를 다시 실행합니다.

- `pnpm install`은 새 버전에서 바뀐 의존성을 반영합니다. 업데이트할 때마다 실행하며, 바뀐 것이 없으면 바로 끝납니다.
- `GYP_DEFINES`는 빌드 스크립트와 같은 Termux용 설정으로, `better-sqlite3` 같은 네이티브 모듈을 다시 컴파일할 때 필요합니다.

### `git pull`이 실패하는 경우

로컬 수정이 있거나 현재 브랜치와 원격 브랜치의 이력이 달라졌을 수 있습니다. 먼저 아래 명령으로 상태를 확인하세요.

```bash
cd ~/PocketRisu-Exp
git status
git branch -vv
git fetch origin
```

이 명령은 기존 파일을 강제로 덮어쓰지 않습니다. **작업 브랜치에서 `git reset --hard origin/main`을 실행하면 개조판 수정 사항이 사라질 수 있으므로 사용하지 마세요.** 현재 브랜치와 수정 파일을 확인한 뒤 병합·재배치 또는 커밋/백업 여부를 결정해야 합니다.


---

## 7. 제한사항

Quick Tunnel은 Termux 패키지의 `cloudflared`가 필요하며, Android의 백그라운드 제한·네트워크 상태에 따라 연결이 끊길 수 있습니다. 수동으로 실행한 `cloudflared`의 동작 확인과 별개로, PocketRisu UI를 통한 전체 흐름은 실제 기기에서 검증해야 합니다.


---

← [PocketRisu README로 돌아가기](../../i18n/README.ko.md)
