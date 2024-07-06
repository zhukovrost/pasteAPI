package v1

import (
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/zhukovrost/pasteAPI/internal/auth"
	"github.com/zhukovrost/pasteAPI/internal/repository"
	"github.com/zhukovrost/pasteAPI/internal/repository/models"
	"github.com/zhukovrost/pasteAPI/pkg/helpers"
	"github.com/zhukovrost/pasteAPI/pkg/validator"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// pasteKey generates the key for searched paste
func pasteKey(id uint16) string {
	return fmt.Sprintf("paste:%d", id)
}

// searchKey generates the key for search result
// pattern: search:title:category:sort:page:pagesize
func searchKey(settings SearchSettings) string {
	return fmt.Sprintf("search:%s:%d:%s:%d:%d", settings.Title, settings.Category, settings.Filters.Sort, settings.Filters.Page, settings.Filters.PageSize)
}

type SearchSettings struct {
	Title    string
	Category uint8
	Filters  models.Filters
}

type ListPastesOutput struct {
	Pastes   []*models.Paste  `json:"pastes"`
	Metadata *models.Metadata `json:"metadata"`
}

// ListPastesHandler retrieves a paste by its ID
//
// @Summary      Retrieve a paste
// @Description  Retrieves a paste from the database by its ID.
// @Tags         pastes
// @Produce      json
// @Param        title     query    string  false  "Title of the paste"
// @Param        category  query    int     false  "Category ID of the paste"
// @Param        sort      query    string  false  "Sort order, e.g., -created_at"
// @Param        page      query    int     false  "Page number for pagination"
// @Param        pageSize  query    int     false  "Number of items per page"
// @Success      200  {object}  ListPastesOutput  "Successfully retrieved paste"
// @Failure      422  {object}  ErrorResponse "Unprocessing data"
// @Failure 429 {object} ErrorResponse "Too many requests, rate limit exceeded"
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/pastes/ [get]
func (h *Handler) ListPastesHandler(w http.ResponseWriter, r *http.Request) {
	var in SearchSettings

	qs := r.URL.Query()
	in.Title = helpers.ReadString(qs, "title", "")
	in.Filters.Sort = helpers.ReadString(qs, "sort", "-created_at")

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

	cacheKey := searchKey(in)
	listOutput := &ListPastesOutput{}
	err := getFromCache(h.service.Deps.Redis, cacheKey, listOutput)
	if err == nil {
		h.service.Deps.Logger.Debugf("pastes list got from cache")
		if err = helpers.WriteJSON(w, http.StatusOK, listOutput, nil); err != nil {
			h.ServerErrorResponse(w, r, err)
		}
		return
	} else if !errors.Is(err, redis.Nil) {
		h.service.Deps.Logger.Error(err)
	}

	pastes, metadata, err := h.service.Models.Pastes.ReadAll(in.Title, in.Category, in.Filters)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
		return
	}
	listOutput = &ListPastesOutput{pastes, metadata}
	// SendEmail a JSON response containing the movie repository.
	err = helpers.WriteJSON(w, http.StatusOK, listOutput, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}

	h.service.Background(func() {
		in, err := existsInCache(h.service.Deps.Redis, cacheKey)
		if err != nil {
			h.service.Deps.Logger.Error(err)
			return
		}

		if !in && setCache(h.service.Deps.Redis, cacheKey, listOutput) != nil {
			h.service.Deps.Logger.Error(err)
		}
	})
}

type PasteResp struct {
	R *models.Paste `json:"paste"`
}

// GetPasteHandler retrieves a paste by its ID
//
// @Summary      Retrieve a paste
// @Description  Retrieves a paste from the database by its ID.
// @Tags         pastes
// @Produce      json
// @Param        id   path   int   true       "Paste ID"
// @Success      200  {object}  PasteResp  "Successfully retrieved paste"
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

	successfulResp := func(p *models.Paste) {
		if err := helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"paste": p}, nil); err != nil {
			h.ServerErrorResponse(w, r, err)
		}
	}

	paste := &models.Paste{}
	err = getFromCache(h.service.Deps.Redis, pasteKey(uint16(id)), paste)
	if err == nil {
		successfulResp(paste)
		h.service.Deps.Logger.Debugf("paste (ID: %d) got from cache", paste.Id)
		return
	} else if !errors.Is(err, redis.Nil) {
		h.service.Deps.Logger.Error(err)
	}

	paste, err = h.service.Models.Pastes.Read(uint16(id))
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			h.NotFoundResponse(w, r)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	successfulResp(paste)

	h.service.Background(func() {
		in, err := existsInCache(h.service.Deps.Redis, pasteKey(paste.Id))
		if err != nil {
			h.service.Deps.Logger.Error(err)
			return
		}

		if !in && setCache(h.service.Deps.Redis, pasteKey(paste.Id), paste) != nil {
			h.service.Deps.Logger.Error(err)
		}
	})
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
// @Failure 429 {object} ErrorResponse "Too many requests, rate limit exceeded"
// @Failure      500  {object} ErrorResponse "Internal server error"
// @Router       /api/v1/pastes/{id} [delete]
func (h *Handler) DeletePasteHandler(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.ReadIDParam(r)
	if err != nil {
		h.NotFoundResponse(w, r)
		return
	}

	err = h.service.Models.Pastes.Delete(uint16(id))
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

	// Delete paste from cache as well
	go func() {
		in, err := existsInCache(h.service.Deps.Redis, pasteKey(uint16(id)))
		if err != nil {
			h.service.Deps.Logger.Error(err)
			return
		}

		if in && deleteFromCache(h.service.Deps.Redis, pasteKey(uint16(id))) != nil {
			h.service.Deps.Logger.Error(err)
		}

		if in && invalidateCache(h.service.Deps.Redis, pasteKey(uint16(id))) != nil {
			h.service.Deps.Logger.Error(err)
		}
	}()
}

