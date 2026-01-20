package valueobject

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

const (
	StorageKeyMaxBytes = 1024
)

var (
	ErrInvalidStorageKey = errors.New("invalid storage key")
)

// StorageKey はMinIO内のオブジェクトキーを表す値オブジェクト
// 形式: {file_id} (UUIDv4)
type StorageKey struct {
	value string
}

// NewStorageKey はファイルIDからStorageKeyを生成します
func NewStorageKey(fileID uuid.UUID) StorageKey {
	return StorageKey{
		value: fileID.String(),
	}
}

// NewStorageKeyFromString は文字列からStorageKeyを生成します
func NewStorageKeyFromString(key string) (StorageKey, error) {
	// UUIDとしてパース可能か検証
	if _, err := uuid.Parse(key); err != nil {
		return StorageKey{}, fmt.Errorf("%w: %v", ErrInvalidStorageKey, err)
	}

	if len(key) > StorageKeyMaxBytes {
		return StorageKey{}, fmt.Errorf("%w: key too long", ErrInvalidStorageKey)
	}

	return StorageKey{value: key}, nil
}

// Value はキー文字列を返します
func (k StorageKey) Value() string {
	return k.value
}

// FileID はファイルIDを取得します
func (k StorageKey) FileID() (uuid.UUID, error) {
	return uuid.Parse(k.value)
}

// String はキー文字列を返します（Stringerインターフェース）
func (k StorageKey) String() string {
	return k.value
}

// IsEmpty はキーが空かどうかを判定します
func (k StorageKey) IsEmpty() bool {
	return k.value == ""
}

// ThumbnailKey はサムネイル用のキーを返します
func (k StorageKey) ThumbnailKey(size string) string {
	return fmt.Sprintf("%s/thumbnails/%s", k.value, size)
}
