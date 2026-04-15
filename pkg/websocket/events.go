package websocket

const (
	EventMessageNew       = "message.new"
	EventMessageUpdated   = "message.updated"
	EventMessageDeleted   = "message.deleted"
	EventReactionAdded    = "reaction.added"
	EventReactionRemoved  = "reaction.removed"
	EventConversationRead = "conversation.read"
	EventMessagingBlocked   = "messaging.blocked"
	EventMessagingUnblocked = "messaging.unblocked"
	EventConversationClosed   = "conversation.closed"
	EventConversationReopened = "conversation.reopened"
	EventImportProgress       = "import.progress"
	EventUserBanned           = "user.banned"
	EventNotificationNew      = "notification.new"
)

type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type MessageDeletedPayload struct {
	MessageID  string `json:"message_id"`
	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
}

type ReactionPayload struct {
	MessageID  string `json:"message_id"`
	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
	UserID     string `json:"user_id"`
	Emoji      string `json:"emoji"`
}

type ConversationReadPayload struct {
	ReaderID string `json:"reader_id"`
	OtherID  string `json:"other_id"`
}

type ConversationClosedPayload struct {
	OtherUserID string `json:"other_user_id"`
}

type MessagingBlockedPayload struct {
	UserID string `json:"user_id"`
	Reason string `json:"reason"`
}

type MessagingUnblockedPayload struct {
	UserID string `json:"user_id"`
	Reason string `json:"reason"`
}

type UserBannedPayload struct {
	UserID string `json:"user_id"`
}

type NotificationPayload struct {
	ID        string  `json:"id"`
	Type      string  `json:"type"`
	ActorID   *string `json:"actor_id,omitempty"`
	ReviewID  *string `json:"review_id,omitempty"`
	CommentID *string `json:"comment_id,omitempty"`
	Message   *string `json:"message,omitempty"`
	CreatedAt string  `json:"created_at"`
}
