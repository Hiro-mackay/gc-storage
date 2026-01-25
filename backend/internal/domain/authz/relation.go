package authz

import (
	"errors"

	"github.com/google/uuid"
)

var (
	ErrInvalidRelationType = errors.New("invalid relation type")
	ErrInvalidSubjectType  = errors.New("invalid subject type")
	ErrInvalidObjectType   = errors.New("invalid object type")
)

// RelationType はリレーションの種類を表す型
type RelationType string

const (
	RelationOwner  RelationType = "owner"
	RelationMember RelationType = "member"
	RelationParent RelationType = "parent"
)

// NewRelationType は文字列からRelationTypeを生成します
func NewRelationType(r string) (RelationType, error) {
	rt := RelationType(r)
	if !rt.IsValid() {
		return "", ErrInvalidRelationType
	}
	return rt, nil
}

// IsValid はリレーションタイプが有効かを判定します
func (r RelationType) IsValid() bool {
	switch r {
	case RelationOwner, RelationMember, RelationParent:
		return true
	default:
		return false
	}
}

// String は文字列を返します
func (r RelationType) String() string {
	return string(r)
}

// SubjectType はサブジェクトの種類を表す型
type SubjectType string

const (
	SubjectTypeUser   SubjectType = "user"
	SubjectTypeGroup  SubjectType = "group"
	SubjectTypeFile   SubjectType = "file"
	SubjectTypeFolder SubjectType = "folder"
)

// NewSubjectType は文字列からSubjectTypeを生成します
func NewSubjectType(s string) (SubjectType, error) {
	st := SubjectType(s)
	if !st.IsValid() {
		return "", ErrInvalidSubjectType
	}
	return st, nil
}

// IsValid はサブジェクトタイプが有効かを判定します
func (s SubjectType) IsValid() bool {
	switch s {
	case SubjectTypeUser, SubjectTypeGroup, SubjectTypeFile, SubjectTypeFolder:
		return true
	default:
		return false
	}
}

// String は文字列を返します
func (s SubjectType) String() string {
	return string(s)
}

// ObjectType はオブジェクトの種類を表す型
type ObjectType string

const (
	ObjectTypeFile   ObjectType = "file"
	ObjectTypeFolder ObjectType = "folder"
	ObjectTypeGroup  ObjectType = "group"
)

// NewObjectType は文字列からObjectTypeを生成します
func NewObjectType(o string) (ObjectType, error) {
	ot := ObjectType(o)
	if !ot.IsValid() {
		return "", ErrInvalidObjectType
	}
	return ot, nil
}

// IsValid はオブジェクトタイプが有効かを判定します
func (o ObjectType) IsValid() bool {
	switch o {
	case ObjectTypeFile, ObjectTypeFolder, ObjectTypeGroup:
		return true
	default:
		return false
	}
}

// String は文字列を返します
func (o ObjectType) String() string {
	return string(o)
}

// Resource はリソースを表す構造体
type Resource struct {
	Type ResourceType
	ID   uuid.UUID
}

// NewResource は新しいResourceを生成します
func NewResource(resourceType ResourceType, id uuid.UUID) Resource {
	return Resource{
		Type: resourceType,
		ID:   id,
	}
}

// Tuple はZanzibar形式のタプルを表す構造体
// subject#relation@object の形式
type Tuple struct {
	SubjectType SubjectType
	SubjectID   uuid.UUID
	Relation    RelationType
	ObjectType  ObjectType
	ObjectID    uuid.UUID
}

// NewTuple は新しいTupleを生成します
func NewTuple(subjectType SubjectType, subjectID uuid.UUID, relation RelationType, objectType ObjectType, objectID uuid.UUID) Tuple {
	return Tuple{
		SubjectType: subjectType,
		SubjectID:   subjectID,
		Relation:    relation,
		ObjectType:  objectType,
		ObjectID:    objectID,
	}
}
