package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type ExportRepository interface {
	GetAllUserData(ctx context.Context, userID uuid.UUID) (*domain.UserDataExport, error)
}

type ExportService interface {
	ExportUserData(ctx context.Context, userID uuid.UUID) (*domain.UserDataExport, error)
}
