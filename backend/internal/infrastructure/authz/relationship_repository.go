package authz

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database/sqlcgen"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// RelationshipRepository はリレーションシップリポジトリの実装です
type RelationshipRepository struct {
	*database.BaseRepository
}

// NewRelationshipRepository は新しいRelationshipRepositoryを作成します
func NewRelationshipRepository(txManager *database.TxManager) *RelationshipRepository {
	return &RelationshipRepository{
		BaseRepository: database.NewBaseRepository(txManager),
	}
}

// Create はリレーションシップを作成します
func (r *RelationshipRepository) Create(ctx context.Context, rel *authz.Relationship) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	_, err := queries.CreateRelationship(ctx, sqlcgen.CreateRelationshipParams{
		ID:          rel.ID,
		SubjectType: rel.SubjectType.String(),
		SubjectID:   rel.SubjectID,
		Relation:    rel.Relation.String(),
		ObjectType:  rel.ObjectType.String(),
		ObjectID:    rel.ObjectID,
		CreatedAt:   rel.CreatedAt,
	})

	return r.HandleError(err)
}

// Delete はリレーションシップを削除します
func (r *RelationshipRepository) Delete(ctx context.Context, id uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteRelationship(ctx, id)
	return r.HandleError(err)
}

// DeleteByTuple はタプルでリレーションシップを削除します
func (r *RelationshipRepository) DeleteByTuple(ctx context.Context, tuple authz.Tuple) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteRelationshipByTuple(ctx, sqlcgen.DeleteRelationshipByTupleParams{
		SubjectType: tuple.SubjectType.String(),
		SubjectID:   tuple.SubjectID,
		Relation:    tuple.Relation.String(),
		ObjectType:  tuple.ObjectType.String(),
		ObjectID:    tuple.ObjectID,
	})
	return r.HandleError(err)
}

// Exists はリレーションシップが存在するかを確認します
func (r *RelationshipRepository) Exists(ctx context.Context, tuple authz.Tuple) (bool, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	exists, err := queries.RelationshipExists(ctx, sqlcgen.RelationshipExistsParams{
		SubjectType: tuple.SubjectType.String(),
		SubjectID:   tuple.SubjectID,
		Relation:    tuple.Relation.String(),
		ObjectType:  tuple.ObjectType.String(),
		ObjectID:    tuple.ObjectID,
	})
	if err != nil {
		return false, r.HandleError(err)
	}

	return exists, nil
}

// FindSubjects はオブジェクトからサブジェクトを検索します
func (r *RelationshipRepository) FindSubjects(ctx context.Context, objectType authz.ObjectType, objectID uuid.UUID, relation authz.RelationType) ([]authz.Resource, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.FindSubjectsByObject(ctx, sqlcgen.FindSubjectsByObjectParams{
		ObjectType: objectType.String(),
		ObjectID:   objectID,
		Relation:   relation.String(),
	})
	if err != nil {
		return nil, r.HandleError(err)
	}

	resources := make([]authz.Resource, 0, len(rows))
	for _, row := range rows {
		resourceType, err := authz.NewResourceType(row.SubjectType)
		if err != nil {
			// Skip if invalid resource type
			continue
		}
		resources = append(resources, authz.NewResource(resourceType, row.SubjectID))
	}

	return resources, nil
}

// FindObjects はサブジェクトからオブジェクトを検索します
func (r *RelationshipRepository) FindObjects(ctx context.Context, subjectType authz.SubjectType, subjectID uuid.UUID, relation authz.RelationType, objectType authz.ObjectType) ([]uuid.UUID, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.FindObjectsBySubject(ctx, sqlcgen.FindObjectsBySubjectParams{
		SubjectType: subjectType.String(),
		SubjectID:   subjectID,
		Relation:    relation.String(),
		ObjectType:  objectType.String(),
	})
	if err != nil {
		return nil, r.HandleError(err)
	}

	return rows, nil
}

// FindByObject はオブジェクトでリレーションシップを検索します
func (r *RelationshipRepository) FindByObject(ctx context.Context, objectType authz.ObjectType, objectID uuid.UUID) ([]*authz.Relationship, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListRelationshipsByObject(ctx, sqlcgen.ListRelationshipsByObjectParams{
		ObjectType: objectType.String(),
		ObjectID:   objectID,
	})
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows)
}

// FindBySubject はサブジェクトでリレーションシップを検索します
func (r *RelationshipRepository) FindBySubject(ctx context.Context, subjectType authz.SubjectType, subjectID uuid.UUID) ([]*authz.Relationship, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListRelationshipsBySubject(ctx, sqlcgen.ListRelationshipsBySubjectParams{
		SubjectType: subjectType.String(),
		SubjectID:   subjectID,
	})
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows)
}

// FindParent は親リソースを検索します
func (r *RelationshipRepository) FindParent(ctx context.Context, objectType authz.ObjectType, objectID uuid.UUID) (*authz.Resource, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.FindParentRelationship(ctx, sqlcgen.FindParentRelationshipParams{
		ObjectType: objectType.String(),
		ObjectID:   objectID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No parent found
		}
		return nil, r.HandleError(err)
	}

	resourceType, err := authz.NewResourceType(row.SubjectType)
	if err != nil {
		return nil, apperror.NewInternalError(err)
	}

	resource := authz.NewResource(resourceType, row.SubjectID)
	return &resource, nil
}

// DeleteByObject はオブジェクトでリレーションシップを一括削除します
func (r *RelationshipRepository) DeleteByObject(ctx context.Context, objectType authz.ObjectType, objectID uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteRelationshipsByObject(ctx, sqlcgen.DeleteRelationshipsByObjectParams{
		ObjectType: objectType.String(),
		ObjectID:   objectID,
	})
	return r.HandleError(err)
}

// DeleteBySubject はサブジェクトでリレーションシップを一括削除します
func (r *RelationshipRepository) DeleteBySubject(ctx context.Context, subjectType authz.SubjectType, subjectID uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteRelationshipsBySubject(ctx, sqlcgen.DeleteRelationshipsBySubjectParams{
		SubjectType: subjectType.String(),
		SubjectID:   subjectID,
	})
	return r.HandleError(err)
}

// toEntity はsqlcgen.Relationshipをauthz.Relationshipに変換します
func (r *RelationshipRepository) toEntity(row sqlcgen.Relationship) (*authz.Relationship, error) {
	subjectType, err := authz.NewSubjectType(row.SubjectType)
	if err != nil {
		return nil, err
	}

	relation, err := authz.NewRelationType(row.Relation)
	if err != nil {
		return nil, err
	}

	objectType, err := authz.NewObjectType(row.ObjectType)
	if err != nil {
		return nil, err
	}

	return authz.ReconstructRelationship(
		row.ID,
		subjectType,
		row.SubjectID,
		relation,
		objectType,
		row.ObjectID,
		row.CreatedAt,
	), nil
}

// toEntities は複数のsqlcgen.Relationshipをauthz.Relationshipに変換します
func (r *RelationshipRepository) toEntities(rows []sqlcgen.Relationship) ([]*authz.Relationship, error) {
	rels := make([]*authz.Relationship, 0, len(rows))
	for _, row := range rows {
		rel, err := r.toEntity(row)
		if err != nil {
			return nil, err
		}
		rels = append(rels, rel)
	}
	return rels, nil
}

// インターフェースの実装を保証
var _ authz.RelationshipRepository = (*RelationshipRepository)(nil)
