package services

import (
	"context"
	"errors"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/google/uuid"
)

type reviewService struct {
	reviewRepo     ports.ReviewRepository
	reviewLikeRepo ports.ReviewLikeRepository
	commentRepo    ports.CommentRepository
	collectionSvc  ports.CollectionService
	userRepo       ports.UserRepository
}

func NewReviewService(
	reviewRepo ports.ReviewRepository,
	reviewLikeRepo ports.ReviewLikeRepository,
	commentRepo ports.CommentRepository,
	collectionSvc ports.CollectionService,
	userRepo ports.UserRepository,
) ports.ReviewService {
	return &reviewService{
		reviewRepo:     reviewRepo,
		reviewLikeRepo: reviewLikeRepo,
		commentRepo:    commentRepo,
		collectionSvc:  collectionSvc,
		userRepo:       userRepo,
	}
}

func (s *reviewService) Create(ctx context.Context, userID uuid.UUID, tmdbID int, input ports.CreateReviewInput) (*domain.Review, error) {
	if input.Rating < 0.5 || input.Rating > 5.0 {
		return nil, domain.ErrInvalidInput
	}
	if input.Rating*2 != float64(int(input.Rating*2)) {
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

	err = s.collectionSvc.RemoveItem(ctx, userID, "to-watch", tmdbID)
	if err != nil && !errors.Is(err, domain.ErrCollectionItemNotFound) {
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

func (s *reviewService) GetByUserID(ctx context.Context, userID uuid.UUID, tmdbID *int, requestingUserID *uuid.UUID, offset, limit int, sort ports.ReviewSort) ([]*ports.ReviewWithMeta, int, error) {
	reviews, err := s.reviewRepo.GetByUserID(ctx, userID, tmdbID, offset, limit, sort)
	if err != nil {
		return nil, 0, domain.ErrInternal
	}

	total, err := s.reviewRepo.CountByUserID(ctx, userID, tmdbID)
	if err != nil {
		return nil, 0, domain.ErrInternal
	}

	result, err := s.enrichReviews(ctx, reviews, requestingUserID)
	if err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

func (s *reviewService) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, input ports.UpdateReviewInput) (*ports.ReviewWithMeta, error) {
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
		if *input.Rating*2 != float64(int(*input.Rating*2)) {
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

	enriched, err := s.enrichReview(ctx, review, &userID)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, domain.ErrInternal
	}
	enriched.User = user

	return enriched, nil
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
	if len(reviews) == 0 {
		return []*ports.ReviewWithMeta{}, nil
	}

	userIDSet := make(map[uuid.UUID]struct{})
	reviewIDs := make([]uuid.UUID, len(reviews))
	for i, r := range reviews {
		userIDSet[r.UserID] = struct{}{}
		reviewIDs[i] = r.ID
	}
	userIDs := make([]uuid.UUID, 0, len(userIDSet))
	for id := range userIDSet {
		userIDs = append(userIDs, id)
	}

	users, err := s.userRepo.GetByIDs(ctx, userIDs)
	if err != nil {
		return nil, domain.ErrInternal
	}
	userMap := make(map[uuid.UUID]*domain.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	likeCounts, err := s.reviewLikeRepo.CountByReviewIDs(ctx, reviewIDs)
	if err != nil {
		return nil, domain.ErrInternal
	}

	commentCounts, err := s.commentRepo.CountByReviewIDs(ctx, reviewIDs)
	if err != nil {
		return nil, domain.ErrInternal
	}

	var likedByUser map[uuid.UUID]bool
	if requestingUserID != nil {
		likedByUser, err = s.reviewLikeRepo.GetLikedByUser(ctx, *requestingUserID, reviewIDs)
		if err != nil {
			return nil, domain.ErrInternal
		}
	}

	result := make([]*ports.ReviewWithMeta, len(reviews))
	for i, review := range reviews {
		result[i] = &ports.ReviewWithMeta{
			Review:       review,
			LikeCount:    likeCounts[review.ID],
			CommentCount: commentCounts[review.ID],
			LikedByUser:  likedByUser[review.ID],
			User:         userMap[review.UserID],
		}
	}

	return result, nil
}

func (s *reviewService) enrichReview(ctx context.Context, review *domain.Review, requestingUserID *uuid.UUID) (*ports.ReviewWithMeta, error) {
	likeCount, err := s.reviewLikeRepo.CountByReviewID(ctx, review.ID)
	if err != nil {
		return nil, domain.ErrInternal
	}

	commentCount, err := s.commentRepo.CountByReviewID(ctx, review.ID)
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
		Review:       review,
		LikeCount:    likeCount,
		LikedByUser:  likedByUser,
		CommentCount: commentCount,
	}, nil
}
