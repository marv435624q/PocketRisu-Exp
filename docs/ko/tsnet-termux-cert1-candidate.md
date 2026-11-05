# Termux tsnet v1.104.5 cert1 수정 후보

2026-11-05. 기준은 `f4a6d99b08ef491d2557692b7ec7b1c6e06c6dde`의 argv1 후보다. **새 후보의 실제 Termux HTTPS 성공은 아직 확인하지 않았다.** 기존 후보·APK·Quick Tunnel·main·사용자 상태는 덮어쓰지 않는다.

## 이번 기준이 된 사용자 실기기 결과

argv1은 사용자 Termux에서 오프라인 검사 6개, 중복 실행 경로 제거, 로그인·등록, Running·Connected·IP 할당까지 성공했다. 이 단계는 더 이상 미확인으로 취급하지 않는다. 다만 인증서는 첫 시도에서 `context deadline exceeded`, 같은 state의 다음 실행에서 `zero serverNoiseKey`로 실패했고, 관리 화면 버전은 `1.104.5-ERR-BuildInfo`였다. 개인 도메인·TXT·ID·키·인증 URL·이메일·상태 파일은 이 저장소에 수록하지 않았다.

## A. 인증서 실패의 근거와 수정

v1.104.5의 호출 경로는 `CertPair → LocalAPI → ACME DNS-01 → LocalBackend.SetDNS → Auto.SetDNS → Direct.setDNSNoise → getNoiseClient`다. 원본 `getNoiseClient`의 `serverNoiseKey.IsZero()` 검사는 실제 DNS RPC 전에 오류를 반환한다. `Running`과 등록 상태만으로 현재 Direct 인스턴스의 키 초기화까지 보장하지 못하는 경우를 방어해야 한다.

정상 `doLogin`은 기존 HTTPS `/key` 응답으로 legacy/Noise 공개키를 설정한다. `LocalBackend.SetDNS`는 현재 `ccAuto`를 잡고, 클라이언트 교체 함수도 `ccAuto`를 갱신한다. 코드에서 별도 잘못된 클라이언트를 가리키는 결함이나 사용자의 정확한 교체·초기화 순서를 확인하지는 못했다. **새 클라이언트·캐시된 상태·인증서 호출 시점 사이의 경쟁은 가능한 설명이며, 사용자 단말에서 입증한 원인은 아니다.** 첫 timeout의 발생 단계도 확보하지 못했으므로 후속 zero-key와 같은 원인으로 단정하지 않는다.

수정은 `SetDNS` 진입 시 키가 비어 있으면 로그인과 동일한 `loadServerPubKeys(ctx, c.httpc, c.serverURL)`로 먼저 보충하는 것이다. 기존 HTTPS 인증서 신뢰와 컨트롤 URL을 사용하며, 가짜 키를 넣거나 원래 zero-key 검사를 제거하지 않는다. 기존 정상 키는 그대로 사용한다. 병렬 호출은 하나로 합치고, 취소·종료를 반영하며, 동시에 로그인에서 채운 키를 덮어쓰지 않는다. 장치 키·persist·로그인 코드는 이 수정에서 바꾸지 않는다.

원본 direct.go에 새 회귀 사례를 적용하면 `zero serverNoiseKey`로 실패했고, 수정본은 합성 `/key` 응답 뒤 합성 `/machine/set-dns` 요청까지 통과했다. 동시 호출·잘못된 키·HTTP 실패·취소 후 재시도·로그인과의 경합·종료된 클라이언트 격리도 시험했다. **실제 Noise 핸드셰이크나 ACME 발급 시험은 아니다.** 기존 Android ACME 패치는 localapi 두 build tag만 바꾸므로 키를 비우는 경로와 구별된다. Android 빌드에 `cert.go`가 계속 포함된다.

인증서 성공 전에는 443 리스너를 만들지 않는다. 기존 인증서 제한 시간 30초는 유지한다. 일반 모드의 재시도는 유지하며, 단일 검증 모드는 인증서·리스너 실패 시 실패로 종료한다.

## B. 버전 표시의 근거와 수정

