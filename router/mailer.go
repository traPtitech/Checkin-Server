package router

import (
	"context"

	"go.uber.org/zap"
)

type Mailer interface {
	SendVerificationEmail(ctx context.Context, email string, verificationURL string) error
}

type MockMailer struct {
	Logger *zap.Logger
}

func (m MockMailer) SendVerificationEmail(_ context.Context, email string, verificationURL string) error {
	m.Logger.Info("mock verification email sent",
		zap.String("to", email),
		zap.String("verification_url", verificationURL),
	)
	return nil
}
