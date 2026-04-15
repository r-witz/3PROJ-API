package services

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	"duskforge-api/pkg/auth"
	"duskforge-api/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	verificationCodeTTL    = 15 * time.Minute
	verificationRateWindow = 15 * time.Minute
)

type authService struct {
	userRepo          ports.UserRepository
	sessionRepo       ports.SessionRepository
	collectionService ports.CollectionService
	emailSender       ports.EmailSender
	verificationRepo  ports.VerificationCodeRepository
	config            TokenConfig
}

func NewAuthService(
	userRepo ports.UserRepository,
	sessionRepo ports.SessionRepository,
	collectionService ports.CollectionService,
	emailSender ports.EmailSender,
	verificationRepo ports.VerificationCodeRepository,
	config TokenConfig,
) ports.AuthService {
	return &authService{
		userRepo:          userRepo,
		sessionRepo:       sessionRepo,
		collectionService: collectionService,
		emailSender:       emailSender,
		verificationRepo:  verificationRepo,
		config:            config,
	}
}

func (s *authService) Register(ctx context.Context, input ports.RegisterInput) (*domain.User, *ports.AuthTokens, error) {
	existing, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, nil, domain.ErrInternal
	}
	if existing != nil {
		return nil, nil, domain.ErrEmailAlreadyExists
	}

	existing, err = s.userRepo.GetByUsername(ctx, input.Username)
	if err != nil {
		return nil, nil, domain.ErrInternal
	}
	if existing != nil {
		return nil, nil, domain.ErrUsernameAlreadyExists
	}

	passwordHash, err := auth.HashPassword(input.Password)
	if err != nil {
		return nil, nil, mapPasswordError(err)
	}

	locale := input.Locale
	if locale == "" {
		locale = domain.UserLocaleEN
	}

	now := time.Now()
	user := &domain.User{
		ID:           uuid.New(),
		Email:        input.Email,
		Username:     input.Username,
		PasswordHash: &passwordHash,
		Role:         domain.UserRoleUser,
		Theme:        domain.UserThemeSystem,
		Locale:       locale,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, nil, domain.ErrInternal
	}

	if s.collectionService != nil {
		if err := s.collectionService.CreateDefaultCollections(ctx, user.ID); err != nil {
			return nil, nil, domain.ErrInternal
		}
	}

	tokens, err := createSession(ctx, s.sessionRepo, user, s.config)
	if err != nil {
		return nil, nil, err
	}

	if s.emailSender != nil && s.verificationRepo != nil {
		if err := s.sendVerificationCodeForUser(ctx, user, false); err != nil {
			logger.Logger.Error("Failed to send verification email on registration", zap.Error(err), zap.String("email", user.Email))
			_ = s.sessionRepo.DeleteByUserID(ctx, user.ID)
			_ = s.userRepo.Delete(ctx, user.ID)
			return nil, nil, domain.ErrInternal
		}
	}

	return user, tokens, nil
}

func (s *authService) Login(ctx context.Context, input ports.LoginInput) (*domain.User, *ports.AuthTokens, error) {
	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, nil, domain.ErrInternal
	}
	if user == nil {
		return nil, nil, domain.ErrInvalidCredentials
	}

	if user.BannedAt != nil {
		return nil, nil, domain.ErrUserBanned
	}

	if user.PasswordHash == nil {
		return nil, nil, domain.ErrInvalidCredentials
	}

	match, err := auth.ComparePassword(*user.PasswordHash, input.Password)
	if err != nil || !match {
		return nil, nil, domain.ErrInvalidCredentials
	}

	if !user.EmailVerified {
		return nil, nil, domain.ErrEmailNotVerified
	}

	tokens, err := createSession(ctx, s.sessionRepo, user, s.config)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

func (s *authService) Refresh(ctx context.Context, refreshToken string) (*ports.AuthTokens, error) {
	claims, err := auth.ValidateRefreshToken(refreshToken, s.config.RefreshTokenSecret)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	tokenHash := auth.HashToken(refreshToken)
	session, err := s.sessionRepo.GetByRefreshTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if session == nil || session.ID != claims.SessionID {
		return nil, domain.ErrInvalidToken
	}

	if time.Now().After(session.ExpiresAt) {
		_ = s.sessionRepo.Delete(ctx, session.ID)
		return nil, domain.ErrSessionExpired
	}

	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil || user == nil {
		return nil, domain.ErrInternal
	}

	if user.BannedAt != nil {
		_ = s.sessionRepo.Delete(ctx, session.ID)
		return nil, domain.ErrUserBanned
	}

	accessToken, err := auth.GenerateAccessToken(
		user.ID, string(user.Role), s.config.AccessTokenSecret, s.config.AccessTokenExpiry,
	)
	if err != nil {
		return nil, domain.ErrInternal
	}

	newRefreshToken, err := auth.GenerateRefreshToken(
		session.ID, s.config.RefreshTokenSecret, s.config.RefreshTokenExpiry,
	)
	if err != nil {
		return nil, domain.ErrInternal
	}

	session.RefreshTokenHash = auth.HashToken(newRefreshToken)
	session.ExpiresAt = time.Now().Add(s.config.RefreshTokenExpiry)
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return nil, domain.ErrInternal
	}

	return &ports.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(s.config.AccessTokenExpiry.Seconds()),
	}, nil
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := auth.HashToken(refreshToken)
	session, err := s.sessionRepo.GetByRefreshTokenHash(ctx, tokenHash)
	if err != nil {
		return domain.ErrInternal
	}
	if session == nil {
		return nil
	}

	return s.sessionRepo.Delete(ctx, session.ID)
}

