## THRM Fork Roadmap

기준일: 2026-09-16

## 프로젝트 목표

이 fork의 주요 목표는 THRM을 한국어 및 Linux 환경에서 안정적으로 사용할 수 있게 유지하는 것이다.

현재 한국어화와 Fedora RPM packaging까지 완료했다.

앞으로는 upstream 유지보수와 Linux package 회귀 검증을 중심으로 관리한다.

## Phase 1 — Korean Localization

상태: 완료

완료 범위:

- `ko-KR` locale 추가
- React/i18next 기반 메인 UI 한국어화
- 사용자 노출 중국어 제거
- Backend runtime message localization
- Persistent UI render-time localization
- 언어 변경 시 기존 오류/경고 재렌더링
- Hotkey 성공/실패 message localization
- Debug UI localization
- 기본 profile 표시 localization
- System tray localization
- OS notification localization
- Frontend locale → Wails → IPC → Core 전달
- Core UI locale 저장 및 복원
- Ubuntu `.deb` 실제 실행 검증

## Phase 2 — Fedora RPM Packaging

상태: 완료

Fedora/RHEL 계열용 RPM package를 추가했다.

생성 artifact:

    thrm-<version>-1.x86_64.rpm

현재 version 기준:

    thrm-3.7.0-1.x86_64.rpm

설치 방식:

    sudo dnf install ./thrm-3.7.0-1.x86_64.rpm

기존 package는 그대로 유지한다.

현재 Linux artifact:

    THRM-linux-amd64-portable.tar.gz
    thrm_<version>_amd64.deb
    thrm-bin-<version>-1-x86_64.pkg.tar.zst
    thrm-<version>-1.x86_64.rpm

## Phase 3 — Linux Packaging Validation

상태: 완료 및 지속 검증

RPM 구현 시 Fedora 42 환경에서 다음을 검증했다.

- RPM build
- `dnf install`
- Runtime dependency 자동 해결
- `rpm -V`
- `ldd`
- 설치 파일 목록
- Package metadata

GitHub Actions에서도 최종 Linux artifact에 다음 네 package가 함께 포함되는 것을 확인했다.

    .tar.gz
    .deb
    .pkg.tar.zst
    .rpm

향후 upstream 변경 후에도 동일한 검증을 반복한다.

## Phase 4 — Upstream Maintenance

상태: 지속 작업

원본 저장소:

    TIANLI0/THRM

Fork:

    hs20789/THRM

원본 프로젝트가 업데이트되면 fork에 변경사항을 반영한다.

기본 흐름:

    git fetch upstream
            ↓
    upstream 변경 확인
            ↓
    upstream/main 병합
            ↓
    merge conflict 해결
            ↓
    locale 변경 확인
            ↓
    frontend build/test
            ↓
    Go build/test
            ↓
    Linux package build
            ↓
    Ubuntu/Fedora 회귀 검증

특히 다음 영역을 확인한다.

- translation resource
- frontend i18n
- Backend error message
- Backend message template
- Wails API
- IPC
- Core UI locale
- System tray
- OS notification
- Linux packaging
- GitHub Actions

## Phase 5 — Localization Maintenance

상태: 지속 작업

지원 locale:

    zh-CN
    en-US
    ja-JP
    ko-KR

Upstream에서 UI가 변경되면 다음을 확인한다.

- 새 translation key
- 삭제된 translation key
- interpolation placeholder 변경
- Backend runtime message 변경
- Tray 문자열 변경
- Notification 문자열 변경

Interpolation 예:

    {{name}}
    {{detail}}
    {{profile}}
    {{gear}}
    {{level}}
    {{rpm}}

알 수 없는 backend 오류는 정보 손실 방지를 위해 raw fallback을 유지한다.

## Phase 6 — Packaging Maintenance

상태: 지속 작업

현재 Linux package:

    .deb
    .rpm
    .pkg.tar.zst
    portable tar.gz

Upstream build 구조나 dependency가 변경되면 네 package가 모두 정상 생성되는지 확인한다.

특히 RPM에서는 ELF 기반 automatic dependency generation이 실제 build 결과와 맞는지 확인한다.

새 library dependency가 추가될 경우 package manager가 올바른 dependency를 생성하는지 검증한다.

## Phase 7 — Updater Strategy

상태: 향후 검토

현재 fork에는 upstream 공식 버전에 없는 변경이 포함돼 있다.

앱 내부 updater가 upstream 공식 Release를 설치하면 fork의 한국어화 및 packaging 변경이 사라질 수 있다.

향후 확인 대상:

- Updater가 참조하는 repository
- Release URL
- Version 비교 방식
- Fork release 지원 가능 여부
- Upstream updater를 그대로 유지할지 여부

Updater 구조를 확인하기 전까지는 fork에서 생성한 package를 직접 설치하는 방식을 기본으로 한다.

## Phase 8 — Distribution

상태: 선택 사항

필요하면 개인 fork의 GitHub Release를 운영한다.

검토 대상:

- Release artifact 자동 생성
- `.deb`
- `.rpm`
- `.pkg.tar.zst`
- portable archive
- Release note 관리
- Version tagging

## Phase 9 — Upstream Contribution

상태: 선택 사항

현재 fork에서 구현한 기능 중 upstream에도 일반적으로 유용한 변경은 PR을 검토할 수 있다.

후보:

- `ko-KR` locale
- Korean translation
- Runtime localization 개선
- System tray localization
- OS notification localization
- Fedora RPM packaging

PR 전에는 개인 fork 전용 구현과 upstream에도 필요한 일반 기능을 구분한다.

## 현재 우선순위

    1. 현재 fork 상태 안정적으로 유지
    2. Upstream 새 version 모니터링
    3. Upstream merge 후 localization 회귀 검증
    4. Linux 4종 package 회귀 검증
    5. 필요 시 fork Release 운영
    6. 필요 시 upstream PR 준비

## 장기 완료 기준

다음 조건을 계속 유지하는 것을 목표로 한다.

- 한국어 UI 정상
- Backend runtime message localization 정상
- System tray 정상
- OS notification 정상
- 장치 protocol/config 계약값 불변
- `.deb` 정상 생성
- `.rpm` 정상 생성
- `.pkg.tar.zst` 정상 생성
- portable package 정상 생성
- Upstream 변경을 안정적으로 반영 가능
