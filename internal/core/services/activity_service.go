package services

import (
	"context"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/google/uuid"
)

type activityService struct {
	activityRepo   ports.ActivityRepository
	userRepo       ports.UserRepository
	reviewRepo     ports.ReviewRepository
	collectionRepo ports.CollectionRepository
	commentRepo    ports.CommentRepository
}

func NewActivityService(
	activityRepo ports.ActivityRepository,
	userRepo ports.UserRepository,
	reviewRepo ports.ReviewRepository,
	collectionRepo ports.CollectionRepository,
	commentRepo ports.CommentRepository,
) ports.ActivityService {
	return &activityService{
		activityRepo:   activityRepo,
		userRepo:       userRepo,
		reviewRepo:     reviewRepo,
		collectionRepo: collectionRepo,
		commentRepo:    commentRepo,
	}
}

func (s *activityService) GetByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*ports.ActivityFeedItem, int, error) {
	activities, err := s.activityRepo.GetByUserIDPaginated(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, domain.ErrInternal
	}

	total, err := s.activityRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, 0, domain.ErrInternal
	}

	items, err := s.enrichActivities(ctx, activities)
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (s *activityService) GetFeedForUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*ports.ActivityFeedItem, int, error) {
	activities, err := s.activityRepo.GetFeedForUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, domain.ErrInternal
	}

	total, err := s.activityRepo.CountFeedForUser(ctx, userID)
	if err != nil {
		return nil, 0, domain.ErrInternal
	}

	items, err := s.enrichActivities(ctx, activities)
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (s *activityService) Create(ctx context.Context, activity *domain.Activity) error {
	return s.activityRepo.Create(ctx, activity)
}

func (s *activityService) DeleteByTypeAndReference(ctx context.Context, userID uuid.UUID, actType domain.ActivityType, reviewID *uuid.UUID, collectionID *uuid.UUID, commentID *uuid.UUID, tmdbID *int) error {
	return s.activityRepo.DeleteByTypeAndReference(ctx, userID, actType, reviewID, collectionID, commentID, tmdbID)
}

func (s *activityService) enrichActivities(ctx context.Context, activities []*domain.Activity) ([]*ports.ActivityFeedItem, error) {
	if len(activities) == 0 {
		return []*ports.ActivityFeedItem{}, nil
	}

	userIDSet := make(map[uuid.UUID]struct{})
	reviewIDSet := make(map[uuid.UUID]struct{})
	collectionIDSet := make(map[uuid.UUID]struct{})
	commentIDSet := make(map[uuid.UUID]struct{})

	for _, a := range activities {
		userIDSet[a.UserID] = struct{}{}
		if a.ReviewID != nil {
			reviewIDSet[*a.ReviewID] = struct{}{}
		}
		if a.CollectionID != nil {
			collectionIDSet[*a.CollectionID] = struct{}{}
		}
		if a.CommentID != nil {
			commentIDSet[*a.CommentID] = struct{}{}
		}
	}

	userIDs := uuidSetToSlice(userIDSet)
	users, err := s.userRepo.GetByIDs(ctx, userIDs)
	if err != nil {
		return nil, domain.ErrInternal
	}
	userMap := make(map[uuid.UUID]*domain.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	reviewMap := make(map[uuid.UUID]*domain.Review)
	if len(reviewIDSet) > 0 {
		reviewIDs := uuidSetToSlice(reviewIDSet)
		reviews, err := s.reviewRepo.GetByIDs(ctx, reviewIDs)
		if err != nil {
			return nil, domain.ErrInternal
		}
		for _, r := range reviews {
			reviewMap[r.ID] = r
		}
	}

	collectionMap := make(map[uuid.UUID]*domain.Collection)
	if len(collectionIDSet) > 0 {
		collectionIDs := uuidSetToSlice(collectionIDSet)
		collections, err := s.collectionRepo.GetByIDs(ctx, collectionIDs)
		if err != nil {
			return nil, domain.ErrInternal
		}
		for _, c := range collections {
			collectionMap[c.ID] = c
		}
	}

	commentMap := make(map[uuid.UUID]*domain.Comment)
	if len(commentIDSet) > 0 {
		commentIDs := uuidSetToSlice(commentIDSet)
		comments, err := s.commentRepo.GetByIDs(ctx, commentIDs)
		if err != nil {
			return nil, domain.ErrInternal
		}
		for _, c := range comments {
			commentMap[c.ID] = c
		}
	}

	result := make([]*ports.ActivityFeedItem, len(activities))
	for i, a := range activities {
		item := &ports.ActivityFeedItem{
			Activity: a,
			User:     userMap[a.UserID],
		}
		if a.ReviewID != nil {
			item.Review = reviewMap[*a.ReviewID]
		}
		if a.CollectionID != nil {
			item.Collection = collectionMap[*a.CollectionID]
		}
		if a.CommentID != nil {
			item.Comment = commentMap[*a.CommentID]
		}
		result[i] = item
	}

	return result, nil
}

func uuidSetToSlice(set map[uuid.UUID]struct{}) []uuid.UUID {
	slice := make([]uuid.UUID, 0, len(set))
	for id := range set {
		slice = append(slice, id)
	}
	return slice
}
