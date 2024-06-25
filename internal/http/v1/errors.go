package v1

import (
	"fmt"
	"net/http"
	"pasteAPI/pkg/helpers"
)

func (h *Handler) LogError(r *http.Request, err error) {
	h.service.Logger.WithFields(map[string]interface{}{
		"request_method": r.Method,
		"request_url":    r.URL.Path,
	}).Error(err)
}

func (h *Handler) ErrorResponse(w http.ResponseWriter, r *http.Request, status int, message interface{}) {
	env := helpers.Envelope{"error": message}
	err := helpers.WriteJSON(w, status, env, nil)
	if err != nil {
		h.LogError(r, err)
		w.WriteHeader(500)
	}
}

func (h *Handler) ServerErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	h.LogError(r, err)
	message := "the server encountered a problem and could not process your request"
	h.ErrorResponse(w, r, http.StatusInternalServerError, message)
}

func (h *Handler) NotFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource could not be found"
	h.ErrorResponse(w, r, http.StatusNotFound, message)
}

func (h *Handler) MethodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not supported for this resource", r.Method)
	h.ErrorResponse(w, r, http.StatusMethodNotAllowed, message)
}

func (h *Handler) BadRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	h.ErrorResponse(w, r, http.StatusBadRequest, err.Error())
}

func (h *Handler) FailedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	h.ErrorResponse(w, r, http.StatusUnprocessableEntity, errors)
}

func (h *Handler) EditConflictResponse(w http.ResponseWriter, r *http.Request) {
	message := "unable to update the record due to an edit conflict, please try again"
	h.ErrorResponse(w, r, http.StatusConflict, message)
}

func (h *Handler) RateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	message := "rate limit exceeded"
	h.ErrorResponse(w, r, http.StatusTooManyRequests, message)
}
func (h *Handler) InvalidCredentialsResponse(w http.ResponseWriter, r *http.Request) {
	message := "invalid authentication credentials"
	h.ErrorResponse(w, r, http.StatusUnauthorized, message)
}
func (h *Handler) InvalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	message := "invalid or missing authentication token"
	h.ErrorResponse(w, r, http.StatusUnauthorized, message)
}

func (h *Handler) AuthenticationRequiredResponse(w http.ResponseWriter, r *http.Request) {
	message := "you must be authenticated to access this resource"
	h.ErrorResponse(w, r, http.StatusUnauthorized, message)
}
func (h *Handler) InactiveAccountResponse(w http.ResponseWriter, r *http.Request) {
	message := "your user account must be activated to access this resource"
	h.ErrorResponse(w, r, http.StatusForbidden, message)
}

func (h *Handler) ForbiddenResponse(w http.ResponseWriter, r *http.Request) {
	message := "you are not allowed to access this resource"
	h.ErrorResponse(w, r, http.StatusForbidden, message)
}
