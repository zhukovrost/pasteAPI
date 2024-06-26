package models

import (
	"pasteAPI/pkg/validator"
	"time"
)

type Paste struct {
	Id        uint16    `json:"id"`
	Title     string    `json:"title"`
	Category  uint8     `json:"category,omitempty"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Minutes   int32     `json:"-"`
	Version   uint32    `json:"version"`
}

func ValidatePaste(v *validator.Validator, p *Paste) {
	v.Check(p.Title != "", "title", "must be provided")
	v.Check(len(p.Title) <= 255, "title", "must not be more than 500 bytes long")

	v.Check(CategoriesList.IsValidCategory(p.Category), "category", "no such category")

	v.Check(p.Text != "", "text", "must be provided")
	v.Check(len(p.Title) <= 500, "title", "must not be more than 500 bytes long")
}
