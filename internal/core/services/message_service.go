package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	"duskforge-api/pkg/storage"

	"github.com/google/uuid"
)

type messageService struct {
	messageRepo    ports.MessageRepository
	followRepo     ports.FollowRepository
	userRepo       ports.UserRepository
	blockRepo      ports.BlockRepository
	attachmentRepo ports.MessageAttachmentRepository
	reactionRepo   ports.MessageReactionRepository
	convStateRepo  ports.ConversationStateRepository
	storage        *storage.MinioStorage
}

func NewMessageService(
	messageRepo ports.MessageRepository,
	followRepo ports.FollowRepository,
	userRepo ports.UserRepository,
	blockRepo ports.BlockRepository,
	attachmentRepo ports.MessageAttachmentRepository,
	reactionRepo ports.MessageReactionRepository,
	convStateRepo ports.ConversationStateRepository,
	storage *storage.MinioStorage,
) ports.MessageService {
	return &messageService{
		messageRepo:    messageRepo,
		followRepo:     followRepo,
		userRepo:       userRepo,
		blockRepo:      blockRepo,
		attachmentRepo: attachmentRepo,
		reactionRepo:   reactionRepo,
		convStateRepo:  convStateRepo,
		storage:        storage,
	}
}

func (s *messageService) SendMessage(ctx context.Context, senderID, receiverID uuid.UUID, content *string, attachments []ports.AttachmentInput) (*domain.Message, []*domain.MessageAttachment, error) {
	if senderID == receiverID {
		return nil, nil, domain.ErrCannotMessageSelf
	}

	hasContent := content != nil && strings.TrimSpace(*content) != ""
	hasAttachments := len(attachments) > 0

	if !hasContent && !hasAttachments {
		return nil, nil, domain.ErrNoContent
	}

	if len(attachments) > 10 {
		return nil, nil, domain.ErrTooManyAttachments
	}

	receiver, err := s.userRepo.GetByID(ctx, receiverID)
	if err != nil {
		return nil, nil, err
	}
	if receiver == nil {
		return nil, nil, domain.ErrUserNotFound
	}

	blocked, err := s.blockRepo.IsBlocked(ctx, senderID, receiverID)
	if err != nil {
		return nil, nil, err
	}
	if blocked {
		return nil, nil, domain.ErrUserBlocked
	}

	senderFollowsReceiver, err := s.followRepo.GetByFollowerIDAndFollowingID(ctx, senderID, receiverID)
	if err != nil {
		return nil, nil, err
	}
	receiverFollowsSender, err := s.followRepo.GetByFollowerIDAndFollowingID(ctx, receiverID, senderID)
	if err != nil {
		return nil, nil, err
	}
	if senderFollowsReceiver == nil || receiverFollowsSender == nil {
		return nil, nil, domain.ErrNotMutualFollow
	}

	var messageContent *string
	if hasContent {
		trimmed := strings.TrimSpace(*content)
		messageContent = &trimmed
	}

	message := &domain.Message{
		ID:         uuid.New(),
		SenderID:   senderID,
		ReceiverID: receiverID,
		Content:    messageContent,
		CreatedAt:  time.Now(),
	}

	if err := s.messageRepo.Create(ctx, message); err != nil {
		return nil, nil, err
	}

	var createdAttachments []*domain.MessageAttachment
	if hasAttachments {
		now := time.Now()
		createdAttachments = make([]*domain.MessageAttachment, 0, len(attachments))
		for i, att := range attachments {
			ext := filepath.Ext(att.FileName)
			objectName := fmt.Sprintf("messages/%s_%d_%d%s", message.ID.String(), now.UnixMilli(), i, ext)

			fileURL, err := s.storage.Upload(ctx, objectName, att.Reader, att.FileSize, att.ContentType)
			if err != nil {
				return nil, nil, domain.ErrInternal
			}

			attachment := &domain.MessageAttachment{
				ID:          uuid.New(),
				MessageID:   message.ID,
				FileURL:     fileURL,
				FileName:    att.FileName,
				FileSize:    int(att.FileSize),
				ContentType: att.ContentType,
				Position:    int16(i),
				CreatedAt:   now,
			}
			createdAttachments = append(createdAttachments, attachment)
		}

		if err := s.attachmentRepo.CreateBatch(ctx, createdAttachments); err != nil {
			return nil, nil, domain.ErrInternal
		}
	}

	// Auto-reopen conversation for both users if closed
	_ = s.convStateRepo.ClearClosedAt(ctx, receiverID, senderID)
	_ = s.convStateRepo.ClearClosedAt(ctx, senderID, receiverID)

	return message, createdAttachments, nil
}

func (s *messageService) GetConversation(ctx context.Context, userID, otherUserID uuid.UUID, offset, limit int) ([]*domain.Message, int, error) {
	return s.messageRepo.GetConversationPaginated(ctx, userID, otherUserID, offset, limit)
}

