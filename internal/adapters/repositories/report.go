package repositories

import (
	"context"
	"errors"

	"duskforge-api/internal/core/domain"
	"duskforge-api/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ReportRepository struct {
	db *database.DB
}

func NewReportRepository(db *database.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

func (r *ReportRepository) Create(ctx context.Context, report *domain.Report) error {
	query := `
		INSERT INTO reports (id, reporter_id, reason, details, status, target_user_id, target_review_id, target_comment_id, created_at, resolved_at, resolver_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.Pool.Exec(ctx, query,
		report.ID, report.ReporterID, report.Reason, report.Details, report.Status,
		report.TargetUserID, report.TargetReviewID, report.TargetCommentID,
		report.CreatedAt, report.ResolvedAt, report.ResolverID,
	)
	return err
}

func (r *ReportRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Report, error) {
	query := `
		SELECT id, reporter_id, reason, details, status, target_user_id, target_review_id, target_comment_id, created_at, resolved_at, resolver_id
		FROM reports WHERE id = $1
	`
	report := &domain.Report{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&report.ID, &report.ReporterID, &report.Reason, &report.Details, &report.Status,
		&report.TargetUserID, &report.TargetReviewID, &report.TargetCommentID,
		&report.CreatedAt, &report.ResolvedAt, &report.ResolverID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return report, err
}

func (r *ReportRepository) GetByStatus(ctx context.Context, status domain.ReportStatus) ([]*domain.Report, error) {
	query := `
		SELECT id, reporter_id, reason, details, status, target_user_id, target_review_id, target_comment_id, created_at, resolved_at, resolver_id
		FROM reports WHERE status = $1 ORDER BY created_at
	`
	rows, err := r.db.Pool.Query(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []*domain.Report
	for rows.Next() {
		report := &domain.Report{}
		if err := rows.Scan(
			&report.ID, &report.ReporterID, &report.Reason, &report.Details, &report.Status,
			&report.TargetUserID, &report.TargetReviewID, &report.TargetCommentID,
			&report.CreatedAt, &report.ResolvedAt, &report.ResolverID,
		); err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, rows.Err()
}

func (r *ReportRepository) Update(ctx context.Context, report *domain.Report) error {
	query := `
		UPDATE reports
		SET reason = $2, details = $3, status = $4, resolved_at = $5, resolver_id = $6
		WHERE id = $1
	`
	_, err := r.db.Pool.Exec(ctx, query,
		report.ID, report.Reason, report.Details, report.Status, report.ResolvedAt, report.ResolverID,
	)
	return err
}

func (r *ReportRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM reports WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}
