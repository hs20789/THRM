## THRM Fork Project Status

기준일: 2026-09-16

## 프로젝트 개요

이 저장소는 `TIANLI0/THRM`을 기반으로 한 개인 fork다.

주요 목표는 다음과 같다.

- THRM의 한국어 지원
- Linux 환경에서의 안정적인 사용
- Ubuntu/Debian 계열 패키지 검증
- Fedora/RHEL 계열 RPM 지원
- Upstream 변경사항을 지속적으로 반영할 수 있는 유지보수 구조 확립

## 저장소 정보

- Upstream: `TIANLI0/THRM`
- Fork: `hs20789/THRM`
- 주요 작업 브랜치: `feat/korean-localization`
- 현재 THRM 버전: `3.7.0`

Remote 구조:

    origin   → git@github.com:hs20789/THRM.git
    upstream → git@github.com:TIANLI0/THRM.git

`origin`은 개인 fork이고 `upstream`은 원본 프로젝트다.

`upstream`에는 직접 push하지 않는다.

## 현재 프로젝트 상태

한국어 지원 작업과 Fedora RPM packaging 작업까지 완료했다.

현재 GitHub Actions의 Linux artifact에서는 다음 4가지 package를 생성한다.

    THRM-linux-amd64-portable.tar.gz
    thrm_3.7.0_amd64.deb
    thrm-bin-3.7.0-1-x86_64.pkg.tar.zst
    thrm-3.7.0-1.x86_64.rpm

GitHub Actions에서 최신 artifact에 네 package가 모두 포함되는 것을 확인했다.

## 완료된 한국어화 작업

다음 작업을 완료했다.

- `ko-KR` locale 추가
- React/i18next 기반 메인 GUI 한국어화
- 사용자에게 노출되는 frontend 하드코딩 중국어 제거
- Backend runtime 메시지의 표시 계층 localization
- Backend raw error/detail 보존
- Persistent UI render-time localization
- 언어 변경 시 기존 오류와 경고 재렌더링
- Hotkey 성공/실패 메시지 localization
- ControlPanel debug UI 한국어화
- AboutPanel 지원 언어 목록에 한국어 추가
- 시스템 기본 profile 표시 localization
- 사용자 입력 profile 이름 원문 보존
- System tray 한국어화
- OS notification 다국어화
- Frontend locale을 Wails → IPC → Core로 전달
- Core UI locale 저장 및 복원
- 언어 변경 시 tray 즉시 갱신
- HID/BLE/config 계약값 보존

## 시스템 기본 프로필 처리

시스템 기본 profile의 저장 원문은 다음과 같다.

    默认

이 값을 실제 저장 데이터에서 `기본`으로 변경하지 않는다.

시스템 기본 profile임을 확실히 확인할 수 있는 경우에만 표시 단계에서 번역한다.

현재 기본 판단 조건:

    id == "default"
    AND
    name == "默认"

ko-KR 화면에서는 다음과 같이 표시한다.

    기본

사용자가 직접 만든 profile 이름은 자동 번역하지 않는다.

## Ubuntu 검증 상태

Ubuntu에서 `.deb` package를 실제 설치하고 실행했다.

검증 완료 항목:

- THRM 실행
- 한국어 메인 UI
- Runtime 오류 및 경고 한국어 표시
- 기본 profile `기본` 표시
- System tray 한국어 표시
- Locale 전달 및 저장
- 동일 version `.deb` 재설치
- 최신 한국어화 변경사항 반영

동일 version의 package를 다시 설치할 때 사용한 방식:

    sudo apt install --reinstall ./thrm_3.7.0_amd64.deb

## Fedora RPM Packaging 상태

Fedora/RHEL 계열용 RPM packaging 구현을 완료했다.

추가된 주요 파일:

    packaging/linux/thrm.spec

GitHub Actions의 기존 Linux build에 RPM 생성 단계를 추가했다.

생성되는 RPM:

    thrm-3.7.0-1.x86_64.rpm

설치 방식:

    sudo dnf install ./thrm-3.7.0-1.x86_64.rpm

## RPM 검증 결과

RPM packaging 구현 과정에서 다음을 검증했다.

- RPM build 성공
- Fedora 42 환경에서 `dnf install` 성공
- Runtime dependency 자동 해결
- `rpm -V` 검증 성공
- `ldd` unresolved library 0
- `rpm -qip` metadata 확인
- `rpm -qlp` 설치 파일 확인
- 기존 `.deb` packaging 유지
- 기존 Arch package 유지
- 기존 portable package 유지
- GitHub Actions artifact에 `.rpm` 포함 확인

RPM 설치 경로는 기존 Linux package와 일관되게 유지한다.

주요 설치 파일:

    /usr/bin/thrm
    /usr/bin/thrm-core
    /usr/lib/udev/rules.d/99-flydigi-fan.rules
    /usr/share/applications/thrm.desktop
    /usr/share/icons/hicolor/256x256/apps/thrm.png
    /usr/share/licenses/thrm/LICENSE

## 현재 Linux Packaging 상태

현재 지원 package:

    Ubuntu / Debian 계열 → .deb
    Fedora / RHEL 계열   → .rpm
    Arch 계열            → .pkg.tar.zst
    일반 Linux           → portable tar.gz

## 주요 검증 명령

Frontend:

    cd frontend
    npx tsc --noEmit
    npm run build
    npm test

Go:

    go build ./...
    go test ./internal/...

Repository:

    git diff --check
    git status

## 최근 주요 커밋

    8d916f5 feat: add Fedora RPM packaging
    0e9a869 fix: localize default profile display
    78f1c11 feat: localize tray and system notifications
    964294e fix: localize remaining runtime UI messages
    044189c chore: remove hardcoded Chinese brand description
    1956835 refactor: polish Korean UI copy
    5407a64 fix: localize remaining user-facing Chinese text
    9ab5a3e feat: add Korean UI localization

## 중요한 내부 값 보존 원칙

다음 값은 단순한 중국어 UI 문자열이 아니다.

    静音
    标准
    强劲
    超频

    低
    中
    高

    挡位工作模式
    自动模式(实时转速)

이 값들은 장치 명령, config 또는 backend/frontend 계약에서 사용될 수 있다.

따라서 내부 값 자체는 변경하지 않는다.

사용자에게 표시할 때만 현재 locale에 맞는 label로 변환한다.

## 현재 프로젝트 단계

    Phase 1
    한국어화
    → 완료

    Phase 2
    Fedora RPM Packaging
    → 완료

    Phase 3
    Linux Package / Runtime 검증
    → 완료 및 지속 검증

    Phase 4
    Upstream 유지보수
    → 지속 작업

    Phase 5
    Distribution / Upstream Contribution
    → 필요 시 진행

현재부터는 새로운 기능 개발보다 upstream 변경사항 추적과 회귀 검증이 주요 유지보수 작업이다.
