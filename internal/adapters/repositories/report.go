package repositories

import (
	"context"
	"errors"
	"fmt"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
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

func (r *ReportRepository) List(ctx context.Context, filter ports.ReportFilter) ([]*domain.Report, error) {
	args := []interface{}{}
	argIndex := 1
	conditions := []string{}

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.TargetUserID != nil {
		conditions = append(conditions, fmt.Sprintf(
			"(target_user_id = $%d OR target_review_id IN (SELECT id FROM reviews WHERE user_id = $%d) OR target_comment_id IN (SELECT id FROM comments WHERE user_id = $%d))",
			argIndex, argIndex, argIndex,
		))
		args = append(args, *filter.TargetUserID)
		argIndex++
	}

	if filter.TargetReviewID != nil {
		conditions = append(conditions, fmt.Sprintf("target_review_id = $%d", argIndex))
		args = append(args, *filter.TargetReviewID)
		argIndex++
	}

	if filter.TargetCommentID != nil {
		conditions = append(conditions, fmt.Sprintf("target_comment_id = $%d", argIndex))
		args = append(args, *filter.TargetCommentID)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			whereClause += " AND " + c
		}
	}

	q := fmt.Sprintf(`
		SELECT id, reporter_id, reason, details, status, target_user_id, target_review_id, target_comment_id, created_at, resolved_at, resolver_id
		FROM reports%s ORDER BY created_at
	`, whereClause)

	rows, err := r.db.Pool.Query(ctx, q, args...)
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
