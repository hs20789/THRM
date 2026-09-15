package coreapp

import (
	"fmt"

	"github.com/TIANLI0/THRM/internal/appmeta"
	hotkeysvc "github.com/TIANLI0/THRM/internal/hotkey"
	"github.com/TIANLI0/THRM/internal/ipc"
	"github.com/TIANLI0/THRM/internal/plugins/fnqpowermode"
	"github.com/TIANLI0/THRM/internal/types"
)

func (a *CoreApp) registerPlugins() {
	if a.pluginManager == nil {
		return
	}
	if !a.legionFnQRegistered.CompareAndSwap(false, true) {
		return
	}

	a.pluginManager.Register(fnqpowermode.New(fnqpowermode.Options{
		Logger: a.logger,
		OnModeChange: func(state fnqpowermode.PowerModeState) {
			a.handleLegionPowerModeChange(state)
		},
	}))
}

func (a *CoreApp) applyCachedLegionFnQSupport(cfg *types.AppConfig) bool {
	if cfg == nil || !cfg.LegionFnQSupport.Checked {
		return false
	}

	a.legionFnQSupportChecked.Store(true)
	a.legionFnQSupported.Store(cfg.LegionFnQSupport.Supported)
	a.logInfo("Lenovo Legion Fn+Q host support loaded from config cache: supported=%v", cfg.LegionFnQSupport.Supported)

	if cfg.LegionFnQSupport.Supported {
		a.registerPlugins()
		return false
	}

	return a.normalizeLegionFnQConfigForHost(cfg)
}

func (a *CoreApp) startLegionFnQSupportDetection() {
	a.safeGo("detectLegionFnQSupport", func() {
		supported, hostInfo, err := fnqpowermode.DetectSupport()
		if err != nil {
			a.logError("failed to detect Lenovo Legion Fn+Q host support: %v", err)
			return
		}

		a.cacheLegionFnQSupportResult(supported)
		a.legionFnQSupportChecked.Store(true)
		if !supported {
			a.logInfo("Lenovo Legion Fn+Q plugin skipped: unsupported host (manufacturer=%s model=%s family=%s product=%s)",
				hostInfo.Manufacturer, hostInfo.Model, hostInfo.Family, hostInfo.Product)
			a.disableLegionFnQConfigForUnsupportedHost()
			a.broadcastLegionFnQSupportUpdate(false)
			return
		}

		a.registerPlugins()
		a.legionFnQSupported.Store(true)
		a.broadcastLegionFnQSupportUpdate(true)

		a.mutex.RLock()
		cfg := a.configManager.Get()
		a.mutex.RUnlock()
		a.applyPluginConfig(cfg)
	})
}

func (a *CoreApp) cacheLegionFnQSupportResult(supported bool) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	cfg := a.configManager.Get()
	if cfg.LegionFnQSupport.Checked && cfg.LegionFnQSupport.Supported == supported {
		return
	}

	cfg.LegionFnQSupport = types.LegionFnQSupportCache{
		Checked:   true,
		Supported: supported,
	}
	a.configManager.Set(cfg)
	if err := a.configManager.Save(); err != nil {
		a.logError("保存 Lenovo Legion Fn+Q 支持缓存失败: %v", err)
	}
	if a.ipcServer != nil {
		a.ipcServer.BroadcastEvent(ipc.EventConfigUpdate, cfg)
	}
}

func (a *CoreApp) disableLegionFnQConfigForUnsupportedHost() {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	cfg := a.configManager.Get()
	if !a.normalizeLegionFnQConfigForHost(&cfg) {
		return
	}

	a.configManager.Set(cfg)
	if err := a.configManager.Save(); err != nil {
		a.logError("保存 Lenovo Legion Fn+Q 配置失败: %v", err)
	}
	if a.ipcServer != nil {
		a.ipcServer.BroadcastEvent(ipc.EventConfigUpdate, cfg)
	}
}

func (a *CoreApp) broadcastLegionFnQSupportUpdate(supported bool) {
	if a.ipcServer == nil {
		return
	}
	a.ipcServer.BroadcastEvent(ipc.EventLegionFnQSupportUpdate, map[string]any{
		"supported": supported,
	})
}

