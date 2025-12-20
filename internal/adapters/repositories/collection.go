package repositories

import (
	"context"
	"errors"

	"duskforge-api/internal/core/domain"
	"duskforge-api/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type CollectionRepository struct {
	db *database.DB
}

func NewCollectionRepository(db *database.DB) *CollectionRepository {
	return &CollectionRepository{db: db}
}

func (r *CollectionRepository) Create(ctx context.Context, collection *domain.Collection) error {
	query := `
		INSERT INTO collections (id, user_id, name, slug, type, visibility, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.Pool.Exec(ctx, query,
		collection.ID, collection.UserID, collection.Name, collection.Slug, collection.Type,
		collection.Visibility, collection.Description, collection.CreatedAt, collection.UpdatedAt,
	)
	return err
}

func (r *CollectionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Collection, error) {
	query := `
		SELECT id, user_id, name, slug, type, visibility, description, created_at, updated_at
		FROM collections WHERE id = $1
	`
	collection := &domain.Collection{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&collection.ID, &collection.UserID, &collection.Name, &collection.Slug, &collection.Type,
		&collection.Visibility, &collection.Description, &collection.CreatedAt, &collection.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return collection, err
}

func (r *CollectionRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Collection, error) {
	query := `
		SELECT id, user_id, name, slug, type, visibility, description, created_at, updated_at
		FROM collections WHERE user_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collections []*domain.Collection
	for rows.Next() {
		collection := &domain.Collection{}
		if err := rows.Scan(
			&collection.ID, &collection.UserID, &collection.Name, &collection.Slug, &collection.Type,
			&collection.Visibility, &collection.Description, &collection.CreatedAt, &collection.UpdatedAt,
		); err != nil {
			return nil, err
		}
		collections = append(collections, collection)
	}
	return collections, rows.Err()
}

func (r *CollectionRepository) GetByUserIDAndSlug(ctx context.Context, userID uuid.UUID, slug string) (*domain.Collection, error) {
	query := `
		SELECT id, user_id, name, slug, type, visibility, description, created_at, updated_at
		FROM collections WHERE user_id = $1 AND slug = $2
	`
	collection := &domain.Collection{}
	err := r.db.Pool.QueryRow(ctx, query, userID, slug).Scan(
		&collection.ID, &collection.UserID, &collection.Name, &collection.Slug, &collection.Type,
		&collection.Visibility, &collection.Description, &collection.CreatedAt, &collection.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return collection, err
}

func (r *CollectionRepository) Update(ctx context.Context, collection *domain.Collection) error {
	query := `
		UPDATE collections
		SET name = $2, slug = $3, type = $4, visibility = $5, description = $6, updated_at = $7
		WHERE id = $1
	`
	_, err := r.db.Pool.Exec(ctx, query,
		collection.ID, collection.Name, collection.Slug, collection.Type,
		collection.Visibility, collection.Description, collection.UpdatedAt,
	)
	return err
}

func (r *CollectionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM collections WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}
