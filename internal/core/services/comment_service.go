package services

import (
	"context"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/google/uuid"
)

type commentService struct {
	commentRepo     ports.CommentRepository
	commentLikeRepo ports.CommentLikeRepository
	reviewRepo      ports.ReviewRepository
}

func NewCommentService(
	commentRepo ports.CommentRepository,
	commentLikeRepo ports.CommentLikeRepository,
	reviewRepo ports.ReviewRepository,
) ports.CommentService {
	return &commentService{
		commentRepo:     commentRepo,
		commentLikeRepo: commentLikeRepo,
		reviewRepo:      reviewRepo,
	}
}

func (s *commentService) Create(ctx context.Context, reviewID uuid.UUID, userID uuid.UUID, input ports.CreateCommentInput) (*domain.Comment, error) {
	review, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if review == nil {
		return nil, domain.ErrReviewNotFound
	}

	if input.Content == "" {
		return nil, domain.ErrInvalidInput
	}

	now := time.Now()
	comment := &domain.Comment{
		ID:               uuid.New(),
		UserID:           userID,
		ReviewID:         reviewID,
		Content:          input.Content,
		ContainsSpoilers: input.ContainsSpoilers,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.commentRepo.Create(ctx, comment); err != nil {
		return nil, domain.ErrInternal
	}

	return comment, nil
}

func (s *commentService) GetByReviewID(ctx context.Context, reviewID uuid.UUID, requestingUserID *uuid.UUID) ([]*ports.CommentWithMeta, error) {
	comments, err := s.commentRepo.GetByReviewID(ctx, reviewID)
	if err != nil {
		return nil, domain.ErrInternal
	}

	result := make([]*ports.CommentWithMeta, len(comments))
	for i, comment := range comments {
		enriched, err := s.enrichComment(ctx, comment, requestingUserID)
		if err != nil {
			return nil, err
		}
		result[i] = enriched
	}

	return result, nil
}

func (s *commentService) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, input ports.UpdateCommentInput) (*domain.Comment, error) {
	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if comment == nil {
		return nil, domain.ErrCommentNotFound
	}

	if comment.UserID != userID {
		return nil, domain.ErrForbidden
	}

	if input.Content != nil {
		if *input.Content == "" {
			return nil, domain.ErrInvalidInput
		}
		comment.Content = *input.Content
	}
	if input.ContainsSpoilers != nil {
		comment.ContainsSpoilers = *input.ContainsSpoilers
	}

	comment.UpdatedAt = time.Now()

	if err := s.commentRepo.Update(ctx, comment); err != nil {
		return nil, domain.ErrInternal
	}

	return comment, nil
}

func (s *commentService) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		return domain.ErrInternal
	}
	if comment == nil {
		return domain.ErrCommentNotFound
	}

	if comment.UserID != userID {
		return domain.ErrForbidden
	}

	if err := s.commentRepo.Delete(ctx, id); err != nil {
		return domain.ErrInternal
	}

	return nil
}

func (s *commentService) Like(ctx context.Context, commentID uuid.UUID, userID uuid.UUID) error {
	comment, err := s.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return domain.ErrInternal
	}
	if comment == nil {
		return domain.ErrCommentNotFound
	}

	existing, err := s.commentLikeRepo.GetByUserIDAndCommentID(ctx, userID, commentID)
	if err != nil {
		return domain.ErrInternal
	}
	if existing != nil {
		return domain.ErrAlreadyLiked
	}

	like := &domain.CommentLike{
		UserID:    userID,
		CommentID: commentID,
		CreatedAt: time.Now(),
	}

	if err := s.commentLikeRepo.Create(ctx, like); err != nil {
		return domain.ErrInternal
	}

	return nil
}

func (s *commentService) Unlike(ctx context.Context, commentID uuid.UUID, userID uuid.UUID) error {
	comment, err := s.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return domain.ErrInternal
	}
	if comment == nil {
		return domain.ErrCommentNotFound
	}

	existing, err := s.commentLikeRepo.GetByUserIDAndCommentID(ctx, userID, commentID)
	if err != nil {
		return domain.ErrInternal
	}
	if existing == nil {
		return domain.ErrNotLiked
	}

	if err := s.commentLikeRepo.Delete(ctx, userID, commentID); err != nil {
		return domain.ErrInternal
	}

	return nil
}

func (s *commentService) enrichComment(ctx context.Context, comment *domain.Comment, requestingUserID *uuid.UUID) (*ports.CommentWithMeta, error) {
	likeCount, err := s.commentLikeRepo.CountByCommentID(ctx, comment.ID)
	if err != nil {
		return nil, domain.ErrInternal
	}

	likedByUser := false
	if requestingUserID != nil {
		existing, err := s.commentLikeRepo.GetByUserIDAndCommentID(ctx, *requestingUserID, comment.ID)
		if err != nil {
			return nil, domain.ErrInternal
		}
		likedByUser = existing != nil
	}

	return &ports.CommentWithMeta{
		Comment:     comment,
		LikeCount:   likeCount,
		LikedByUser: likedByUser,
	}, nil
}