func (a *CoreApp) handleLegionPowerModeChange(state fnqpowermode.PowerModeState) {
	if !a.legionFnQSupported.Load() {
		return
	}

	a.logInfo("Lenovo Legion Fn+Q power mode changed: raw=%d mapped=%d mode=%s source=%s",
		state.Raw, state.Mapped, state.Mode, state.Source)

	if a.ipcServer != nil {
		a.ipcServer.BroadcastEvent(ipc.EventLegionPowerModeUpdate, state)
	}

	a.applyLegionFnQFanMapping(state)
}

func (a *CoreApp) applyPluginConfig(cfg types.AppConfig) {
	if a.pluginManager == nil || !a.legionFnQSupported.Load() {
		return
	}

	if cfg.LegionFnQ.Enabled {
		if err := a.pluginManager.Start(fnqpowermode.PluginID); err != nil {
			a.logError("failed to start Lenovo Legion Fn+Q plugin: %v", err)
		}
		return
	}

	if err := a.pluginManager.Stop(fnqpowermode.PluginID); err != nil {
		a.logError("failed to stop Lenovo Legion Fn+Q plugin: %v", err)
	}
}

func (a *CoreApp) applyLegionFnQFanMapping(state fnqpowermode.PowerModeState) {
	cfg := a.configManager.Get()
	cfg.LegionFnQ = types.NormalizeLegionFnQConfig(cfg.LegionFnQ)
	if !cfg.LegionFnQ.Enabled || !cfg.LegionFnQ.TakeOverFan {
		return
	}
	if !a.isConnected {
		a.logDebug("Lenovo Legion Fn+Q takeover skipped: device not connected")
		return
	}

	target, ok := cfg.LegionFnQ.ModeMapping[state.Mode]
	if !ok {
		a.logDebug("Lenovo Legion Fn+Q takeover skipped: no mapping for mode=%s", state.Mode)
		return
	}

	if cfg.AutoControl || cfg.CustomSpeedEnabled {
		cfg.AutoControl = false
		cfg.CustomSpeedEnabled = false
		a.configManager.Set(cfg)
		if err := a.configManager.Save(); err != nil {
			a.logError("failed to save Lenovo Legion Fn+Q takeover config: %v", err)
		}
		if a.ipcServer != nil {
			a.ipcServer.BroadcastEvent(ipc.EventConfigUpdate, cfg)
		}
	}

	a.safeGo("applyLegionFnQFanMapping", func() {
		if ok := a.SetManualGear(target.Gear, target.Level); !ok {
			a.logError("Lenovo Legion Fn+Q takeover failed: mode=%s gear=%s level=%s", state.Mode, target.Gear, target.Level)
			return
		}
		a.logInfo("Lenovo Legion Fn+Q takeover applied: mode=%s gear=%s level=%s", state.Mode, target.Gear, target.Level)
	})
}

func (a *CoreApp) normalizeLegionFnQConfigForHost(cfg *types.AppConfig) bool {
	if cfg == nil || !a.legionFnQSupportChecked.Load() || a.legionFnQSupported.Load() {
		return false
	}

	changed := false
	if cfg.LegionFnQ.Enabled {
		cfg.LegionFnQ.Enabled = false
		changed = true
	}
	if cfg.LegionFnQ.TakeOverFan {
		cfg.LegionFnQ.TakeOverFan = false
		changed = true
	}

	return changed
}

func normalizeHotkeyConfig(cfg *types.AppConfig) bool {
	if cfg == nil {
		return false
	}

	changed := false
	if cfg.ManualGearToggleHotkey != "" {
		if _, _, err := hotkeysvc.ParseShortcut(cfg.ManualGearToggleHotkey); err != nil {
			cfg.ManualGearToggleHotkey = types.GetDefaultConfig(false).ManualGearToggleHotkey
			changed = true
		}
	}
	if cfg.AutoControlToggleHotkey != "" {
		if _, _, err := hotkeysvc.ParseShortcut(cfg.AutoControlToggleHotkey); err != nil {
			cfg.AutoControlToggleHotkey = types.GetDefaultConfig(false).AutoControlToggleHotkey
			changed = true
		}
	}
	if cfg.CurveProfileToggleHotkey != "" {
		if _, _, err := hotkeysvc.ParseShortcut(cfg.CurveProfileToggleHotkey); err != nil {
			cfg.CurveProfileToggleHotkey = types.GetDefaultConfig(false).CurveProfileToggleHotkey
			changed = true
		}
	}

	return changed
}

