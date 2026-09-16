package tray

import (
	"fyne.io/systray"
	"github.com/TIANLI0/THRM/internal/appmeta"
	"github.com/TIANLI0/THRM/internal/uilocale"
)

func (m *Manager) localeSnapshot() uilocale.Snapshot {
	if value := m.locale.Load(); value != nil {
		return value.(uilocale.Snapshot)
	}
	return uilocale.Snapshot(uilocale.Default)
}

// SetLocale wakes the status worker without replacing menu items or their channels.
// A pending signal also covers updates received before the tray becomes ready.
func (m *Manager) SetLocale(locale uilocale.Snapshot) {
	locale = uilocale.Snapshot(uilocale.Normalize(string(locale)))
	if m.localeSnapshot() == locale {
		return
	}
	m.locale.Store(locale)
	select {
	case m.localeChanged <- struct{}{}:
	default:
	}
}

func (m *Manager) applyStaticLabels(locale uilocale.Snapshot) {
	items := m.menuItems
	for _, entry := range []struct {
		item *systray.MenuItem
		key  string
	}{
		{items.Show, "show"}, {items.DeviceStatus, "device"},
		{items.CPUTemperature, "cpuTemperature"}, {items.GPUTemperature, "gpuTemperature"},
		{items.CPUPower, "cpuPower"}, {items.GPUPower, "gpuPower"},
		{items.FanSpeed, "fanSpeed"}, {items.CurveSelect, "curveSelect"},
		{items.AutoControl, "autoControl"}, {items.Quit, "quit"},
	} {
		if entry.item == nil {
			continue
		}
		entry.item.SetTitle(locale.Text("nativeUI.tray."+entry.key, nil))
		entry.item.SetTooltip(locale.Text("nativeUI.tray."+entry.key+"Tooltip", nil))
	}
}

func localizedTrayTooltip(status Status, locale uilocale.Snapshot) string {
	key := "nativeUI.tray.deviceDisconnected"
	if status.Connected {
		key = "nativeUI.tray.manualMode"
		if status.AutoControlState {
			key = "nativeUI.tray.smartControlActive"
		}
	}
	header := locale.Text(key, map[string]any{"app": appmeta.AppName})
	if !status.Connected {
		return header
	}
	return buildTrayTooltip(header, formatTooltipReadings(status, locale))
}

func curveDisplayName(option CurveOption, locale uilocale.Snapshot) string {
	return locale.ProfileName(option.ID, option.Name)
}
