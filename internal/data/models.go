package data

import (
	"database/sql"
	"errors"
	"time"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type Models struct {
	Pastes interface {
		Create(p *Paste) error
		Read(id uint16) (*Paste, error)
		ReadAll(title string, category uint8, filters Filters) ([]*Paste, *Metadata, error)
		Update(p *Paste) error
		Delete(id uint16) error
	}
	Users interface {
		Create(u *User) error
		GetByEmail(email string) (*User, error)
		Update(u *User) error
		GetForToken(tokenScope, tokenPlaintext string) (*User, error)
	}
	Tokens interface {
		New(userID int64, ttl time.Duration, scope string) (*Token, error)
		DeleteAllForUser(scope string, userID int64) error
	}
	Permissions interface {
		SetWritePermission(userId int64, pasteId uint16) error
		GetWritePermission(userId int64, pasteId uint16) (bool, error)
	}
}

func NewModels(db *sql.DB) Models {
	return Models{
		Pastes:      &PasteModel{DB: db},
		Users:       &UserModel{DB: db},
		Tokens:      &TokenModel{DB: db},
		Permissions: &PermissionModel{DB: db},
	}
}
