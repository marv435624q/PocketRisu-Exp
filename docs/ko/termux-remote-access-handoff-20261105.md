# PocketRisu-Exp / Termux 원격 접속 작업 인계 — 2026-11-05

> **목적:** 다른 세션·에이전트가 Quick Tunnel과 Tailscale(tsnet) 작업을 혼동하지 않고, 실기기 로그에서 이어서 작업하기 위한 문서.
>
> **현황:** Quick Tunnel은 사용자 보고 기준 Termux의 기존 설정 UI에서 시작·종료 확인. tsnet은 **Android ARM64 바이너리 실행과 Go 오류 출력까지만 확인**, 명령줄 인자 처리 문제로 중단. **Tailscale 엔진 시작·로그인·등록·HTTPS 연결은 미검증**.
>
> 이 문서는 이미 진행한 내용과 후속 요구사항의 기록이다. 문서화 자체가 코드 수정·재빌드·실기기 검증을 뜻하지 않는다.

## 1. 저장소·브랜치·경계

- 저장소: https://github.com/marv435624q/PocketRisu-Exp (문서 작성 시점 public).
- 원본: https://github.com/PocketRisu/PocketRisu
- 기본 브랜치: `main`. **이 작업에서 수정·병합하지 않는다**.
- Quick Tunnel: `feat/termux-quick-tunnel-ui`, 이전 확인된 커밋 `e33ba89e48fe2deaaa8c57c6c82194f5ee27d04f`.
- Tailscale 독립 시험: `test/termux-tsnet-v1044`, 인계 작성 전 커밋 `088fa507008909f24297899e981b5992b42da334`. Quick Tunnel 브랜치에서 파생되었지만 tsnet을 Node/UI에 통합한 것은 아니다.
- 별도 Termux 시험 폴더: `$HOME/PocketRisu-Exp-tsnet-v1044`. 기존 Quick Tunnel 서버 작업 폴더와 구별한다.
- 기존 APK용 tsnet 바이너리와 PocketRisu-Android 소스·장치 데이터·인증 정보는 변경/복사/삭제 금지. 별도 분기에서 격리하여 실험할 것.
- 참고: 현재 공개 저장소인데 Quick Tunnel 브랜치 `README.md`에 “This repository is private”가 남아 있는 것은 오래된 안내다.

## 2. Quick Tunnel — 구현·테스트와 **필수 Y/N 요구**

### 확인된 구현 (GitHub 정적 소스)

- `src/lib/Setting/Pages/RemoteAccessSettings.svelte`: Android에서 설정 UI를 가리던 제한을 없애고 기존 “원격 접속 열기/닫기” 컨트롤을 유지.
- `server/node/server.cjs`: `/api/tunnel/status`, `/api/tunnel/start`, `/api/tunnel/stop`를 통해 Node 서버가 `cloudflared` 프로세스를 시작·관리·종료한다. 별도의 수동 CLI 실행을 요구하지 않는다.
- `server/node/cloudflared-utils.cjs`: Termux에서는 `$PREFIX/bin/cloudflared`를 사용하고 stderr에서 Quick Tunnel URL 추출. Termux에 패키지가 없을 때는 서버 API가 `pkg install -y cloudflared` 안내 오류를 반환하고 Linux ARM64 일반 바이너리 다운로드로 대체하지 않는다. 다른 데스크톱 경로는 보존.
- 현재 `scripts/termux/build.sh`의 설치 명령은 `pkg install -y nodejs-lts python make clang pkg-config tar curl cloudflared`로 **cloudflared를 무조건 설치**한다.

### 사용자 실기기 보고 (서로 다른 검증 범위 구분)

- Termux에서 PocketRisu 빌드, `node server/node/server.cjs` 실행, 휴대폰 `http://localhost:6001` 열기 성공.
- **기존 설정 UI에서 Quick Tunnel 열기 및 닫기 정상 동작** 확인. 앱 UI에서 Node를 통해 설치된 `cloudflared`를 관리하는 것이 구현 목표다.
- 별도 `cloudflared` CLI 시험에서 동일 네트워크 PC 및 Wi-Fi→셀룰러 전환 후에도 터널 접속된 보고가 있음. **CLI의 네트워크 시험을 UI 통합 버전의 동일 조건 검증으로 혼동하지 말 것**.
- 데스크톱 전체 회귀/장기간 운용/상위 upstream PR·병합은 완료 판정하지 않는다.

### 사용자 확정 요구: **설치 중 Quick Tunnel 의존성 Y/N 선택** (아직 미구현)

