package websocket

const (
	EventMessageNew      = "message.new"
	EventMessageUpdated  = "message.updated"
	EventMessageDeleted  = "message.deleted"
	EventReactionAdded   = "reaction.added"
	EventReactionRemoved = "reaction.removed"
	EventConversationRead = "conversation.read"
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
	ReaderID  string `json:"reader_id"`
	OtherID   string `json:"other_id"`
}
