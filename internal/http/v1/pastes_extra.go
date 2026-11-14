package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/zhukovrost/pasteAPI/internal/auth"
	"github.com/zhukovrost/pasteAPI/internal/repository"
	"github.com/zhukovrost/pasteAPI/internal/service"
	"github.com/zhukovrost/pasteAPI/pkg/helpers"
)

// SearchPastesHandler ищет пасты по точному заголовку.
//
// @Summary      Search pastes by title
// @Tags         pastes
// @Produce      json
// @Param        title  query  string  true  "Exact paste title"
// @Success      200  {object}  object
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/pastes/search [get]
func (h *Handler) SearchPastesHandler(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Query().Get("title")

	pastes, err := h.services.Pastes.SearchPastes(title)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"pastes": pastes}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

// SafeSearchPastesHandler ищет пасты по заголовку с сортировкой из whitelist.
//
// @Summary      Search pastes with safe sorting
// @Tags         pastes
// @Produce      json
// @Param        title  query  string  true   "Paste title"
// @Param        sort   query  string  false  "Sort order, e.g., -created_at"
// @Success      200  {object}  object
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/pastes/search/safe [get]
func (h *Handler) SafeSearchPastesHandler(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Query().Get("title")
	sort := r.URL.Query().Get("sort")

	pastes, err := h.services.Pastes.SearchPastesSafe(title, sort)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"pastes": pastes}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

// RawPasteHandler отдаёт текст пасты для встраивания по ссылке.
//
// @Summary      Get raw paste text
// @Tags         pastes
// @Produce      plain
// @Param        id  path  int  true  "Paste ID"
// @Security BearerAuth
// @Success      200  {string}  string  "Raw paste text"
// @Failure      404  {object}  ErrorResponse "Paste not found"
// @Router       /api/v1/pastes/{id}/raw [get]
func (h *Handler) RawPasteHandler(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.ReadIDParam(r)
	if err != nil {
		h.NotFoundResponse(w, r)
		return
	}

	user := auth.ContextGetUser(r)
	text, err := h.services.Pastes.GetRawPaste(id, user)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			h.NotFoundResponse(w, r)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(text))
}

// PasteViewsHandler увеличивает и возвращает счётчик просмотров пасты.
//
// @Summary      Increment paste views
// @Tags         pastes
// @Produce      json
// @Param        id  path  int  true  "Paste ID"
// @Success      200  {object}  object
// @Failure      404  {object}  ErrorResponse "Paste not found"
// @Router       /api/v1/pastes/{id}/views [get]
func (h *Handler) PasteViewsHandler(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.ReadIDParam(r)
	if err != nil {
		h.NotFoundResponse(w, r)
		return
	}

	views, err := h.services.Pastes.IncrementPasteViews(id)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			h.NotFoundResponse(w, r)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"views": views}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

// PasteHistoryHandler возвращает историю версий пасты.
//
// @Summary      Get paste history
// @Tags         pastes
// @Produce      json
// @Param        id  path  int  true  "Paste ID"
// @Success      200  {object}  object
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/pastes/{id}/history [get]
func (h *Handler) PasteHistoryHandler(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.ReadIDParam(r)
	if err != nil {
		h.NotFoundResponse(w, r)
		return
	}

	history, err := h.services.Pastes.GetPasteHistory(id)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"history": history}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

