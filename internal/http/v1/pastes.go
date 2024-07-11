package v1

import (
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/zhukovrost/pasteAPI/internal/auth"
	"github.com/zhukovrost/pasteAPI/internal/repository"
	"github.com/zhukovrost/pasteAPI/internal/repository/models"
	"github.com/zhukovrost/pasteAPI/internal/service"
	"github.com/zhukovrost/pasteAPI/pkg/helpers"
	"github.com/zhukovrost/pasteAPI/pkg/validator"
	"net/http"
	"strconv"
	"time"
)

// ListPastesHandler retrieves a paste by its ID
//
// @Summary      Retrieve a paste
// @Description  Retrieves a paste from the database by its ID.
// @Tags         pastes
// @Produce      json
// @Param        title     query    string  false  "Title of the paste"
// @Param        category  query    int     false  "Category ID of the paste"
// @Param        onlyUsers  query    bool     false  "Get pastes, which current user can update"
// @Param        sort      query    string  false  "Sort order, e.g., -created_at"
// @Param        page      query    int     false  "Page number for pagination"
// @Param        pageSize  query    int     false  "Number of items per page"
// @Success      200  {object}  service.ListPastesOutput  "Successfully retrieved paste"
// @Failure      422  {object}  ErrorResponse "Unprocessing data"
// @Failure      429 {object} ErrorResponse "Too many requests, rate limit exceeded"
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/pastes/ [get]
func (h *Handler) ListPastesHandler(w http.ResponseWriter, r *http.Request) {
	var in service.SearchSettings

	qs := r.URL.Query()
	in.Title = helpers.ReadString(qs, "title", "")
	in.Filters.Sort = helpers.ReadString(qs, "sort", "-created_at")
	in.OnlyUsers = helpers.ReadBool(qs, "onlyUsers", false)

	v := validator.New()
	in.Category = uint8(helpers.ReadInt(qs, "category", 0, v))
	in.Filters.Page = uint32(helpers.ReadInt(qs, "page", 1, v))
	in.Filters.PageSize = uint32(helpers.ReadInt(qs, "pageSize", 5, v))
	in.Filters.SortSafelist = []string{"id", "-id", "title", "-title", "created_at", "-created_at", "expires_at", "-expires_at"}

	models.ValidateFilters(v, in.Filters)
	if !v.Valid() {
		h.FailedValidationResponse(w, r, v.Errors)
		return
	}

	user := auth.ContextGetUser(r)

	listOutput, err := h.services.Pastes.GetList(in, user)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
		return
	}

	if err = helpers.WriteJSON(w, http.StatusOK, listOutput, nil); err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

// GetPasteHandler retrieves a paste by its ID
//
// @Summary      Retrieve a paste
// @Description  Retrieves a paste from the database by its ID.
// @Tags         pastes
// @Produce      json
// @Param        id   path   int   true       "Paste ID"
// @Success      200  {object}  service.PasteResp  "Successfully retrieved paste"
// @Failure      404  {object}  ErrorResponse "Paste not found"
// @Failure 429 {object} ErrorResponse "Too many requests, rate limit exceeded"
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/pastes/{id} [get]
func (h *Handler) GetPasteHandler(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.ReadIDParam(r)
	if err != nil {
		h.NotFoundResponse(w, r)
		return
	}

	user := auth.ContextGetUser(r)
	paste, err := h.services.Pastes.GetPaste(uint16(id), user)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			h.NotFoundResponse(w, r)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	if err := helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"paste": paste}, nil); err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

