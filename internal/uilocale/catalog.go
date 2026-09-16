// Package uilocale localizes native UI without changing device configuration.
package uilocale

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/TIANLI0/THRM/frontend/src/app/locales"
)

const Default = "zh-CN"

var supported = []string{Default, "en-US", "ja-JP", "ko-KR"}
var catalogs = loadCatalogs()
var placeholder = regexp.MustCompile(`\{\{([a-zA-Z0-9_]+)\}\}`)

// Normalize follows frontend/src/app/lib/i18n.tsx.
func Normalize(value string) string {
	lower := strings.ToLower(value)
	switch {
	case strings.HasPrefix(lower, "zh"):
		return Default
	case strings.HasPrefix(lower, "en"):
		return "en-US"
	case strings.HasPrefix(lower, "ja"):
		return "ja-JP"
	case lower == "ko" || strings.HasPrefix(lower, "ko-"):
		return "ko-KR"
	default:
		return Default
	}
}

// Snapshot is immutable so a notification's title and body always use one locale.
type Snapshot string

func (s Snapshot) template(key string) (string, bool) {
	if text, ok := catalogs[Normalize(string(s))][key]; ok {
		return text, true
	}
	text, ok := catalogs[Default][key]
	return text, ok
}

func (s Snapshot) Text(key string, params map[string]any) string {
	template, ok := s.template(key)
	if !ok {
		return key
	}
	// One pass: diagnostic text and user names containing {{...}} stay literal.
	return placeholder.ReplaceAllStringFunc(template, func(token string) string {
		if value, ok := params[token[2:len(token)-2]]; ok {
			return fmt.Sprint(value)
		}
		return token
	})
}

func (s Snapshot) ProfileName(id, name string) string {
	if id == "default" && name == "默认" {
		return s.Text("displayErrors.defaultProfile", nil)
	}
	return name
}

func loadCatalogs() map[string]map[string]string {
	result := make(map[string]map[string]string)
	for _, locale := range supported {
		data, err := locales.Resources.ReadFile(locale + "/translation.json")
		if err != nil {
			panic(err) // Embedded resources are validated at build/test time.
		}
		var tree map[string]any
		if err := json.Unmarshal(data, &tree); err != nil {
			panic(err)
		}
		flat := make(map[string]string)
		var walk func(string, map[string]any)
		walk = func(prefix string, entries map[string]any) {
			for key, value := range entries {
				path := prefix + key
				switch v := value.(type) {
				case string:
					flat[path] = v
				case map[string]any:
					walk(path+".", v)
				}
			}
		}
		walk("", tree)
		result[locale] = flat
	}
	return result
}
