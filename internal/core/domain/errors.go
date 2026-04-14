package domain

import "errors"

var (
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource already exists")
	ErrInvalidInput  = errors.New("invalid input")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrInternal      = errors.New("internal error")
)

var (
	ErrInvalidCredentials    = errors.New("invalid email or password")
	ErrEmailAlreadyExists    = errors.New("email already registered")
	ErrEmailRequired         = errors.New("email is required")
	ErrUsernameAlreadyExists = errors.New("username already taken")
	ErrUsernameRequired      = errors.New("username is required")
	ErrUsernameTooShort      = errors.New("username must be at least 3 characters")
	ErrUsernameTooLong       = errors.New("username must be at most 50 characters")
	ErrInvalidEmailFormat    = errors.New("invalid email format")
	ErrInvalidToken          = errors.New("invalid or expired token")
	ErrSessionExpired        = errors.New("session has expired")
	ErrUserBanned            = errors.New("user account is banned")
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrNoPasswordSet     = errors.New("no password set for this account")
	ErrIncorrectPassword = errors.New("current password is incorrect")
)

var (
	ErrPasswordRequired      = errors.New("password is required")
	ErrPasswordTooShort      = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong       = errors.New("password must be at most 72 characters")
	ErrPasswordNoUppercase   = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNoLowercase   = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNoDigit       = errors.New("password must contain at least one digit")
	ErrPasswordNoSpecialChar = errors.New("password must contain at least one special character")
)

var (
	ErrMovieNotFound = errors.New("movie not found")
	ErrActorNotFound = errors.New("actor not found")
	ErrTMDBError     = errors.New("external movie service error")
)

var (
	ErrOAuthAccountNotFound      = errors.New("oauth account not found")
	ErrOAuthAccountAlreadyLinked = errors.New("oauth account already linked to another user")
	ErrOAuthProviderNotSupported = errors.New("oauth provider not supported")
	ErrOAuthStateMismatch        = errors.New("invalid oauth state")
	ErrCannotUnlinkOnlyAuth      = errors.New("cannot unlink the only authentication method")
)

var (
	ErrCollectionNotFound            = errors.New("collection not found")
	ErrCollectionAlreadyExists       = errors.New("collection already exists")
	ErrCannotModifySystemCollection  = errors.New("cannot modify system collection")
	ErrCannotDeleteSystemCollection  = errors.New("cannot delete system collection")
	ErrCollectionItemAlreadyExists   = errors.New("item already exists in collection")
	ErrCollectionItemNotFound        = errors.New("collection item not found")
)

var (
	ErrReviewNotFound      = errors.New("review not found")
	ErrReviewAlreadyExists = errors.New("review already exists for this movie")
	ErrCommentNotFound     = errors.New("comment not found")
	ErrAlreadyLiked        = errors.New("already liked")
	ErrNotLiked            = errors.New("not liked")
)

var (
	ErrCannotFollowSelf = errors.New("cannot follow yourself")
	ErrAlreadyFollowing = errors.New("already following this user")
	ErrNotFollowing     = errors.New("not following this user")
)

var (
	ErrNotMutualFollow   = errors.New("users must follow each other to message")
	ErrMessageNotFound   = errors.New("message not found")
	ErrCannotMessageSelf = errors.New("cannot send a message to yourself")
)

var (
	ErrCannotBlockSelf = errors.New("cannot block yourself")
	ErrAlreadyBlocked  = errors.New("user is already blocked")
	ErrNotBlocked      = errors.New("user is not blocked")
	ErrUserBlocked     = errors.New("action blocked due to user block")
)

var (
	ErrNoContent             = errors.New("message must have content or attachments")
	ErrTooManyAttachments    = errors.New("too many attachments")
	ErrAttachmentTooLarge    = errors.New("attachment file is too large")
	ErrInvalidAttachmentType = errors.New("invalid attachment content type")
)

var (
	ErrReactionAlreadyExists = errors.New("reaction already exists")
	ErrReactionNotFound      = errors.New("reaction not found")
	ErrNotParticipant        = errors.New("user is not a participant of this conversation")
)

var (
	ErrConversationAlreadyClosed = errors.New("conversation is already closed")
	ErrConversationNotClosed     = errors.New("conversation is not closed")
)

var (
	ErrCannotBanAdmin        = errors.New("cannot ban an admin or super-admin")
	ErrCannotBanSelf         = errors.New("cannot ban yourself")
	ErrUserAlreadyBanned     = errors.New("user is already banned")
	ErrUserNotBanned         = errors.New("user is not banned")
	ErrCannotDeleteSuperAdmin = errors.New("super-admin account cannot be deleted")
	ErrCannotChangeOwnRole    = errors.New("cannot change your own role")
	ErrInsufficientRole      = errors.New("insufficient permissions for this action")
	ErrInvalidRole           = errors.New("invalid role")
	ErrReportNotFound        = errors.New("report not found")
	ErrReportAlreadyResolved = errors.New("report is already resolved or dismissed")
	ErrInvalidReportTarget   = errors.New("report must target exactly one entity")
)

var (
	ErrInvalidImportFile  = errors.New("invalid import file")
	ErrImportFileTooLarge = errors.New("import file too large")
)

var (
	ErrNotificationNotFound = errors.New("notification not found")
)
