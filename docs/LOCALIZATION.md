## THRM Localization Architecture

기준일: 2026-09-16

## 목적

이 문서는 THRM fork에서 사용하는 localization 구조와 유지보수 원칙을 기록한다.

한국어 지원뿐 아니라 기존 `zh-CN`, `en-US`, `ja-JP` 동작을 유지하는 것이 목표다.

## 지원 Locale

    zh-CN
    en-US
    ja-JP
    ko-KR

Frontend는 `i18next`와 `react-i18next`를 사용한다.

주요 translation resource:

    frontend/src/app/locales/
    ├── zh-CN/translation.json
    ├── en-US/translation.json
    ├── ja-JP/translation.json
    └── ko-KR/translation.json

## 전체 Localization 구조

THRM에는 React UI와 Go native UI가 모두 존재한다.

Locale 흐름:

    Frontend locale
          │
          ├── React / i18next
          │
          └── Wails
               ↓
              IPC
               ↓
           Core UI locale
               │
               ├── System Tray
               └── OS Notification

Frontend에서 선택한 locale을 Core에도 전달한다.

이를 통해 React 외부에서 생성되는 tray와 notification도 같은 언어를 사용한다.

## Translation Resource 공유

가능하면 frontend와 Go native UI가 동일한 translation resource를 사용한다.

Go 전용 번역 내용을 별도 독립 dictionary로 복사해 관리하지 않는다.

Translation resource 변경 시 네 locale의 key 구조와 placeholder를 함께 확인한다.

## 표시값과 내부값 분리

Localization에서 가장 중요한 원칙이다.

    내부 원본 데이터
            ↓
    저장 / 장치 제어 / IPC
            ↓
        표시 계층
            ↓
      현재 locale 번역

사용자에게 보여주는 값만 번역한다.

Backend raw message, 장치 제어 값, config key, protocol identifier는 번역된 문자열로 덮어쓰지 않는다.

## 변경하면 안 되는 내부 값

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

이 값들은 장치 명령, config 저장 및 backend/frontend 계약에 사용될 수 있다.

따라서 내부 원문을 유지한다.

예:

    internal value:
    静音

    ko-KR display:
    저소음

내부 값 `静音` 자체는 변경하지 않는다.

## 시스템 기본 프로필

시스템 기본 profile의 원본 이름:

    默认

저장 원문을 `기본`으로 변경하지 않는다.

다음 조건으로 시스템 기본 profile임을 확실히 확인할 수 있을 때만 표시 단계에서 번역한다.

    id == "default"
    AND
    name == "默认"

예:

    stored:
    {id: "default", name: "默认"}

    ko-KR display:
    기본

사용자가 일반 profile의 이름을 직접 `默认`으로 지정한 경우에는 번역하지 않는다.

## 사용자 입력 이름

사용자가 직접 작성한 profile 이름은 localization 대상이 아니다.

특히 다음 종류의 경로에서는 번역된 display 값을 저장 API에 전달하지 않는다.

- Profile rename input
- Profile save API
- Profile fallback/save path

Display name과 stored name을 분리한다.

## Backend Runtime Message

Go backend가 생성하는 중국어 error가 frontend에 그대로 전달되면 ko-KR UI에 중국어가 섞일 수 있다.

현재 구조에서는 알려진 backend message를 display localization 계층에서 인식하고 locale에 맞게 표시한다.

기본 원칙:

    Known backend message
        ↓
    translation key/template
        ↓
    localized display

    Unknown backend message
        ↓
    raw fallback

Unknown error는 정보 손실 방지를 위해 원문을 유지한다.

알 수 없는 detail을 임의의 일반 오류로 덮어쓰지 않는다.

## Persistent UI

오류 banner나 status처럼 지속적으로 화면에 표시되는 UI에는 번역 완료 문자열을 canonical state로 저장하지 않는 것을 원칙으로 한다.

가능하면 다음 정보를 저장한다.

    raw message
    translation key
    translation params
    DisplayMessage descriptor

그리고 React render 시점에 현재 locale로 번역한다.

이를 통해 다음 동작이 가능하다.

    zh-CN에서 오류 발생
            ↓
    descriptor 저장
            ↓
    ko-KR로 언어 변경
            ↓
    같은 descriptor를 한국어로 다시 렌더링

## Toast

일부 toast는 생성 시점의 locale을 사용한다.

이미 생성된 일시적 toast를 언어 변경 후 다시 렌더링하지 않는 것은 현재 허용한다.

Persistent UI와 toast를 동일하게 처리할 필요는 없다.

## Core Error Detail

Backend error에 포함된 실제 진단 detail은 가능한 한 유지한다.

Multiline detail도 손실하지 않는다.

Unknown detail을 삭제해서 localization을 완성하려고 하지 않는다.

정보 보존이 번역 완전성보다 우선한다.

## System Tray

System tray는 React가 아니라 Go Core가 생성한다.

Frontend locale을 Wails → IPC → Core로 전달한다.

언어 변경 시 systray 자체를 재시작하지 않는다.

기존 menu item의 title과 tooltip을 갱신한다.

다음 상태는 유지해야 한다.

- Click handler
- Check state
- Visibility
- Current device state

Tray가 아직 생성되지 않은 상태에서 locale이 변경되면 다음 tray 생성 시 최신 locale을 사용한다.

## OS Notification

OS notification은 Go에서 생성된다.

알림 생성 시점의 Core UI locale snapshot을 사용해 제목과 본문을 번역한다.

이미 발송된 notification을 언어 변경 후 다시 번역할 필요는 없다.

알 수 없는 error detail은 그대로 보존한다.

## Core UI Locale 저장

UI locale은 장치 설정과 분리한다.

장치 `AppConfig`에 UI locale을 섞지 않는다.

Core가 GUI 없이 실행되더라도 마지막 언어를 복원할 수 있도록 UI 전용 locale storage를 사용한다.

GUI가 다시 연결되면 frontend의 현재 locale을 Core에 다시 동기화한다.

## Locale 초기화 주의

Frontend 초기 렌더의 임시 기본 locale이 사용자가 저장해둔 locale을 덮어쓰면 안 된다.

저장된 locale 초기화가 완료된 후 Core에 locale을 전달한다.

예:

    stored locale:
    ko-KR

    React initial fallback:
    zh-CN

이 경우 초기 `zh-CN` 값을 Core에 먼저 저장해서 기존 `ko-KR`을 덮어쓰면 안 된다.

## 새 Translation 추가 원칙

새 key를 추가할 때 다음 네 locale을 확인한다.

    zh-CN
    en-US
    ja-JP
    ko-KR

Interpolation placeholder도 동일하게 유지한다.

예:

    {{detail}}
    {{profile}}
    {{gear}}
    {{level}}
    {{rpm}}

## User-facing Chinese 검색 시 주의

Repository에서 중국어 문자열을 검색했다고 해서 모두 번역 대상은 아니다.

검색 결과를 최소한 다음으로 분류한다.

    A. 실제 사용자 UI
    B. 내부 데이터
    C. Protocol / identifier
    D. Comment / log
    E. User-provided data

특히 protocol과 config 값은 UI 중국어로 오인하지 않는다.

## 회귀 검증

Localization 변경 후 최소한 다음을 확인한다.

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

추가 확인:

- 4 locale key 구조
- Placeholder 일치
- 내부 protocol 값 변경 여부
- 기본 profile 원문 보존
- 사용자 profile 이름 보존
- Tray locale 동기화
- Notification locale 동기화
