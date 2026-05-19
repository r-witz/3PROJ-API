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
	if filter.TargetUsername != nil && filter.TargetUserID == nil {
		user, err := s.userRepo.GetByUsername(ctx, *filter.TargetUsername)
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, domain.ErrUserNotFound
		}
		filter.TargetUserID = &user.ID
	}

	reports, err := s.reportRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	reviewIDSet := make(map[uuid.UUID]struct{})
	commentIDSet := make(map[uuid.UUID]struct{})
	for _, r := range reports {
		if r.TargetReviewID != nil {
			reviewIDSet[*r.TargetReviewID] = struct{}{}
		}
		if r.TargetCommentID != nil {
			commentIDSet[*r.TargetCommentID] = struct{}{}
		}
	}

	reviewIDs := make([]uuid.UUID, 0, len(reviewIDSet))
	for id := range reviewIDSet {
		reviewIDs = append(reviewIDs, id)
	}
	commentIDs := make([]uuid.UUID, 0, len(commentIDSet))
	for id := range commentIDSet {
		commentIDs = append(commentIDs, id)
	}

	reviewMap := make(map[uuid.UUID]*domain.Review)
	if len(reviewIDs) > 0 {
		reviews, err := s.reviewRepo.GetByIDs(ctx, reviewIDs)
		if err != nil {
			return nil, err
		}
		for _, rv := range reviews {
			reviewMap[rv.ID] = rv
		}
	}

	commentMap := make(map[uuid.UUID]*domain.Comment)
	if len(commentIDs) > 0 {
		comments, err := s.commentRepo.GetByIDs(ctx, commentIDs)
		if err != nil {
			return nil, err
		}
		for _, c := range comments {
			commentMap[c.ID] = c
		}
	}

	userIDSet := make(map[uuid.UUID]struct{})
	for _, r := range reports {
		if r.TargetUserID != nil {
			userIDSet[*r.TargetUserID] = struct{}{}
		}
		if r.TargetReviewID != nil {
			if rv, ok := reviewMap[*r.TargetReviewID]; ok {
				userIDSet[rv.UserID] = struct{}{}
			}
		}
		if r.TargetCommentID != nil {
			if c, ok := commentMap[*r.TargetCommentID]; ok {
				userIDSet[c.UserID] = struct{}{}
			}
		}
	}

	userIDs := make([]uuid.UUID, 0, len(userIDSet))
	for id := range userIDSet {
		userIDs = append(userIDs, id)
	}

	userMap := make(map[uuid.UUID]*domain.User)
	if len(userIDs) > 0 {
		users, err := s.userRepo.GetByIDs(ctx, userIDs)
		if err != nil {
			return nil, err
		}
		for _, u := range users {
			userMap[u.ID] = u
		}
	}

	result := make([]*ports.ReportWithContext, 0, len(reports))
	for _, r := range reports {
		ctxOut := &ports.ReportWithContext{Report: r}
		if r.TargetReviewID != nil {
			if rv, ok := reviewMap[*r.TargetReviewID]; ok {
				ctxOut.Review = rv
				ctxOut.User = userMap[rv.UserID]
			}
		}
		if r.TargetCommentID != nil {
			if c, ok := commentMap[*r.TargetCommentID]; ok {
				ctxOut.Comment = c
				ctxOut.User = userMap[c.UserID]
			}
		}
		if r.TargetUserID != nil {
			ctxOut.User = userMap[*r.TargetUserID]
		}
		result = append(result, ctxOut)
	}
	return result, nil
}

func (s *ReportService) enrich(ctx context.Context, report *domain.Report) (*ports.ReportWithContext, error) {
	ctxOut := &ports.ReportWithContext{Report: report}

	var targetUserID *uuid.UUID
	if report.TargetUserID != nil {
		targetUserID = report.TargetUserID
	}
	if report.TargetReviewID != nil {
		review, err := s.reviewRepo.GetByID(ctx, *report.TargetReviewID)
		if err != nil {
			return nil, err
		}
		ctxOut.Review = review
		if review != nil && targetUserID == nil {
			targetUserID = &review.UserID
		}
	}
	if report.TargetCommentID != nil {
		comment, err := s.commentRepo.GetByID(ctx, *report.TargetCommentID)
		if err != nil {
			return nil, err
		}
		ctxOut.Comment = comment
		if comment != nil && targetUserID == nil {
			targetUserID = &comment.UserID
		}
	}
	if targetUserID != nil {
		user, err := s.userRepo.GetByID(ctx, *targetUserID)
		if err != nil {
			return nil, err
		}
		ctxOut.User = user
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
