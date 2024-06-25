package models

import "time"

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
