# Termux tsnet cert1 — 오프라인 검증 기록

2026-11-05. 기준 argv1 `f4a6d99b08ef491d2557692b7ec7b1c6e06c6dde`.
새 후보 Android 파일은 크로스 빌드만 했으며 실행하지 않았다.
바이너리/소스 해시와 버전은 `native/tsnet/SOURCE.json`에 있다.

## 결과와 증거

| 파일 | 의미 |
| --- | --- |
| sidecar-race.log | 기존 15 포함 24개 통과, 선택형 브라우저 fixture 1개 생략 |
| control-race.log | 누락된 키·동시 호출·실패/취소/종료 격리 새 race 7개 통과 |
| tsnet-guard-race.log | 인증키 조회·명시적 대화형 로그인 차단 2개 통과 |
| host-cli-results.json | 기존 14 포함 19개 통과, 새 state 미생성·합성 state 불변 |
| verifier-python-final.log | 개인정보 whitelist·TLS 판정·인증 환경 제거·종료 정리 모의 시험 6개 통과 |
| baseline-missing-key.log | 원본 Direct에 새 사례를 적용하면 zero serverNoiseKey로 실패. 기대한 회귀 재현 |
| version-unstamped.json | 스탬프 없는 비교 빌드에서 Long/Short ERR-BuildInfo 재현 |
| network-interfaces.txt | 실행 namespace에 loopback만 존재. IPv4/IPv6 default route도 없음 |
| summary.json | 위 실행 결과와 검증 경계 요약 |

Go 1.27.3, Linux amd64, race 시험은 CGO=1, 후보/호스트 실행 파일은 CGO=0.
스탬프 없는 race 실행 파일에서도 키 수정이 통과하므로 버전 수정과 분리했다.
스탬프가 있는 호스트 --build-info는 기반 1.104.5·custom Long·전체 sourceId를 확인한다.
원래 인자 정규화 및 API/프록시/인증/스트리밍/WebSocket 시험을 유지했다.
로그의 loopback 주소와 TLS 오류는 합성 로컬 시험이다. 사용자 장치 정보가 아니다.

## 실행 방법과 한계

소스 archive를 복원한 Linux/WSL, Go 1.27.3에서 시험 실행 파일을 컴파일한다.
sidecar는 native/tsnet의 `go test -c -race`, controlclient는
`go test -c -race tailscale.com/control/controlclient`를 사용했다.
control 실행은 새 `TestSetDNS*` 7개만 선택하며 upstream 전체 통합 시험은 아니다.
등록 가드는 vendor 모듈의 tsnet 디렉터리에서 전체 생산 파일 두 개와
`require_existing_registration_test.go`만 명시해 `go test -c -race`로 컴파일했다.
원본 모듈 밖에서 파일 목록으로 시험하면 internal 패키지 경계 오류가 발생한다.
전체 upstream tsnet 통합 시험은 서버·클라우드 시험 전용 캐시가 부족해 컴파일하지 못했다.
추가 다운로드 없이 범위 내 등록 가드 시험을 분리했으며 이 실패를 통과로 세지 않았다.

호스트 CLI 도우미는 `scripts/termux/test-tsnet-host-cli.py`의
`--binary <Linux host> --output <새 기록 폴더> --identity <matching build-flags.json>`이다.
검증기 모의 시험은 `python3 scripts/termux/test_verify_tsnet_cert1.py`다.
시험은 `unshare --net`, loopback up, 외부 default route 없음의 namespace에서 실행했다.
준비 단계의 sysfs 인터페이스 집계와 결과 집계 도구 누락은 별도 기록에 보존했으며,
이미 통과한 Go/CLI 시험을 그 이유로 다시 실행하지 않았다.

실제 컨트롤 Noise 핸드셰이크·ACME DNS/CA·Termux 443·외부 기기 HTTPS·
새 후보 관리 화면 표시는 미검증이다. 로컬 합성 CA/TLS 프록시 시험은 이 증거가 아니다.
원격 CI·Actions·실계정·장치 상태를 쓰는 온라인 시험은 수행하지 않았다.
사용자 후속 명령과 안전 경계는 `docs/ko/tsnet-termux-cert1-candidate.md`에 있다.

최종 일반 실행 환경 호환 수정 후 sidecar 24개와 공개 CLI 도우미 19개를 다시 확인했다. 컨트롤 시험 실행 파일은 이전 검증본과 SHA-256까지 동일해 7개 결과를 재사용했다. vendor 및 Python 소스가 그대로인 등록 가드 2개·검증기 6개도 재사용했다.
