package v1

import (
	"errors"
	"net/http"
	"pasteAPI/internal/repository"
	"pasteAPI/internal/repository/models"
	"pasteAPI/pkg/helpers"
	"pasteAPI/pkg/validator"
	"time"
)

func (h *Handler) CreateAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := helpers.ReadJSON(w, r, &in)
	if err != nil {
		h.BadRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	models.ValidateEmail(v, in.Email)
	models.ValidatePasswordPlaintext(v, in.Password)
	if !v.Valid() {
		h.FailedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := h.models.Users.GetByEmail(in.Email)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			h.InvalidCredentialsResponse(w, r)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	match, err := user.Password.Matches(in.Password)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
		return
	}

	if !match {
		h.InvalidCredentialsResponse(w, r)
		return
	}

	token, err := h.models.Tokens.New(user.ID, 24*time.Hour, repository.ScopeAuthentication)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
		return
	}

	err = helpers.WriteJSON(w, http.StatusCreated, helpers.Envelope{"authentication_token": token}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}
