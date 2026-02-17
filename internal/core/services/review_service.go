package services

import (
	"context"
	"errors"
	"math"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/google/uuid"
)

type reviewService struct {
	reviewRepo     ports.ReviewRepository
	reviewLikeRepo ports.ReviewLikeRepository
	collectionSvc  ports.CollectionService
	userRepo       ports.UserRepository
}

func NewReviewService(
	reviewRepo ports.ReviewRepository,
	reviewLikeRepo ports.ReviewLikeRepository,
	collectionSvc ports.CollectionService,
	userRepo ports.UserRepository,
) ports.ReviewService {
	return &reviewService{
		reviewRepo:     reviewRepo,
		reviewLikeRepo: reviewLikeRepo,
		collectionSvc:  collectionSvc,
		userRepo:       userRepo,
	}
}

func (s *reviewService) Create(ctx context.Context, userID uuid.UUID, tmdbID int, input ports.CreateReviewInput) (*domain.Review, error) {
	if input.Rating < 0.5 || input.Rating > 5.0 {
		return nil, domain.ErrInvalidInput
	}
	if math.Mod(input.Rating, 0.5) != 0 {
		return nil, domain.ErrInvalidInput
	}

	existing, err := s.reviewRepo.GetByUserIDAndTMDBID(ctx, userID, tmdbID)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if existing != nil {
		return nil, domain.ErrReviewAlreadyExists
	}

	now := time.Now()
	review := &domain.Review{
		ID:               uuid.New(),
		UserID:           userID,
		TMDBID:           tmdbID,
		Rating:           input.Rating,
		Content:          input.Content,
		ContainsSpoilers: input.ContainsSpoilers,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.reviewRepo.Create(ctx, review); err != nil {
		return nil, domain.ErrInternal
	}

	_, err = s.collectionSvc.AddItem(ctx, userID, "watched", tmdbID)
	if err != nil && !errors.Is(err, domain.ErrCollectionItemAlreadyExists) {
		// silently ignore — non-critical
	}

	return review, nil
}

func (s *reviewService) GetByID(ctx context.Context, id uuid.UUID, requestingUserID *uuid.UUID) (*ports.ReviewWithMeta, error) {
	review, err := s.reviewRepo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if review == nil {
		return nil, domain.ErrReviewNotFound
	}

	enriched, err := s.enrichReview(ctx, review, requestingUserID)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, review.UserID)
	if err != nil {
		return nil, domain.ErrInternal
	}
	enriched.User = user

	return enriched, nil
}

func (s *reviewService) GetByTMDBID(ctx context.Context, tmdbID int, requestingUserID *uuid.UUID, offset, limit int, sort ports.ReviewSort) ([]*ports.ReviewWithMeta, int, error) {
	reviews, err := s.reviewRepo.GetByTMDBID(ctx, tmdbID, offset, limit, sort)
	if err != nil {
		return nil, 0, domain.ErrInternal
	}

	total, err := s.reviewRepo.CountByTMDBID(ctx, tmdbID)
	if err != nil {
		return nil, 0, domain.ErrInternal
	}

	result, err := s.enrichReviews(ctx, reviews, requestingUserID)
	if err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

func (s *reviewService) GetByUserID(ctx context.Context, userID uuid.UUID, requestingUserID *uuid.UUID, offset, limit int, sort ports.ReviewSort) ([]*ports.ReviewWithMeta, int, error) {
	reviews, err := s.reviewRepo.GetByUserID(ctx, userID, offset, limit, sort)
	if err != nil {
		return nil, 0, domain.ErrInternal
	}

	total, err := s.reviewRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, 0, domain.ErrInternal
	}

	result, err := s.enrichReviews(ctx, reviews, requestingUserID)
	if err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

func (s *reviewService) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, input ports.UpdateReviewInput) (*domain.Review, error) {
	review, err := s.reviewRepo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if review == nil {
		return nil, domain.ErrReviewNotFound
	}

	if review.UserID != userID {
		return nil, domain.ErrForbidden
	}

	if input.Rating != nil {
		if *input.Rating < 0.5 || *input.Rating > 5.0 {
			return nil, domain.ErrInvalidInput
		}
		if math.Mod(*input.Rating, 0.5) != 0 {
			return nil, domain.ErrInvalidInput
		}
		review.Rating = *input.Rating
	}
	if input.Content != nil {
		review.Content = input.Content
	}
	if input.ContainsSpoilers != nil {
		review.ContainsSpoilers = *input.ContainsSpoilers
	}

	review.UpdatedAt = time.Now()

	if err := s.reviewRepo.Update(ctx, review); err != nil {
		return nil, domain.ErrInternal
	}

	return review, nil
}

