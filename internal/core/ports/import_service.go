package ports

import (
	"context"
	"io"

	"github.com/google/uuid"
)

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
	ImportLetterboxd(ctx context.Context, userID uuid.UUID, zipReader io.ReaderAt, zipSize int64) (*ImportResult, error)
}
