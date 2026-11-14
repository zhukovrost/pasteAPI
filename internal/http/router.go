package router

import (
	"expvar"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/zhukovrost/pasteAPI/internal/http/v1"
	"net/http"
)

func NewRouter(handler *v1.Handler) http.Handler {
	r := chi.NewRouter()

	r.NotFound(handler.NotFoundResponse)

	r.Get("/api/debug/vars", expvar.Handler().ServeHTTP)
	r.Get("/api/swagger/*", httpSwagger.WrapHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/healthcheck", handler.HealthcheckHandler)

		r.Route("/pastes", func(r chi.Router) {
			r.Get("/", handler.ListPastesHandler)
			r.Post("/", handler.CreatePasteHandler)
			r.Get("/search", handler.SearchPastesHandler)
			r.Get("/search/safe", handler.SafeSearchPastesHandler)
			r.Post("/batch", handler.BatchCreatePastesHandler)

			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", handler.GetPasteHandler)
				r.Delete("/", handler.RequireAllowedToWriteUser(handler.DeletePasteHandler))
				r.Patch("/", handler.RequireAllowedToWriteUser(handler.UpdatePasteHandler))

				r.Put("/permission/{userLogin}", handler.RequireAllowedToWriteUser(handler.PermissionHandler))

				r.Get("/raw", handler.RequireAuthenticatedUser(handler.RawPasteHandler))
				r.Get("/views", handler.PasteViewsHandler)
				r.Get("/history", handler.PasteHistoryHandler)
				r.Post("/moderate", handler.ModeratePasteHandler)
			})
		})

		r.Route("/categories", func(r chi.Router) {
			r.Post("/", handler.CreateCategoryHandler)
		})

		r.Get("/link-preview", handler.LinkPreviewHandler)
		r.Get("/link-preview/safe", handler.SafeLinkPreviewHandler)

		r.Route("/attachments", func(r chi.Router) {
			r.Get("/", handler.DownloadAttachmentHandler)
			r.Delete("/", handler.DeleteAttachmentHandler)
		})

		r.Get("/public-config", handler.PublicConfigHandler)

		r.Route("/users", func(r chi.Router) {
			r.Post("/", handler.RegisterUserHandler)
			r.Put("/activated", handler.ActivateUserHandler)
			r.Put("/password", handler.UpdatePasswordHandler)
			r.Post("/activation-email", handler.RequireAuthenticatedUser(handler.ResendActivationEmail))
			r.Get("/", handler.RequireAuthenticatedUser(handler.GetLoggedUserData))
			r.Post("/legacy-password-check", handler.LegacyPasswordCheckHandler)
		})

		r.Route("/tokens", func(r chi.Router) {
			r.Post("/authentication", handler.CreateAuthenticationTokenHandler)
			r.Post("/password-reset", handler.PasswordResetTokenHandler)
		})
	})

	return handler.Metrics(handler.RecoverPanic(handler.EnableCORS(handler.RateLimit(handler.Authenticate(handler.DebugRequest(r))))))
}
