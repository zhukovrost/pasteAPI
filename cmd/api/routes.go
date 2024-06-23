package main

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (app *application) newRouter() http.Handler {
	r := chi.NewRouter()
	r.NotFound(app.notFoundResponse)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/healthcheck", app.healthcheckHandler)

		// /api/v1/pastes/
		r.Route("/pastes", func(r chi.Router) {
			r.Get("/", app.listPastesHandler)
			r.Post("/", app.requireActivatedUser(app.createPasteHandler))

			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", app.getPasteHandler)
				r.Delete("/", app.requireAllowedToWriteUser(app.deletePasteHandler))
				r.Patch("/", app.requireAllowedToWriteUser(app.updatePasteHandler))
			})
		})
		// /api/v1/users/
		r.Route("/users", func(r chi.Router) {
			r.Post("/", app.registerUserHandler)
			r.Put("/activated", app.activateUserHandler)
		})

		// /api/v1/tokens/authentication/
		r.Post("/tokens/authentication", app.createAuthenticationTokenHandler)
	})

	return app.recoverPanic(app.rateLimit(app.authenticate(app.debugRequest(r))))
}
