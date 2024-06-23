package main

import (
	"errors"
	"fmt"
	"net/http"
	"pasteAPI/internal/data"
	"pasteAPI/internal/validator"
	"strings"
	"time"
)

func (app *application) listPastesHandler(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Title    string
		Category uint8
		Filters  data.Filters
	}

	qs := r.URL.Query()
	in.Title = app.readString(qs, "title", "")
	in.Filters.Sort = app.readString(qs, "sort", "-created_at")

	v := validator.New()
	in.Category = uint8(app.readInt(qs, "category", 0, v))
	in.Filters.Page = uint32(app.readInt(qs, "page", 1, v))
	in.Filters.PageSize = uint32(app.readInt(qs, "pageSize", 5, v))
	in.Filters.SortSafelist = []string{"id", "-id", "title", "-title", "created_at", "-created_at", "expires_at", "-expires_at"}

	data.ValidateFilters(v, in.Filters)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	pastes, metadata, err := app.models.Pastes.ReadAll(in.Title, in.Category, in.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	// Send a JSON response containing the movie data.
	err = app.writeJSON(w, http.StatusOK, envelope{"pastes": pastes, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) getPasteHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	paste, err := app.models.Pastes.Read(uint16(id))
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"paste": paste}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deletePasteHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	err = app.models.Pastes.Delete(uint16(id))
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusNoContent, nil, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) createPasteHandler(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Title    string `json:"title"`
		Category uint8  `json:"category,omitempty"`
		Text     string `json:"text"`
		Minutes  int32  `json:"minutes"`
	}

	err := app.readJSON(w, r, &in)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	paste := &data.Paste{
		Title:    in.Title,
		Category: in.Category,
		Text:     in.Text,
		Minutes:  in.Minutes,
		Version:  1,
	}

	v := validator.New()
	v.Check(paste.Minutes > 0, "minutes", "must be greater than zero")
	if data.ValidatePaste(v, paste); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Pastes.Create(paste)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if user := app.contextGetUser(r); !user.IsAnonymous() {
		err = app.models.Permissions.SetWritePermission(user.ID, paste.Id)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("api/v1/pastes/%d", paste.Id))

	err = app.writeJSON(w, http.StatusCreated, envelope{"paste": paste}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updatePasteHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	paste, err := app.models.Pastes.Read(uint16(id))
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	var in struct {
		Title    *string `json:"title"`
		Category *uint8  `json:"category,omitempty"`
		Text     *string `json:"text"`
		Minutes  *int32  `json:"minutes"`
	}

	err = app.readJSON(w, r, &in)
	if err != nil {
		app.badRequestResponse(w, r, err)
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

	if data.ValidatePaste(v, paste); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Pastes.Update(paste)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"paste": paste}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
