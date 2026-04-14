package domain

import (
	"time"

	"github.com/google/uuid"
)

type VerificationCodePurpose string

const (
	VerificationCodePurposeEmailVerify  VerificationCodePurpose = "email_verify"
	VerificationCodePurposePasswordReset VerificationCodePurpose = "password_reset"
	VerificationCodePurposeEmailChange  VerificationCodePurpose = "email_change"
)

type VerificationCode struct {
	UserID    uuid.UUID               `json:"user_id"`
	Email     string                  `json:"email"`
	Code      string                  `json:"code"`
	Purpose   VerificationCodePurpose `json:"purpose"`
	ExpiresAt time.Time               `json:"expires_at"`
}