func (s *authService) SendVerificationCode(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return domain.ErrInternal
	}
	if user == nil {
		return nil // silent success to prevent email enumeration
	}

	if user.EmailVerified {
		return domain.ErrEmailAlreadyVerified
	}

	if err := s.sendVerificationCodeForUser(ctx, user, true); err != nil {
		if errors.Is(err, domain.ErrVerificationCodeRateLimit) {
			return err
		}
		logger.Logger.Error("Failed to send verification code", zap.Error(err), zap.String("email", user.Email))
		return domain.ErrInternal
	}
	return nil
}

func (s *authService) VerifyEmail(ctx context.Context, email string, code string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return domain.ErrInternal
	}
	if user == nil {
		return domain.ErrVerificationCodeInvalid
	}

	if user.EmailVerified {
		return domain.ErrEmailAlreadyVerified
	}

	stored, err := s.verificationRepo.Get(ctx, email, domain.VerificationCodePurposeEmailVerify)
	if err != nil {
		return domain.ErrInternal
	}
	if stored == nil || stored.Code != code {
		return domain.ErrVerificationCodeInvalid
	}

	if err := s.userRepo.SetEmailVerified(ctx, user.ID, true); err != nil {
		return domain.ErrInternal
	}

	_ = s.verificationRepo.Delete(ctx, email, domain.VerificationCodePurposeEmailVerify)
	return nil
}

func (s *authService) RequestPasswordReset(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return domain.ErrInternal
	}
	if user == nil {
		return nil // silent success to prevent email enumeration
	}

	canRequest, err := s.verificationRepo.CanRequest(ctx, email, domain.VerificationCodePurposePasswordReset)
	if err != nil {
		return domain.ErrInternal
	}
	if !canRequest {
		return domain.ErrVerificationCodeRateLimit
	}

	code, err := generateVerificationCode()
	if err != nil {
		return domain.ErrInternal
	}

	vc := &domain.VerificationCode{
		UserID:    user.ID,
		Email:     email,
		Code:      code,
		Purpose:   domain.VerificationCodePurposePasswordReset,
		ExpiresAt: time.Now().Add(verificationCodeTTL),
	}

	if err := s.verificationRepo.Store(ctx, vc, verificationCodeTTL); err != nil {
		return domain.ErrInternal
	}

	if err := s.verificationRepo.RecordRequest(ctx, email, domain.VerificationCodePurposePasswordReset, verificationRateWindow); err != nil {
		return domain.ErrInternal
	}

	if err := s.emailSender.SendPasswordResetCode(ctx, email, code); err != nil {
		logger.Logger.Error("Failed to send password reset email", zap.Error(err), zap.String("email", email))
		return domain.ErrInternal
	}

	return nil
}

func (s *authService) ResetPassword(ctx context.Context, input ports.ResetPasswordInput) error {
	stored, err := s.verificationRepo.Get(ctx, input.Email, domain.VerificationCodePurposePasswordReset)
	if err != nil {
		return domain.ErrInternal
	}
	if stored == nil || stored.Code != input.Code {
		return domain.ErrVerificationCodeInvalid
	}

	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil || user == nil {
		return domain.ErrInternal
	}

	passwordHash, err := auth.HashPassword(input.NewPassword)
	if err != nil {
		return mapPasswordError(err)
	}

	user.PasswordHash = &passwordHash
	user.UpdatedAt = time.Now()
	if err := s.userRepo.Update(ctx, user); err != nil {
		return domain.ErrInternal
	}

	_ = s.verificationRepo.Delete(ctx, input.Email, domain.VerificationCodePurposePasswordReset)
	_ = s.sessionRepo.DeleteByUserID(ctx, user.ID)

	return nil
}

