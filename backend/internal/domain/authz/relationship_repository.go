package authz

import (
	"context"

	"github.com/google/uuid"
)

// RelationshipRepository はリレーションシップリポジトリのインターフェース
type RelationshipRepository interface {
	// 基本CRUD
	Create(ctx context.Context, rel *Relationship) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByTuple(ctx context.Context, tuple Tuple) error

	// 存在チェック
	Exists(ctx context.Context, tuple Tuple) (bool, error)

	// サブジェクト検索（object から subject を探す）
	FindSubjects(ctx context.Context, objectType ObjectType, objectID uuid.UUID, relation RelationType) ([]Resource, error)

	// オブジェクト検索（subject から object を探す）
	FindObjects(ctx context.Context, subjectType SubjectType, subjectID uuid.UUID, relation RelationType, objectType ObjectType) ([]uuid.UUID, error)

	// オブジェクトでの検索
	FindByObject(ctx context.Context, objectType ObjectType, objectID uuid.UUID) ([]*Relationship, error)

	// サブジェクトでの検索
	FindBySubject(ctx context.Context, subjectType SubjectType, subjectID uuid.UUID) ([]*Relationship, error)

	// 親リソース検索
	FindParent(ctx context.Context, objectType ObjectType, objectID uuid.UUID) (*Resource, error)

	// 一括削除
	DeleteByObject(ctx context.Context, objectType ObjectType, objectID uuid.UUID) error
	DeleteBySubject(ctx context.Context, subjectType SubjectType, subjectID uuid.UUID) error
}