// DeletePasteHandler deletes a paste by its ID
//
// @Summary      Deletes a paste
// @Description  Deletes a paste from the database by its ID.
// @Tags         pastes
// @Produce      json
// @Security BearerAuth
// @Param        id   path     int   true   "Paste ID"
// @Success      204  "Successfully deleted paste"
// @Failure      403  {object}  ErrorResponse "User is not allowed to edit this paste"
// @Failure      404  {object} ErrorResponse "Paste not found"
// @Failure      429 {object} ErrorResponse "Too many requests, rate limit exceeded"
// @Failure      500  {object} ErrorResponse "Internal server error"
// @Router       /api/v1/pastes/{id} [delete]
func (h *Handler) DeletePasteHandler(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.ReadIDParam(r)
	if err != nil {
		h.NotFoundResponse(w, r)
		return
	}

	err = h.services.Pastes.Delete(uint16(id))
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			h.NotFoundResponse(w, r)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	if err = helpers.WriteJSON(w, http.StatusNoContent, nil, nil); err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

// CreatePasteHandler creates a new paste by input data
//
// @Summary      Create a new paste
// @Description  Creates a new paste in the database by input data.
// @Tags         pastes
// @Accept       json
// @Produce      json
// @Param        body  body     service.CreatePasteInput  true  "Paste creation input"
// @Security BearerAuth
// @Success      201  {object}  service.PasteResp  "Successfully created paste"
// @Header       201 {string} Location "URL of the newly created paste"
// @Failure      400  {object}  ErrorResponse "Bad request"
// @Failure      422  {object}  ErrorResponse "Unprocessable data"
// @Failure      429 {object} ErrorResponse "Too many requests, rate limit exceeded"
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/pastes/ [post]
func (h *Handler) CreatePasteHandler(w http.ResponseWriter, r *http.Request) {
	in := &service.CreatePasteInput{}

	err := helpers.ReadJSON(w, r, in)
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

	user := auth.ContextGetUser(r)

	err = h.services.Pastes.Create(paste, user)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("api/v1/pastes/%d", paste.Id))

	err = helpers.WriteJSON(w, http.StatusCreated, helpers.Envelope{"paste": paste}, headers)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

// UpdatePasteHandler updates a new paste by ID and input data
//
// @Summary      Update the paste
// @Description  Updates the paste in the database by ID and input data.
// @Tags         pastes
// @Accept       json
// @Produce      json
// @Param        id   path     int   true   "Paste ID"
// @Param        body  body     service.UpdatePasteInput  false  "Paste update input"
// @Security BearerAuth
// @Success      200  {object}  service.PasteResp  "Successfully updated paste"
// @Failure      400  {object}  ErrorResponse "Bad request"
// @Failure      403  {object}  ErrorResponse "User is not allowed to edit this paste"
// @Failure      404  {object}  ErrorResponse "Not found"
// @Failure      409  {object}  ErrorResponse "Conflict"
// @Failure      422  {object}  ErrorResponse "Unprocessable data"
// @Failure      429 {object} ErrorResponse "Too many requests, rate limit exceeded"
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/pastes/{id} [patch]
func (h *Handler) UpdatePasteHandler(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.ReadIDParam(r)
	if err != nil {
		h.NotFoundResponse(w, r)
		return
	}

	var in service.UpdatePasteInput

	err = helpers.ReadJSON(w, r, &in)
	if err != nil {
		h.BadRequestResponse(w, r, err)
		return
	}

	pasteForUpdate, err := h.services.Pastes.GetPasteForUpdate(uint16(id), in)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			h.NotFoundResponse(w, r)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	v := validator.New()

	expiration := pasteForUpdate.ExpiresAt.Add(time.Duration(pasteForUpdate.Minutes) * time.Minute)
	v.Check(expiration.After(pasteForUpdate.CreatedAt), "minutes", "paste can't be expired before creation")

	if service.ValidatePaste(v, pasteForUpdate); !v.Valid() {
		h.FailedValidationResponse(w, r, v.Errors)
		return
	}

	if err = h.services.Pastes.Update(pasteForUpdate); err != nil {
		switch {
		case errors.Is(err, repository.ErrEditConflict):
			h.EditConflictResponse(w, r)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"paste": pasteForUpdate}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

// PermissionHandler gives a write permission by input data
//
// @Summary      Gives a write permission
// @Description  Gives a permission to user in the url to update / delete paste in the url.
// @Tags         pastes
// @Tags         users
// @Produce      json
// @Param        id   path   int   true       "Paste ID"
// @Param        user_id   path   int   true       "User ID"
// @Security BearerAuth
// @Header 200 {string} Location "URL of the newly created paste"
// @Success      200  {object}  service.PastePermissionResponse  "Successfully gave permission"
// @Failure      404  {object}  ErrorResponse "Not found"
// @Failure 429 {object} ErrorResponse "Too many requests, rate limit exceeded"
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/pastes/{id}/permission/{user_id} [put]
func (h *Handler) PermissionHandler(w http.ResponseWriter, r *http.Request) {
	pasteId, err := helpers.ReadIDParam(r)
	if err != nil {
		h.NotFoundResponse(w, r)
		return
	}

	userIdStr := chi.URLParam(r, "user_id")
	userId, err := strconv.ParseInt(userIdStr, 10, 64)
	if err != nil || userId <= 0 {
		h.NotFoundResponse(w, r)
		return
	}

	resp, err := h.services.Pastes.GivePermission(uint16(pasteId), userId)

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("api/v1/pastes/%d", pasteId))

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"permission": resp}, headers)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}
