package repositories

import (
	"context"
	"errors"

	"duskforge-api/internal/core/domain"
	"duskforge-api/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type CollectionItemRepository struct {
	db *database.DB
}

func NewCollectionItemRepository(db *database.DB) *CollectionItemRepository {
	return &CollectionItemRepository{db: db}
}

func (r *CollectionItemRepository) Create(ctx context.Context, item *domain.CollectionItem) error {
	query := `
		INSERT INTO collection_items (collection_id, tmdb_id, added_at, runtime, metadata)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Pool.Exec(ctx, query,
		item.CollectionID, item.TMDBID, item.AddedAt, item.Runtime, item.Metadata,
	)
	return err
}

func (r *CollectionItemRepository) GetByCollectionID(ctx context.Context, collectionID uuid.UUID) ([]*domain.CollectionItem, error) {
	query := `
		SELECT collection_id, tmdb_id, added_at, runtime, metadata
		FROM collection_items WHERE collection_id = $1 ORDER BY added_at DESC
	`
	rows, err := r.db.Pool.Query(ctx, query, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*domain.CollectionItem
	for rows.Next() {
		item := &domain.CollectionItem{}
		if err := rows.Scan(
			&item.CollectionID, &item.TMDBID, &item.AddedAt, &item.Runtime, &item.Metadata,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *CollectionItemRepository) GetByCollectionIDPaginated(ctx context.Context, collectionID uuid.UUID, offset, limit int) ([]*domain.CollectionItem, error) {
	query := `
		SELECT collection_id, tmdb_id, added_at, runtime, metadata
		FROM collection_items WHERE collection_id = $1 ORDER BY added_at DESC LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Pool.Query(ctx, query, collectionID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*domain.CollectionItem
	for rows.Next() {
		item := &domain.CollectionItem{}
		if err := rows.Scan(
			&item.CollectionID, &item.TMDBID, &item.AddedAt, &item.Runtime, &item.Metadata,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *CollectionItemRepository) CountByCollectionID(ctx context.Context, collectionID uuid.UUID) (int, error) {
	var count int
	err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM collection_items WHERE collection_id = $1`, collectionID).Scan(&count)
	return count, err
}

func (r *CollectionItemRepository) CountByCollectionIDs(ctx context.Context, collectionIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	counts := make(map[uuid.UUID]int, len(collectionIDs))
	if len(collectionIDs) == 0 {
		return counts, nil
	}

	query := `SELECT collection_id, COUNT(*) FROM collection_items WHERE collection_id = ANY($1) GROUP BY collection_id`
	rows, err := r.db.Pool.Query(ctx, query, collectionIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		var count int
		if err := rows.Scan(&id, &count); err != nil {
			return nil, err
		}
		counts[id] = count
	}
	return counts, rows.Err()
}

func (r *CollectionItemRepository) GetByCollectionIDAndTMDBID(ctx context.Context, collectionID uuid.UUID, tmdbID int) (*domain.CollectionItem, error) {
	query := `
		SELECT collection_id, tmdb_id, added_at, runtime, metadata
		FROM collection_items WHERE collection_id = $1 AND tmdb_id = $2
	`
	item := &domain.CollectionItem{}
	err := r.db.Pool.QueryRow(ctx, query, collectionID, tmdbID).Scan(
		&item.CollectionID, &item.TMDBID, &item.AddedAt, &item.Runtime, &item.Metadata,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return item, err
}

func (r *CollectionItemRepository) Delete(ctx context.Context, collectionID uuid.UUID, tmdbID int) error {
	query := `DELETE FROM collection_items WHERE collection_id = $1 AND tmdb_id = $2`
	_, err := r.db.Pool.Exec(ctx, query, collectionID, tmdbID)
	return err
}
