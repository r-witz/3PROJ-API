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
	userRepo        ports.UserRepository
	blockRepo       ports.BlockRepository
	activityRepo    ports.ActivityRepository
}

func NewCommentService(
	commentRepo ports.CommentRepository,
	commentLikeRepo ports.CommentLikeRepository,
	reviewRepo ports.ReviewRepository,
	userRepo ports.UserRepository,
	blockRepo ports.BlockRepository,
	activityRepo ports.ActivityRepository,
) ports.CommentService {
	return &commentService{
		commentRepo:     commentRepo,
		commentLikeRepo: commentLikeRepo,
		reviewRepo:      reviewRepo,
		userRepo:        userRepo,
		blockRepo:       blockRepo,
		activityRepo:    activityRepo,
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

	if review.UserID != userID {
		blocked, err := s.blockRepo.IsBlocked(ctx, userID, review.UserID)
		if err != nil {
			return nil, domain.ErrInternal
		}
		if blocked {
			return nil, domain.ErrUserBlocked
		}
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

	_ = s.activityRepo.Create(ctx, &domain.Activity{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      domain.ActivityTypeCommentCreated,
		CommentID: &comment.ID,
		CreatedAt: now,
	})

	return comment, nil
}

func (s *commentService) GetByReviewID(ctx context.Context, reviewID uuid.UUID, requestingUserID *uuid.UUID, offset, limit int) ([]*ports.CommentWithMeta, int, error) {
	comments, err := s.commentRepo.GetByReviewID(ctx, reviewID, offset, limit)
	if err != nil {
		return nil, 0, domain.ErrInternal
	}

	total, err := s.commentRepo.CountByReviewID(ctx, reviewID)
	if err != nil {
		return nil, 0, domain.ErrInternal
	}

	if len(comments) == 0 {
		return []*ports.CommentWithMeta{}, total, nil
	}

	userIDSet := make(map[uuid.UUID]struct{})
	commentIDs := make([]uuid.UUID, len(comments))
	for i, c := range comments {
		userIDSet[c.UserID] = struct{}{}
		commentIDs[i] = c.ID
	}
	userIDs := make([]uuid.UUID, 0, len(userIDSet))
	for id := range userIDSet {
		userIDs = append(userIDs, id)
	}

	users, err := s.userRepo.GetByIDs(ctx, userIDs)
	if err != nil {
		return nil, 0, domain.ErrInternal
	}
	userMap := make(map[uuid.UUID]*domain.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	likeCounts, err := s.commentLikeRepo.CountByCommentIDs(ctx, commentIDs)
	if err != nil {
		return nil, 0, domain.ErrInternal
	}

	var likedByUser map[uuid.UUID]bool
	if requestingUserID != nil {
		likedByUser, err = s.commentLikeRepo.GetLikedByUser(ctx, *requestingUserID, commentIDs)
		if err != nil {
			return nil, 0, domain.ErrInternal
		}
	}

	result := make([]*ports.CommentWithMeta, len(comments))
	for i, comment := range comments {
		result[i] = &ports.CommentWithMeta{
			Comment:     comment,
			LikeCount:   likeCounts[comment.ID],
			LikedByUser: likedByUser[comment.ID],
			User:        userMap[comment.UserID],
		}
	}

	return result, total, nil
}

func (s *commentService) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, input ports.UpdateCommentInput) (*ports.CommentWithMeta, error) {
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

	likeCount, err := s.commentLikeRepo.CountByCommentID(ctx, comment.ID)
	if err != nil {
		return nil, domain.ErrInternal
	}

	likedByUser := false
	existing, err := s.commentLikeRepo.GetByUserIDAndCommentID(ctx, userID, comment.ID)
	if err != nil {
		return nil, domain.ErrInternal
	}
	likedByUser = existing != nil

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, domain.ErrInternal
	}

	return &ports.CommentWithMeta{
		Comment:     comment,
		LikeCount:   likeCount,
		LikedByUser: likedByUser,
		User:        user,
	}, nil
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

	_ = s.activityRepo.DeleteByTypeAndReference(ctx, userID, domain.ActivityTypeCommentCreated, nil, nil, &id, nil)

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

	if comment.UserID != userID {
		blocked, err := s.blockRepo.IsBlocked(ctx, userID, comment.UserID)
		if err != nil {
			return domain.ErrInternal
		}
		if blocked {
			return domain.ErrUserBlocked
		}
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

	_ = s.activityRepo.Create(ctx, &domain.Activity{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      domain.ActivityTypeCommentLiked,
		CommentID: &commentID,
		CreatedAt: time.Now(),
	})

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

	if comment.UserID != userID {
		blocked, err := s.blockRepo.IsBlocked(ctx, userID, comment.UserID)
		if err != nil {
			return domain.ErrInternal
		}
		if blocked {
			return domain.ErrUserBlocked
		}
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

	_ = s.activityRepo.DeleteByTypeAndReference(ctx, userID, domain.ActivityTypeCommentLiked, nil, nil, &commentID, nil)

	return nil
}

