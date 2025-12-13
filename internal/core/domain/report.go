package domain

import (
	"time"

	"github.com/google/uuid"
)

type ReportReason string

const (
	ReportReasonSpam          ReportReason = "spam"
	ReportReasonHarassment    ReportReason = "harassment"
	ReportReasonSpoiler       ReportReason = "spoiler"
	ReportReasonInappropriate ReportReason = "inappropriate"
	ReportReasonOther         ReportReason = "other"
)

type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "pending"
	ReportStatusResolved  ReportStatus = "resolved"
	ReportStatusDismissed ReportStatus = "dismissed"
)

type Report struct {
	ID              uuid.UUID    `json:"id" db:"id"`
	ReporterID      uuid.UUID    `json:"reporter_id" db:"reporter_id"`
	Reason          ReportReason `json:"reason" db:"reason"`
	Details         *string      `json:"details,omitempty" db:"details"`
	Status          ReportStatus `json:"status" db:"status"`
	TargetUserID    *uuid.UUID   `json:"target_user_id,omitempty" db:"target_user_id"`
	TargetReviewID  *uuid.UUID   `json:"target_review_id,omitempty" db:"target_review_id"`
	TargetCommentID *uuid.UUID   `json:"target_comment_id,omitempty" db:"target_comment_id"`
	CreatedAt       time.Time    `json:"created_at" db:"created_at"`
	ResolvedAt      *time.Time   `json:"resolved_at,omitempty" db:"resolved_at"`
	ResolverID      *uuid.UUID   `json:"resolver_id,omitempty" db:"resolver_id"`
}
