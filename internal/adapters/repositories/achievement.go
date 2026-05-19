package repositories

import (
	"context"
	"errors"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	"duskforge-api/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AchievementRepository struct {
	db *database.DB
}

func NewAchievementRepository(db *database.DB) *AchievementRepository {
	return &AchievementRepository{db: db}
}

const achievementColumns = "id, code, name, description, category, tier, icon_url, criterion, secret, active, system, sort_order, created_at, updated_at"

func scanAchievement(row pgx.Row, a *domain.Achievement) error {
	return row.Scan(
		&a.ID, &a.Code, &a.Name, &a.Description, &a.Category, &a.Tier,
		&a.IconURL, &a.Criterion, &a.Secret, &a.Active, &a.System,
		&a.SortOrder, &a.CreatedAt, &a.UpdatedAt,
	)
}

func (r *AchievementRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Achievement, error) {
	a := &domain.Achievement{}
	err := scanAchievement(
		r.db.Pool.QueryRow(ctx, `SELECT `+achievementColumns+` FROM achievements WHERE id = $1`, id),
		a,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (r *AchievementRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.Achievement, error) {
	if len(ids) == 0 {
		return []*domain.Achievement{}, nil
	}
	rows, err := r.db.Pool.Query(ctx,
		`SELECT `+achievementColumns+` FROM achievements WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Achievement
	for rows.Next() {
		a := &domain.Achievement{}
		if err := scanAchievement(rows, a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *AchievementRepository) List(ctx context.Context, filter ports.AchievementListFilter) ([]*domain.Achievement, error) {
	query := `SELECT ` + achievementColumns + ` FROM achievements`
	args := []any{}
	where := []string{}

	if filter.OnlyActive {
		where = append(where, "active = TRUE")
	}
	if filter.Category != nil {
		args = append(args, *filter.Category)
		where = append(where, "category = $1")
	}

	if len(where) > 0 {
		query += " WHERE "
		for i, clause := range where {
			if i > 0 {
				query += " AND "
			}
			query += clause
		}
	}
	query += " ORDER BY sort_order ASC, created_at ASC"

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Achievement
	for rows.Next() {
		a := &domain.Achievement{}
		if err := scanAchievement(rows, a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *AchievementRepository) GetUnlockedIDsByUser(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]struct{}, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT achievement_id FROM user_achievements WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	set := make(map[uuid.UUID]struct{})
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		set[id] = struct{}{}
	}
	return set, rows.Err()
}

func (r *AchievementRepository) GetUnlockedByUser(ctx context.Context, userID uuid.UUID) ([]*domain.UserAchievement, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT user_id, achievement_id, unlocked_at
		FROM user_achievements
		WHERE user_id = $1
		ORDER BY unlocked_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.UserAchievement
	for rows.Next() {
		ua := &domain.UserAchievement{}
		if err := rows.Scan(&ua.UserID, &ua.AchievementID, &ua.UnlockedAt); err != nil {
			return nil, err
		}
		out = append(out, ua)
	}
	return out, rows.Err()
}

func (r *AchievementRepository) GetRecentUnlocksByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.UserAchievement, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT user_id, achievement_id, unlocked_at
		FROM user_achievements
		WHERE user_id = $1
		ORDER BY unlocked_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.UserAchievement
	for rows.Next() {
		ua := &domain.UserAchievement{}
		if err := rows.Scan(&ua.UserID, &ua.AchievementID, &ua.UnlockedAt); err != nil {
			return nil, err
		}
		out = append(out, ua)
	}
	return out, rows.Err()
}

func (r *AchievementRepository) CountUnlockedByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM user_achievements WHERE user_id = $1`, userID,
	).Scan(&count)
	return count, err
}

// Unlock inserts a user_achievements row idempotently. Returns true when the
// row is newly inserted (i.e. this caller caused the unlock), false if it
// already existed.
func (r *AchievementRepository) Unlock(ctx context.Context, userID, achievementID uuid.UUID) (bool, error) {
	tag, err := r.db.Pool.Exec(ctx, `
		INSERT INTO user_achievements (user_id, achievement_id, unlocked_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, achievement_id) DO NOTHING
	`, userID, achievementID, time.Now())
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (r *AchievementRepository) CountCommentsByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM comments WHERE user_id = $1`, userID).Scan(&count)
	return count, err
}

func (r *AchievementRepository) CountWrittenReviewsByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM reviews WHERE user_id = $1 AND content IS NOT NULL AND btrim(content) <> ''`,
		userID,
	).Scan(&count)
	return count, err
}

func (r *AchievementRepository) CountReviewsByUserWithRating(ctx context.Context, userID uuid.UUID, rating float64) (int, error) {
	var count int
	err := r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM reviews WHERE user_id = $1 AND rating = $2`,
		userID, rating,
	).Scan(&count)
	return count, err
}

func (r *AchievementRepository) CountCustomCollectionsByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM collections WHERE user_id = $1 AND type = 'custom'`,
		userID,
	).Scan(&count)
	return count, err
}
