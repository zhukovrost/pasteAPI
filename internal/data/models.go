package data

import (
	"database/sql"
	"errors"
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
}

func NewModels(db *sql.DB) Models {
	return Models{
		Pastes: &PasteModel{DB: db},
	}
}
