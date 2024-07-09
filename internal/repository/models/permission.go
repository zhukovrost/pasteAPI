package models

type Permission struct {
	PasteId uint16 `json:"paste_id"`
	UserId  int64  `json:"user_id"`
}