// BatchCreatePastesHandler создаёт несколько паст за один запрос.
//
// @Summary      Create pastes in batch
// @Tags         pastes
// @Accept       json
// @Produce      json
// @Param        body  body  []service.CreatePasteInput  true  "Paste creation input"
// @Security BearerAuth
// @Success      201  {object}  object
// @Failure      400  {object}  ErrorResponse "Bad request"
// @Router       /api/v1/pastes/batch [post]
func (h *Handler) BatchCreatePastesHandler(w http.ResponseWriter, r *http.Request) {
	h.services.Deps.Logger.Debugf("batch create: Authorization header %q", r.Header.Get("Authorization"))

	var in []service.CreatePasteInput
	_ = json.NewDecoder(r.Body).Decode(&in)

	user := auth.ContextGetUser(r)
	created, err := h.services.Pastes.CreatePastesBatch(in, user)
	if err != nil {
		h.BadRequestResponse(w, r, err)
		return
	}

	err = helpers.WriteJSON(w, http.StatusCreated, helpers.Envelope{"created": created}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

// CreateCategoryHandler создаёт новую категорию паст.
//
// @Summary      Create category
// @Tags         categories
// @Accept       json
// @Produce      json
// @Success      201  {object}  object
// @Failure      400  {object}  ErrorResponse "Bad request"
// @Router       /api/v1/categories [post]
func (h *Handler) CreateCategoryHandler(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)

	if err := h.services.Pastes.CreateCategory(in.Name); err != nil {
		h.BadRequestResponse(w, r, err)
		return
	}

	err := helpers.WriteJSON(w, http.StatusCreated, helpers.Envelope{"message": "category created"}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

// ModeratePasteHandler проверяет текст пасты во внешнем сервисе модерации.
//
// @Summary      Moderate paste
// @Tags         pastes
// @Produce      json
// @Param        id  path  int  true  "Paste ID"
// @Security BearerAuth
// @Success      200  {object}  object
// @Failure      404  {object}  ErrorResponse "Paste not found"
// @Router       /api/v1/pastes/{id}/moderate [post]
func (h *Handler) ModeratePasteHandler(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.ReadIDParam(r)
	if err != nil {
		h.NotFoundResponse(w, r)
		return
	}

	moderation, err := h.services.Pastes.ModeratePaste(id)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			h.NotFoundResponse(w, r)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"moderation": moderation}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

// LinkPreviewHandler загружает превью страницы по ссылке из пасты.
//
// @Summary      Fetch link preview
// @Tags         pastes
// @Produce      json
// @Param        url  query  string  true  "Target URL"
// @Success      200  {object}  object
// @Failure      400  {object}  ErrorResponse "Bad request"
// @Router       /api/v1/link-preview [get]
func (h *Handler) LinkPreviewHandler(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("url")

	preview, err := h.services.Pastes.FetchLinkPreview(target)
	if err != nil {
		h.BadRequestResponse(w, r, err)
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{
		"url":     target,
		"preview": preview,
	}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

// SafeLinkPreviewHandler загружает превью страницы с таймаутом и проверкой статуса.
//
// @Summary      Fetch link preview safely
// @Tags         pastes
// @Produce      json
// @Param        url  query  string  true  "Target URL"
// @Success      200  {object}  object
// @Failure      400  {object}  ErrorResponse "Bad request"
// @Router       /api/v1/link-preview/safe [get]
func (h *Handler) SafeLinkPreviewHandler(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("url")

	preview, err := h.services.Pastes.FetchLinkPreviewSafe(target)
	if err != nil {
		h.BadRequestResponse(w, r, err)
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{
		"url":     target,
		"preview": preview,
	}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

// DownloadAttachmentHandler отдаёт вложение пасты по имени файла.
//
// @Summary      Download attachment
// @Tags         attachments
// @Produce      octet-stream
// @Param        name  query  string  true  "Attachment file name"
// @Success      200  {file}  binary  "Attachment content"
// @Failure      404  {object}  ErrorResponse "Attachment not found"
// @Router       /api/v1/attachments [get]
func (h *Handler) DownloadAttachmentHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")

	data, err := h.services.Pastes.DownloadAttachment(name)
	if err != nil {
		h.NotFoundResponse(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
	_, _ = w.Write(data)
}

// DeleteAttachmentHandler удаляет вложение пасты.
//
// @Summary      Delete attachment
// @Tags         attachments
// @Param        name  query  string  true  "Attachment file name"
// @Success      204  "Attachment deleted"
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/attachments [delete]
func (h *Handler) DeleteAttachmentHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")

	if err := h.services.Pastes.DeleteAttachment(name); err != nil {
		h.ServerErrorResponse(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// LegacyPasswordCheckHandler проверяет пароль по старой md5-схеме.
//
// @Summary      Check legacy password
// @Tags         users
// @Accept       json
// @Produce      json
// @Success      200  {object}  object
// @Failure      400  {object}  ErrorResponse "Bad request"
// @Failure      404  {object}  ErrorResponse "User not found"
// @Router       /api/v1/users/legacy-password-check [post]
func (h *Handler) LegacyPasswordCheckHandler(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	if err := helpers.ReadJSON(w, r, &in); err != nil {
		h.BadRequestResponse(w, r, err)
		return
	}

	valid, err := h.services.Users.CheckLegacyPassword(in.Login, in.Password)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrUserNotFound):
			h.NotFoundResponse(w, r)
		default:
			h.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"valid": valid}, nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}

// PublicConfigHandler возвращает публичные настройки клиента.
//
// @Summary      Get public config
// @Tags         config
// @Produce      json
// @Success      200  {object}  object
// @Router       /api/v1/public-config [get]
func (h *Handler) PublicConfigHandler(w http.ResponseWriter, r *http.Request) {
	err := helpers.WriteJSON(w, http.StatusOK, h.services.PublicConfig(), nil)
	if err != nil {
		h.ServerErrorResponse(w, r, err)
	}
}
