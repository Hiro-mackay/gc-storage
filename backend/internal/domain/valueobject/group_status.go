package valueobject

import "errors"

var (
	ErrInvalidGroupStatus = errors.New("invalid group status")
)

// GroupStatus はグループの状態を表す値オブジェクト
// Note: Groupは論理削除をサポートしないため、常に"active"のみ
type GroupStatus string

const (
	GroupStatusActive GroupStatus = "active"
)

// NewGroupStatus は文字列からGroupStatusを生成します
func NewGroupStatus(status string) (GroupStatus, error) {
	s := GroupStatus(status)
	if !s.IsValid() {
		return "", ErrInvalidGroupStatus
	}
	return s, nil
}

// IsValid は状態が有効かを判定します
func (s GroupStatus) IsValid() bool {
	return s == GroupStatusActive
}

// String は文字列を返します
func (s GroupStatus) String() string {
	return string(s)
}
