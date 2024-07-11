package service

import (
	"context"
	"encoding/json"
	"github.com/zhukovrost/pasteAPI/internal/repository/models"
	"github.com/zhukovrost/pasteAPI/pkg/rabbitmq"
	"time"
)

type emailService struct {
	mailer rabbitmq.Mailer
	*emailServiceConfig
}

type emailServiceConfig struct {
	ActivationLink string
	ResetLink      string
}

func newEmailService(mailer rabbitmq.Mailer, config *emailServiceConfig) *emailService {
	return &emailService{mailer: mailer, emailServiceConfig: config}
}

func (s *emailService) sendEmail(email *models.Email) error {
	emailJSON, err := json.Marshal(email)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.mailer.PublishMessage(ctx, "application/json", emailJSON)
}

func (s *emailService) sendActivationEmail(email *models.Email) error {
	email.Message = s.ActivationLink + email.Message
	return s.sendEmail(email)
}

func (s *emailService) sendResetEmail(email *models.Email) error {
	email.Message = s.ResetLink + email.Message
	return s.sendEmail(email)
}
