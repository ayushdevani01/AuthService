package service

import (
	"context"
	"fmt"
	"log"

	"github.com/resend/resend-go/v2"
)

type EmailService struct {
	client      *resend.Client
	platformURL string
	fromAddress string
}

func NewEmailService(apiKey, platformURL, fromAddress string) *EmailService {
	var client *resend.Client
	if apiKey != "" {
		client = resend.NewClient(apiKey)
	}
	if fromAddress == "" {
		fromAddress = "noreply@yourplatform.com"
	}
	return &EmailService{
		client:      client,
		platformURL: platformURL,
		fromAddress: fromAddress,
	}
}

func (s *EmailService) SendPasswordReset(ctx context.Context, appID, toEmail, userName, rawToken string) error {
	resetLink := fmt.Sprintf("%s/?mode=reset&app_id=%s&token=%s", s.platformURL, appID, rawToken)

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<body style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <h2>Reset your password</h2>
  <p>Hi %s,</p>
  <p>We received a request to reset your password. Click the button below to set a new one.</p>
  <a href="%s" style="display:inline-block;padding:12px 24px;background:#4F46E5;color:#fff;border-radius:6px;text-decoration:none;font-weight:bold;">
    Reset Password
  </a>
  <p style="color:#888;font-size:12px;margin-top:24px;">This link expires in 15 minutes. If you didn't request a reset, ignore this email.</p>
</body>
</html>`, userName, resetLink)

	if s.client == nil || s.fromAddress == "" {
		log.Printf("[DEV] Password reset link for %s: %s", toEmail, resetLink)
		return nil
	}

	_, err := s.client.Emails.Send(&resend.SendEmailRequest{
		From:    s.fromAddress,
		To:      []string{toEmail},
		Subject: "Reset your password",
		Html:    html,
	})
	return err
}

func (s *EmailService) SendEmailVerification(ctx context.Context, appID, toEmail, userName, verifyToken string) error {
	verifyLink := fmt.Sprintf("%s/?mode=verify&app_id=%s&token=%s", s.platformURL, appID, verifyToken)

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<body style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <h2>Verify your email address</h2>
  <p>Hi %s,</p>
  <p>Thanks for signing up! Please verify your email address to get started.</p>
  <a href="%s" style="display:inline-block;padding:12px 24px;background:#4F46E5;color:#fff;border-radius:6px;text-decoration:none;font-weight:bold;">
    Verify Email
  </a>
  <p style="color:#888;font-size:12px;margin-top:24px;">This link expires in 24 hours.</p>
</body>
</html>`, userName, verifyLink)

	if s.client == nil || s.fromAddress == "" {
		log.Printf("[DEV] Email verification link for %s: %s", toEmail, verifyLink)
		return nil
	}

	_, err := s.client.Emails.Send(&resend.SendEmailRequest{
		From:    s.fromAddress,
		To:      []string{toEmail},
		Subject: "Verify your email address",
		Html:    html,
	})
	return err
}
