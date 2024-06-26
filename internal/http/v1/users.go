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

type RegistrationInput struct {
	Login    string `json:"login"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResp struct {
	U *models.User `json:"user"`
}

// RegisterUserHandler creates a new user by input data
//
// @Summary      Registration
// @Description  Creates a new user in the database by input data.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body     RegistrationInput  true  "User registration input"
// @Success      202  {object}  UserResp  "Successfully accepted"
// @Failure      400  {object}  ErrorResponse "Bad request"
// @Failure      422  {object}  ErrorResponse "Unprocessable data"
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/users/ [post]
func (h *Handler) RegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	var input RegistrationInput

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

	if models.ValidateUser(v, user); !v.Valid() {
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

type ActivateUserInput struct {
	TokenPlainText string `json:"token"`
}

// ActivateUserHandler activates the user by input token
//
// @Summary      Activation
// @Description  Activates the user by input token.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body     ActivateUserInput  true  "User activation input"
// @Success      202  {object}  UserResp  "Successfully accepted"
// @Failure      400  {object}  ErrorResponse "Bad request"
// @Failure      422  {object}  ErrorResponse "Unprocessable data"
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/users/activated/ [put]
func (h *Handler) ActivateUserHandler(w http.ResponseWriter, r *http.Request) {
	var in ActivateUserInput

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
