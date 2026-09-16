package coreapp

import (
	"encoding/json"
	"testing"

	"github.com/TIANLI0/THRM/internal/ipc"
	"github.com/TIANLI0/THRM/internal/tray"
	"github.com/TIANLI0/THRM/internal/uilocale"
)

func TestSetUILocaleIPCWithoutDeviceOrAppConfig(t *testing.T) {
	dir := t.TempDir()
	a := &CoreApp{uiLocale: uilocale.NewStore(dir), trayManager: tray.NewManager(nil, nil)}
	for _, locale := range []string{"ko-KR", "en-US", "ja-JP", "zh-CN", "ko-KR"} {
		data, _ := json.Marshal(ipc.SetStringParams{Value: locale})
		response := a.handleIPCRequest(ipc.Request{Type: ipc.ReqSetUILocale, Data: data})
		if !response.Success {
			t.Fatal(response.Error)
		}
		if got := string(a.uiLocale.Snapshot()); got != locale {
			t.Fatal(got)
		}
		if got := string(uilocale.NewStore(dir).Snapshot()); got != locale {
			t.Fatal("restart did not restore locale")
		}
	}
}
