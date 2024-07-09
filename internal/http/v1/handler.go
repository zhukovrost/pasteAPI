package v1

import (
	"github.com/zhukovrost/pasteAPI/internal/service"
)

type Handler struct {
	services *service.Services
}

func NewHandler(service *service.Services) *Handler {
	return &Handler{
		services: service,
	}
}
