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

type ReportWithContext struct {
	Report  *domain.Report
	User    *domain.User
	Review  *domain.Review
	Comment *domain.Comment
}

type ReportService interface {
	Create(ctx context.Context, reporterID uuid.UUID, input CreateReportInput) (*domain.Report, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ReportWithContext, error)
	List(ctx context.Context, filter ReportFilter) ([]*ReportWithContext, error)
	Resolve(ctx context.Context, reportID uuid.UUID, resolverID uuid.UUID, input ResolveReportInput) (*ReportWithContext, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
