package service

import (
	"github.com/sirupsen/logrus"
	"github.com/zhukovrost/pasteAPI/internal/repository"
	"github.com/zhukovrost/pasteAPI/internal/repository/models"
	"github.com/zhukovrost/pasteAPI/pkg/cache"
	"github.com/zhukovrost/pasteAPI/pkg/postgres"
	"github.com/zhukovrost/pasteAPI/pkg/rabbitmq"
	"sync"
)

type Services struct {
	Config
	Deps   Dependencies
	Pastes Pastes
	Users  Users
}

type Pastes interface {
	GetList(settings SearchSettings, user *models.User) (*ListPastesOutput, error)
	GetPaste(id uint16, user *models.User) (*models.Paste, error)
	Delete(id uint16) error
	Create(paste *models.Paste, creator *models.User) error
	GetPasteForUpdate(pasteId uint16, in UpdatePasteInput) (*models.Paste, error)
	Update(paste *models.Paste) error
	GivePermission(pasteId uint16, userLogin string) (*PastePermissionResponse, error)
}

type Users interface {
	Register(user *models.User) error
	Login(email, password string) (*models.Token, error)
	Activate(token string) (*models.User, error)
	ActivationRequest(user *models.User) error
	UpdatePassword(token, newPassword string) error
	ResetPasswordRequest(email string) error
}

type emails interface {
	sendActivationEmail(email *rabbitmq.Email) error
	sendResetEmail(email *rabbitmq.Email) error
}

// Config represents 'super-config' for all services
type Config struct {
	Host           string
	Port           int
	Env            string
	Status         string
	ActivationLink string
	ResetLink      string
	Limiter        struct {
		RPS     float64
		Burst   int
		Enabled bool
	}
	CORS struct {
		TrustedOrigins []string
	}
	BuildTime string
	Version   string
}

type Dependencies struct {
	Logger *logrus.Logger
	DB     postgres.Database
	Cache  cache.Cache
	Mailer *rabbitmq.Connection
	Models *repository.Models
	Wg     *sync.WaitGroup
}

func New(cfg Config, deps Dependencies) *Services {
	return &Services{
		Config: cfg,
		Deps:   deps,
		Pastes: newPasteService(
			deps.Models.Pastes,
			deps.Models.Permissions,
			deps.Logger,
			deps.Cache,
			deps.Wg,
		),
		Users: newUserService(
			&UserServiceConfig{
				Environment: cfg.Env,
			},
			deps.Models.Users,
			deps.Models.Tokens,
			newEmailService(
				deps.Mailer,
				&emailServiceConfig{
					ActivationLink: cfg.ActivationLink,
					ResetLink:      cfg.ResetLink,
				},
			),
			deps.Logger,
		),
	}
}

type SearchSettings struct {
	Title     string
	Category  uint8
	OnlyUsers bool
	Filters   models.Filters
}

type ListPastesOutput struct {
	Pastes   []*models.Paste  `json:"pastes"`
	Metadata *models.Metadata `json:"metadata"`
}

type PasteResp struct {
	R *models.Paste `json:"paste"`
}

type CreatePasteInput struct {
	Title    string `json:"title"`
	Category uint8  `json:"category,omitempty"`
	Text     string `json:"text"`
	Minutes  int32  `json:"minutes"`
}

type UpdatePasteInput struct {
	Title    *string `json:"title"`
	Category *uint8  `json:"category,omitempty"`
	Text     *string `json:"text"`
	Minutes  *int32  `json:"minutes"`
}

type PastePermissionResponse struct {
	Permission models.Permission `json:"permission"`
}

type RegistrationInput struct {
	Login    string `json:"login"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResp struct {
	U *models.User `json:"user"`
}

type ActivateUserInput struct {
	TokenPlainText string `json:"token"`
}

type UpdatePasswordInput struct {
	TokenPlainText string `json:"token"`
	NewPassword    string `json:"password"`
}

type MessageResp struct {
	Message string `json:"message"`
}

type AuthInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResp struct {
	*models.Token `json:"authentication_token"`
}

type ResetPasswordInput struct {
	Email string `json:"email"`
}
