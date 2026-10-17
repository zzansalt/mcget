package models

import (
	"embed"
	"encoding/json"
	"fmt"
	"sync"
)

//go:embed locales/*.json
var localeFiles embed.FS

var (
	localeOnce sync.Once
	locales    map[string]map[string]string
)

func loadLocales() {
	locales = make(map[string]map[string]string)
	for _, lang := range []string{"en", "ko"} {
		data, err := localeFiles.ReadFile("locales/" + lang + ".json")
		if err != nil {
			continue
		}
		var values map[string]string
		if json.Unmarshal(data, &values) == nil {
			locales[lang] = values
		}
	}
}

func tr(state *AppState, key string, args ...any) string {
	localeOnce.Do(loadLocales)
	lang := "en"
	if state != nil && state.Language == "ko" {
		lang = "ko"
	}
	value := locales[lang][key]
	if value == "" {
		value = locales["en"][key]
	}
	if value == "" {
		value = key
	}
	if len(args) > 0 {
		return fmt.Sprintf(value, args...)
	}
	return value
}
