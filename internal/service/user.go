package service

import (
	"github.com/sirupsen/logrus"
	"github.com/zhukovrost/pasteAPI/internal/repository"
	"github.com/zhukovrost/pasteAPI/internal/repository/models"
	"github.com/zhukovrost/pasteAPI/pkg/rabbitmq"
	"github.com/zhukovrost/pasteAPI/pkg/validator"
	"time"
)

type UserService struct {
	repo      repository.Users
	tokenRepo repository.Tokens
	log       *logrus.Logger
	mailer    emails
	*UserServiceConfig
}

type UserServiceConfig struct {
	Environment string
}

func newUserService(repo repository.Users, tokenRepo repository.Tokens, mailer *emailService, log *logrus.Logger) *UserService {
	return &UserService{
		repo:      repo,
		tokenRepo: tokenRepo,
		mailer:    mailer,
		log:       log,
	}
}

func (s *UserService) Register(user *models.User) error {
	err := s.repo.Create(user)
	if err != nil {
		return err
	}

	token, err := s.tokenRepo.New(user.ID, 8*time.Hour, repository.ScopeActivation)
	if err != nil {
		return err
	}

	if s.Environment == "development" {
		s.log.Infof("New activation tocken for user %s (id: %d): %s. "+
			"Go to (PUT) http://localhost:8080/api/v1/users/activated with token in th request body to activate user.",
			user.Login, user.ID, token.Plaintext,
		)
	}

	email := &rabbitmq.Email{
		To: rabbitmq.Receiver{
			Email: user.Email,
			Login: user.Login,
			ID:    user.ID,
		},
		Type:    rabbitmq.Activation,
		Message: token.Plaintext,
	}

	return s.mailer.sendActivationEmail(email)
}

func (s *UserService) Activate(token string) (*models.User, error) {
	user, err := s.repo.GetForToken(repository.ScopeActivation, token)
	if err != nil {
		return nil, err
	}
	user.Activated = true
	err = s.repo.Update(user)
	if err != nil {
		return nil, err
	}
	err = s.tokenRepo.DeleteAllForUser(repository.ScopeActivation, user.ID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) UpdatePassword(token, newPasswordPlain string) error {
	user, err := s.repo.GetForToken(repository.ScopePasswordReset, token)
	if err != nil {
		return err
	}

	newPassword := models.Password{}
	if err = newPassword.Set(newPasswordPlain); err != nil {
		return err
	}

	user.Password = newPassword

	err = s.repo.Update(user)
	if err != nil {
		return err
	}

	return s.tokenRepo.DeleteAllForUser(repository.ScopePasswordReset, user.ID)
}

func (s *UserService) Login(email, password string) (*models.Token, error) {
	user, err := s.repo.GetByEmail(email)
	if err != nil {
		return nil, err
	}

	match, err := user.Password.Matches(password)
	if err != nil {
		return nil, err
	}

	if !match {
		return nil, repository.ErrUnauthorized
	}

	return s.tokenRepo.New(user.ID, 24*time.Hour, repository.ScopeAuthentication)
}

func (s *UserService) ResetPasswordRequest(emailAddr string) error {
	user, err := s.repo.GetByEmail(emailAddr)
	if err != nil {
		return err
	}

	if !user.Activated {
		return repository.ErrUnactivated
	}

	token, err := s.tokenRepo.New(user.ID, 45*time.Minute, repository.ScopePasswordReset)
	if err != nil {
		return err
	}

	if s.Environment == "development" {
		s.log.Infof("New reset password tocken for user %s (id: %d): %s. "+
			"Go to (PUT) http://localhost:8080/api/v1/users/password with token and new password in th request body to reset user's password.",
			user.Login, user.ID, token.Plaintext,
		)
	}

	email := &rabbitmq.Email{
		To: rabbitmq.Receiver{
			Email: user.Email,
			Login: user.Login,
			ID:    user.ID,
		},
		Type:    rabbitmq.PasswordReset,
		Message: token.Plaintext,
	}

	return s.mailer.sendResetEmail(email)
}

func ValidateEmail(v *validator.MyValidator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}

func ValidatePasswordPlaintext(v *validator.MyValidator, password string) {
	v.Check(password != "", "Password", "must be provided")
	v.Check(len(password) >= 8, "Password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "Password", "must not be more than 72 bytes long")
}

func ValidateLogin(v *validator.MyValidator, login string) {
	v.Check(login != "", "login", "must be provided")
	v.Check(validator.Matches(login, validator.LoginRX), "login", "contains incorrect symbols")
}

func ValidateUser(v *validator.MyValidator, user *models.User) {
	ValidateLogin(v, user.Login)
	ValidateEmail(v, user.Email)

	if user.Password.Plaintext != nil {
		ValidatePasswordPlaintext(v, *user.Password.Plaintext)
	}

	if user.Password.Hash == nil {
		panic("missing Password hash for user")
	}
}
