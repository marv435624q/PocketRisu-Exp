# Termux tsnet 인증서·버전 후속 인계 — 2026-11-05

작업 가지는 `fix/termux-tsnet-cert-version-20261105`, 기준은 `f4a6d99b08ef491d2557692b7ec7b1c6e06c6dde`다. 새 후보는 **실기기 확인 전**이다. [후보 설명·명령·원인과 한계](tsnet-termux-cert1-candidate.md), [검증 기록](../../verification/termux-tsnet-cert-version-20261105/README.md)을 먼저 읽는다.

## 이어받을 사실

1. 기존 argv1의 Termux 인자 문제는 사용자의 오프라인 6개 검사로 해결 확인됐다. 로그인·장치 등록·Running·Connected·주소 할당도 성공했다. 같은 성공을 계속 미확인으로 되돌리지 않는다.
2. argv1의 인증서 실패는 timeout 뒤 `zero serverNoiseKey`다. 정확한 첫 timeout 원인과 단말에서 키가 비게 된 순서는 미확인이다. DNS 요청 전 키 준비 누락 경로는 원본 소스로 재현했고 새 후보에 방어 수정을 넣었다. 실제 ACME 발급 성공은 아직 아니다.
3. `ERR-BuildInfo`는 VCS 정보와 스탬프가 모두 빠진 빌드로 별도 재현했다. 공식 Long/Short 스탬프와 커스텀 sourceId를 넣었다. 새 후보의 실제 관리 화면 표시는 아직 확인하지 않았다.
4. 기존 Android 앱 내장 Tailscale은 R2 이전부터 사용자 실사용이 정상이고 R2도 정상이다. Quick Tunnel도 사용자가 11-04 실사용했다. 이번 Exp 후보의 미검증 범위와 혼동하지 않는다.

## 새 산출물

- `prebuilt/termux/arm64/tsnet-v1.104.5-cert1/libpocketrisu_tsnet_exec.so`
- 21,889,318 bytes; SHA-256 `9e1e30367c8a361b92b38e0085f6bb67492d4e4b5b4be915fc6a887e8b65c5bb`
- Go 1.27.3, Tailscale v1.104.5. Long `1.104.5-pocketrisu-exp-cert1-s0db6aff6de9f`
- 출처/소스 묶음/라이선스/입력 해시는 같은 폴더와 `native/tsnet/SOURCE.json`에 있다.
- 실제 vendor는 Git에서 제외하고 대응 소스 archive로 복원한다. `prepare-tsnet-source.py`는 기존 수정을 덮어쓰지 않는다.

## 다음 단계

사용자가 새 후보를 시험할 때에는 위 설명의 검증 스크립트를 쓴다. 기존 로그인에 성공한 상태 디렉터리와 호스트명을 그대로 사용하고, 같은 state에 두 프로세스를 동시에 띄우지 않는다. 상태 초기화·장치 삭제·불필요한 재로그인은 하지 않는다. 실제 상태 파일이나 계정 자료를 수집하지 않는다.

결과에서는 `binaryVerified`, `probes.build-info`, `https.statuses`, `https.verification`, 전체 `ok`를 본다. `TLSVerified=true`와 200–499, 정상 종료가 모두 있어야 자체 HTTPS 시험 성공이다. 별도 기기 접속 성공은 따로 사용자 결과를 받아야 한다. `zero-noise-key`가 재발하면 새 후보의 sourceId/hash를 먼저 확인하고, whitelist 상태 순서로 재조사한다. 도메인/TXT/키를 포함한 원문 로그를 요구하지 않는다.

`deadline`은 그 자체로 이전 zero-key와 같은 원인이라는 뜻이 아니다. 소스 패치가 오직 키 누락 경로를 고쳤다는 점을 유지한다. `certificate`·`http-response`·`https-request`는 다른 단계의 실패로 분리한다. 외부 계정·CI·추가 온라인 시험은 새로 승인된 범위인지 먼저 확인한다.

## 보존 및 별도 작업

main·Quick Tunnel 가지·v1.104.4·argv1 후보는 그대로다. Quick Tunnel 설치 시 Y/N, Node/UI 통합, Android APK 재빌드·교체는 이번 작업이 아니다. 새 후보의 `--require-registered`·`--verify-https`는 opt-in이고 기본 실행 경로를 바꾸지 않는다. 기존 프록시 인증·WebSocket·스트리밍과 일반 종료는 회귀 시험으로 확인했다. 실기기 네트워크 전환·중지 후 재시작·신원 유지의 새 후보 결과는 아직 없다.

푸시·커밋·원격 CI·병합은 별개다. 이 문서가 존재하는 커밋은 Git 로그에서 확인하고, 로컬 보고서의 실제 원격 결과와 대조한다. main 병합이나 CI 통과를 추정하지 않는다.