1. `scripts/termux/build.sh`에서 공통 필수 패키지 설치와 `cloudflared` 설치를 분리한다. Y/N은 사용자에게 실제 설치 시 질문한다.
2. **Y**: Termux 패키지 `cloudflared` 설치. **N**: 패키지 설치는 건너뛰되 PocketRisu 설치·빌드는 정상 진행.
3. 이미 설치된 `cloudflared`가 있다면 기존 Node 경로에서 그대로 사용. N 선택으로 UI·설정·시작/종료 기능을 삭제하거나 수동 터미널 명령으로 대체하지 않는다.
4. 미설치 상태에서 시작 버튼 사용 시 패키지 누락을 명확히 알려야 하며 Android에서 잘못된 Linux 바이너리를 자동 다운로드하면 안 된다.
5. Y/N 대소문자, 재실행, 패키지 설치 실패, 비대화형 호출, 기존 설치 여부, Y와 N 모두 시험. **기본 답변값 및 비대화형 입력 처리 정책은 아직 확정되지 않았으므로 임의로 자동 설치로 되돌리지 말 것**.
6. 기존 다른 플랫폼 동작 및 UI 회귀를 검사할 것. 설치 시 Y/N만을 위해 원격 접속 UX를 재설계하지 않는다.

**중요:** 이번 인계 커밋은 Markdown만 변경한다. 위 Y/N 기능은 후속 구현 작업이며 아직 스크립트에 반영되지 않았다.

## 3. tsnet 시험 준비본 / 소스와 무결성

- 파일: `prebuilt/termux/arm64/libpocketrisu_tsnet_exec.so`. 확장자는 `.so`이지만 실제 Android/ARM64 Go 실행 파일(기존 APK 바이너리 무수정 복사).
- 크기 **21,758,246 bytes**; SHA-256 **`c9b5c75f4c9792b79b1f840986582289d2ff1ad822e88fc2206b1a551fe5a71b`**.
- ELF64 AArch64 PIE, interpreter `/system/bin/linker64`, `DT_NEEDED` 없음. Go 1.27.3, Tailscale v1.104.4, `GOOS=android GOARCH=arm64 CGO_ENABLED=0`.
- 원본: `marv435624q/PocketRisu-Android` `prebuilt/jniLibs/arm64-v8a/libpocketrisu_tsnet_exec.so`, 확인 원본 커밋 `a67a75217f94c0d756f4066197e9b0a458bdebe6`. 바이트 동일 기록이 있으며 이번 사용자 기기에서 SHA-256 일치 확인.
- 출처·라이선스: `prebuilt/termux/arm64/PROVENANCE.md`, `BUILDINFO.txt`, `provenance.json`, `SHA256SUMS`, `LICENSES/`.
- 대응 소스: `prebuilt/termux/arm64/tsnet-v1.104.4-source.tar.gz`, **81,151,418 bytes**. 보존된 소스와 고지 자료일 뿐 재현 빌드 결과가 아니다.
- 추가 오프라인 `go mod verify`는 의존성 캐시 메타데이터가 부족해 exit 1로 **미완료**. Go 재빌드/CI/실기기 네트워크 검증을 마쳤다고 보고하지 말 것.
- 원본 Go 소스: https://github.com/marv435624q/PocketRisu-Android/blob/a67a75217f94c0d756f4066197e9b0a458bdebe6/native/tsnet/main.go
- 초기 시험 안내: `docs/ko/tsnet-termux-smoke.md`. 원본 문서의 `--help` 예상 exit 0 및 “기기 미실행” 기록은 **2026-11-05 실기기 로그를 반영하기 전 상태**였음.

## 4. Termux 사용자 실기기 로그 (2026-11-05)

사용자는 Termux를 종료했다가 재시작하고 테스트 브랜치를 `$HOME/PocketRisu-Exp-tsnet-v1044`에 새로 clone했다. 수신 57,797 objects / 165.10 MiB, deltas 해결 성공. 이번 Termux 세션에서 PocketRisu Node 서버는 시작하지 않았다.

`BIN="$PWD/prebuilt/termux/arm64/libpocketrisu_tsnet_exec.so"` 지정, `sha256sum "$BIN"`이 위의 **c9b5…fe5a71b**와 정확히 일치. `chmod u+x "$BIN"` 후 다음을 시험함:

