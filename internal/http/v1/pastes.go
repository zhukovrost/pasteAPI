package v1

import (
	"errors"
	"fmt"
	"net/http"
	"pasteAPI/internal/auth"
	"pasteAPI/internal/repository/models"
	"pasteAPI/internal/service"

	"pasteAPI/internal/repository"
	"pasteAPI/pkg/helpers"
	"pasteAPI/pkg/validator"
	"strings"
	"time"
)

func (h *Handler) ListPastesHandler(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Title    string
		Category uint8
		Filters  service.Filters
	}

	qs := r.URL.Query()
	in.Title = helpers.ReadString(qs, "title", "")
	in.Filters.Sort = helpers.ReadString(qs, "sort", "-created_at")

	v := validator.New()
	in.Category = uint8(helpers.ReadInt(qs, "category", 0, v))
	in.Filters.Page = uint32(helpers.ReadInt(qs, "page", 1, v))
	in.Filters.PageSize = uint32(helpers.ReadInt(qs, "pageSize", 5, v))
	in.Filters.SortSafelist = []string{"id", "-id", "title", "-title", "created_at", "-created_at", "expires_at", "-expires_at"}

	service.ValidateFilters(v, in.Filters)
	if !v.Valid() {
		h.FailedValidationResponse(w, r, v.Errors)
		return
	}

	pastes, metadata, err := h.models.Pastes.ReadAll(in.Title, in.Category, in.Filters)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
		return
	}
	// SendEmail a JSON response containing the movie repository.
	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"pastes": pastes, "metadata": metadata}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

func (h *Handler) GetPasteHandler(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.ReadIDParam(r)
	if err != nil {
		h.NotFoundResponse(w, r)
		return
	}

	paste, err := h.models.Pastes.Read(uint16(id))
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			h.NotFoundResponse(w, r)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"paste": paste}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

func (h *Handler) DeletePasteHandler(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.ReadIDParam(r)
	if err != nil {
		h.NotFoundResponse(w, r)
		return
	}

	err = h.models.Pastes.Delete(uint16(id))
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			h.NotFoundResponse(w, r)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = helpers.WriteJSON(w, http.StatusNoContent, nil, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

func (h *Handler) CreatePasteHandler(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Title    string `json:"title"`
		Category uint8  `json:"category,omitempty"`
		Text     string `json:"text"`
		Minutes  int32  `json:"minutes"`
	}

	err := helpers.ReadJSON(w, r, &in)
	if err != nil {
		h.BadRequestResponse(w, r, err)
		return
	}

	paste := &models.Paste{
		Title:    in.Title,
		Category: in.Category,
		Text:     in.Text,
		Minutes:  in.Minutes,
		Version:  1,
	}

	v := validator.New()
	v.Check(paste.Minutes > 0, "minutes", "must be greater than zero")
	if service.ValidatePaste(v, paste); !v.Valid() {
		h.FailedValidationResponse(w, r, v.Errors)
		return
	}

	err = h.models.Pastes.Create(paste)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
		return
	}

	if user := auth.ContextGetUser(r); !user.IsAnonymous() {
		err = h.models.Permissions.SetWritePermission(user.ID, paste.Id)
		if err != nil {
			h.ServerErrorResponse(w, r, err)
			return
		}
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("api/v1/pastes/%d", paste.Id))

	err = helpers.WriteJSON(w, http.StatusCreated, helpers.Envelope{"paste": paste}, headers)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

func (h *Handler) UpdatePasteHandler(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.ReadIDParam(r)
	if err != nil {
		h.NotFoundResponse(w, r)
		return
	}

	paste, err := h.models.Pastes.Read(uint16(id))
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			h.NotFoundResponse(w, r)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	var in struct {
		Title    *string `json:"title"`
		Category *uint8  `json:"category,omitempty"`
		Text     *string `json:"text"`
		Minutes  *int32  `json:"minutes"`
	}

	err = helpers.ReadJSON(w, r, &in)
	if err != nil {
		h.BadRequestResponse(w, r, err)
		return
	}

	if in.Title != nil {
		paste.Title = strings.TrimSpace(*in.Title)
	}
	if in.Category != nil {
		paste.Category = *in.Category
	}
	if in.Text != nil {
		paste.Text = strings.TrimSpace(*in.Text)
	}
	if in.Minutes != nil {
		paste.Minutes = *in.Minutes
		paste.ExpiresAt = paste.ExpiresAt.Add(time.Duration(paste.Minutes) * time.Minute)
	}

	v := validator.New()

	expiration := paste.ExpiresAt.Add(time.Duration(paste.Minutes) * time.Minute)
	v.Check(expiration.After(paste.CreatedAt), "minutes", "paste can't be expired before creation")

	if service.ValidatePaste(v, paste); !v.Valid() {
		h.FailedValidationResponse(w, r, v.Errors)
		return
	}

	err = h.models.Pastes.Update(paste)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrEditConflict):
			h.EditConflictResponse(w, r)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"paste": paste}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}
