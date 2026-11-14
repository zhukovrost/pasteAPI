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
	ErrDuplicateUser  = errors.New("email or login already exists")
	ErrUserNotFound   = errors.New("user not found")
)

type Users interface {
	Create(u *models.User) error
	GetByEmail(email string) (*models.User, error)
	Update(u *models.User) error
	GetForToken(tokenScope, tokenPlaintext string) (*models.User, error)
	ReadPasswordHash(login string) (string, error)
}

type Pastes interface {
	Create(p *models.Paste) error
	Read(id uint16) (*models.Paste, error)
	ReadAll(title string, category uint8, filters models.Filters) ([]*models.Paste, *models.Metadata, error)
	ReadUserPastes(title string, category uint8, user *models.User, filters models.Filters) ([]*models.Paste, *models.Metadata, error)
	Update(p *models.Paste) error
	Delete(id uint16) error

	SearchByTitle(title string) ([]*models.Paste, error)
	SearchByTitleSafe(title, sortColumn string) ([]*models.Paste, error)
	ReadText(id int64) (string, error)
	IncrementViews(id uint16) (int64, error)
	ReadHistory(id int64) ([]*models.PasteHistoryEntry, error)
	CreateBatch(pastes []*models.Paste) error
	CreateCategory(name string) error
}

type Attachments interface {
	Read(name string) ([]byte, error)
	Delete(name string) error
}

type Tokens interface {
	New(userID int64, ttl time.Duration, scope string) (*models.Token, error)
	DeleteAllForUser(scope string, userID int64) error
}

type Permissions interface {
	SetWritePermissionByLogin(userLogin string, pasteId uint16) (int64, error)
	SetWritePermissionById(userId int64, pasteId uint16) error
	CheckWritePermission(userId int64, pasteId uint16) (bool, error)
}

type Models struct {
	Pastes      Pastes
	Users       Users
	Tokens      Tokens
	Permissions Permissions
	Attachments Attachments
}

func NewModels(db postgres.Database) *Models {
	return &Models{
		Pastes:      &PasteModel{DB: db},
		Users:       &UserModel{DB: db},
		Tokens:      &TokenModel{DB: db},
		Permissions: &PermissionModel{DB: db},
		Attachments: &AttachmentModel{Dir: "uploads"},
	}
}
