package storage

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// OwnerType はオーナータイプを表します
type OwnerType string

const (
	OwnerTypeUser  OwnerType = "user"
	OwnerTypeGroup OwnerType = "group"
)

// StorageKey はMinIO内のオブジェクトキーを表します
type StorageKey struct {
	OwnerType OwnerType
	OwnerID   uuid.UUID
	FileID    uuid.UUID
	Version   int
}

// NewStorageKey は新しいStorageKeyを作成します
func NewStorageKey(ownerType OwnerType, ownerID, fileID uuid.UUID, version int) StorageKey {
	return StorageKey{
		OwnerType: ownerType,
		OwnerID:   ownerID,
		FileID:    fileID,
		Version:   version,
	}
}

// String はキー文字列を返します
// 形式: {owner_type}/{owner_id}/{file_id}/v{version}
func (k StorageKey) String() string {
	return fmt.Sprintf("%s/%s/%s/v%d", k.OwnerType, k.OwnerID, k.FileID, k.Version)
}

// ParseStorageKey はキー文字列をパースします
func ParseStorageKey(key string) (StorageKey, error) {
	parts := strings.Split(key, "/")
	if len(parts) != 4 {
		return StorageKey{}, fmt.Errorf("invalid storage key format: %s", key)
	}

	ownerID, err := uuid.Parse(parts[1])
	if err != nil {
		return StorageKey{}, fmt.Errorf("invalid owner_id: %w", err)
	}

	fileID, err := uuid.Parse(parts[2])
	if err != nil {
		return StorageKey{}, fmt.Errorf("invalid file_id: %w", err)
	}

	var version int
	if _, err := fmt.Sscanf(parts[3], "v%d", &version); err != nil {
		return StorageKey{}, fmt.Errorf("invalid version: %w", err)
	}

	return StorageKey{
		OwnerType: OwnerType(parts[0]),
		OwnerID:   ownerID,
		FileID:    fileID,
		Version:   version,
	}, nil
}

// ThumbnailKey はサムネイルのキーを返します
func (k StorageKey) ThumbnailKey(size string) string {
	return fmt.Sprintf("%s/%s/%s/thumbnails/%s/v%d", k.OwnerType, k.OwnerID, k.FileID, size, k.Version)
}

// Directory はキーのディレクトリ部分を返します
func (k StorageKey) Directory() string {
	return fmt.Sprintf("%s/%s/%s", k.OwnerType, k.OwnerID, k.FileID)
}