func (s *reviewService) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	review, err := s.reviewRepo.GetByID(ctx, id)
	if err != nil {
		return domain.ErrInternal
	}
	if review == nil {
		return domain.ErrReviewNotFound
	}

	if review.UserID != userID {
		return domain.ErrForbidden
	}

	if err := s.reviewRepo.Delete(ctx, id); err != nil {
		return domain.ErrInternal
	}

	return nil
}

func (s *reviewService) Like(ctx context.Context, reviewID uuid.UUID, userID uuid.UUID) error {
	review, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return domain.ErrInternal
	}
	if review == nil {
		return domain.ErrReviewNotFound
	}

	existing, err := s.reviewLikeRepo.GetByUserIDAndReviewID(ctx, userID, reviewID)
	if err != nil {
		return domain.ErrInternal
	}
	if existing != nil {
		return domain.ErrAlreadyLiked
	}

	like := &domain.ReviewLike{
		UserID:    userID,
		ReviewID:  reviewID,
		CreatedAt: time.Now(),
	}

	if err := s.reviewLikeRepo.Create(ctx, like); err != nil {
		return domain.ErrInternal
	}

	return nil
}

func (s *reviewService) Unlike(ctx context.Context, reviewID uuid.UUID, userID uuid.UUID) error {
	review, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return domain.ErrInternal
	}
	if review == nil {
		return domain.ErrReviewNotFound
	}

	existing, err := s.reviewLikeRepo.GetByUserIDAndReviewID(ctx, userID, reviewID)
	if err != nil {
		return domain.ErrInternal
	}
	if existing == nil {
		return domain.ErrNotLiked
	}

	if err := s.reviewLikeRepo.Delete(ctx, userID, reviewID); err != nil {
		return domain.ErrInternal
	}

	return nil
}

func (s *reviewService) enrichReviews(ctx context.Context, reviews []*domain.Review, requestingUserID *uuid.UUID) ([]*ports.ReviewWithMeta, error) {
	// Collect unique user IDs
	userIDSet := make(map[uuid.UUID]struct{})
	for _, r := range reviews {
		userIDSet[r.UserID] = struct{}{}
	}
	userIDs := make([]uuid.UUID, 0, len(userIDSet))
	for id := range userIDSet {
		userIDs = append(userIDs, id)
	}

	// Batch fetch users
	users, err := s.userRepo.GetByIDs(ctx, userIDs)
	if err != nil {
		return nil, domain.ErrInternal
	}
	userMap := make(map[uuid.UUID]*domain.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	result := make([]*ports.ReviewWithMeta, len(reviews))
	for i, review := range reviews {
		enriched, err := s.enrichReview(ctx, review, requestingUserID)
		if err != nil {
			return nil, err
		}
		enriched.User = userMap[review.UserID]
		result[i] = enriched
	}

	return result, nil
}

func (s *reviewService) enrichReview(ctx context.Context, review *domain.Review, requestingUserID *uuid.UUID) (*ports.ReviewWithMeta, error) {
	likeCount, err := s.reviewLikeRepo.CountByReviewID(ctx, review.ID)
	if err != nil {
		return nil, domain.ErrInternal
	}

	likedByUser := false
	if requestingUserID != nil {
		existing, err := s.reviewLikeRepo.GetByUserIDAndReviewID(ctx, *requestingUserID, review.ID)
		if err != nil {
			return nil, domain.ErrInternal
		}
		likedByUser = existing != nil
	}

	return &ports.ReviewWithMeta{
		Review:      review,
		LikeCount:   likeCount,
		LikedByUser: likedByUser,
	}, nil
}
