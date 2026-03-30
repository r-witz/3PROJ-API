package services

import (
	"context"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/google/uuid"
)

type statsService struct {
	statsRepo ports.StatsRepository
	userRepo  ports.UserRepository
}

func NewStatsService(statsRepo ports.StatsRepository, userRepo ports.UserRepository) ports.StatsService {
	return &statsService{statsRepo: statsRepo, userRepo: userRepo}
}

func (s *statsService) GetUserStats(ctx context.Context, userID uuid.UUID) (*ports.UserStats, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if user == nil {
		return nil, domain.ErrUserNotFound
	}

	stats, err := s.statsRepo.GetUserStats(ctx, userID)
	if err != nil {
		return nil, domain.ErrInternal
	}

	return stats, nil
}
