package v1

import (
	"errors"
	"github.com/zhukovrost/pasteAPI/internal/repository"
	"github.com/zhukovrost/pasteAPI/internal/repository/models"
	"github.com/zhukovrost/pasteAPI/pkg/helpers"
	"github.com/zhukovrost/pasteAPI/pkg/validator"
	"net/http"
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
// @Failure 429 {object} ErrorResponse "Too many requests, rate limit exceeded"
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

	err = h.service.Models.Users.Create(user)
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

	token, err := h.service.Models.Tokens.New(user.ID, 8*time.Hour, repository.ScopeActivation)
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
		err = h.service.Deps.Mailer.SendEmail(user.Email, "welcome.tmpl", tmplData)

		if h.service.Config.Env == "development" {
			h.service.Deps.Logger.Infof("New activation tocken for user %s (id: %d): %s. "+
				"Go to (PUT) http://localhost:8080/api/v1/users/activated/ with token in th request body to activate user.",
				user.Login, user.ID, token.Plaintext,
			)
		}

		if err != nil {
			h.service.Deps.Logger.Error(err)
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
// @Failure 429 {object} ErrorResponse "Too many requests, rate limit exceeded"
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/users/activated [put]
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

	user, err := h.service.Models.Users.GetForToken(repository.ScopeActivation, in.TokenPlainText)
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

	err = h.service.Models.Users.Update(user)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrEditConflict):
			h.EditConflictResponse(w, r)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = h.service.Models.Tokens.DeleteAllForUser(repository.ScopeActivation, user.ID)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"user": user}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}
