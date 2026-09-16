package uilocale

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestStoreRestoresLocaleWithoutGUIOrDeviceConfig(t *testing.T) {
	dir := t.TempDir()
	config := []byte(`{"manualGear":"静音","manualLevel":"中"}`)
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, config, 0600); err != nil {
		t.Fatal(err)
	}
	store := NewStore(dir)
	if store.Snapshot() != Snapshot(Default) {
		t.Fatal("new installation must retain Chinese fallback")
	}
	if err := store.Set("ko-KR"); err != nil {
		t.Fatal(err)
	}
	if got := NewStore(dir).Snapshot(); got != "ko-KR" {
		t.Fatalf("restart locale = %q", got)
	}
	data, _ := os.ReadFile(path)
	if string(data) != string(config) {
		t.Fatal("device config changed")
	}
	if err := os.WriteFile(filepath.Join(dir, "ui-locale.json"), []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if NewStore(dir).Snapshot() != Snapshot(Default) {
		t.Fatal("corrupt preference did not fall back")
	}
}

func TestFailedSaveKeepsPreviousPreference(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	if err := store.Set("ko-KR"); err != nil {
		t.Fatal(err)
	}
	store.path = filepath.Join(dir, "blocked", "ui-locale.json")
	if err := os.WriteFile(filepath.Join(dir, "blocked"), []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := store.Set("en-US"); err == nil {
		t.Fatal("expected storage error")
	}
	if store.Snapshot() != "ko-KR" || NewStore(dir).Snapshot() != "ko-KR" {
		t.Fatal("failed write changed locale")
	}
}

func TestNativeKeysAndPlaceholdersMatch(t *testing.T) {
	for key, source := range catalogs[Default] {
		if !strings.HasPrefix(key, "nativeUI.") {
			continue
		}
		for _, locale := range supported {
			translated, ok := catalogs[locale][key]
			if !ok {
				t.Fatalf("%s missing %s", locale, key)
			}
			if !reflect.DeepEqual(placeholder.FindAllString(source, -1), placeholder.FindAllString(translated, -1)) {
				t.Fatalf("%s %s placeholders differ", locale, key)
			}
		}
	}
}

func TestNormalizationMatchesFrontend(t *testing.T) {
	for input, want := range map[string]string{"": Default, "fr-FR": Default, "zh-TW": Default, "EN-gb": "en-US", "ja": "ja-JP", "ko": "ko-KR", "KO-kr": "ko-KR", "kok": Default} {
		if got := Normalize(input); got != want {
			t.Errorf("%s: %s != %s", input, got, want)
		}
	}
}

func TestHotkeyDisplayPreservesRawValuesAndNames(t *testing.T) {
	ko := Snapshot("ko-KR")
	params := map[string]any{"gear": "静音", "level": "中", "rpm": 1700}
	before := map[string]any{"gear": "静音", "level": "中", "rpm": 1700}
	got := ko.HotkeyMessage("store.hotkey.manualGearWithRpm", params, "raw")
	if got != "수동 속도 단계: 저소음 중간 (1700 RPM)" {
		t.Fatal(got)
	}
	if !reflect.DeepEqual(params, before) {
		t.Fatal("protocol params mutated")
	}
	if got := ko.HotkeyMessage("store.hotkey.curveSwitched", map[string]any{"profileId": "default", "profile": "默认"}, "raw"); got != "팬 커브를 전환했습니다: 기본" {
		t.Fatal(got)
	}
	for _, name := range []string{"默认", "新曲线", "方案1", "时段 1", "{{profile}}"} {
		if got := ko.ProfileName("user-id", name); got != name {
			t.Fatalf("user name changed: %q", got)
		}
	}
	if got := ko.ProfileName("default", "내 프로필"); got != "내 프로필" {
		t.Fatal(got)
	}
}

func TestKnownFailuresAndUnknownMultilineDetails(t *testing.T) {
	ko := Snapshot("ko-KR")
	if got := ko.Error("自定义转速模式下无法开启智能变频"); got != "고정 속도 모드에서는 스마트 제어를 켤 수 없습니다" {
		t.Fatal(got)
	}
	raw := " \n未知详细错误: {{detail}}\npermission denied\n "
	if got := ko.HotkeyMessage("unknown.future.key", nil, raw); got != raw {
		t.Fatalf("raw detail lost: %q", got)
	}
	prefix := "切换到手动模式失败: 保存配置到用户配置目录失败: "
	got := ko.Error(prefix + raw)
	if !strings.HasSuffix(got, raw) || strings.Contains(got, "切换到手动模式失败") {
		t.Fatalf("nested error = %q", got)
	}
	for _, key := range hotkeyErrorKeys {
		source, _ := Snapshot(Default).template(key)
		if got := Snapshot(Default).Error(source); got != source {
			t.Fatalf("Chinese changed: %q", got)
		}
	}
}
