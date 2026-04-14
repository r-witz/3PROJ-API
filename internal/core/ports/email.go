package ports

import "context"

type EmailSender interface {
	SendVerificationCode(ctx context.Context, to string, code string) error
	SendPasswordResetCode(ctx context.Context, to string, code string) error
}
