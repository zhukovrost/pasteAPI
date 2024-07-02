package v1

import (
	"github.com/zhukovrost/pasteAPI/internal/repository"
	"github.com/zhukovrost/pasteAPI/internal/service"
)

type Handler struct {
	service *service.Service
	models  *repository.Models
}

func NewHandler(service *service.Service, models *repository.Models) *Handler {
	return &Handler{
		service: service,
		models:  models,
	}
}
