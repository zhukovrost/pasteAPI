package models

import "github.com/zhukovrost/pasteAPI/pkg/rabbitmq"

type Email struct {
	To      rabbitmq.Receiver  `json:"to"`
	Type    rabbitmq.EmailType `json:"type"`
	Message string             `json:"message"`
}