func (s *messageService) GetConversations(ctx context.Context, userID uuid.UUID, includeClosed bool, offset, limit int) ([]*ports.ConversationResponse, int, error) {
	var previews []*ports.ConversationPreview
	var total int
	var err error

	// Always exclude blocked users from conversation list
	excludeIDs := make(map[uuid.UUID]struct{})
	if blockerIDs, berr := s.blockRepo.GetBlockerIDs(ctx, userID); berr == nil {
		for _, id := range blockerIDs {
			excludeIDs[id] = struct{}{}
		}
	}
	if blockedIDs, berr := s.blockRepo.GetBlockedIDs(ctx, userID); berr == nil {
		for _, id := range blockedIDs {
			excludeIDs[id] = struct{}{}
		}
	}

	if !includeClosed {
		closedPartnerIDs, cerr := s.convStateRepo.GetClosedConversationPartnerIDs(ctx, userID)
		if cerr != nil {
			return nil, 0, cerr
		}
		for _, id := range closedPartnerIDs {
			excludeIDs[id] = struct{}{}
		}
	}

	excludeSlice := make([]uuid.UUID, 0, len(excludeIDs))
	for id := range excludeIDs {
		excludeSlice = append(excludeSlice, id)
	}

	previews, total, err = s.messageRepo.GetConversationsFiltered(ctx, userID, excludeSlice, offset, limit)
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

func (s *messageService) UpdateMessage(ctx context.Context, messageID, userID uuid.UUID, content string) (*domain.Message, error) {
	message, err := s.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if message == nil {
		return nil, domain.ErrMessageNotFound
	}

	if message.SenderID != userID {
		return nil, domain.ErrForbidden
	}

	message.Content = &content
	now := time.Now()
	message.UpdatedAt = &now

	if err := s.messageRepo.Update(ctx, message); err != nil {
		return nil, domain.ErrInternal
	}

	return message, nil
}

func (s *messageService) DeleteMessage(ctx context.Context, messageID, userID uuid.UUID) (*domain.Message, error) {
	message, err := s.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if message == nil {
		return nil, domain.ErrMessageNotFound
	}

	if message.SenderID != userID {
		return nil, domain.ErrForbidden
	}

	// Delete attachments from storage
	attachments, err := s.attachmentRepo.DeleteByMessageID(ctx, messageID)
	if err != nil {
		return nil, domain.ErrInternal
	}
	for _, a := range attachments {
		_ = s.storage.DeleteByURL(ctx, a.FileURL)
	}

	if err := s.messageRepo.Delete(ctx, messageID); err != nil {
		return nil, domain.ErrInternal
	}

	return message, nil
}

func (s *messageService) GetMessageByID(ctx context.Context, messageID, userID uuid.UUID) (*domain.Message, error) {
	message, err := s.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if message == nil {
		return nil, domain.ErrMessageNotFound
	}

	if message.SenderID != userID && message.ReceiverID != userID {
		return nil, domain.ErrNotParticipant
	}

	return message, nil
}

func (s *messageService) AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) (*domain.MessageReaction, error) {
	message, err := s.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if message == nil {
		return nil, domain.ErrMessageNotFound
	}

	if message.SenderID != userID && message.ReceiverID != userID {
		return nil, domain.ErrNotParticipant
	}

	otherUserID := message.SenderID
	if message.SenderID == userID {
		otherUserID = message.ReceiverID
	}

	blocked, err := s.blockRepo.IsBlocked(ctx, userID, otherUserID)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if blocked {
		return nil, domain.ErrUserBlocked
	}

	exists, err := s.reactionRepo.Exists(ctx, messageID, userID, emoji)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if exists {
		return nil, domain.ErrReactionAlreadyExists
	}

	reaction := &domain.MessageReaction{
		MessageID: messageID,
		UserID:    userID,
		Emoji:     emoji,
		CreatedAt: time.Now(),
	}

	if err := s.reactionRepo.Create(ctx, reaction); err != nil {
		return nil, domain.ErrInternal
	}

	return reaction, nil
}

func (s *messageService) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	exists, err := s.reactionRepo.Exists(ctx, messageID, userID, emoji)
	if err != nil {
		return domain.ErrInternal
	}
	if !exists {
		return domain.ErrReactionNotFound
	}

	if err := s.reactionRepo.Delete(ctx, messageID, userID, emoji); err != nil {
		return domain.ErrInternal
	}

	return nil
}

func (s *messageService) CloseConversation(ctx context.Context, userID, otherUserID uuid.UUID) error {
	existing, err := s.convStateRepo.GetByUserAndOther(ctx, userID, otherUserID)
	if err != nil {
		return domain.ErrInternal
	}
	if existing != nil && existing.ClosedAt != nil {
		return domain.ErrConversationAlreadyClosed
	}

	now := time.Now()
	state := &domain.ConversationState{
		UserID:      userID,
		OtherUserID: otherUserID,
		ClosedAt:    &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.convStateRepo.Upsert(ctx, state); err != nil {
		return domain.ErrInternal
	}

	return nil
}

func (s *messageService) ReopenConversation(ctx context.Context, userID, otherUserID uuid.UUID) error {
	existing, err := s.convStateRepo.GetByUserAndOther(ctx, userID, otherUserID)
	if err != nil {
		return domain.ErrInternal
	}
	if existing == nil || existing.ClosedAt == nil {
		return domain.ErrConversationNotClosed
	}

	if err := s.convStateRepo.ClearClosedAt(ctx, userID, otherUserID); err != nil {
		return domain.ErrInternal
	}

	return nil
}

func (s *messageService) GetAttachmentsByMessageIDs(ctx context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]*domain.MessageAttachment, error) {
	return s.attachmentRepo.GetByMessageIDs(ctx, messageIDs)
}

func (s *messageService) GetReactionsByMessageIDs(ctx context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]*domain.MessageReaction, error) {
	return s.reactionRepo.GetByMessageIDs(ctx, messageIDs)
}
