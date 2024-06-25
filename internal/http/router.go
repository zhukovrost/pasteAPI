package router

import (
	"expvar"
	"github.com/go-chi/chi/v5"
	"net/http"
	"pasteAPI/internal/http/v1"
)

func NewRouter(handler *v1.Handler) http.Handler {
	r := chi.NewRouter()

	r.NotFound(handler.NotFoundResponse)

	//debug
	r.Get("/api/debug/vars", expvar.Handler().ServeHTTP)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/healthcheck", handler.HealthcheckHandler)

		// /api/v1/pastes/
		r.Route("/pastes", func(r chi.Router) {
			r.Get("/", handler.ListPastesHandler)
			r.Post("/", handler.RequireActivatedUser(handler.CreatePasteHandler))

			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", handler.GetPasteHandler)
				r.Delete("/", handler.RequireAllowedToWriteUser(handler.DeletePasteHandler))
				r.Patch("/", handler.RequireAllowedToWriteUser(handler.UpdatePasteHandler))
			})
		})
		// /api/v1/users/
		r.Route("/users", func(r chi.Router) {
			r.Post("/", handler.RegisterUserHandler)
			r.Put("/activated", handler.ActivateUserHandler)
		})

		// /api/v1/tokens/authentication/
		r.Post("/tokens/authentication", handler.CreateAuthenticationTokenHandler)
	})

	return handler.Metrics(handler.RecoverPanic(handler.EnableCORS(handler.RateLimit(handler.Authenticate(handler.DebugRequest(r))))))
}
