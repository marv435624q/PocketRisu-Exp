# Termux에서 기존 tsnet 바이너리 독립 실행 확인

이 브랜치는 APK에서 쓰던 **Tailscale v1.104.4 기반 바이너리를 바이트 변경 없이** 가져온 시험 준비본입니다. **2026-11-05 Termux 실기기에서 해시 일치 및 Go 코드의 JSON 오류 출력까지 확인했지만, 명령줄 인자 인식 문제로 `tsnet.Server.Start()`에는 진입하지 못했습니다.** 로그인·장치 등록·HTTPS 연결 성공이 아닙니다. 기존 Quick Tunnel, 서버 실행 경로와 PocketRisu 설정 UI는 바꾸지 않았습니다. v1.104.5 업그레이드와 재빌드도 하지 않았습니다.

> **최신 인계:** [Quick Tunnel 및 tsnet 문제·실측 로그·다음 작업](termux-remote-access-handoff-20261105.md). 이 안내서의 최초 작성 당시 `--help` 성공 예상은 실제 Termux 실행 결과와 달랐으며, 아래에 정정했습니다. 수정 바이너리가 실기기에서 검증되기 전에는 이 문서 4절의 온라인 연결 명령을 실행하지 마세요.

## 포함한 파일과 출처

| 항목 | 값 |
| --- | --- |
| 파일 | `prebuilt/termux/arm64/libpocketrisu_tsnet_exec.so` |
| 크기 | 21,758,246 bytes |
| SHA-256 | `c9b5c75f4c9792b79b1f840986582289d2ff1ad822e88fc2206b1a551fe5a71b` |
| ELF | 64비트 little-endian, AArch64(ARM64), ET_DYN/PIE 실행 파일 |
| 인터프리터 | `/system/bin/linker64` |
| 동적 의존성 | `DT_NEEDED` 항목 없음 |
| 빌드 기록 | Go 1.27.3, Tailscale 1.104.4, `GOOS=android GOARCH=arm64 CGO_ENABLED=0` |
| 원본 | PocketRisu-Android의 `prebuilt/jniLibs/arm64-v8a/libpocketrisu_tsnet_exec.so` |
| 원본 확인 커밋 | `a67a75217f94c0d756f4066197e9b0a458bdebe6` |

이름의 `.so`는 APK 포장용입니다. Go 프로그램의 진입점과 Android 인터프리터가 있는 별도 실행 파일입니다. Android 인증서 API를 포함하도록 수정한 Tailscale 소스를 사용했습니다. 상세 출처·고지·대응 소스는 같은 prebuilt 디렉터리의 `PROVENANCE.md`와 라이선스 자료를 확인하세요.

## 1. 기존 설치와 분리해서 받기

다음은 **사용자가 Termux에서 직접 수행할 명령**입니다. ARM64 Termux의 앱 전용 홈 아래에서 사용합니다. `/sdcard` 등 공유 저장소에 놓으면 실행 권한 문제를 구분하기 어렵습니다. 아래 clone은 기존 `~/PocketRisu-Exp`를 건드리지 않습니다. 대상 폴더가 이미 있으면 clone은 중단됩니다.

```bash
pkg install -y git coreutils file binutils
git clone --single-branch --branch test/termux-tsnet-v1044 \
  https://github.com/marv435624q/PocketRisu-Exp.git \
  "$HOME/PocketRisu-Exp-tsnet-v1044"
cd "$HOME/PocketRisu-Exp-tsnet-v1044"
```

각 명령의 성공을 확인하고 다음 단계로 진행합니다. 이 준비본에 웹 설치나 빌드를 실행할 필요는 없습니다.

## 2. 파일 확인과 네트워크 없는 실행 확인

```bash
cd "$HOME/PocketRisu-Exp-tsnet-v1044"
BIN="$PWD/prebuilt/termux/arm64/libpocketrisu_tsnet_exec.so"
uname -m
sha256sum "$BIN"
(cd prebuilt/termux/arm64 && sha256sum -c SHA256SUMS)
file "$BIN"
readelf -h "$BIN"
readelf -l "$BIN"
readelf -d "$BIN"
```

`uname -m`은 `aarch64`, SHA-256은 위 값, ELF 인터프리터는 `/system/bin/linker64`여야 합니다. `readelf -d`에는 `NEEDED`가 없어야 합니다. 다르면 실행하지 않고 결과부터 확인합니다.

