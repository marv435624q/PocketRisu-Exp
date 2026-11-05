# Termux tsnet v1.104.5 인자 수정 후보

2026-11-05. 기존 인계 `b3bcd71632370208733fd63f5b02561eb1924c7c`를 이어 만든 **실행 인자 확인용 후보**다. 기존 v1.104.4 파일·APK·장치 신원은 바꾸지 않는다. Quick Tunnel/UI/설치 Y/N는 이번 변경에 포함하지 않는다.

## 기존 실패와 수정 근거

사용자가 시험한 v1.104.4는 Android 실행 파일에 진입했지만 `--help`, 잘못된 플래그, 올바른 `--state-dir`, 안전하지 않은 exec 경로 검사가 모두 같은 state-dir 오류로 끝났다. `starting`·엔진·로그인·장치 등록에 도달했다는 증거는 없다. 원문과 한계는 [기존 인계](termux-remote-access-handoff-20261105.md)에 보존한다.

[Termux exec의 Go 인자 문제](https://github.com/termux-play-store/termux-exec/issues/9)와 구현상 추가 경로 인자는 이 증상과 일치한다. 다만 사용자 단말의 실제 `os.Args`는 아직 확보하지 않았으므로 원인 확정이나 단말 해결 완료로 보고하지 않는다. `LD_PRELOAD`를 제거해 다시 시험할 필요는 없다. 이미 permission denied 126이 기록되어 있다.

후보는 Android에서 `TERMUX_EXEC__PROC_SELF_EXE`가 있고 첫 추가 인자가 절대 경로이며, 두 경로가 **동일한 일반 파일**일 때만 인자 하나를 제거한다. 상대 marker는 현재 작업 폴더 기준으로 해석한다. 파일 이름만 비교하거나 임의의 위치 인자를 삭제하지 않는다. 다른 파일·marker 없음·일반 APK 인자·Linux 실행은 유지한다.

`--diagnose-argv`를 단독으로 사용하면 개수와 조건 충족 여부만 JSON으로 내고 종료한다. 경로·호스트명·인자 값은 출력하지 않는다. `removedExecutableArg:true`는 이 조건에 맞는 인자가 실제로 제거됐다는 뜻이다. false라면 marker/경로 조건 등 결과를 근거로 다음 수정을 판단해야 한다.

## 후보와 출처

- 파일: `prebuilt/termux/arm64/tsnet-v1.104.5-argv1/libpocketrisu_tsnet_exec.so`
- 크기: **21,758,246 bytes**
- SHA-256: **`49c998c1a1701340d0855068f8f5d42126ff948dadd9998cd44d32438658662e`**
- Go **1.27.3**, Tailscale **1.104.5**, Android arm64, `CGO_ENABLED=0`
- ELF64 little-endian AArch64, ET_DYN/PIE, interpreter `/system/bin/linker64`, `DT_NEEDED` 없음, LOAD 정렬 65,536 bytes
- 공식 Tailscale v1.104.5 소스와 기존 공개 sidecar 소스를 사용했다. Android ACME localapi의 기존 두 build-tag 패치를 유지했다. 공식 소스 다운로드 후 SHA-256은 계산했으며, 별도 서명으로 검증한 해시는 아니다.

원본 v1.104.4 SHA-256은 계속 `c9b5c75f4c9792b79b1f840986582289d2ff1ad822e88fc2206b1a551fe5a71b`다. 새 후보로 덮어쓰지 않는다. 이 후보는 공식 Tailscale 배포 바이너리가 아니다.

버전·출처·해시·시험 범위는 `native/tsnet/SOURCE.json`에 고정한다. 실제 빌드 정보와 소스 파일별 해시는 후보 폴더의 `buildinfo.txt`, `BUILD_MANIFEST.json`, `provenance.json`에 있다.

## 현재 확인 범위

Android ARM64 크로스 빌드, ELF/내장 Go buildinfo, ACME `cert.go`의 Android 빌드 포함, `go mod verify`의 `all modules verified`를 확인했다. 최초 검사는 캐시에 memconn 메타데이터가 없어 실패했고, 정확한 공개 모듈 메타데이터를 받은 뒤 다시 검사해 통과했다. go.mod/go.sum은 이 과정에서 바뀌지 않았다.

새 네트워크 namespace에 loopback만 두고 외부 IPv4/IPv6 경로가 없는 Linux PC에서 Go race 시험 **15개**(인자 정규화 하위 사례 11개 포함), 호스트 CLI 사례 **14개**가 통과했다. 기존 인증 전달·인증서 검증·스트리밍·WebSocket의 로컬 시험도 포함한다. 새 state 미생성 및 기존 합성 state 불변을 검사했다. Android 분기는 함수 시험에서 조건을 주어 검증했으며 Linux 실행 파일이 실제 Termux를 대신하는 것은 아니다.

**Android 후보를 실제 Termux에서 실행하지 않았다.** 로그인·장치 등록·Tailnet/HTTPS 접속·네트워크 전환은 시험하지 않았다. 원격 CI도 실행하지 않았다.

## 후보를 받은 뒤 실행할 오프라인 검사

기존 Termux 저장소에서 변경한 파일과 이전 실행 기록을 보존하려면 새 worktree를 사용한다. 아래는 기존 clone의 최상위 폴더에서 실행한다. `<새 폴더>` 대신 아직 없는 형제 경로를 고른다.

```bash
git fetch origin fix/termux-tsnet-argv-v1045-20261105
git worktree add --detach ../PocketRisu-Exp-tsnet-v1045 origin/fix/termux-tsnet-argv-v1045-20261105
cd ../PocketRisu-Exp-tsnet-v1045
bash scripts/termux/test-tsnet-args.sh
```

이미 같은 새 폴더가 있다면 다른 새 경로를 사용한다. 기존 폴더를 삭제하거나 reset하지 않는다. 스크립트는 후보 SHA-256을 검사하고 다음 여섯 검사만 실행한다.

| 검사 | 예상 결과 |
| --- | --- |
| 단독 `--diagnose-argv` | JSON `type:argv-diagnostic`, exit 0 |
| `--help` | 도움말, exit 0 |
| 잘못된 플래그 | 값이 없는 고정 오류, exit 2 |
| 상대 state 경로 | 고정 state-dir 오류, exit 1 |
| `--exec-cloudflared /dev/null` | 안전하지 않은 경로 거부, exit 2 |
| `--check-args --state-dir <새 경로>/tailscale-state --hostname ...` | JSON `type:args-check`, exit 0, state 생성 없음 |

로그는 `${XDG_STATE_HOME:-$HOME/.local/state}/pocketrisu-exp-tsnet-argv-probe/run.XXXXXXXX/`에 보존하며 `LAST_RUN`에 마지막 폴더 위치를 기록한다. APK 상태와 분리하고 기존 신원을 읽거나 등록 해제하지 않는다. raw 인자나 환경 전체를 수집하지 않는다.

`--check-args`는 일반 tsnet 옵션 해석과 경로 형식만 확인하고 mkdir·엔진·로그아웃 전에 종료한다. **`--exec-cloudflared`와 결합하지 않는다.** 특수 exec는 일반 파서보다 먼저 처리되는 기존 규약을 유지한다. `/dev/null` 거부 확인은 실제 cloudflared 실행 성공의 증거가 아니다. Termux에서 `os.Executable()`이 linker를 가리키는 경우와 sibling 검증의 실동작은 별도 미확인 사항이다.

이 스크립트에는 정상 시작·로그인·등록·HTTPS 연결 명령이 없다. 인자 검사가 통과해도 다음 네트워크 시험은 별도 범위로 진행한다. 이전 실패 명령을 반복해서 정상 시작을 시도할 필요는 없다.

## 대응 소스와 개발자 빌드

새 후보의 대응 소스는 다음 두 묶음을 함께 사용한다.

1. 후보 폴더의 `tsnet-v1.104.5-argv1-source.tar.gz`: 이번 sidecar·시험·패치된 Tailscale v1.104.5·라이선스·빌드 기록.
2. `prebuilt/termux/arm64/tsnet-v1.104.4-source.tar.gz`: 후보에도 동일하게 링크된 외부 Go 모듈 27개의 `module-sources/`와 Go 라이선스·특허 고지. 예전 sidecar/Tailscale 트리는 새 후보의 소스로 쓰지 않는다.

두 번째 묶음의 SHA-256은 `f5a17e6896c1ba42f21f860a09614c29a21b48a9e2878eeee6fd42ccd0bdece7`다. 후보 buildinfo와 27개 모듈의 경로·버전·Go h1을 대조했다. 기존 `LICENSES/`의 원 저작권·라이선스·특허 고지도 함께 유지한다. 공개 소스에 포함된 예제·시험 fixture는 사용자 상태가 아니다. APK 데이터·인증키·장치 신원은 포함하지 않는다.

Git checkout의 `native/tsnet/third_party/`는 추적하지 않으며 새 후보 소스 묶음의 고정 해시로 복원한다. 개발용 Linux/WSL에서 Go 1.27.3을 설치한 뒤 다음 빌드 도우미를 쓴다.

```bash
bash scripts/termux/build-tsnet-candidate.sh
```

기존 source 수정이나 심볼릭 링크를 덮어쓰지 않으며 출력은 매번 새 `native/tsnet/output/` 하위 폴더에 만든다. prebuilt/장치 state에 쓰거나 후보를 실행하지 않는다. Go 캐시가 비어 있으면 공개 모듈 다운로드가 필요하다. 빈 캐시에서 완전한 오프라인 빌드나 독립적인 재현 빌드를 입증한 것은 아니다. archive를 직접 푼 경우 `native/tsnet`에서 BUILD_MANIFEST의 명령으로 빌드할 수 있다.
