package repository

import (
	"database/sql"
	"errors"
	"pasteAPI/internal/repository/models"
	"time"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type Models struct {
	Pastes Pastes
	Users  Users
	Tokens interface {
		New(userID int64, ttl time.Duration, scope string) (*models.Token, error)
		DeleteAllForUser(scope string, userID int64) error
	}
	Permissions interface {
		SetWritePermission(userId int64, pasteId uint16) error
		GetWritePermission(userId int64, pasteId uint16) (bool, error)
	}
}

func NewModels(db *sql.DB) *Models {
	return &Models{
		Pastes:      &PasteModel{DB: db},
		Users:       &UserModel{DB: db},
		Tokens:      &TokenModel{DB: db},
		Permissions: &PermissionModel{DB: db},
	}
}
