package repository

import (
	"errors"
	"github.com/zhukovrost/pasteAPI/internal/repository/models"
	"github.com/zhukovrost/pasteAPI/pkg/postgres"
	"time"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
	ErrUnauthorized   = errors.New("invalid authentication credentials")
	ErrUnactivated    = errors.New("user email must be activated")
)

type Users interface {
	Create(u *models.User) error
	GetByEmail(email string) (*models.User, error)
	Update(u *models.User) error
	GetForToken(tokenScope, tokenPlaintext string) (*models.User, error)
}

type Pastes interface {
	Create(p *models.Paste) error
	Read(id uint16, user *models.User) (*models.Paste, error)
	ReadAll(title string, category uint8, user *models.User, filters models.Filters) ([]*models.Paste, *models.Metadata, error)
	ReadUserPastes(title string, category uint8, user *models.User, filters models.Filters) ([]*models.Paste, *models.Metadata, error)
	Update(p *models.Paste) error
	Delete(id uint16) error
}

type Tokens interface {
	New(userID int64, ttl time.Duration, scope string) (*models.Token, error)
	DeleteAllForUser(scope string, userID int64) error
}

type Permissions interface {
	SetWritePermission(userId int64, pasteId uint16) error
	CheckWritePermission(userId int64, pasteId uint16) (bool, error)
}

type Models struct {
	Pastes      Pastes
	Users       Users
	Tokens      Tokens
	Permissions Permissions
}

func NewModels(db postgres.Database) *Models {
	return &Models{
		Pastes:      &PasteModel{DB: db},
		Users:       &UserModel{DB: db},
		Tokens:      &TokenModel{DB: db},
		Permissions: &PermissionModel{DB: db},
	}
}
