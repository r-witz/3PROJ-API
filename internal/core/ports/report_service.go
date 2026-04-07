package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type CreateReportInput struct {
	Reason          domain.ReportReason
	Details         *string
	TargetUserID    *uuid.UUID
	TargetReviewID  *uuid.UUID
	TargetCommentID *uuid.UUID
}

type ResolveReportInput struct {
	Status domain.ReportStatus
}

type ReportService interface {
	Create(ctx context.Context, reporterID uuid.UUID, input CreateReportInput) (*domain.Report, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Report, error)
	List(ctx context.Context, filter ReportFilter) ([]*domain.Report, error)
	Resolve(ctx context.Context, reportID uuid.UUID, resolverID uuid.UUID, input ResolveReportInput) (*domain.Report, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