원본 Go `flag` 구현상 정상 인자가 전달된다면 `--help`는 소스의 인자 처리 단계에서 종료하며 `tsnet.Server.Start()`를 호출하지 않아야 합니다. 실행 권한만 복사본에 설정하며 파일 바이트는 바꾸지 않습니다. **다만 사용자 Termux 실기기에서는 실제로 아래 명령이 도움말 대신 `state-dir` 경로 오류를 반환하고 종료 코드 1로 끝났습니다.**

```bash
chmod u+x "$BIN"
"$BIN" --help
printf 'help exit=%s\n' "$?"
```

**소스 기준 정상 기대 결과는 도움말과 종료 코드 0이지만, 2026-11-05 사용자 실측은 다음과 같습니다.**

```text
{"type":"status","status":"error","error":"state-dir must be an absolute path ending in tailscale-state"}
help exit=1
```

즉, 바이너리는 실행되어 오류 JSON을 출력했지만 **플래그 처리가 정상이라고 볼 수 없습니다**. 인자 오류의 유력 후보는 Termux `linker64`/Go `os.Args` 중복 전달 이슈([termux-exec #9](https://github.com/termux-play-store/termux-exec/issues/9))이나, 본 기기에서 `os.Args` 계측 전이므로 확정하지 않습니다. 원인 분리·수정은 위 인계 문서를 따릅니다.

원한다면 잘못된 상태 경로를 거부하는 분기도 확인할 수 있습니다. 이 경로는 의도적으로 `tailscale-state`로 끝나지 않습니다. 소스는 디렉터리를 만들거나 서버를 시작하기 전에 거부합니다.

```bash
"$BIN" --state-dir "$HOME/.local/state/pocketrisu-exp-tsnet-smoke/invalid-state-name"
printf 'invalid state-dir exit=%s\n' "$?"
```

기대 결과는 `state-dir must be an absolute path ending in tailscale-state`가 들어 있는 JSON 오류와 종료 코드 1입니다. 여기까지가 로그인 없이 할 수 있는 실행 확인입니다.

## 3. 실제 인자 규약

원본 `native/tsnet/main.go`의 `flag` 정의, 상태 경로 검사, stdin 처리와 대조했습니다.

| 인자 | 실제 동작 |
| --- | --- |
| `--state-dir ABSOLUTE_PATH` | 필수. 절대 경로이며 마지막 폴더 이름은 정확히 `tailscale-state`여야 합니다. 권한 0700으로 만들거나 설정합니다. |
| `--hostname NAME` | 기본값 `pocketrisu`. 이 시험에서는 APK와 구분되는 `pocketrisu-exp-termux-v1044`를 사용합니다. |
| `--logout-only` | 장치 로그아웃에 성공하면 지정한 상태 디렉터리를 삭제합니다. 이번 시험과 일반 중지에는 사용하지 않습니다. |
| `--client-target HOST:PORT` | APK 클라이언트용 로컬 프록시 모드입니다. 이번 서버용 독립 시험에는 넣지 않습니다. |

이 바이너리에는 `--port`, `--authkey`, `--version` 인자가 없습니다. 서버용 모드는 Tailnet의 443 포트를 열어 **`http://127.0.0.1:6001`**로 전달하도록 고정돼 있습니다. 별도 Tailscale CLI나 `tailscale serve`를 실행하는 프로그램도 아닙니다.

## 4. 후속 로그인·네트워크 시험용 실행 예시

**주의: 현재 제공된 바이너리로는 여기서 더 진행하지 마세요.** 이 절은 후속 연결 시험에 사용할 명령 예시입니다. 원래 준비 작업에서는 실행하지 않았지만, **2026-11-05 사용자가 실제로 시도했을 때 정상 형태의 `--state-dir`를 전달했음에도 이전과 동일한 경로 검증 오류가 발생하여 서버 시작 전에 종료됐습니다.** 원인을 수정하고 먼저 오프라인 인자 검증을 통과한 **새 후보 바이너리**에 대해서만, 별도 사용자 승인 후 로그인·장치 등록·네트워크 시험을 진행합니다. 그 과정은 네트워크 통신·인증 링크 생성·장치 신원 저장을 일으킬 수 있습니다.

상태는 저장소 밖의 새 테스트 폴더에 둡니다. APK의 상태 폴더를 복사하거나 공유하지 않습니다. 매번 새 `run.XXXXXXXX` 폴더를 만들므로 이전 시험의 장치 신원도 덮어쓰지 않습니다. 재시작 때 같은 시험 신원을 유지하려면 터미널의 `STATE_DIR` 값을 그대로 사용합니다.

```bash
cd "$HOME/PocketRisu-Exp-tsnet-v1044"
BIN="$PWD/prebuilt/termux/arm64/libpocketrisu_tsnet_exec.so"
umask 077
STATE_BASE="${XDG_STATE_HOME:-$HOME/.local/state}/pocketrisu-exp-tsnet-smoke"
mkdir -p "$STATE_BASE"
TEST_RUN="$(mktemp -d "$STATE_BASE/run.XXXXXXXX")"
STATE_DIR="$TEST_RUN/tailscale-state"
printf 'test state directory: %s\n' "$STATE_DIR"
"$BIN" --state-dir "$STATE_DIR" --hostname pocketrisu-exp-termux-v1044
```

`STATE_BASE`는 절대 경로여야 합니다. 표준 입력을 열어 둔 채 전경에서 실행합니다. stdin을 `/dev/null`로 돌리거나 입력 파이프가 닫히면 프로그램은 종료를 요청합니다. 부모 프로세스 종료 시에도 함께 종료하도록 작성돼 있습니다.

일반 중지는 **Ctrl+C**입니다. 기존 상태를 삭제하지 않습니다. stdout은 줄마다 JSON 상태이며, 향후 `{"action":"stop"}` 한 줄을 stdin으로 보내는 부모 프로세스 제어도 가능합니다. 이번에는 Node 관리부나 UI에 연결하지 않았습니다.

실제 HTTPS 시험에는 별도로 실행 중인 로컬 PocketRisu 서버, Tailnet의 MagicDNS·HTTPS 인증서 설정, 같은 Tailnet에 접속한 클라이언트가 필요합니다. 그 검증 결과는 아직 없습니다. 시작 오류와 부모·stdin 종료 문제는 HTTPS 성공 여부와 나눠 기록합니다.

상태 폴더에는 장치 신원과 인증 관련 자료가 생길 수 있습니다. 이 폴더와 `authUrl`을 포함할 수 있는 실행 출력은 Git에 추가하지 않습니다. 공유가 필요하면 인증 URL·키·장치 정보를 제거한 오류 부분만 사용합니다. 이 브랜치에 포함한 것은 기존 바이너리와 출처·라이선스·실행 준비 자료뿐입니다.

## 검증 범위와 2026-11-05 실기기 갱신

**준비 작업 당시의 정적 확인:** 원본·복사본의 크기와 SHA-256, ELF 헤더·인터프리터·동적 의존성, 원본 소스 및 빌드 기록, 문서의 인자와 셸 문법을 확인했습니다. PC에서 Android 바이너리를 실행하지 않았고, 준비 시점에는 Termux도 실행하지 않았습니다.

**추가 사용자 실기기 확인 (2026-11-05):** Termux 새 세션에서 시험 브랜치를 clone하고 SHA-256이 위의 값과 정확히 일치함을 확인했습니다. `--help`, 유효한 `--state-dir`, 미지 플래그, `--exec-cloudflared /dev/null` 모두 동일한 경로 JSON 오류를 반환했습니다. `LD_PRELOAD`에 `libtermux-exec.so`가 있었고, 이를 제거한 시험은 `Permission denied`(exit 126)로 프로그램 시작 이전에 종료됐습니다. `__argv_padding__ --help`는 Go `flag` 특성상 올바른 판별 시험이 아니었습니다. **Tailscale 시작·로그인·등록·실제 Tailnet/HTTPS 연결·재빌드는 여전히 수행하지 않았습니다.** 상세한 명령·결과·가설·후속 작업은 [인계 문서](termux-remote-access-handoff-20261105.md)를 참조하세요.

추가 오프라인 `go mod verify`는 캐시에 필요한 모듈 정보가 없어 종료 코드 1로 완료되지 않았습니다. 다운로드는 하지 않았고 `go.mod`·`go.sum`도 바꾸지 않았습니다. 모듈 캐시 무결성 검사 통과나 전체 빌드 그래프·재현 빌드 검증으로 보고하지 않습니다. 바이너리·복사본 및 소스 묶음의 파일/해시 비교와는 구분합니다.
