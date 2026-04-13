package ports

import (
	"context"
	"io"

	"github.com/google/uuid"
)

type ImportStatus string

const (
	ImportStatusProcessing ImportStatus = "processing"
	ImportStatusCompleted  ImportStatus = "completed"
	ImportStatusFailed     ImportStatus = "failed"
)

type ImportProgress struct {
	Status   ImportStatus        `json:"status"`
	Phase    string              `json:"phase"`
	Resolved int                 `json:"resolved"`
	Total    int                 `json:"total"`
	Result   *ImportResult       `json:"result,omitempty"`
	Error    string              `json:"error,omitempty"`
}

type ImportResult struct {
	Watched   ImportSectionResult `json:"watched"`
	Watchlist ImportSectionResult `json:"watchlist"`
	Ratings   ImportSectionResult `json:"ratings"`
	Reviews   ImportSectionResult `json:"reviews"`
	Failed    []ImportFailure     `json:"failed"`
}

type ImportSectionResult struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
}

type ImportFailure struct {
	Name   string `json:"name"`
	Year   int    `json:"year"`
	Reason string `json:"reason"`
}

type ImportService interface {
	StartImportLetterboxd(ctx context.Context, userID uuid.UUID, zipReader io.ReaderAt, zipSize int64) (*ImportProgress, error)
	GetImportStatus(userID uuid.UUID) *ImportProgress
}
