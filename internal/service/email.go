package service

import (
	"context"
	"encoding/json"
	"github.com/zhukovrost/pasteAPI/pkg/rabbitmq"
)

type emailService struct {
	*rabbitmq.Connection
	*emailServiceConfig
}

type emailServiceConfig struct {
	ActivationLink string
	ResetLink      string
}

func newEmailService(conn *rabbitmq.Connection, config *emailServiceConfig) *emailService {
	return &emailService{Connection: conn, emailServiceConfig: config}
}

func (s *emailService) sendEmail(email *rabbitmq.Email) error {
	emailJSON, err := json.Marshal(email)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.Timeout)
	defer cancel()

	return s.PublishMessage(ctx, "application/json", emailJSON)
}

func (s *emailService) sendActivationEmail(email *rabbitmq.Email) error {
	email.Message = s.ActivationLink + email.Message
	return s.sendEmail(email)
}

func (s *emailService) sendResetEmail(email *rabbitmq.Email) error {
	email.Message = s.ResetLink + email.Message
	return s.sendEmail(email)
}
