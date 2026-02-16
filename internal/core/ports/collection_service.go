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
	GetByID(ctx context.Context, collectionID uuid.UUID, requestingUserID *uuid.UUID) (*domain.Collection, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, requestingUserID *uuid.UUID) ([]*domain.Collection, error)
	Update(ctx context.Context, collectionID uuid.UUID, userID uuid.UUID, input UpdateCollectionInput) (*domain.Collection, error)
	Delete(ctx context.Context, collectionID uuid.UUID, userID uuid.UUID) error
	AddItem(ctx context.Context, collectionID uuid.UUID, userID uuid.UUID, tmdbID int, runtime int16) (*domain.CollectionItem, error)
	RemoveItem(ctx context.Context, collectionID uuid.UUID, userID uuid.UUID, tmdbID int) error
	GetItems(ctx context.Context, collectionID uuid.UUID, requestingUserID *uuid.UUID) ([]*domain.CollectionItem, error)
}
