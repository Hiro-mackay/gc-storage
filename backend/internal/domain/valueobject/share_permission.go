package valueobject

import "errors"

var (
	ErrInvalidSharePermission = errors.New("invalid share permission")
)

// SharePermission は共有リンクの権限を表す値オブジェクト
type SharePermission string

const (
	SharePermissionRead  SharePermission = "read"
	SharePermissionWrite SharePermission = "write"
)

// NewSharePermission は文字列からSharePermissionを生成します
func NewSharePermission(permission string) (SharePermission, error) {
	p := SharePermission(permission)
	if !p.IsValid() {
		return "", ErrInvalidSharePermission
	}
	return p, nil
}

// IsValid は権限が有効かを判定します
func (p SharePermission) IsValid() bool {
	switch p {
	case SharePermissionRead, SharePermissionWrite:
		return true
	default:
		return false
	}
}

// String は文字列を返します
func (p SharePermission) String() string {
	return string(p)
}

// IsRead は読み取り権限かを判定します
func (p SharePermission) IsRead() bool {
	return p == SharePermissionRead
}

// IsWrite は書き込み権限かを判定します
func (p SharePermission) IsWrite() bool {
	return p == SharePermissionWrite
}

// CanDownload はダウンロード可能かを判定します
func (p SharePermission) CanDownload() bool {
	return p == SharePermissionRead || p == SharePermissionWrite
}

// CanUpload はアップロード可能かを判定します
func (p SharePermission) CanUpload() bool {
	return p == SharePermissionWrite
}

// Level は権限のレベルを返します
func (p SharePermission) Level() int {
	switch p {
	case SharePermissionWrite:
		return 2
	case SharePermissionRead:
		return 1
	default:
		return 0
	}
}

// Includes は指定された権限を含むかを判定します
func (p SharePermission) Includes(other SharePermission) bool {
	return p.Level() >= other.Level()
}