type CreatePasteInput struct {
	Title    string `json:"title"`
	Category uint8  `json:"category,omitempty"`
	Text     string `json:"text"`
	Minutes  int32  `json:"minutes"`
}

// CreatePasteHandler creates a new paste by input data
//
// @Summary      Create a new paste
// @Description  Creates a new paste in the database by input data.
// @Tags         pastes
// @Accept       json
// @Produce      json
// @Param        body  body     CreatePasteInput  true  "Paste creation input"
// @Security BearerAuth
// @Success      201  {object}  PasteResp  "Successfully created paste"
// @Header 201 {string} Location "URL of the newly created paste"
// @Failure      400  {object}  ErrorResponse "Bad request"
// @Failure      422  {object}  ErrorResponse "Unprocessable data"
// @Failure 429 {object} ErrorResponse "Too many requests, rate limit exceeded"
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/pastes/ [post]
func (h *Handler) CreatePasteHandler(w http.ResponseWriter, r *http.Request) {
	var in CreatePasteInput

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
	if models.ValidatePaste(v, paste); !v.Valid() {
		h.FailedValidationResponse(w, r, v.Errors)
		return
	}

	err = h.service.Models.Pastes.Create(paste)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
		return
	}

	if user := auth.ContextGetUser(r); !user.IsAnonymous() {
		err = h.service.Models.Permissions.SetWritePermission(user.ID, paste.Id)
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

	h.service.Background(func() {
		if err = setCache(h.service.Deps.Redis, pasteKey(paste.Id), paste); err == nil {
			if err = invalidateCache(h.service.Deps.Redis, "search:*"); err == nil {
				h.service.Deps.Logger.Debugf("paste (ID: %d) added to cache", paste.Id)
				return
			}
		}
		h.service.Deps.Logger.Errorf("paste (ID: %d) not added to cache due to the error: %s", paste.Id, err)
	})
}

type UpdatePasteInput struct {
	Title    *string `json:"title"`
	Category *uint8  `json:"category,omitempty"`
	Text     *string `json:"text"`
	Minutes  *int32  `json:"minutes"`
}

// UpdatePasteHandler updates a new paste by ID and input data
//
// @Summary      Update the paste
// @Description  Updates the paste in the database by ID and input data.
// @Tags         pastes
// @Accept       json
// @Produce      json
// @Param        id   path     int   true   "Paste ID"
// @Param        body  body     UpdatePasteInput  false  "Paste update input"
// @Security BearerAuth
// @Success      200  {object}  PasteResp  "Successfully updated paste"
// @Failure      400  {object}  ErrorResponse "Bad request"
// @Failure      403  {object}  ErrorResponse "User is not allowed to edit this paste"
// @Failure      404  {object}  ErrorResponse "Not found"
// @Failure      422  {object}  ErrorResponse "Unprocessable data"
// @Failure 429 {object} ErrorResponse "Too many requests, rate limit exceeded"
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/pastes/{id} [patch]
func (h *Handler) UpdatePasteHandler(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.ReadIDParam(r)
	if err != nil {
		h.NotFoundResponse(w, r)
		return
	}

	paste := &models.Paste{}
	err = getFromCache(h.service.Deps.Redis, pasteKey(uint16(id)), paste)
	if err != nil {
		paste, err = h.service.Models.Pastes.Read(uint16(id))
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrRecordNotFound):
				h.NotFoundResponse(w, r)
			default:
				h.ServerErrorResponse(w, r, err)
			}
			return
		}
	}

	var in UpdatePasteInput

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

	if models.ValidatePaste(v, paste); !v.Valid() {
		h.FailedValidationResponse(w, r, v.Errors)
		return
	}

	err = h.service.Models.Pastes.Update(paste)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrEditConflict):
			h.EditConflictResponse(w, r)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	if err = setCache(h.service.Deps.Redis, pasteKey(paste.Id), paste); err != nil {
		h.service.Deps.Logger.Errorf("paste (ID: %d) not added to cache due to the error: %s", paste.Id, err)
	} else {
		h.service.Deps.Logger.Debugf("paste (ID: %d) added to cache", paste.Id)
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"paste": paste}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}

	h.service.Background(func() {
		if err = invalidateCache(h.service.Deps.Redis, "search:*"); err != nil {
			h.service.Deps.Logger.Error(err)
		}
	})
}

type PastePermissionResponse struct {
	R struct {
		P uint16 `json:"paste_id"`
		U int64  `json:"user_id"`
	} `json:"permission"`
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
// @Success      200  {object}  PastePermissionResponse  "Successfully gave permission"
// @Header 200 {string} Location "URL of the newly created paste"
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

	if err = h.models.Permissions.SetWritePermission(userId, uint16(pasteId)); err != nil {
		h.ServerErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("api/v1/pastes/%d", pasteId))

	resp := struct {
		P uint16 `json:"paste_id"`
		U int64  `json:"user_id"`
	}{uint16(pasteId), userId}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"permission": resp}, headers)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}
