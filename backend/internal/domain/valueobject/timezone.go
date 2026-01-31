package valueobject

import (
	"fmt"
	"time"
)

// Timezone はタイムゾーンを表す値オブジェクトです
// IANA Time Zone Database形式 (例: "Asia/Tokyo", "America/New_York", "UTC")
type Timezone struct {
	value string
}

// NewTimezone は新しいTimezoneを作成します
// IANA Time Zone Database形式で検証されます
func NewTimezone(value string) (Timezone, error) {
	if value == "" {
		return Timezone{}, fmt.Errorf("timezone cannot be empty")
	}

	// time.LoadLocationでIANA形式を検証
	_, err := time.LoadLocation(value)
	if err != nil {
		return Timezone{}, fmt.Errorf("invalid timezone: %s", value)
	}

	return Timezone{value: value}, nil
}

// String はタイムゾーンを文字列で返します
func (t Timezone) String() string {
	return t.value
}

// Equals は2つのTimezoneが等しいかを判定します
func (t Timezone) Equals(other Timezone) bool {
	return t.value == other.value
}

// Location はtime.Locationを返します
func (t Timezone) Location() *time.Location {
	loc, _ := time.LoadLocation(t.value)
	return loc
}

// DefaultTimezone はデフォルトのタイムゾーン(Asia/Tokyo)を返します
func DefaultTimezone() Timezone {
	return Timezone{value: "Asia/Tokyo"}
}
