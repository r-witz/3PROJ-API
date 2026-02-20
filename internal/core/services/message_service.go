package services

import (
	"context"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/google/uuid"
)

type messageService struct {
	messageRepo ports.MessageRepository
	followRepo  ports.FollowRepository
	userRepo    ports.UserRepository
}

func NewMessageService(messageRepo ports.MessageRepository, followRepo ports.FollowRepository, userRepo ports.UserRepository) ports.MessageService {
	return &messageService{
		messageRepo: messageRepo,
		followRepo:  followRepo,
		userRepo:    userRepo,
	}
}

func (s *messageService) SendMessage(ctx context.Context, senderID, receiverID uuid.UUID, content string) (*domain.Message, error) {
	if senderID == receiverID {
		return nil, domain.ErrCannotMessageSelf
	}

	receiver, err := s.userRepo.GetByID(ctx, receiverID)
	if err != nil {
		return nil, err
	}
	if receiver == nil {
		return nil, domain.ErrUserNotFound
	}

	senderFollowsReceiver, err := s.followRepo.GetByFollowerIDAndFollowingID(ctx, senderID, receiverID)
	if err != nil {
		return nil, err
	}
	receiverFollowsSender, err := s.followRepo.GetByFollowerIDAndFollowingID(ctx, receiverID, senderID)
	if err != nil {
		return nil, err
	}
	if senderFollowsReceiver == nil || receiverFollowsSender == nil {
		return nil, domain.ErrNotMutualFollow
	}

	message := &domain.Message{
		ID:         uuid.New(),
		SenderID:   senderID,
		ReceiverID: receiverID,
		Content:    content,
		CreatedAt:  time.Now(),
	}

	if err := s.messageRepo.Create(ctx, message); err != nil {
		return nil, err
	}

	return message, nil
}

func (s *messageService) GetConversation(ctx context.Context, userID, otherUserID uuid.UUID, offset, limit int) ([]*domain.Message, int, error) {
	return s.messageRepo.GetConversationPaginated(ctx, userID, otherUserID, offset, limit)
}

func (s *messageService) GetConversations(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*ports.ConversationResponse, int, error) {
	previews, total, err := s.messageRepo.GetConversations(ctx, userID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	userIDs := make([]uuid.UUID, len(previews))
	for i, p := range previews {
		userIDs[i] = p.OtherUserID
	}

	users, err := s.userRepo.GetByIDs(ctx, userIDs)
	if err != nil {
		return nil, 0, err
	}

	userMap := make(map[uuid.UUID]*domain.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	conversations := make([]*ports.ConversationResponse, 0, len(previews))
	for _, p := range previews {
		conv := &ports.ConversationResponse{
			OtherUser:   userMap[p.OtherUserID],
			LastMessage: p.LastMessage,
			UnreadCount: p.UnreadCount,
		}
		conversations = append(conversations, conv)
	}

	return conversations, total, nil
}

func (s *messageService) MarkAsRead(ctx context.Context, userID, otherUserID uuid.UUID) error {
	return s.messageRepo.MarkConversationAsRead(ctx, userID, otherUserID)
}
