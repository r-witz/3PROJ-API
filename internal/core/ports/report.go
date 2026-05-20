package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type ReportFilter struct {
	Status           *domain.ReportStatus
	TargetUserID     *uuid.UUID
	TargetUsername   *string
	TargetReviewID   *uuid.UUID
	TargetCommentID  *uuid.UUID
}

type ReportRepository interface {
	Create(ctx context.Context, report *domain.Report) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Report, error)
	List(ctx context.Context, filter ReportFilter) ([]*domain.Report, error)
	Update(ctx context.Context, report *domain.Report) error
	Delete(ctx context.Context, id uuid.UUID) error
}
