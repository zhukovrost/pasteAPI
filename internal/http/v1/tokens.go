package v1

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/zhukovrost/pasteAPI/internal/repository"
	"github.com/zhukovrost/pasteAPI/internal/repository/models"
	"github.com/zhukovrost/pasteAPI/pkg/helpers"
	"github.com/zhukovrost/pasteAPI/pkg/rabbitmq"
	"github.com/zhukovrost/pasteAPI/pkg/validator"
	"net/http"
	"time"
)

type AuthInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResp struct {
	*models.Token `json:"authentication_token"`
}

// CreateAuthenticationTokenHandler creates a new authentication token by input data
//
// @Summary      Authentication
// @Description  Creates a new user token in the database by input data.
// @Tags         users
// @Tags         tokens
// @Accept       json
// @Produce      json
// @Param        body  body     AuthInput  true  "User registration input"
// @Success      201  {object}  AuthResp  "Successfully created"
// @Failure      400  {object}  ErrorResponse "Bad request"
// @Failure      401  {object}  ErrorResponse "Unauthorized"
// @Failure      422  {object}  ErrorResponse "Unprocessable data"
// @Failure 429 {object} ErrorResponse "Too many requests, rate limit exceeded"
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/tokens/authentication [post]
func (h *Handler) CreateAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var in AuthInput

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

	user, err := h.service.Models.Users.GetByEmail(in.Email)
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

	token, err := h.service.Models.Tokens.New(user.ID, 24*time.Hour, repository.ScopeAuthentication)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
		return
	}

	err = helpers.WriteJSON(w, http.StatusCreated, helpers.Envelope{"authentication_token": token}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

type ResetPasswordInput struct {
	Email string `json:"email"`
}

type ResetPasswordResp struct {
	Message string `json:"message"`
}

// PasswordResetTokenHandler creates a new password reset token by input email
//
// @Summary      Password reset
// @Description  Creates a new password reset token in the database by input email. Sends a reset password email.
// @Tags         users
// @Tags         tokens
// @Accept       json
// @Produce      json
// @Param        body  body     ResetPasswordInput  true  "Input email"
// @Success      202  {object}  ResetPasswordResp  "Successfully accepted"
// @Failure      400  {object}  ErrorResponse "Bad request"
// @Failure      422  {object}  ErrorResponse "Unprocessable data"
// @Failure 429 {object} ErrorResponse "Too many requests, rate limit exceeded"
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/tokens/password-reset [post]
func (h *Handler) PasswordResetTokenHandler(w http.ResponseWriter, r *http.Request) {
	var in ResetPasswordInput

	err := helpers.ReadJSON(w, r, &in)
	if err != nil {
		h.BadRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	models.ValidateEmail(v, in.Email)
	if !v.Valid() {
		h.FailedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := h.service.Models.Users.GetByEmail(in.Email)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			v.AddError("email", "no matching email address found")
			h.FailedValidationResponse(w, r, v.Errors)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	if !user.Activated {
		v.AddError("email", "user account must be activated")
		h.FailedValidationResponse(w, r, v.Errors)
		return
	}

	token, err := h.service.Models.Tokens.New(user.ID, 45*time.Minute, repository.ScopePasswordReset)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
		return
	}

	if h.service.Config.Env == "development" {
		h.service.Deps.Logger.Infof("New reset password tocken for user %s (id: %d): %s. "+
			"Go to (PUT) http://localhost:8080/api/v1/users/password with token and new password in th request body to reset user's password.",
			user.Login, user.ID, token.Plaintext,
		)
	}

	email := rabbitmq.Email{
		To: rabbitmq.Receiver{
			Email: user.Email,
			Login: user.Login,
			ID:    user.ID,
		},
		Type:    rabbitmq.PasswordReset,
		Message: h.service.Config.ResetLink + token.Plaintext,
	}

	emailJSON, err := json.Marshal(email)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.service.Deps.Mailer.Timeout)
	defer cancel()

	err = h.service.Deps.Mailer.PublishMessage(ctx, "application/json", emailJSON)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
		return
	}

	err = helpers.WriteJSON(w, http.StatusAccepted, helpers.Envelope{"message": "email with password reset was successfully sent"}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}
