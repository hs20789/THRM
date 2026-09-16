## THRM Fork Build / Release Checklist

기준일: 2026-09-16

## 목적

이 문서는 fork의 변경사항을 검증하고 Linux artifact를 생성할 때 사용하는 checklist다.

현재 Linux artifact는 네 종류다.

    THRM-linux-amd64-portable.tar.gz
    thrm_<version>_amd64.deb
    thrm-bin-<version>-1-x86_64.pkg.tar.zst
    thrm-<version>-1.x86_64.rpm

## 1. Git 상태 확인

현재 branch:

    git branch --show-current

Working tree:

    git status

Remote:

    git remote -v

확인:

    origin   → hs20789/THRM
    upstream → TIANLI0/THRM

`upstream`에는 직접 push하지 않는다.

## 2. Frontend 검증

    cd frontend

    npx tsc --noEmit
    npm run build
    npm test

모두 성공해야 한다.

## 3. Go 검증

Repository root에서:

    go build ./...
    go test ./internal/...

필요하면 관련 frontend Go package도 함께 검사한다.

## 4. Repository 검증

    git diff --check
    git status

Unintended file이나 generated temporary file이 남아 있지 않은지 확인한다.

## 5. Localization 검증

확인 대상:

- `zh-CN`
- `en-US`
- `ja-JP`
- `ko-KR`

확인:

- JSON syntax
- Duplicate key
- Translation key parity
- Interpolation placeholder
- User-facing Chinese
- Protocol/config 값 불변

다음 내부 값은 번역 대상으로 취급하지 않는다.

    静音
    标准
    强劲
    超频
    低
    中
    高
    挡位工作模式
    自动模式(实时转速)

## 6. GitHub Actions 실행

Workflow:

    Build and Release

기능 branch 테스트용 기본 설정:

    Branch: feat/korean-localization
    prerelease_tag: 빈칸
    publish_release: false

`publish_release`를 반드시 확인한다.

테스트 artifact 생성만 원하는 경우 `false`를 사용한다.

## 7. Workflow Commit 확인

GitHub Actions 실행 상세에서 build 대상 SHA가 현재 HEAD와 동일한지 확인한다.

로컬 HEAD:

    git rev-parse HEAD

GitHub Actions의 target commit과 비교한다.

## 8. Linux Artifact 확인

Actions 성공 후 `THRM-linux-<commit SHA>` artifact를 확인한다.

압축 안에 다음 네 파일이 있어야 한다.

    THRM-linux-amd64-portable.tar.gz
    thrm_<version>_amd64.deb
    thrm-bin-<version>-1-x86_64.pkg.tar.zst
    thrm-<version>-1.x86_64.rpm

하나라도 없으면 packaging workflow를 확인한다.

## 9. Debian Package 검증

Package:

    thrm_<version>_amd64.deb

설치:

    sudo apt install ./thrm_<version>_amd64.deb

동일 version 재설치:

    sudo apt install --reinstall ./thrm_<version>_amd64.deb

확인:

- 설치 성공
- 실행 성공
- Desktop entry
- Icon
- 한국어 UI
- System tray
- OS notification
- Locale 유지

## 10. RPM Metadata 검증

Package:

    thrm-<version>-1.x86_64.rpm

Metadata:

    rpm -qip ./thrm-<version>-1.x86_64.rpm

설치 파일:

    rpm -qlp ./thrm-<version>-1.x86_64.rpm

예상 주요 파일:

    /usr/bin/thrm
    /usr/bin/thrm-core
    /usr/lib/udev/rules.d/99-flydigi-fan.rules
    /usr/share/applications/thrm.desktop
    /usr/share/icons/hicolor/256x256/apps/thrm.png
    /usr/share/licenses/thrm/LICENSE

## 11. Fedora RPM 설치 검증

    sudo dnf install ./thrm-<version>-1.x86_64.rpm

확인:

- Dependency 해결
- 설치 성공
- Application 실행
- Desktop entry
- Icon
- System tray
- 한국어 UI
- OS notification
- Bluetooth/HID

## 12. RPM 설치 무결성

설치 후:

    rpm -V thrm

정상 package file에 예상하지 않은 변경이 없어야 한다.

## 13. Dynamic Library 확인

필요하면 다음을 확인한다.

    ldd /usr/bin/thrm
    ldd /usr/bin/thrm-core

`not found` 항목이 없어야 한다.

## 14. Arch Package

Package:

    thrm-bin-<version>-1-x86_64.pkg.tar.zst

RPM 변경 때문에 기존 `PKGBUILD`나 Arch packaging이 의도치 않게 변경되지 않았는지 확인한다.

## 15. Portable Package

Package:

    THRM-linux-amd64-portable.tar.gz

기존 portable artifact가 계속 생성되는지 확인한다.

## 16. Runtime Localization 검증

한국어 환경에서 확인:

- 메인 GUI
- Runtime 오류
- 기본 profile 표시
- System tray
- OS notification

기본 profile의 저장 원문:

    默认

ko-KR 표시:

    기본

저장 원문 자체는 변경하지 않는다.

## 17. Locale 변경 검증

가능하면 다음을 확인한다.

    zh-CN → ko-KR
    ko-KR → en-US
    en-US → ja-JP

Persistent UI와 tray가 현재 locale을 반영해야 한다.

이미 발생한 일시적 toast나 OS notification은 생성 당시 locale을 유지할 수 있다.

## 18. Package 제거 검증

Fedora:

    sudo dnf remove thrm

Ubuntu:

    sudo apt remove thrm

필요하면 재설치도 확인한다.

## 19. Release 여부 확인

테스트 목적이면:

    publish_release: false

공식 fork Release를 만들 계획이 있는 경우에만 release publishing을 별도로 검토한다.

Upstream 공식 Release와 혼동하지 않는다.

## 20. 최종 확인

다음을 확인한 후 작업을 완료한다.

- Git working tree 상태 확인
- Local HEAD 확인
- Origin SHA 확인
- Build/test 성공
- Linux artifact 4종 생성
- Ubuntu package 정상
- Fedora RPM 정상
- Localization 회귀 없음
- Protocol/config 값 변경 없음

큰 작업이 완료되면 다음 문서도 갱신한다.

    docs/PROJECT_STATUS.md
    docs/ROADMAP.md
    docs/MAINTENANCE.md
    docs/LOCALIZATION.md
    docs/RELEASE_CHECKLIST.md
