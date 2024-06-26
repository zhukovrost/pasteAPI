package router

import (
	"expvar"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"

	"net/http"
	"pasteAPI/internal/http/v1"
)

func NewRouter(handler *v1.Handler) http.Handler {
	r := chi.NewRouter()

	r.NotFound(handler.NotFoundResponse)

	r.Get("/api/debug/vars", expvar.Handler().ServeHTTP)
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/healthcheck", handler.HealthcheckHandler)

		r.Route("/pastes", func(r chi.Router) {
			r.Get("/", handler.ListPastesHandler)
			r.Post("/", handler.CreatePasteHandler)

			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", handler.GetPasteHandler)
				r.Delete("/", handler.RequireAllowedToWriteUser(handler.DeletePasteHandler))
				r.Patch("/", handler.RequireAllowedToWriteUser(handler.UpdatePasteHandler))
			})
		})

		r.Route("/users", func(r chi.Router) {
			r.Post("/", handler.RegisterUserHandler)
			r.Put("/activated", handler.ActivateUserHandler)
		})

		r.Post("/tokens/authentication", handler.CreateAuthenticationTokenHandler)
	})

	return handler.Metrics(handler.RecoverPanic(handler.EnableCORS(handler.RateLimit(handler.Authenticate(handler.DebugRequest(r))))))
}
