package notifier

import (
	"testing"

	"github.com/TIANLI0/THRM/internal/uilocale"
)

func TestNotificationUsesOneSnapshotAndPreservesRawDetail(t *testing.T) {
	store := uilocale.NewStore(t.TempDir())
	if err := store.Set("ko-KR"); err != nil {
		t.Fatal(err)
	}
	locale := store.Snapshot()
	if err := store.Set("en-US"); err != nil {
		t.Fatal(err)
	}
	var title, body string
	m := NewManager(testLogger{}, nil)
	m.send = func(gotTitle, gotBody string, _ any) error { title, body = gotTitle, gotBody; return nil }
	m.Notify(locale, locale.Text("nativeUI.notification.hotkey", map[string]any{"app": "THRM"}), locale.HotkeyMessage("store.hotkey.smartControlOn", nil, "智能变频已开启"))
	if title != "THRM 단축키" || body != "스마트 제어를 켰습니다" {
		t.Fatalf("mixed snapshot: %q / %q", title, body)
	}
	raw := " \nunknown diagnostic\n原始错误 {{detail}}\n "
	m.Notify(locale, "", locale.Error(raw))
	if title != "기능 설정이 변경되었습니다" || body != raw {
		t.Fatalf("fallback/detail: %q / %q", title, body)
	}
	m.Notify(locale, locale.Text("nativeUI.notification.hotkeyFailed", map[string]any{"app": "THRM"}), locale.Error("应用手动挡位失败"))
	if title != "THRM 단축키 실행 실패" || body != "수동 속도 단계를 적용하지 못했습니다" {
		t.Fatalf("failure not translated: %q / %q", title, body)
	}
}
