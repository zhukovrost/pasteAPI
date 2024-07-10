package v1

import (
	"errors"
	"github.com/zhukovrost/pasteAPI/internal/repository"
	"github.com/zhukovrost/pasteAPI/internal/service"
	"github.com/zhukovrost/pasteAPI/pkg/helpers"
	"github.com/zhukovrost/pasteAPI/pkg/validator"
	"net/http"
)

// CreateAuthenticationTokenHandler creates a new authentication token by input data
//
// @Summary      Authentication
// @Description  Creates a new user token in the database by input data.
// @Tags         users
// @Tags         tokens
// @Accept       json
// @Produce      json
// @Param        body  body     service.AuthInput  true  "User registration input"
// @Success      201  {object}  service.AuthResp  "Successfully created"
// @Failure      400  {object}  ErrorResponse "Bad request"
// @Failure      401  {object}  ErrorResponse "Unauthorized"
// @Failure      422  {object}  ErrorResponse "Unprocessable data"
// @Failure 429 {object} ErrorResponse "Too many requests, rate limit exceeded"
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/tokens/authentication [post]
func (h *Handler) CreateAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var in service.AuthInput

	err := helpers.ReadJSON(w, r, &in)
	if err != nil {
		h.BadRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	service.ValidateEmail(v, in.Email)
	service.ValidatePasswordPlaintext(v, in.Password)
	if !v.Valid() {
		h.FailedValidationResponse(w, r, v.Errors)
		return
	}

	token, err := h.services.Users.Login(in.Email, in.Password)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound) || errors.Is(err, repository.ErrUnauthorized):
			h.InvalidCredentialsResponse(w, r)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = helpers.WriteJSON(w, http.StatusCreated, helpers.Envelope{"authentication_token": token}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

// PasswordResetTokenHandler creates a new password reset token by input email
//
// @Summary      Password reset
// @Description  Creates a new password reset token in the database by input email. Sends a reset password email.
// @Tags         users
// @Tags         tokens
// @Accept       json
// @Produce      json
// @Param        body  body     service.ResetPasswordInput  true  "Input email"
// @Success      202  {object}  service.MessageResp  "Successfully accepted"
// @Failure      400  {object}  ErrorResponse "Bad request"
// @Failure      422  {object}  ErrorResponse "Unprocessable data"
// @Failure 429 {object} ErrorResponse "Too many requests, rate limit exceeded"
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/tokens/password-reset [post]
func (h *Handler) PasswordResetTokenHandler(w http.ResponseWriter, r *http.Request) {
	var in service.ResetPasswordInput

	err := helpers.ReadJSON(w, r, &in)
	if err != nil {
		h.BadRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	service.ValidateEmail(v, in.Email)
	if !v.Valid() {
		h.FailedValidationResponse(w, r, v.Errors)
		return
	}

	err = h.services.Users.ResetPasswordRequest(in.Email)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			v.AddError("email", "no matching email address found")
			h.FailedValidationResponse(w, r, v.Errors)
		case errors.Is(err, repository.ErrUnactivated):
			v.AddError("email", "user account must be activated")
			h.FailedValidationResponse(w, r, v.Errors)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = helpers.WriteJSON(w, http.StatusAccepted, helpers.Envelope{"message": "email with password reset was successfully sent"}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}