func normalizeManualGearMemoryConfig(cfg *types.AppConfig) bool {
	if cfg == nil {
		return false
	}

	changed := false
	if cfg.ManualGearLevels == nil {
		cfg.ManualGearLevels = map[string]string{}
		changed = true
	}

	for _, gear := range []string{"静音", "标准", "强劲", "超频"} {
		if level, ok := cfg.ManualGearLevels[gear]; !ok {
			cfg.ManualGearLevels[gear] = "中"
			changed = true
		} else {
			normalized := normalizeManualLevel(level)
			if normalized != level {
				cfg.ManualGearLevels[gear] = normalized
				changed = true
			}
		}
	}

	normalizedCurrent := normalizeManualLevel(cfg.ManualLevel)
	if normalizedCurrent != cfg.ManualLevel {
		cfg.ManualLevel = normalizedCurrent
		changed = true
	}

	if cfg.ManualGear != "" {
		if remembered, ok := cfg.ManualGearLevels[cfg.ManualGear]; !ok || remembered != normalizedCurrent {
			cfg.ManualGearLevels[cfg.ManualGear] = normalizedCurrent
			changed = true
		}
	}

	return changed
}

func (a *CoreApp) applyHotkeyBindings(cfg types.AppConfig) {
	if a.hotkeyManager == nil {
		return
	}
	if err := a.hotkeyManager.UpdateBindings(cfg.ManualGearToggleHotkey, cfg.AutoControlToggleHotkey, cfg.CurveProfileToggleHotkey); err != nil {
		a.logError("更新全局快捷键失败: %v", err)
	}
}

func (a *CoreApp) handleHotkeyAction(action hotkeysvc.Action, shortcut string) {
	a.safeGo("handleHotkeyAction", func() {
		var result hotkeyResult
		success := true

		switch action {
		case hotkeysvc.ActionToggleManualGear:
			res, err := a.toggleManualGearByHotkey()
			if err != nil {
				success = false
				result = hotkeyResult{Text: err.Error()}
			} else {
				result = res
			}
		case hotkeysvc.ActionToggleAutoMode:
			res, err := a.toggleAutoControlByHotkey()
			if err != nil {
				success = false
				result = hotkeyResult{Text: err.Error()}
			} else {
				result = res
			}
		case hotkeysvc.ActionToggleCurveProfile:
			res, err := a.toggleCurveProfileByHotkey()
			if err != nil {
				success = false
				result = hotkeyResult{Text: err.Error()}
			} else {
				result = res
			}
		default:
			success = false
			result = hotkeyResult{Text: "未知快捷键动作"}
		}

		message := result.Text

		if a.ipcServer != nil {
			// messageKey/messageParams 是新增字段，message 保持原样：旧版前端与系统通知
			// 继续用中文原文，认识 messageKey 的前端才走 i18n 分支。
			a.ipcServer.BroadcastEvent(ipc.EventHotkeyTriggered, map[string]any{
				"action":        string(action),
				"shortcut":      shortcut,
				"success":       success,
				"message":       message,
				"messageKey":    result.Key,
				"messageParams": result.Params,
			})
		}

		title := appmeta.AppName + " 快捷键"
		if !success {
			title = appmeta.AppName + " 快捷键失败"
		}
		if a.notifier != nil {
			a.notifier.Notify(title, message)
		}
	})
}

// hotkeyResult 是一次快捷键动作的结果。Key/Params 给前端做 i18n（核心服务不知道 GUI
// 当前语言），Text 是等价的中文原文，留给系统通知和不认识 messageKey 的旧版前端。
//
// Params 里的挡位/档级放的是 GearCommands 的原始键（"静音"、"中"），不是本地化文案：
// 那是设备协议值，前端拿到后用 getManualGearLabel/getManualLevelLabel 翻译再插值。
type hotkeyResult struct {
	Key    string
	Params map[string]any
	Text   string
}

func (a *CoreApp) toggleCurveProfileByHotkey() (hotkeyResult, error) {
	profile, err := a.CycleFanCurveProfile()
	if err != nil {
		return hotkeyResult{}, err
	}
	return hotkeyResult{
		Key:    types.HotkeyKeyCurveSwitched,
		Params: map[string]any{"profile": profile.Name},
		Text:   fmt.Sprintf("温控曲线已切换: %s", profile.Name),
	}, nil
}

