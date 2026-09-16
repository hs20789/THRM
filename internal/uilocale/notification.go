package uilocale

import "strings"

var gearKeys = map[string]string{
	"静音": "manualGear.gears.quiet", "标准": "manualGear.gears.standard",
	"强劲": "manualGear.gears.strong", "超频": "manualGear.gears.overclock",
}
var levelKeys = map[string]string{
	"低": "manualGear.levels.low", "中": "manualGear.levels.medium", "高": "manualGear.levels.high",
}

func (s Snapshot) HotkeyMessage(key string, params map[string]any, raw string) string {
	if _, ok := s.template(key); key == "" || !ok {
		return s.Error(raw)
	}
	display := make(map[string]any, len(params))
	for name, value := range params {
		display[name] = value
	}
	if gear, ok := params["gear"].(string); ok {
		if key, known := gearKeys[gear]; known {
			display["gear"] = s.Text(key, nil)
		}
	}
	if level, ok := params["level"].(string); ok {
		if key, known := levelKeys[level]; known {
			display["level"] = s.Text(key, nil)
		}
	}
	if name, ok := params["profile"].(string); ok {
		id, _ := params["profileId"].(string)
		display["profile"] = s.ProfileName(id, name)
	}
	return s.Text(key, display)
}

// Recognize only errors reachable from hotkey actions. The Chinese resource is
// the existing backend template; no second translation dictionary is maintained.
var hotkeyErrorKeys = []string{
	"displayErrors.unknownHotkey", "displayErrors.fixedSpeedSmartControl",
	"displayErrors.manualGearFailed", "nativeUI.notification.noCurves",
	"displayErrors.configDirectory", "displayErrors.configSave",
	"displayErrors.manualSwitchFailed", "displayErrors.curvePoints",
	"displayErrors.curveTemperatureOrder", "displayErrors.curveRpmOrder",
}

func (s Snapshot) Error(raw string) string { return s.formatError(raw, 0) }

func (s Snapshot) formatError(raw string, depth int) string {
	if depth >= 8 {
		return raw
	}
	for _, key := range hotkeyErrorKeys {
		template, _ := Snapshot(Default).template(key)
		if raw == template {
			return s.Text(key, nil)
		}
		prefix, suffix, hasDetail := strings.Cut(template, "{{detail}}")
		if hasDetail && strings.HasPrefix(raw, prefix) && strings.HasSuffix(raw, suffix) && len(raw) >= len(prefix)+len(suffix) {
			detail := raw[len(prefix) : len(raw)-len(suffix)]
			return s.Text(key, map[string]any{"detail": s.formatError(detail, depth+1)})
		}
	}
	return raw
}
