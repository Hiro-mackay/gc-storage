package valueobject

import "fmt"

// サポートするロケール
var supportedLocales = map[string]bool{
	"ja": true,
	"en": true,
}

// Locale はロケールを表す値オブジェクトです
type Locale struct {
	value string
}

// NewLocale は新しいLocaleを作成します
// サポートされるロケール: "ja", "en"
func NewLocale(value string) (Locale, error) {
	if value == "" {
		return Locale{}, fmt.Errorf("locale cannot be empty")
	}

	if !supportedLocales[value] {
		return Locale{}, fmt.Errorf("unsupported locale: %s (supported: ja, en)", value)
	}

	return Locale{value: value}, nil
}

// String はロケールを文字列で返します
func (l Locale) String() string {
	return l.value
}

// Equals は2つのLocaleが等しいかを判定します
func (l Locale) Equals(other Locale) bool {
	return l.value == other.value
}

// IsJapanese はロケールが日本語かを判定します
func (l Locale) IsJapanese() bool {
	return l.value == "ja"
}

// IsEnglish はロケールが英語かを判定します
func (l Locale) IsEnglish() bool {
	return l.value == "en"
}

// DefaultLocale はデフォルトのロケール(ja)を返します
func DefaultLocale() Locale {
	return Locale{value: "ja"}
}