원본 `version.Long()`/`Short()`는 스탬프가 없고 `vcs.revision`·`vcs.time`도 없으면 `ERR-BuildInfo`를 붙인다. 기존 `-buildvcs=false -ldflags='-s -w'`는 이 조건과 일치한다. 모듈 버전이나 `node:tsVersion = 1.104.5`는 이 표시 오류와 모순되지 않는다.

공식 linker 스탬프 `tailscale.com/version.shortStamp`, `longStamp`를 사용한다.

- Short: `1.104.5`
- Long: `1.104.5-pocketrisu-exp-cert1-s0db6aff6de9f`
- 전체 sourceId: `0db6aff6de9fc04413fef2af30ecfaeeb0c3028132c0d0a79c6a1e0eaa065c35`

`-buildvcs=false`는 유지한다. 수정된 소스를 공식 배포물이나 원본의 깨끗한 VCS checkout으로 가장하지 않는다. sourceId는 production Go·전체 패치된 Tailscale 트리·go.mod/go.sum·CANDIDATE.json·두 빌드 스크립트의 상대 경로/해시를 정렬한 JSON의 SHA-256이다. 생성 산출물과 SOURCE.json은 순환 참조를 피하려고 제외한다. 정확한 입력은 BUILD_MANIFEST에 있다.

스탬프 없는 비교 실행 파일에서 `ERR-BuildInfo`를 재현하고, 스탬프를 넣은 Linux 후보의 `--build-info`로 위 값을 확인했다. 키 처리 시험은 스탬프 없는 race 실행 파일에서 별도로 통과해 두 수정의 근거를 분리했다. 새 Android 후보의 실제 관리 화면 표시는 사용자 후속 확인이 필요하다.

## 파일·소스·라이선스

| 항목 | 값 |
| --- | --- |
| 바이너리 | `prebuilt/termux/arm64/tsnet-v1.104.5-cert1/libpocketrisu_tsnet_exec.so` |
| 크기 | 21,889,318 bytes |
| SHA-256 | `9e1e30367c8a361b92b38e0085f6bb67492d4e4b5b4be915fc6a887e8b65c5bb` |
| 컴파일러/기반 | Go 1.27.3 / Tailscale v1.104.5 / CGO_ENABLED=0 |
| ELF | ELF64 little-endian AArch64 PIE, `/system/bin/linker64`, DT_NEEDED 없음, LOAD 정렬 65,536 bytes |
| 원본 Tailscale commit | `73a48057aaf5f60e4f770b398e9b7af576b01dbe` |

후보 폴더의 `provenance.json`, `BUILD_MANIFEST.json`, `buildinfo.txt`, `SHA256SUMS`와 `native/tsnet/SOURCE.json`을 함께 확인한다. 대응 소스는 같은 폴더의 `tsnet-v1.104.5-cert1-source.tar.gz`와 기존 `prebuilt/termux/arm64/tsnet-v1.104.4-source.tar.gz`의 외부 모듈 27개를 함께 사용한다. 링크된 모듈 경로·버전·h1은 argv1과 같다. 이전 묶음의 옛 sidecar/Tailscale은 새 후보 소스가 아니다.

sidecar의 저장소 GPL-3.0, Tailscale BSD-3-Clause, 기존 `LICENSES/` 및 모듈별 저작권·특허 고지를 보존했다. 패치는 `ANDROID_ACME.patch`, `CONTROL_NOISE_KEY.patch`, `REQUIRE_EXISTING_REGISTRATION.patch`에 있다. Go 캐시가 비었을 때 완전한 오프라인 빌드나 독립 재현 빌드를 입증한 것은 아니다.

## 확인한 범위

| 검증 | 결과·한계 |
| --- | --- |
| sidecar Go race | 24 통과, 기존 15 포함. 선택형 브라우저 fixture 1개 생략 |
| 컨트롤 키 Go race | 새 7개 통과. HTTP/Noise 전송은 합성 |
| 등록 차단 Go race | 새 2개 통과. callback 합성, 실제 계정 사용 없음 |
| Linux CLI | 19 통과, 기존 14 포함. state 미생성·합성 기존 state 불변 |
| Python 검증기 | 모의 단위 시험 6개 통과. 비밀 필드 제거·환경 제거·종료 정리 |
| 로컬 HTTPS | 합성 CA/loopback TLS·프록시·리디렉션 차단·5xx 실패 확인 |
| 정적 검사 | Android 크로스 빌드·ELF·buildinfo·소스/바이너리 해시·모듈 검증 |
| 새 후보 실기기 | 미실시. 로그인 유지·ACME 발급·443·실제 Tailnet/외부 기기 HTTPS·관리 화면 버전 미확인 |

