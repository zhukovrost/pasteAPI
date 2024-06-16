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

		r.Route("/pastes", func(r chi.Router) {
			r.Get("/", app.listPastesHandler)
			r.Post("/", app.createPasteHandler)

			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", app.getPasteHandler)
				r.Delete("/", app.deletePasteHandler)
				r.Patch("/", app.updatePasteHandler)
			})
		})
	})

	return app.recoverPanic(app.debugRequest(r))
}
