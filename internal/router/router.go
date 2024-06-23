package router

import (
	"github.com/go-chi/chi/v5"
	"pasteAPI/internal/handlers"
)

func New() *chi.Mux {
	r := chi.NewRouter()

	r.Get("/healthcheck", handlers.HealthcheckHandler)

	r.Route("/pastes", func(r chi.Router) {
		r.Get("/", handlers.ListPastesHandler)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", handlers.GetPasteHandler)
			r.Delete("/", handlers.DeletePasteHandler)
		})
	})

	return r
}