func (s *authService) RequestEmailChange(ctx context.Context, userID uuid.UUID, newEmail string) error {
	// Check the new email isn't already taken
	existing, err := s.userRepo.GetByEmail(ctx, newEmail)
	if err != nil {
		return domain.ErrInternal
	}
	if existing != nil {
		return domain.ErrEmailAlreadyExists
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return domain.ErrInternal
	}

	canRequest, err := s.verificationRepo.CanRequest(ctx, newEmail, domain.VerificationCodePurposeEmailChange)
	if err != nil {
		return domain.ErrInternal
	}
	if !canRequest {
		return domain.ErrVerificationCodeRateLimit
	}

	code, err := generateVerificationCode()
	if err != nil {
		return domain.ErrInternal
	}

	vc := &domain.VerificationCode{
		UserID:    userID,
		Email:     newEmail,
		Code:      code,
		Purpose:   domain.VerificationCodePurposeEmailChange,
		ExpiresAt: time.Now().Add(verificationCodeTTL),
	}

	if err := s.verificationRepo.Store(ctx, vc, verificationCodeTTL); err != nil {
		return domain.ErrInternal
	}

	if err := s.verificationRepo.StorePendingEmail(ctx, userID, newEmail, verificationCodeTTL); err != nil {
		return domain.ErrInternal
	}

	if err := s.verificationRepo.RecordRequest(ctx, newEmail, domain.VerificationCodePurposeEmailChange, verificationRateWindow); err != nil {
		return domain.ErrInternal
	}

	if err := s.emailSender.SendVerificationCode(ctx, newEmail, code); err != nil {
		logger.Logger.Error("Failed to send email change verification", zap.Error(err), zap.String("email", newEmail))
		return domain.ErrInternal
	}

	return nil
}

func (s *authService) ConfirmEmailChange(ctx context.Context, userID uuid.UUID, code string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return domain.ErrInternal
	}

	// Find the pending email change code for this user
	// We need to scan all email_change keys for this user, but we stored the code keyed by the NEW email.
	// The code contains the userID, so we retrieve it by iterating... but that's not efficient.
	// Instead, we store a pointer from userID -> newEmail in Redis alongside the code.
	newEmail, err := s.verificationRepo.GetPendingEmail(ctx, userID)
	if err != nil {
		return domain.ErrInternal
	}
	if newEmail == "" {
		return domain.ErrVerificationCodeInvalid
	}

	stored, err := s.verificationRepo.Get(ctx, newEmail, domain.VerificationCodePurposeEmailChange)
	if err != nil {
		return domain.ErrInternal
	}
	if stored == nil || stored.Code != code || stored.UserID != userID {
		return domain.ErrVerificationCodeInvalid
	}

	// Check the new email is still available
	existing, err := s.userRepo.GetByEmail(ctx, newEmail)
	if err != nil {
		return domain.ErrInternal
	}
	if existing != nil {
		return domain.ErrEmailAlreadyExists
	}

	user.Email = newEmail
	user.EmailVerified = true
	user.UpdatedAt = time.Now()
	if err := s.userRepo.Update(ctx, user); err != nil {
		return domain.ErrInternal
	}

	_ = s.verificationRepo.Delete(ctx, newEmail, domain.VerificationCodePurposeEmailChange)
	_ = s.verificationRepo.DeletePendingEmail(ctx, userID)

	return nil
}

func (s *authService) sendVerificationCodeForUser(ctx context.Context, user *domain.User, checkRateLimit bool) error {
	if checkRateLimit {
		canRequest, err := s.verificationRepo.CanRequest(ctx, user.Email, domain.VerificationCodePurposeEmailVerify)
		if err != nil {
			return fmt.Errorf("check rate limit: %w", err)
		}
		if !canRequest {
			return domain.ErrVerificationCodeRateLimit
		}
	}

	code, err := generateVerificationCode()
	if err != nil {
		return fmt.Errorf("generate code: %w", err)
	}

	vc := &domain.VerificationCode{
		UserID:    user.ID,
		Email:     user.Email,
		Code:      code,
		Purpose:   domain.VerificationCodePurposeEmailVerify,
		ExpiresAt: time.Now().Add(verificationCodeTTL),
	}

	if err := s.verificationRepo.Store(ctx, vc, verificationCodeTTL); err != nil {
		return fmt.Errorf("store code: %w", err)
	}

	if err := s.verificationRepo.RecordRequest(ctx, user.Email, domain.VerificationCodePurposeEmailVerify, verificationRateWindow); err != nil {
		return fmt.Errorf("record request: %w", err)
	}

	if err := s.emailSender.SendVerificationCode(ctx, user.Email, code); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}

func generateVerificationCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func mapPasswordError(err error) error {
	if errors.Is(err, auth.ErrPasswordTooShort) {
		return domain.ErrPasswordTooShort
	}
	if errors.Is(err, auth.ErrPasswordTooLong) {
		return domain.ErrPasswordTooLong
	}
	if errors.Is(err, auth.ErrPasswordNoUppercase) {
		return domain.ErrPasswordNoUppercase
	}
	if errors.Is(err, auth.ErrPasswordNoLowercase) {
		return domain.ErrPasswordNoLowercase
	}
	if errors.Is(err, auth.ErrPasswordNoDigit) {
		return domain.ErrPasswordNoDigit
	}
	if errors.Is(err, auth.ErrPasswordNoSpecialChar) {
		return domain.ErrPasswordNoSpecialChar
	}
	return domain.ErrInternal
}