| 실행/조치 | 사용자 실측 | 판정 |
| --- | --- | --- |
| `"$BIN" --help` | `state-dir must be an absolute path ending in tailscale-state`, exit 1 | 잘못된 예상(exit 0); 인자 처리 장애 |
| `umask 077; mkdir -p "$STATE_BASE"; TEST_RUN="$(mktemp -d "$STATE_BASE/run.XXXXXXXX")"; STATE_DIR="$TEST_RUN/tailscale-state"` 후 `"$BIN" --state-dir "$STATE_DIR" --hostname pocketrisu-exp-termux-v1044` | 같은 JSON 오류, 즉시 복귀 | 정상 모양의 상태 경로도 인식 실패 |
| `"$BIN" --definitely-invalid-tsnet-flag` | 같은 JSON 오류, exit 1 | 정상 Go flag의 미지 플래그 오류와 다름 |
| `"$BIN" --exec-cloudflared /dev/null` | 같은 JSON 오류, exit 1 | 원본 `main()`의 특수 경로라면 exit 2여야 함 |
| `"$BIN" __argv_padding__ --help` | 같은 JSON 오류, exit 1 | **판별 시험으로 부적절**. Go `flag`는 첫 일반 인자에서 옵션 해석을 중단 |
| `printf 'LD_PRELOAD=<%s>\n' "$LD_PRELOAD"` | `/data/data/com.termux/files/usr/lib/libtermux-exec.so` | Termux 실행 후킹 사용 확인 |
| `LD_PRELOAD= "$PREFIX/bin/env" -u LD_PRELOAD "$BIN" --help` | `Permission denied`, exit 126 | 실행 자체가 막힘; 인자 원인의 직접 증명은 아님 |

모든 JSON 오류 전문: `{"type":"status","status":"error","error":"state-dir must be an absolute path ending in tailscale-state"}`.

**이 로그로 확인된 것:** 정해진 해시의 바이너리가 Android/Termux에서 Go 코드의 JSON 오류 출력까지 도달함. 플래그와 특수 인자가 정상 처리되지 않음. `starting`은 한 번도 출력되지 않았으므로 `tsnet.Server.Start()`, 로그인·장치 등록·실제 Tailnet 연결·HTTPS 연결은 **미시작/미시험**. 오류는 로컬 PocketRisu 서버가 꺼져 있어서 생긴 것이 아니다. 셸이 테스트 기본 폴더와 `run.XXXXXXXX`를 생성한 것과 Tailscale 상태/신원 저장 성공은 다르다. 폴더를 임의로 지우지 말 것.

## 5. 가장 유력한 원인: Termux linker64 / Go 인자 배열 문제 (아직 확정 아님)

관련 이슈: https://github.com/termux-play-store/termux-exec/issues/9 (2026-10-02, *Go binaries receive their own path as argv[1]*).

이슈는 Google Play 배포 Termux에서 Android `linker64` 경유 Go 실행 파일이 `os.Args[1]`에 **자기 실행 파일 경로를 추가 위치 인자**로 받는 증상을 기록한다. `LD_PRELOAD`를 제거하면 `Permission denied`인 현상도 유사하다. 단, **사용자의 Termux 배포판·`termux-exec` 버전과 실제 `os.Args` 배열은 아직 계측하지 않았으므로 같은 결함이라고 단정하지 말 것**.

원본 `native/tsnet/main.go`의 순서:
1. `main()`에서 먼저 `os.Args[1] == "--exec-cloudflared"`인지 검사.
2. `run()`이 `flag.String("state-dir",...)` 등 등록 후 `flag.Parse()`.
3. 상태 경로가 절대 경로/끝 이름 `tailscale-state`인지 검증.
4. 정상일 때만 디렉터리 생성, `starting` 출력, `tsnet.Server.Start()`.

**가설:** `os.Args[1]`에 실행 경로가 중복 삽입 → 특수 분기 불발 및 `flag.Parse()`가 첫 일반 인자에서 정지 → `state-dir` 기본 빈 문자열 유지 → 모든 인자 패턴에 같은 오류. 이 설명은 지금까지의 관찰과 일관되지만 **실제 os.Args 캡처 또는 통제된 최소 재현이 없어 미확정**.

## 6. 후속 에이전트의 작업 순서 (우선순위)

### A. 오프라인 원인 분리 / 격리 후보 수정 **최우선**

