package authz

import (
	"time"

	"github.com/google/uuid"
)

// Relationship はZanzibar形式のリレーションシップを表すエンティティ
// 例: user:123#owner@folder:456 (user 123 is owner of folder 456)
type Relationship struct {
	ID          uuid.UUID
	SubjectType SubjectType
	SubjectID   uuid.UUID
	Relation    RelationType
	ObjectType  ObjectType
	ObjectID    uuid.UUID
	CreatedAt   time.Time
}

// NewRelationship は新しいRelationshipを生成します
func NewRelationship(
	subjectType SubjectType,
	subjectID uuid.UUID,
	relation RelationType,
	objectType ObjectType,
	objectID uuid.UUID,
) *Relationship {
	return &Relationship{
		ID:          uuid.New(),
		SubjectType: subjectType,
		SubjectID:   subjectID,
		Relation:    relation,
		ObjectType:  objectType,
		ObjectID:    objectID,
		CreatedAt:   time.Now(),
	}
}

// ReconstructRelationship はDBから復元するためのコンストラクタ
func ReconstructRelationship(
	id uuid.UUID,
	subjectType SubjectType,
	subjectID uuid.UUID,
	relation RelationType,
	objectType ObjectType,
	objectID uuid.UUID,
	createdAt time.Time,
) *Relationship {
	return &Relationship{
		ID:          id,
		SubjectType: subjectType,
		SubjectID:   subjectID,
		Relation:    relation,
		ObjectType:  objectType,
		ObjectID:    objectID,
		CreatedAt:   createdAt,
	}
}

// ToTuple はTupleに変換します
func (r *Relationship) ToTuple() Tuple {
	return NewTuple(r.SubjectType, r.SubjectID, r.Relation, r.ObjectType, r.ObjectID)
}

// IsOwnerRelation はオーナーリレーションかを判定します
func (r *Relationship) IsOwnerRelation() bool {
	return r.Relation == RelationOwner
}

// IsMemberRelation はメンバーリレーションかを判定します
func (r *Relationship) IsMemberRelation() bool {
	return r.Relation == RelationMember
}

// IsParentRelation は親リレーションかを判定します
func (r *Relationship) IsParentRelation() bool {
	return r.Relation == RelationParent
}

// NewOwnerRelationship はオーナーリレーションシップを生成します
func NewOwnerRelationship(userID uuid.UUID, objectType ObjectType, objectID uuid.UUID) *Relationship {
	return NewRelationship(SubjectTypeUser, userID, RelationOwner, objectType, objectID)
}

// NewParentRelationship は親リレーションシップを生成します
func NewParentRelationship(parentType ObjectType, parentID uuid.UUID, childType ObjectType, childID uuid.UUID) *Relationship {
	// parent_folder --parent--> child_resource
	return NewRelationship(SubjectType(parentType), parentID, RelationParent, childType, childID)
}

// NewMemberRelationship はメンバーリレーションシップを生成します
func NewMemberRelationship(userID uuid.UUID, groupID uuid.UUID) *Relationship {
	return NewRelationship(SubjectTypeUser, userID, RelationMember, ObjectTypeGroup, groupID)
}
