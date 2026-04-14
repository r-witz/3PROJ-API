package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const brevoAPIURL = "https://api.brevo.com/v3/smtp/email"

type BrevoSender struct {
	apiKey  string
	from    emailAddress
	client  *http.Client
}

type emailAddress struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type brevoRequest struct {
	Sender      emailAddress   `json:"sender"`
	To          []emailAddress `json:"to"`
	Subject     string         `json:"subject"`
	HTMLContent string         `json:"htmlContent"`
}

func NewBrevoSender(apiKey, fromEmail, fromName string) *BrevoSender {
	return &BrevoSender{
		apiKey: apiKey,
		from: emailAddress{
			Email: fromEmail,
			Name:  fromName,
		},
		client: &http.Client{},
	}
}

func (s *BrevoSender) SendVerificationCode(ctx context.Context, to string, code string) error {
	subject := "Duskforge - Verify your email"
	html := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 480px; margin: 0 auto; padding: 32px;">
			<h2 style="color: #333;">Verify your email</h2>
			<p style="color: #555;">Your verification code is:</p>
			<div style="background: #f4f4f4; padding: 16px; text-align: center; border-radius: 8px; margin: 24px 0;">
				<span style="font-size: 32px; font-weight: bold; letter-spacing: 8px; color: #111;">%s</span>
			</div>
			<p style="color: #888; font-size: 14px;">This code expires in 15 minutes. If you didn't create an account, you can ignore this email.</p>
		</div>
	`, code)
	return s.send(ctx, to, subject, html)
}

func (s *BrevoSender) SendPasswordResetCode(ctx context.Context, to string, code string) error {
	subject := "Duskforge - Reset your password"
	html := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 480px; margin: 0 auto; padding: 32px;">
			<h2 style="color: #333;">Reset your password</h2>
			<p style="color: #555;">Your password reset code is:</p>
			<div style="background: #f4f4f4; padding: 16px; text-align: center; border-radius: 8px; margin: 24px 0;">
				<span style="font-size: 32px; font-weight: bold; letter-spacing: 8px; color: #111;">%s</span>
			</div>
			<p style="color: #888; font-size: 14px;">This code expires in 15 minutes. If you didn't request a password reset, you can ignore this email.</p>
		</div>
	`, code)
	return s.send(ctx, to, subject, html)
}

func (s *BrevoSender) send(ctx context.Context, to, subject, htmlContent string) error {
	reqBody := brevoRequest{
		Sender:      s.from,
		To:          []emailAddress{{Email: to}},
		Subject:     subject,
		HTMLContent: htmlContent,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal email request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, brevoAPIURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create email request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("brevo API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}
