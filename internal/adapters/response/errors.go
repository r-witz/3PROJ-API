package response

import (
	"errors"
	"net/http"

	"duskforge-api/internal/core/domain"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type ErrorMapping struct {
	Status  int
	Code    string
	Message string
}

var errorMappings = map[error]ErrorMapping{
	domain.ErrNotFound:              {http.StatusNotFound, "NOT_FOUND", "Resource not found"},
	domain.ErrAlreadyExists:         {http.StatusConflict, "ALREADY_EXISTS", "Resource already exists"},
	domain.ErrInvalidInput:          {http.StatusBadRequest, "INVALID_INPUT", "Invalid input provided"},
	domain.ErrUnauthorized:          {http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized"},
	domain.ErrForbidden:             {http.StatusForbidden, "FORBIDDEN", "Access denied"},
	domain.ErrInternal:              {http.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred"},
	domain.ErrInvalidCredentials:    {http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password"},
	domain.ErrEmailAlreadyExists:    {http.StatusConflict, "EMAIL_EXISTS", "Email is already registered"},
	domain.ErrEmailRequired:         {http.StatusBadRequest, "EMAIL_REQUIRED", "Email is required"},
	domain.ErrUsernameAlreadyExists: {http.StatusConflict, "USERNAME_EXISTS", "Username is already taken"},
	domain.ErrUsernameRequired:      {http.StatusBadRequest, "USERNAME_REQUIRED", "Username is required"},
	domain.ErrUsernameTooShort:      {http.StatusBadRequest, "USERNAME_TOO_SHORT", "Username must be at least 3 characters"},
	domain.ErrUsernameTooLong:       {http.StatusBadRequest, "USERNAME_TOO_LONG", "Username must be at most 50 characters"},
	domain.ErrInvalidEmailFormat:    {http.StatusBadRequest, "INVALID_EMAIL", "Email must be a valid email address"},
	domain.ErrInvalidToken:          {http.StatusUnauthorized, "INVALID_TOKEN", "Invalid or expired token"},
	domain.ErrSessionExpired:        {http.StatusUnauthorized, "SESSION_EXPIRED", "Session has expired"},
	domain.ErrUserBanned:            {http.StatusForbidden, "USER_BANNED", "Account has been banned"},
	domain.ErrUserNotFound:       {http.StatusNotFound, "USER_NOT_FOUND", "User not found"},
	domain.ErrNoPasswordSet:      {http.StatusConflict, "NO_PASSWORD_SET", "No password set for this account"},
	domain.ErrIncorrectPassword:  {http.StatusUnauthorized, "INCORRECT_PASSWORD", "Current password is incorrect"},

	domain.ErrPasswordRequired:      {http.StatusBadRequest, "PASSWORD_REQUIRED", "Password is required"},
	domain.ErrPasswordTooShort:      {http.StatusBadRequest, "PASSWORD_TOO_SHORT", "Password must be at least 8 characters"},
	domain.ErrPasswordTooLong:       {http.StatusBadRequest, "PASSWORD_TOO_LONG", "Password must be at most 72 characters"},
	domain.ErrPasswordNoUppercase:   {http.StatusBadRequest, "PASSWORD_NO_UPPERCASE", "Password must contain at least one uppercase letter"},
	domain.ErrPasswordNoLowercase:   {http.StatusBadRequest, "PASSWORD_NO_LOWERCASE", "Password must contain at least one lowercase letter"},
	domain.ErrPasswordNoDigit:       {http.StatusBadRequest, "PASSWORD_NO_DIGIT", "Password must contain at least one digit"},
	domain.ErrPasswordNoSpecialChar: {http.StatusBadRequest, "PASSWORD_NO_SPECIAL", "Password must contain at least one special character"},

	domain.ErrMovieNotFound:      {http.StatusNotFound, "MOVIE_NOT_FOUND", "Movie not found"},
	domain.ErrActorNotFound:      {http.StatusNotFound, "ACTOR_NOT_FOUND", "Actor not found"},
	domain.ErrTMDBError:             {http.StatusBadGateway, "EXTERNAL_SERVICE_ERROR", "Movie service is temporarily unavailable"},

	domain.ErrOAuthAccountNotFound:      {http.StatusNotFound, "OAUTH_NOT_FOUND", "OAuth account not found"},
	domain.ErrOAuthAccountAlreadyLinked: {http.StatusConflict, "OAUTH_ALREADY_LINKED", "OAuth account is already linked to another user"},
	domain.ErrOAuthProviderNotSupported: {http.StatusBadRequest, "OAUTH_PROVIDER_INVALID", "OAuth provider not supported"},
	domain.ErrOAuthStateMismatch:        {http.StatusBadRequest, "OAUTH_STATE_INVALID", "Invalid OAuth state parameter"},
	domain.ErrCannotUnlinkOnlyAuth:      {http.StatusBadRequest, "CANNOT_UNLINK", "Cannot remove the only authentication method"},

	domain.ErrCollectionNotFound:           {http.StatusNotFound, "COLLECTION_NOT_FOUND", "Collection not found"},
	domain.ErrCollectionAlreadyExists:      {http.StatusConflict, "COLLECTION_EXISTS", "A collection with this name already exists"},
	domain.ErrCannotModifySystemCollection: {http.StatusForbidden, "SYSTEM_COLLECTION", "Cannot modify a system collection"},
	domain.ErrCannotDeleteSystemCollection: {http.StatusForbidden, "SYSTEM_COLLECTION", "Cannot delete a system collection"},
	domain.ErrCollectionItemAlreadyExists:  {http.StatusConflict, "ITEM_EXISTS", "Item already exists in this collection"},
	domain.ErrCollectionItemNotFound:       {http.StatusNotFound, "ITEM_NOT_FOUND", "Item not found in this collection"},

	domain.ErrNotificationNotFound: {http.StatusNotFound, "NOTIFICATION_NOT_FOUND", "Notification not found"},

	domain.ErrAchievementNotFound: {http.StatusNotFound, "ACHIEVEMENT_NOT_FOUND", "Achievement not found"},

	domain.ErrReviewNotFound:      {http.StatusNotFound, "REVIEW_NOT_FOUND", "Review not found"},
	domain.ErrReviewAlreadyExists: {http.StatusConflict, "REVIEW_EXISTS", "You have already reviewed this movie"},
	domain.ErrCommentNotFound:     {http.StatusNotFound, "COMMENT_NOT_FOUND", "Comment not found"},
	domain.ErrAlreadyLiked:        {http.StatusConflict, "ALREADY_LIKED", "Already liked"},
	domain.ErrNotLiked:            {http.StatusNotFound, "NOT_LIKED", "Not liked"},

	domain.ErrCannotFollowSelf: {http.StatusBadRequest, "CANNOT_FOLLOW_SELF", "Cannot follow yourself"},
	domain.ErrAlreadyFollowing: {http.StatusConflict, "ALREADY_FOLLOWING", "Already following this user"},
	domain.ErrNotFollowing:     {http.StatusNotFound, "NOT_FOLLOWING", "Not following this user"},

	domain.ErrNotMutualFollow:   {http.StatusForbidden, "NOT_MUTUAL_FOLLOW", "Users must follow each other to message"},
	domain.ErrMessageNotFound:   {http.StatusNotFound, "MESSAGE_NOT_FOUND", "Message not found"},
	domain.ErrCannotMessageSelf: {http.StatusBadRequest, "CANNOT_MESSAGE_SELF", "Cannot send a message to yourself"},

	domain.ErrCannotBlockSelf:  {http.StatusBadRequest, "CANNOT_BLOCK_SELF", "Cannot block yourself"},
	domain.ErrCannotBlockAdmin: {http.StatusForbidden, "CANNOT_BLOCK_ADMIN", "Cannot block an admin or super-admin"},
	domain.ErrAlreadyBlocked:   {http.StatusConflict, "ALREADY_BLOCKED", "User is already blocked"},
	domain.ErrNotBlocked:      {http.StatusNotFound, "NOT_BLOCKED", "User is not blocked"},
	domain.ErrUserBlocked:     {http.StatusForbidden, "USER_BLOCKED", "Action blocked due to user block"},

	domain.ErrNoContent:             {http.StatusBadRequest, "NO_CONTENT", "Message must have content or attachments"},
	domain.ErrTooManyAttachments:    {http.StatusBadRequest, "TOO_MANY_ATTACHMENTS", "Too many attachments"},
	domain.ErrAttachmentTooLarge:    {http.StatusBadRequest, "ATTACHMENT_TOO_LARGE", "Attachment file is too large"},
	domain.ErrInvalidAttachmentType: {http.StatusBadRequest, "INVALID_ATTACHMENT_TYPE", "Invalid attachment content type"},

	domain.ErrReactionAlreadyExists: {http.StatusConflict, "REACTION_EXISTS", "Reaction already exists"},
	domain.ErrReactionNotFound:      {http.StatusNotFound, "REACTION_NOT_FOUND", "Reaction not found"},
	domain.ErrNotParticipant:        {http.StatusForbidden, "NOT_PARTICIPANT", "User is not a participant of this conversation"},
	domain.ErrTooManyReactionTypes:  {http.StatusBadRequest, "TOO_MANY_REACTION_TYPES", "Maximum of 5 different emoji reactions per message"},

	domain.ErrConversationAlreadyClosed: {http.StatusConflict, "CONVERSATION_ALREADY_CLOSED", "Conversation is already closed"},
	domain.ErrConversationNotClosed:     {http.StatusConflict, "CONVERSATION_NOT_CLOSED", "Conversation is not closed"},

	domain.ErrCannotDeleteSuperAdmin: {http.StatusForbidden, "CANNOT_DELETE_SUPERADMIN", "Super-admin account cannot be deleted"},
	domain.ErrCannotBanAdmin:         {http.StatusForbidden, "CANNOT_BAN_ADMIN", "Cannot ban an admin or super-admin"},
	domain.ErrCannotBanSelf:         {http.StatusBadRequest, "CANNOT_BAN_SELF", "Cannot ban yourself"},
	domain.ErrUserAlreadyBanned:     {http.StatusConflict, "USER_ALREADY_BANNED", "User is already banned"},
	domain.ErrUserNotBanned:         {http.StatusConflict, "USER_NOT_BANNED", "User is not banned"},
	domain.ErrCannotChangeOwnRole:   {http.StatusBadRequest, "CANNOT_CHANGE_OWN_ROLE", "Cannot change your own role"},
	domain.ErrInsufficientRole:      {http.StatusForbidden, "INSUFFICIENT_ROLE", "Insufficient permissions for this action"},
	domain.ErrInvalidRole:           {http.StatusBadRequest, "INVALID_ROLE", "Invalid role"},
	domain.ErrReportNotFound:        {http.StatusNotFound, "REPORT_NOT_FOUND", "Report not found"},
	domain.ErrReportAlreadyResolved: {http.StatusConflict, "REPORT_ALREADY_RESOLVED", "Report is already resolved or dismissed"},
	domain.ErrInvalidReportTarget:   {http.StatusBadRequest, "INVALID_REPORT_TARGET", "Report must target exactly one entity"},

	domain.ErrInvalidImportFile:  {http.StatusBadRequest, "INVALID_IMPORT_FILE", "Invalid import file"},
	domain.ErrImportFileTooLarge: {http.StatusBadRequest, "IMPORT_FILE_TOO_LARGE", "Import file exceeds maximum size"},
	domain.ErrImportFileEmpty:    {http.StatusBadRequest, "IMPORT_FILE_EMPTY", "Zip file contains no Letterboxd data (expected watched.csv, ratings.csv, reviews.csv, or watchlist.csv)"},

	domain.ErrEmailAlreadyVerified:      {http.StatusConflict, "EMAIL_ALREADY_VERIFIED", "Email is already verified"},
	domain.ErrVerificationCodeInvalid:   {http.StatusBadRequest, "INVALID_CODE", "Invalid or expired verification code"},
	domain.ErrVerificationCodeRateLimit: {http.StatusTooManyRequests, "RATE_LIMIT", "Too many code requests, please try again later"},
	domain.ErrEmailNotVerified:          {http.StatusForbidden, "EMAIL_NOT_VERIFIED", "Email address has not been verified"},
}

func HandleError(c *gin.Context, err error) {
	for domainErr, mapping := range errorMappings {
		if errors.Is(err, domainErr) {
			Error(c, mapping.Status, mapping.Code, mapping.Message, nil)
			return
		}
	}

	InternalError(c)
}

func HandleValidationError(c *gin.Context, err error) bool {
	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return false
	}

	for _, fieldErr := range validationErrors {
		switch fieldErr.Field() {
		case "Email":
			switch fieldErr.Tag() {
			case "required":
				HandleError(c, domain.ErrEmailRequired)
				return true
			case "email":
				HandleError(c, domain.ErrInvalidEmailFormat)
				return true
			}
		case "Username":
			switch fieldErr.Tag() {
			case "required":
				HandleError(c, domain.ErrUsernameRequired)
				return true
			case "min":
				HandleError(c, domain.ErrUsernameTooShort)
				return true
			case "max":
				HandleError(c, domain.ErrUsernameTooLong)
				return true
			}
		case "Password", "NewPassword", "CurrentPassword":
			switch fieldErr.Tag() {
			case "required":
				HandleError(c, domain.ErrPasswordRequired)
				return true
			case "min":
				HandleError(c, domain.ErrPasswordTooShort)
				return true
			case "max":
				HandleError(c, domain.ErrPasswordTooLong)
				return true
			}
		}
	}

	return false
}