func (a *CoreApp) toggleAutoControlByHotkey() (hotkeyResult, error) {
	cfg := a.configManager.Get()
	target := !cfg.AutoControl
	if err := a.SetAutoControl(target); err != nil {
		return hotkeyResult{}, err
	}
	if target {
		return hotkeyResult{Key: types.HotkeyKeySmartControlOn, Text: "智能变频已开启"}, nil
	}
	return hotkeyResult{Key: types.HotkeyKeySmartControlOff, Text: "智能变频已关闭"}, nil
}

func (a *CoreApp) toggleManualGearByHotkey() (hotkeyResult, error) {
	cfg := a.configManager.Get()

	if cfg.AutoControl {
		if err := a.SetAutoControl(false); err != nil {
			return hotkeyResult{}, fmt.Errorf("切换到手动模式失败: %w", err)
		}
	}

	nextGear, nextLevel := a.getNextManualGearWithMemory(cfg.ManualGear, cfg.ManualLevel)
	if ok := a.SetManualGear(nextGear, nextLevel); !ok {
		return hotkeyResult{}, fmt.Errorf("应用手动挡位失败")
	}

	rpm := cfg.ResolveGearRPM(nextGear, nextLevel)
	if rpm > 0 {
		return hotkeyResult{
			Key:    types.HotkeyKeyManualGearWithRPM,
			Params: map[string]any{"gear": nextGear, "level": nextLevel, "rpm": rpm},
			Text:   fmt.Sprintf("手动挡位: %s %s (%d RPM)", nextGear, nextLevel, rpm),
		}, nil
	}
	return hotkeyResult{
		Key:    types.HotkeyKeyManualGear,
		Params: map[string]any{"gear": nextGear, "level": nextLevel},
		Text:   fmt.Sprintf("手动挡位: %s %s", nextGear, nextLevel),
	}, nil
}

func (a *CoreApp) getNextManualGearWithMemory(currentGear, currentLevel string) (string, string) {
	sequence := []string{"静音", "标准", "强劲", "超频"}
	nextIndex := 0

	for i, gear := range sequence {
		if gear == currentGear {
			nextIndex = (i + 1) % len(sequence)
			break
		}
	}

	a.rememberManualGearLevel(currentGear, currentLevel)
	fallbackLevel := normalizeManualLevel(currentLevel)
	level := a.getRememberedManualLevel(sequence[nextIndex], fallbackLevel)

	return sequence[nextIndex], level
}

func normalizeManualLevel(level string) string {
	if level == "低" || level == "中" || level == "高" {
		return level
	}
	return "中"
}

func cloneManualGearLevels(source map[string]string) map[string]string {
	cloned := map[string]string{}
	for _, gear := range []string{"静音", "标准", "强劲", "超频"} {
		if source == nil {
			cloned[gear] = "中"
			continue
		}
		cloned[gear] = normalizeManualLevel(source[gear])
	}
	return cloned
}

func (a *CoreApp) syncManualGearLevelMemory(cfg types.AppConfig) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.syncManualGearLevelMemoryLocked(cfg)
}

func (a *CoreApp) syncManualGearLevelMemoryLocked(cfg types.AppConfig) {

	if a.manualGearLevelMemory == nil {
		a.manualGearLevelMemory = map[string]string{}
	}

	defaultLevel := normalizeManualLevel(cfg.ManualLevel)
	for _, gear := range []string{"静音", "标准", "强劲", "超频"} {
		if fromCfg, ok := cfg.ManualGearLevels[gear]; ok {
			a.manualGearLevelMemory[gear] = normalizeManualLevel(fromCfg)
			continue
		}
		a.manualGearLevelMemory[gear] = defaultLevel
	}

	a.manualGearLevelMemory[cfg.ManualGear] = normalizeManualLevel(cfg.ManualLevel)
}

func (a *CoreApp) rememberManualGearLevel(gear, level string) {
	if gear != "静音" && gear != "标准" && gear != "强劲" && gear != "超频" {
		return
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.manualGearLevelMemory == nil {
		a.manualGearLevelMemory = map[string]string{}
	}
	a.manualGearLevelMemory[gear] = normalizeManualLevel(level)
}

func (a *CoreApp) getRememberedManualLevel(gear, fallback string) string {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	if a.manualGearLevelMemory == nil {
		return normalizeManualLevel(fallback)
	}
	if level, ok := a.manualGearLevelMemory[gear]; ok {
		return normalizeManualLevel(level)
	}
	return normalizeManualLevel(fallback)
}
