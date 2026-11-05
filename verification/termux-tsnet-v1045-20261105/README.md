# tsnet v1.104.5 인자 수정 후보 검증

2026-11-05, Exp 기준 `b3bcd71632370208733fd63f5b02561eb1924c7c`에서 별도 후보를 만들었다. 정상 Tailscale 시작·로그인·등록·Termux 실행·외부 서비스 접속·원격 CI는 하지 않았다.

## 통과한 확인

- Go 1.27.3으로 Android ARM64 `CGO_ENABLED=0` 후보 빌드. SHA-256 `49c998c1a1701340d0855068f8f5d42126ff948dadd9998cd44d32438658662e`, 21,758,246 bytes.
- ELF64 LE AArch64 ET_DYN/PIE, interpreter `/system/bin/linker64`, `DT_NEEDED` 0개, 네 LOAD 정렬 각각 65,536 bytes.
- 내장 Go buildinfo에 Tailscale v1.104.5, local patched source, android/arm64 및 CGO=0 확인. Android localapi 파일 목록에 ACME `cert.go` 포함, disabled stub 제외.
- 정확한 공개 모듈 메타데이터 확보 후 `go mod verify`: `all modules verified`. go.mod/go.sum 빌드 전후 불변.
- 새 Linux network namespace에서 loopback만 활성화, 외부 인터페이스 및 IPv4/IPv6 기본 경로 없음. 여기서 호스트 race 시험 15개와 CLI 사례 14개 통과. Android 바이너리는 실행하지 않음.
- 인자 정규화의 하위 사례 11개: 절대/상대 marker, marker 없음, Linux, 다른 파일의 같은 이름, 디렉터리, 상대 첫 인자, 일반 옵션, 하나만 제거, 인자 없음, symlink로 같은 파일.
- 기존 프록시의 인증·Origin·HTTPS 인증서 검증·스트리밍·WebSocket·로컬 인증서 API 시험 유지. 합성 데이터와 loopback 서버만 사용.
- CLI의 help 0 / unknown 2 / 잘못된 state 1 / 특수 exec 안전 거부 2 / check-args 0 / 진단 0 및 인자 값 비노출 확인. state 미생성 및 기존 합성 state 내용·mode 불변.
- 후보와 배포 파일/소스 archive 전 항목 해시 일치. 기존 Exp 및 APK의 v1.104.4 해시 `c9b5c75f4c9792b79b1f840986582289d2ff1ad822e88fc2206b1a551fe5a71b` 유지.
- worker 검토: 링크된 외부 모듈 27개 경로·버전 및 원문으로 계산한 Go h1 모두 기존 대응소스와 일치. 라이선스 색인 54개 항목과 모듈 원문 사본 일치. 전체 법률 검토는 아님.
- 바이너리의 private-key material, Tailscale auth/API key, GitHub/AWS token, private user path 표식 검사 0건. 패턴 검사는 모든 민감정보의 부재를 입증하지 않음. 배포 입력은 공개 소스와 지정 빌드 산출물로 제한, 사용자 state/세션 자료 미사용.

## 실패 기록과 검증 한계

첫 후보 빌드 후 offline `go mod verify`는 memconn 메타데이터 부재로 실패했다. 그 산출물과 로그는 보존하고 배포하지 않았다. 이후 파서 오류에 입력값이 그대로 나타나는 문제를 수정하고 새 산출물에 대해 전체 호스트 시험과 정적 검증을 했다. 최초 소스 포장 검사에서는 `cert.go` 파일 목록의 대괄호를 처리하지 않은 검사기 assert가 실패했다. 대상 산출물 변경 없이 검사기를 고쳐 실제 포함 여부를 확인했다.

소스 복원 도우미는 기존 파일을 덮어쓰지 않고 source 하위 symlink도 거부한다. worker가 지적한 목표 폴더 밖 symlink 쓰기 가능성을 차단했다.

사용자 단말의 실제 argv는 확보하지 않았다. Termux exec 소스/공식 이슈와 기존 오류의 일관성에 근거한 후보이며 실제 단말에서 해결됐다고 판단하지 않는다. 정상 Tailnet 연결과 `os.Executable()`에 의존하는 실제 cloudflared sibling 실행은 미확인이다.

빈 Go cache의 완전 오프라인 빌드, 독립 환경에서의 재현 빌드, 모든 의존성에 대한 보안 감사도 수행하지 않았다. 시험용 APK 생성이나 Android 앱 업데이트가 아니다.

사용법과 대응소스 구성은 [후보 안내](../../docs/ko/tsnet-termux-v1045-candidate.md), 고정 버전/출처는 [SOURCE.json](../../native/tsnet/SOURCE.json)에 있다. 원시 로컬 기록은 작업 PC `D:\# Develope\PocketRisu-Android\.local-dev\termux-tsnet-v1045_20261105\`에 보존한다.
