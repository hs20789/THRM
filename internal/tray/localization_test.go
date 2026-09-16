package tray

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/systray"
	"github.com/TIANLI0/THRM/internal/uilocale"
)

func TestLocaleRefreshWakesUnchangedStatusWithoutRestart(t *testing.T) {
	m := NewManager(testLogger{}, nil)
	atomic.StoreInt32(&m.initialized, 1)
	atomic.StoreInt32(&m.readyState, 1)
	atomic.StoreInt32(&m.instanceCount, 1)
	m.getStatus = func() Status { return Status{Connected: true, CPUTemp: 50} }
	done, stopped := make(chan struct{}), make(chan struct{})
	go func() { defer close(stopped); m.updateMenuStatus(done) }()
	defer func() { close(done); <-stopped }()
	for _, locale := range []uilocale.Snapshot{"ko-KR", "en-US", "ja-JP", "zh-CN", "ko-KR"} {
		m.SetLocale(locale)
		select {
		case render := <-m.uiQueue:
			render() // No native menu is needed to verify scheduling.
		case <-time.After(time.Second):
			t.Fatalf("%s waited for the five-second status timer", locale)
		}
		if m.localeSnapshot() != locale || atomic.LoadInt32(&m.instanceCount) != 1 || !m.IsReady() {
			t.Fatal("locale change restarted the tray")
		}
	}
}

func TestExistingMenuItemsAndClickChannelsSurviveLanguageChanges(t *testing.T) {
	// Unregistered menu items exercise systray's in-memory label update without
	// an OS tray. They do not have native IDs and update() performs no native call.
	item := func() *systray.MenuItem { return &systray.MenuItem{ClickedCh: make(chan struct{})} }
	m := NewManager(testLogger{}, nil)
	m.menuItems = &MenuItems{Show: item(), DeviceStatus: item(), CPUTemperature: item(), GPUTemperature: item(), CPUPower: item(), GPUPower: item(), FanSpeed: item(), CurveSelect: item(), AutoControl: item(), Quit: item()}
	m.menuItems.AutoControl.Check()
	m.menuItems.CPUTemperature.Disable()
	menu, clicks := m.menuItems, m.menuItems.Show.ClickedCh
	for _, tc := range []struct {
		locale     uilocale.Snapshot
		show, quit string
	}{
		{"ko-KR", "메인 창 표시", "종료"}, {"en-US", "Show main window", "Quit"},
		{"ja-JP", "メインウィンドウを表示", "終了"}, {"zh-CN", "显示主窗口", "退出"},
	} {
		m.applyStaticLabels(tc.locale)
		if !strings.Contains(menu.Show.String(), tc.show) || !strings.Contains(menu.Quit.String(), tc.quit) {
			t.Fatal("labels not updated")
		}
		if m.menuItems != menu || menu.Show.ClickedCh != clicks || !menu.AutoControl.Checked() || !menu.CPUTemperature.Disabled() {
			t.Fatal("menu behavior changed")
		}
	}
}

func TestLocalizedTooltipsAndProfileNames(t *testing.T) {
	status := Status{Connected: true, AutoControlState: true, CPUTemp: 50, GPUTemp: 40, CurrentRPM: 1700}
	for _, locale := range []uilocale.Snapshot{"ko-KR", "en-US", "ja-JP", "zh-CN"} {
		tooltip := localizedTrayTooltip(status, locale)
		if utf16Len(tooltip) > trayTooltipMaxUnits || !strings.Contains(tooltip, "50°C") {
			t.Fatalf("invalid tooltip %q", tooltip)
		}
		if locale == "ko-KR" && (!strings.Contains(tooltip, "스마트 제어") || !strings.Contains(tooltip, "팬 1700 RPM")) {
			t.Fatal(tooltip)
		}
	}
	ko := uilocale.Snapshot("ko-KR")
	if got := localizedTrayTooltip(Status{}, ko); got != "THRM - 장치 연결 안 됨" {
		t.Fatal(got)
	}
	if got := curveDisplayName(CurveOption{ID: "default", Name: "默认"}, ko); got != "기본" {
		t.Fatal(got)
	}
	if got := curveDisplayName(CurveOption{ID: "user", Name: "默认"}, ko); got != "默认" {
		t.Fatal("user name translated")
	}
	if got := curveDisplayName(CurveOption{ID: "user", Name: ""}, ko); got != "" {
		t.Fatal("blank user name replaced")
	}
}
