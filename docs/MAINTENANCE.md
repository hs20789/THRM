## THRM Fork Maintenance Guide

기준일: 2026-09-16

## 목적

이 문서는 `hs20789/THRM` fork를 장기간 유지하기 위한 절차를 기록한다.

주요 유지보수 대상은 다음과 같다.

- Upstream 변경 반영
- Korean localization 유지
- Runtime message localization 유지
- System tray / OS notification 유지
- Linux packaging 유지
- Ubuntu/Fedora 회귀 검증

## Remote 구조

    origin
    └── hs20789/THRM
        개인 fork

    upstream
    └── TIANLI0/THRM
        원본 프로젝트

원칙:

- 일반 작업 결과는 `origin`에 push한다.
- `upstream`에는 직접 push하지 않는다.
- 특별한 이유가 없는 한 force push하지 않는다.
- 작업 전에 현재 branch와 working tree를 확인한다.

## 작업 시작 전 확인

    git branch --show-current
    git status
    git remote -v

문서와 실제 repository 상태가 다르면 실제 Git 상태와 코드를 우선한다.

## Upstream 변경 확인

    git fetch upstream

현재 branch에 없는 upstream commit 확인:

    git log --oneline HEAD..upstream/main

변경량이 크면 먼저 commit별 diff를 확인한다.

## Upstream 변경 병합

현재 주요 작업 branch:

    feat/korean-localization

기본 병합:

    git checkout feat/korean-localization
    git status
    git merge upstream/main

공유된 branch history를 불필요하게 다시 쓰지 않기 위해 기본적으로 merge를 우선한다.

## Merge Conflict 처리 원칙

충돌이 발생했다고 기존 fork 코드를 무조건 유지하지 않는다.

먼저 upstream에서 해당 기능이 어떻게 변경됐는지 확인한다.

특히 다음 영역을 주의한다.

    frontend i18n
    translation.json
    backend runtime message
    backend message templates
    Wails API
    IPC
    Core UI locale
    system tray
    OS notification
    build.sh
    Linux packaging
    GitHub Actions

Upstream에 동일하거나 개선된 구현이 추가됐다면 현재 fork 코드를 새 구조에 맞게 통합한다.

## Localization 유지보수

지원 locale:

    zh-CN
    en-US
    ja-JP
    ko-KR

Upstream merge 후 locale key 구조를 비교한다.

새 key가 추가되면 `ko-KR`에도 대응 key를 추가한다.

삭제된 upstream key는 필요 여부를 확인한다.

Interpolation placeholder도 비교한다.

예:

    {{name}}
    {{detail}}
    {{profile}}
    {{gear}}
    {{level}}
    {{rpm}}

Placeholder mismatch는 runtime UI 문제를 만들 수 있다.

## Backend Runtime Message 유지보수

알려진 backend message는 display localization 계층에서 현재 locale에 맞게 표시한다.

Unknown backend message는 정보 손실을 방지하기 위해 raw fallback을 사용한다.

Upstream에서 Go error 문자열이 변경되면 기존 template matcher가 더 이상 인식하지 못할 수 있다.

따라서 upstream merge 후 사용자 노출 error 문자열을 확인한다.

## 내부 Protocol 값

다음 값은 UI 번역 문자열로 간주하지 않는다.

    静音
    标准
    强劲
    超频

    低
    中
    高

    挡位工作模式
    自动模式(实时转速)

이 값들은 장치 명령, config 또는 backend/frontend 계약에 사용될 수 있다.

내부 원문은 변경하지 않는다.

사용자에게 표시할 때만 번역한다.

다음 항목도 임의로 변경하지 않는다.

- HID/BLE frame
- command byte
- checksum
- UUID
- VID/PID
- sensor key
- profile ID
- config key

## 사용자 데이터

사용자가 직접 입력한 profile 이름은 자동 번역하지 않는다.

시스템 기본 profile은 확실하게 식별되는 경우에만 표시 단계에서 번역한다.

현재 판단 조건:

    id == "default"
    AND
    name == "默认"

저장 원문은 변경하지 않는다.

## Frontend 검증

    cd frontend

    npx tsc --noEmit
    npm run build
    npm test

## Go 검증

    go build ./...
    go test ./internal/...

필요하면 관련 frontend Go package도 함께 검증한다.

로컬에 Go toolchain이 없으면 기존 검증 방식에 따라 Docker 환경을 사용할 수 있다.

GUI 관련 test가 X11 문제로 실패하면 코드 문제와 실행 환경 문제를 구분한다.

## Repository 검증

    git diff --check
    git status

Commit 전에는 실제 diff를 검토한다.

## Linux Package

현재 Linux package는 네 종류다.

    THRM-linux-amd64-portable.tar.gz
    thrm_<version>_amd64.deb
    thrm-bin-<version>-1-x86_64.pkg.tar.zst
    thrm-<version>-1.x86_64.rpm

Upstream merge 후 네 package가 모두 생성되는지 확인한다.

## RPM 검증

RPM metadata:

    rpm -qip ./thrm-<version>-1.x86_64.rpm

RPM 설치 파일:

    rpm -qlp ./thrm-<version>-1.x86_64.rpm

Fedora 설치:

    sudo dnf install ./thrm-<version>-1.x86_64.rpm

설치 후 무결성:

    rpm -V thrm

필요하면 binary dependency를 확인한다.

    ldd /usr/bin/thrm
    ldd /usr/bin/thrm-core

Unresolved library가 없어야 한다.

## Ubuntu Package 검증

동일 version `.deb` 재설치:

    sudo apt install --reinstall ./thrm_<version>_amd64.deb

확인:

- 앱 실행
- 한국어 UI
- System tray
- OS notification
- 기본 profile 표시
- Locale 유지

## Fedora Package 검증

    sudo dnf install ./thrm-<version>-1.x86_64.rpm

확인:

- 앱 실행
- Desktop entry
- Icon
- System tray
- 한국어 UI
- OS notification
- Bluetooth/HID
- Locale 유지
- Package 제거
- Package 재설치

## GitHub Actions

Workflow:

    Build and Release

기능 branch에서 테스트 artifact를 생성할 때 기본 설정:

    Branch: 작업 branch
    prerelease_tag: 빈칸
    publish_release: false

실행 대상 commit SHA가 현재 HEAD와 같은지 확인한다.

현재 Linux artifact에는 다음 네 종류가 포함돼야 한다.

    .tar.gz
    .deb
    .pkg.tar.zst
    .rpm

## 앱 Updater 주의

현재 fork에는 upstream 공식 버전에 없는 변경이 존재한다.

앱 내부 updater가 upstream 공식 Release를 설치하면 fork 변경이 사라질 수 있다.

Updater 동작을 확인하기 전까지는 fork에서 생성한 package를 직접 설치하는 방식을 기본으로 한다.

## 큰 작업 완료 후 문서 갱신

다음 문서를 확인한다.

    docs/PROJECT_STATUS.md
    docs/ROADMAP.md
    docs/MAINTENANCE.md
    docs/LOCALIZATION.md
    docs/RELEASE_CHECKLIST.md

현재 상태가 달라졌다면 `PROJECT_STATUS.md`를 우선 갱신한다.

앞으로 해야 할 작업이 달라졌다면 `ROADMAP.md`도 갱신한다.
