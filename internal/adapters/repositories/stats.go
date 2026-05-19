package repositories

import (
	"context"
	"fmt"

	"duskforge-api/internal/core/ports"
	"duskforge-api/pkg/database"

	"github.com/google/uuid"
)

type StatsRepository struct {
	db *database.DB
}

func NewStatsRepository(db *database.DB) *StatsRepository {
	return &StatsRepository{db: db}
}

func (r *StatsRepository) GetUserStats(ctx context.Context, userID uuid.UUID) (*ports.UserStats, error) {
	stats := &ports.UserStats{
		RatingDistrib: make(map[string]int),
	}

	scanInto := func(query string, dest ...interface{}) error {
		return r.db.Pool.QueryRow(ctx, query, userID).Scan(dest...)
	}

	if err := scanInto(`
		SELECT COUNT(*), AVG(rating)
		FROM reviews WHERE user_id = $1
	`, &stats.ReviewCount, &stats.AverageRating); err != nil {
		return nil, err
	}

	rows, err := r.db.Pool.Query(ctx, `
		SELECT rating, COUNT(*)
		FROM reviews WHERE user_id = $1
		GROUP BY rating ORDER BY rating
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var rating float64
		var count int
		if err := rows.Scan(&rating, &count); err != nil {
			return nil, err
		}
		key := fmt.Sprintf("%.1f", rating)
		stats.RatingDistrib[key] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := scanInto(`
		SELECT COUNT(*)
		FROM review_likes rl
		JOIN reviews r ON rl.review_id = r.id
		WHERE r.user_id = $1
	`, &stats.LikesReceived); err != nil {
		return nil, err
	}

	if err := scanInto(`
		SELECT COUNT(*)
		FROM comments c
		JOIN reviews r ON c.review_id = r.id
		WHERE r.user_id = $1 AND c.user_id != $1
	`, &stats.CommentsReceived); err != nil {
		return nil, err
	}

	if err := scanInto(`
		SELECT COUNT(*) FROM collections WHERE user_id = $1
	`, &stats.CollectionCount); err != nil {
		return nil, err
	}

	if err := scanInto(`
		SELECT COUNT(*)
		FROM collection_items ci
		JOIN collections c ON ci.collection_id = c.id
		WHERE c.user_id = $1
	`, &stats.TotalItems); err != nil {
		return nil, err
	}

	if err := scanInto(`
		SELECT COUNT(*), COALESCE(SUM(ci.runtime), 0)
		FROM collection_items ci
		JOIN collections c ON ci.collection_id = c.id
		WHERE c.user_id = $1 AND c.slug = 'watched'
	`, &stats.WatchedMovies, &stats.WatchedRuntime); err != nil {
		return nil, err
	}

	if err := scanInto(`
		SELECT COUNT(*) FROM follows f
		JOIN users u ON u.id = f.follower_id
		WHERE f.following_id = $1 AND u.banned_at IS NULL
	`, &stats.FollowersCount); err != nil {
		return nil, err
	}

	if err := scanInto(`
		SELECT COUNT(*) FROM follows f
		JOIN users u ON u.id = f.following_id
		WHERE f.follower_id = $1 AND u.banned_at IS NULL
	`, &stats.FollowingCount); err != nil {
		return nil, err
	}

	if err := scanInto(`
		SELECT COUNT(*) FROM user_achievements WHERE user_id = $1
	`, &stats.AchievementCount); err != nil {
		return nil, err
	}

	return stats, nil
}