실행 시험은 loopback만 있고 외부 IPv4/IPv6 경로가 없는 Linux namespace에서 했다. 원격 CI·Actions·실계정 시험은 실행하지 않았다. 준비 중 캐시 부족, Go internal 패키지 경계, namespace 인터페이스 확인, 집계 도구 누락 때문에 실패한 명령은 별도 기록에 보존했다. 이를 기능 시험 성공으로 세지 않았다.

## 사용자 후속 실행

새 브랜치를 별도 worktree로 받은 다음 저장소 최상위에서 아래 명령 하나로 **오프라인 검사만** 실행한다.

```bash
python3 scripts/termux/verify-tsnet-cert1.py
```

바이너리 SHA/크기·기반 버전·전체 sourceId·커스텀 Long을 먼저 검사한다. 여섯 기존 probe와 `--build-info`만 실행하며 state를 만들거나 연결하지 않는다. 이전에 통과한 argv 검사를 다시 수동으로 하나씩 할 필요는 없다.

다음 명령은 **사용자가 실제 HTTPS 후속 시험을 하기로 한 경우에만** 쓴다. `STATE_DIR`에는 로그인에 성공했던 기존 테스트 `tailscale-state`의 절대 경로를, `TS_HOSTNAME`에는 그때 쓰던 호스트명을 그대로 사용한다. 값이 없거나 상태 파일이 없으면 중단하며 새 등록을 만들기 위해 초기화하지 않는다. 같은 state를 사용하는 옛 sidecar가 종료된 상태에서 실행한다. PocketRisu의 `127.0.0.1:6001` 서버는 켜 둔다.

```bash
python3 scripts/termux/verify-tsnet-cert1.py --https --state-dir "$STATE_DIR" --hostname "$TS_HOSTNAME"
```

`--require-registered`는 기존 nonempty 상태 파일을 엔진 시작 전에 요구하고, 내부 tsnet의 인증키 조회와 명시적 대화형 로그인을 차단한다. inherited `TS_*`/`TSNET_FORCE_LOGIN`도 검증기에서 제거한다. 상태를 삭제·로그아웃·재로그인시키는 코드가 아니다. 정상 시작 과정의 컨트롤 통신·기존 등록 갱신·상태 쓰기까지 차단하는 읽기 전용 모드는 아니다. 인증이 다시 필요하면 실패로 끝내고 그 결과만 전달한다.

단일 검증은 인증서 발급/확인 → `ListenTLS(:443)` → 자체 tsnet dialer를 통한 HTTPS GET → 신뢰 체인/HTTP 응답 확인 후 정상 종료한다. 성공에는 TLS 검증과 HTTP 200–499가 모두 필요하다. 401/403은 인증된 앱 사용 성공을 뜻하지 않는다. 자체 HTTPS 성공도 별도 PC·폰의 ACL/경로/접속 성공을 입증하지 않는다. 그 단계는 사용자 평소 접속 결과로 구별한다.

기본 전체 대기는 180초다. 자식에게 stop을 보내고 남으면 자신이 만든 프로세스 그룹만 정리한다. 인증서 실패·키 오류·등록 요구·취소를 성공으로 처리하지 않는다. 결과는 `$HOME/.local/state/pocketrisu-exp-tsnet-cert1-probe/run-*/results.json`에 허용된 상태명·오류 분류·HTTP 코드·TLS 성공 여부만 저장하며 `LAST_RUN`에는 폴더 이름만 쓴다. 원문 stdout/stderr·실제 경로·호스트명·인증 URL·키·도메인·TXT는 저장하지 않는다.

다른 세션은 [인계 문서](termux-tsnet-cert-version-handoff-20261105.md)에서 이어간다.
