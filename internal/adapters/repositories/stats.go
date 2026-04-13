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

	// Review stats: count, average rating
	err := r.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*), AVG(rating)
		FROM reviews WHERE user_id = $1
	`, userID).Scan(&stats.ReviewCount, &stats.AverageRating)
	if err != nil {
		return nil, err
	}

	// Rating distribution
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

	// Likes received on user's reviews
	err = r.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM review_likes rl
		JOIN reviews r ON rl.review_id = r.id
		WHERE r.user_id = $1
	`, userID).Scan(&stats.LikesReceived)
	if err != nil {
		return nil, err
	}

	// Comments received on user's reviews (excluding self-comments)
	err = r.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM comments c
		JOIN reviews r ON c.review_id = r.id
		WHERE r.user_id = $1 AND c.user_id != $1
	`, userID).Scan(&stats.CommentsReceived)
	if err != nil {
		return nil, err
	}

	// Collection stats: count, total items
	err = r.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM collections WHERE user_id = $1
	`, userID).Scan(&stats.CollectionCount)
	if err != nil {
		return nil, err
	}

	err = r.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM collection_items ci
		JOIN collections c ON ci.collection_id = c.id
		WHERE c.user_id = $1
	`, userID).Scan(&stats.TotalItems)
	if err != nil {
		return nil, err
	}

	// Watched stats (from "watched" system collection)
	err = r.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(ci.runtime), 0)
		FROM collection_items ci
		JOIN collections c ON ci.collection_id = c.id
		WHERE c.user_id = $1 AND c.slug = 'watched'
	`, userID).Scan(&stats.WatchedMovies, &stats.WatchedRuntime)
	if err != nil {
		return nil, err
	}

	// Social stats (exclude banned users)
	err = r.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM follows f
		JOIN users u ON u.id = f.follower_id
		WHERE f.following_id = $1 AND u.banned_at IS NULL
	`, userID).Scan(&stats.FollowersCount)
	if err != nil {
		return nil, err
	}

	err = r.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM follows f
		JOIN users u ON u.id = f.following_id
		WHERE f.follower_id = $1 AND u.banned_at IS NULL
	`, userID).Scan(&stats.FollowingCount)
	if err != nil {
		return nil, err
	}

	return stats, nil
}
