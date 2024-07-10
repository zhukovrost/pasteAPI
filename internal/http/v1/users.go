package v1

import (
	"errors"
	"github.com/zhukovrost/pasteAPI/internal/repository"
	"github.com/zhukovrost/pasteAPI/internal/repository/models"
	"github.com/zhukovrost/pasteAPI/internal/service"
	"github.com/zhukovrost/pasteAPI/pkg/helpers"
	"github.com/zhukovrost/pasteAPI/pkg/validator"
	"net/http"
)

// RegisterUserHandler creates a new user by input data
//
// @Summary      Registration
// @Description  Creates a new user in the database by input data.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body     service.RegistrationInput  true  "User registration input"
// @Success      202  {object}  service.UserResp  "Successfully accepted"
// @Failure      400  {object}  ErrorResponse "Bad request"
// @Failure      422  {object}  ErrorResponse "Unprocessable data"
// @Failure 429 {object} ErrorResponse "Too many requests, rate limit exceeded"
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/users/ [post]
func (h *Handler) RegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	var input service.RegistrationInput

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

	if err = h.services.Users.Register(user); err != nil {
		switch {
		case errors.Is(err, repository.ErrDuplicate):
			v.AddError("user", "a user with this email/login already exists")
			h.FailedValidationResponse(w, r, v.Errors)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = helpers.WriteJSON(w, http.StatusAccepted, helpers.Envelope{"user": user}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

// ActivateUserHandler activates the user by input token
//
// @Summary      Activation
// @Description  Activates the user by input token.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body     service.ActivateUserInput  true  "User activation input"
// @Success      202  {object}  service.UserResp  "Successfully accepted"
// @Failure      400  {object}  ErrorResponse "Bad request"
// @Failure      422  {object}  ErrorResponse "Unprocessable data"
// @Failure 429 {object} ErrorResponse "Too many requests, rate limit exceeded"
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/users/activated [put]
func (h *Handler) ActivateUserHandler(w http.ResponseWriter, r *http.Request) {
	var in service.ActivateUserInput

	err := helpers.ReadJSON(w, r, &in)
	if err != nil {
		h.BadRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	if service.ValidateTokenPlaintext(v, in.TokenPlainText); !v.Valid() {
		h.FailedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := h.services.Users.Activate(in.TokenPlainText)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			v.AddError("token", "invalid or expired activation token")
			h.FailedValidationResponse(w, r, v.Errors)
		case errors.Is(err, repository.ErrEditConflict):
			h.EditConflictResponse(w, r)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"user": user}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

// UpdatePasswordHandler updates user's password by input token
//
// @Summary      Update password
// @Description  Update user's password by input data.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body     service.UpdatePasswordInput  true  "User activation input"
// @Success      200  {object}  service.UpdatePasswordResponse  "Successfully reset"
// @Failure      400  {object}  ErrorResponse "Bad request"
// @Failure      422  {object}  ErrorResponse "Unprocessable data"
// @Failure      429 {object} ErrorResponse "Too many requests, rate limit exceeded"
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/users/password [put]
func (h *Handler) UpdatePasswordHandler(w http.ResponseWriter, r *http.Request) {
	var in service.UpdatePasswordInput

	err := helpers.ReadJSON(w, r, &in)
	if err != nil {
		h.BadRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	service.ValidateTokenPlaintext(v, in.TokenPlainText)
	service.ValidatePasswordPlaintext(v, in.NewPassword)
	if !v.Valid() {
		h.FailedValidationResponse(w, r, v.Errors)
		return
	}

	err = h.services.Users.UpdatePassword(in.TokenPlainText, in.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			v.AddError("token", "invalid or expired activation token")
			h.FailedValidationResponse(w, r, v.Errors)
		case errors.Is(err, repository.ErrEditConflict):
			h.EditConflictResponse(w, r)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"message": "your password has been successfully reset"}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}
