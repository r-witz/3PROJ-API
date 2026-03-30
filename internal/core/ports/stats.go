package ports

import (
	"context"

	"github.com/google/uuid"
)

type UserStats struct {
	ReviewCount      int
	AverageRating    *float64
	RatingDistrib    map[string]int
	LikesReceived    int
	CommentsReceived int
	CollectionCount  int
	TotalItems       int
	WatchedMovies    int
	WatchedRuntime   int
	FollowersCount   int
	FollowingCount   int
}

type StatsRepository interface {
	GetUserStats(ctx context.Context, userID uuid.UUID) (*UserStats, error)
}
