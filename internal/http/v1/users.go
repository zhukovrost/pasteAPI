package v1

import (
	"errors"
	"net/http"
	"pasteAPI/internal/repository"
	"pasteAPI/internal/repository/models"
	"pasteAPI/internal/service"
	"pasteAPI/pkg/helpers"
	"pasteAPI/pkg/validator"
	"time"
)

func (h *Handler) RegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	// Create an anonymous struct to hold the expected repository from the request body.
	var input struct {
		Login    string `json:"login"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := helpers.ReadJSON(w, r, &input)
	if err != nil {
		h.BadRequestResponse(w, r, err)
		return
	}

	user := &models.User{
		Login:     input.Login,
		Email:     input.Email,
		Activated: false,
	}

	err = user.Password.Set(input.Password)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
		return
	}
	v := validator.New()

	if service.ValidateUser(v, user); !v.Valid() {
		h.FailedValidationResponse(w, r, v.Errors)
		return
	}

	err = h.models.Users.Create(user)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrDuplicate):
			v.AddError("user", "a user with this email/login already exists")
			h.FailedValidationResponse(w, r, v.Errors)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	token, err := h.models.Tokens.New(user.ID, 8*time.Hour, repository.ScopeActivation)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
		return
	}

	// TODO: rabbitmq
	h.service.Background(func() {
		tmplData := map[string]interface{}{
			"activationCode": token.Plaintext,
			"ID":             user.ID,
			"Login":          user.Login,
		}
		err = h.service.Mailer.SendEmail(user.Email, "welcome.tmpl", tmplData)
		if err != nil {
			h.service.Logger.Error(err)
		}
	})

	err = helpers.WriteJSON(w, http.StatusAccepted, helpers.Envelope{"user": user}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

func (h *Handler) ActivateUserHandler(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TokenPlainText string `json:"token"`
	}

	err := helpers.ReadJSON(w, r, &in)
	if err != nil {
		h.BadRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	if repository.ValidateTokenPlaintext(v, in.TokenPlainText); !v.Valid() {
		h.FailedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := h.models.Users.GetForToken(repository.ScopeActivation, in.TokenPlainText)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			v.AddError("token", "invalid or expired activation token")
			h.FailedValidationResponse(w, r, v.Errors)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	user.Activated = true

	err = h.models.Users.Update(user)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrEditConflict):
			h.EditConflictResponse(w, r)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = h.models.Tokens.DeleteAllForUser(repository.ScopeActivation, user.ID)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"user": user}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}
