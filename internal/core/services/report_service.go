package services

import (
	"context"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/google/uuid"
)

type ReportService struct {
	reportRepo  ports.ReportRepository
	userRepo    ports.UserRepository
	reviewRepo  ports.ReviewRepository
	commentRepo ports.CommentRepository
}

func NewReportService(
	reportRepo ports.ReportRepository,
	userRepo ports.UserRepository,
	reviewRepo ports.ReviewRepository,
	commentRepo ports.CommentRepository,
) *ReportService {
	return &ReportService{
		reportRepo:  reportRepo,
		userRepo:    userRepo,
		reviewRepo:  reviewRepo,
		commentRepo: commentRepo,
	}
}

func (s *ReportService) Create(ctx context.Context, reporterID uuid.UUID, input ports.CreateReportInput) (*domain.Report, error) {
	targets := 0
	if input.TargetUserID != nil {
		targets++
	}
	if input.TargetReviewID != nil {
		targets++
	}
	if input.TargetCommentID != nil {
		targets++
	}
	if targets != 1 {
		return nil, domain.ErrInvalidReportTarget
	}

	if input.TargetUserID != nil {
		user, err := s.userRepo.GetByID(ctx, *input.TargetUserID)
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, domain.ErrUserNotFound
		}
	}

	if input.TargetReviewID != nil {
		review, err := s.reviewRepo.GetByID(ctx, *input.TargetReviewID)
		if err != nil {
			return nil, err
		}
		if review == nil {
			return nil, domain.ErrReviewNotFound
		}
	}

	if input.TargetCommentID != nil {
		comment, err := s.commentRepo.GetByID(ctx, *input.TargetCommentID)
		if err != nil {
			return nil, err
		}
		if comment == nil {
			return nil, domain.ErrCommentNotFound
		}
	}

	report := &domain.Report{
		ID:              uuid.New(),
		ReporterID:      reporterID,
		Reason:          input.Reason,
		Details:         input.Details,
		Status:          domain.ReportStatusPending,
		TargetUserID:    input.TargetUserID,
		TargetReviewID:  input.TargetReviewID,
		TargetCommentID: input.TargetCommentID,
		CreatedAt:       time.Now(),
	}

	if err := s.reportRepo.Create(ctx, report); err != nil {
		return nil, err
	}

	return report, nil
}

func (s *ReportService) GetByID(ctx context.Context, id uuid.UUID) (*ports.ReportWithContext, error) {
	report, err := s.reportRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if report == nil {
		return nil, domain.ErrReportNotFound
	}
	return s.enrich(ctx, report)
}

func (s *ReportService) List(ctx context.Context, filter ports.ReportFilter) ([]*ports.ReportWithContext, error) {
	reports, err := s.reportRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	result := make([]*ports.ReportWithContext, 0, len(reports))
	for _, r := range reports {
		enriched, err := s.enrich(ctx, r)
		if err != nil {
			return nil, err
		}
		result = append(result, enriched)
	}
	return result, nil
}

func (s *ReportService) enrich(ctx context.Context, report *domain.Report) (*ports.ReportWithContext, error) {
	ctxOut := &ports.ReportWithContext{Report: report}

	if report.TargetUserID != nil {
		user, err := s.userRepo.GetByID(ctx, *report.TargetUserID)
		if err != nil {
			return nil, err
		}
		ctxOut.User = user
	}
	if report.TargetReviewID != nil {
		review, err := s.reviewRepo.GetByID(ctx, *report.TargetReviewID)
		if err != nil {
			return nil, err
		}
		ctxOut.Review = review
	}
	if report.TargetCommentID != nil {
		comment, err := s.commentRepo.GetByID(ctx, *report.TargetCommentID)
		if err != nil {
			return nil, err
		}
		ctxOut.Comment = comment
	}
	return ctxOut, nil
}

func (s *ReportService) Resolve(ctx context.Context, reportID uuid.UUID, resolverID uuid.UUID, input ports.ResolveReportInput) (*domain.Report, error) {
	report, err := s.reportRepo.GetByID(ctx, reportID)
	if err != nil {
		return nil, err
	}
	if report == nil {
		return nil, domain.ErrReportNotFound
	}

	if report.Status != domain.ReportStatusPending {
		return nil, domain.ErrReportAlreadyResolved
	}

	now := time.Now()
	report.Status = input.Status
	report.ResolvedAt = &now
	report.ResolverID = &resolverID

	if err := s.reportRepo.Update(ctx, report); err != nil {
		return nil, err
	}

	return report, nil
}

func (s *ReportService) Delete(ctx context.Context, id uuid.UUID) error {
	report, err := s.reportRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if report == nil {
		return domain.ErrReportNotFound
	}
	return s.reportRepo.Delete(ctx, id)
}
