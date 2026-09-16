// Package locales exposes the same translation resources used by the frontend.
package locales

import "embed"

//go:embed zh-CN/translation.json en-US/translation.json ja-JP/translation.json ko-KR/translation.json
var Resources embed.FS