1. 소스·브랜치·이슈 및 위 로그를 함께 읽는다. **일치한 SHA ≠ 재현 빌드 성공**, JSON 오류 출력 ≠ Tailscale 연결 성공.
2. 인증/네트워크를 호출하지 않는 **최소 Go 인자 출력 재현 프로그램** 또는 통제된 진단 빌드로 `os.Args` 삽입 여부를 확인. `termux-info`와 `termux-exec` 버전도 수집하면 재현 환경을 구분할 수 있음.
3. 인자 중복이 확인되면 Termux에서만 적용되는 최소 정규화 방식을 검토. 원본 Android APK 정상 인자, `--exec-cloudflared`, `--client-target`, `--logout-only`를 깨뜨리지 않아야 함. 진단 중 비밀·URL·장치 정보를 로그에 남기지 않는다.
4. 원본 고정 바이너리를 덮어쓰지 말고 **별도 후보 바이너리** 경로·SHA-256·출처/빌드 정보를 남긴다. 빌드 의존성·소스 일치성 점검; 기존 오프라인 `go mod verify` 미완료 상태도 기록.
5. 네트워크 전 시험: 정상 `--help`는 사용법 출력과 0, 미지의 플래그는 Go `flag` 오류와 2, `--exec-cloudflared /dev/null`은 안전한 거부와 2, 무효 `--state-dir`는 JSON 오류와 1을 기대. **유효한 상태 경로로 바로 실제 연결을 실행하지 말고** 플래그 인식 확인을 오프라인에서 분리한다.
6. 정적 검사/모의 시험/로컬 빌드와 Termux 실기기 재현을 구분한다. 외부 모델/API, Actions·원격 CI, 고비용 벤치마크, 외부 계정·데이터 변경은 별도 승인 전 실행하지 않는다.

### B. 오프라인 수정 검증 **이후** Tailscale 연결 (별도 승인)

- 사용자 승인 후에만 새 격리 상태 폴더에서 Tailscale 로그인·장치 등록. APK 상태·기기 신원을 복사하지 않는다.
- `--state-dir`는 절대 경로이고 끝 이름이 정확히 `tailscale-state`여야 함. 기본 예시: `$HOME/.local/state/pocketrisu-exp-tsnet-smoke/run.XXXXXXXX/tailscale-state`. 실제 폴더 내용과 `authUrl` 등은 공유/커밋하지 말 것.
- 전경 프로세스에서 stdin 유지(닫히면 종료될 수 있음). 일반 종료는 `Ctrl+C`. `--logout-only`는 신원 삭제를 동반하므로 일반 종료용으로 사용하지 않음.
- tsnet 인증·등록·장치 접속과 PocketRisu HTTPS 프록시 사용은 **다른 시험**. 프록시 대상은 `http://127.0.0.1:6001`이며, 최종 접근 시험에는 별도 PocketRisu Node 서버 실행과 Tailnet MagicDNS·HTTPS 설정이 필요.
- 이후에야 tsnet을 Node 관리부·기존 원격 접속 설정 UI에 통합하는 별도 변경 검토. 최종 UX를 수동 실행으로 퇴행시키지 말 것.

### C. Quick Tunnel 마무리 (별도 범위)

- `feat/termux-quick-tunnel-ui`의 설치 스크립트에 **Y/N** 선택 구현 및 설치 선택/미설치 오류/재실행/실패/다른 OS 회귀 검증.
- 구현·테스트 여부를 분리 기록. 기존 Quick Tunnel 브랜치 변경, 상위 PR·병합·릴리스는 별도의 작업으로 취급한다.

## 7. 사용자에게 전달할 다음 행동

1. **지금은 Termux를 재시작·재설치·서버 실행·재클론할 필요가 없음.** tsnet 플래그 인식 문제가 먼저.
2. 이 문서와 `docs/ko/tsnet-termux-smoke.md`를 다음 작업 세션에 전달하고, 우선 **A단계(`os.Args` 최소 재현 및 격리 수정/오프라인 검증)**를 맡긴다.
3. 후보 바이너리의 경로·SHA-256·커밋을 확인한 다음에만 Termux에서 새 바이너리의 오프라인 인자 시험. 기존 `$HOME/PocketRisu-Exp-tsnet-v1044`는 유지.
4. 차후 Git에서 변경을 받을 때 `git status --short`로 현지 수정 확인 후 안전하게 갱신. `reset --hard`, `clean -fd`, 시험용 상태 디렉터리 삭제 금지.
5. **온라인 Tailscale 로그인/장치 등록은 오프라인 문제 해결 뒤 별도로 승인**. Quick Tunnel Y/N 구현은 다른 분기의 독립 후속 작업.

## 8. 최종 상태 체크 (작성 시점)

| 항목 | 상태 |
| --- | --- |
| Quick Tunnel UI 시작·종료 | 사용자 실기기 확인 보고 |
| Quick Tunnel 설치 단계 Y/N | **미구현 / 후속 필수 요구** |
| tsnet 원본 SHA 일치 | 사용자 Termux 실측 PASS |
| tsnet Termux 실행 및 Go 오류 도달 | 실측 PASS |
| tsnet `os.Args` 원인 확정 / 인자 정상 처리 | **미확정 / FAIL** |
| tsnet 서버 시작 / 로그인 / 등록 / HTTPS | **미시험** |
| APK 바이너리 재빌드 / Node·UI 통합 | **미실시** |
| 이 인계 작업의 범위 | **Markdown 문서만** |
