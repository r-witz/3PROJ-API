package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type CreateCollectionInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Visibility  string  `json:"visibility"`
}

type UpdateCollectionInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Visibility  *string `json:"visibility"`
}

type CollectionService interface {
	CreateDefaultCollections(ctx context.Context, userID uuid.UUID) error
	Create(ctx context.Context, userID uuid.UUID, input CreateCollectionInput) (*domain.Collection, error)
	GetBySlug(ctx context.Context, userID uuid.UUID, slug string, requestingUserID *uuid.UUID) (*domain.Collection, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, requestingUserID *uuid.UUID) ([]*domain.Collection, error)
	Update(ctx context.Context, userID uuid.UUID, slug string, input UpdateCollectionInput) (*domain.Collection, error)
	Delete(ctx context.Context, userID uuid.UUID, slug string) error
	AddItem(ctx context.Context, userID uuid.UUID, slug string, tmdbID int, runtime int16) (*domain.CollectionItem, error)
	RemoveItem(ctx context.Context, userID uuid.UUID, slug string, tmdbID int) error
	GetItems(ctx context.Context, userID uuid.UUID, slug string, requestingUserID *uuid.UUID) ([]*domain.CollectionItem, error)
}
