package v1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"github.com/zhukovrost/pasteAPI/internal/repository"
	"github.com/zhukovrost/pasteAPI/internal/repository/models"
	"github.com/zhukovrost/pasteAPI/internal/service"
)

type fakePastes struct {
	service.Pastes

	searchFn   func(title string) ([]*models.Paste, error)
	viewsFn    func(id int64) (int64, error)
	historyFn  func(id int64) ([]*models.PasteHistoryEntry, error)
	categoryFn func(name string) error
}

func (f *fakePastes) SearchPastes(title string) ([]*models.Paste, error) {
	return f.searchFn(title)
}

func (f *fakePastes) IncrementPasteViews(id int64) (int64, error) {
	return f.viewsFn(id)
}

func (f *fakePastes) GetPasteHistory(id int64) ([]*models.PasteHistoryEntry, error) {
	return f.historyFn(id)
}

func (f *fakePastes) CreateCategory(name string) error {
	return f.categoryFn(name)
}

type fakeUsers struct {
	service.Users

	login    string
	password string
	valid    bool
	err      error
}

func (f *fakeUsers) CheckLegacyPassword(login, password string) (bool, error) {
	f.login = login
	f.password = password
	return f.valid, f.err
}

func testHandler(pastes service.Pastes, users service.Users) *Handler {
	return NewHandler(&service.Services{
		Pastes: pastes,
		Users:  users,
		Deps:   service.Dependencies{Logger: logrus.New()},
	})
}

func withURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func decodeBody(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v (body=%q)", err, rr.Body.String())
	}
	return body
}

func TestSearchPastesHandler(t *testing.T) {
	var gotTitle string
	handler := testHandler(&fakePastes{
		searchFn: func(title string) ([]*models.Paste, error) {
			gotTitle = title
			return []*models.Paste{{Id: 1, Title: "hello"}}, nil
		},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pastes/search?title=hello", nil)
	rr := httptest.NewRecorder()
	handler.SearchPastesHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if gotTitle != "hello" {
		t.Fatalf("service got title %q, want %q", gotTitle, "hello")
	}

	body := decodeBody(t, rr)
	pastes, ok := body["pastes"].([]any)
	if !ok || len(pastes) != 1 {
		t.Fatalf("unexpected pastes: %#v", body["pastes"])
	}
	paste := pastes[0].(map[string]any)
	if paste["title"] != "hello" {
		t.Fatalf("unexpected paste title: %v", paste["title"])
	}
}

func TestSearchPastesHandlerServiceError(t *testing.T) {
	handler := testHandler(&fakePastes{
		searchFn: func(string) ([]*models.Paste, error) {
			return nil, errors.New("db is down")
		},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pastes/search?title=hello", nil)
	rr := httptest.NewRecorder()
	handler.SearchPastesHandler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestPasteViewsHandler(t *testing.T) {
	handler := testHandler(&fakePastes{
		viewsFn: func(id int64) (int64, error) {
			if id != 7 {
				t.Errorf("service got id %d, want 7", id)
			}
			return 42, nil
		},
	}, nil)

	req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/v1/pastes/7/views", nil), "id", "7")
	rr := httptest.NewRecorder()
	handler.PasteViewsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	body := decodeBody(t, rr)
	if body["views"] != float64(42) {
		t.Fatalf("views = %v, want 42", body["views"])
	}
}

func TestPasteHistoryHandler(t *testing.T) {
	var gotID int64
	handler := testHandler(&fakePastes{
		historyFn: func(id int64) ([]*models.PasteHistoryEntry, error) {
			gotID = id
			return []*models.PasteHistoryEntry{{Version: 2, Title: "second"}}, nil
		},
	}, nil)

	req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/v1/pastes/42/history", nil), "id", "42")
	rr := httptest.NewRecorder()
	handler.PasteHistoryHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if gotID != 42 {
		t.Fatalf("service got id %d, want 42", gotID)
	}

	body := decodeBody(t, rr)
	history, ok := body["history"].([]any)
	if !ok || len(history) != 1 {
		t.Fatalf("unexpected history: %#v", body["history"])
	}
}

func TestCreateCategoryHandler(t *testing.T) {
	var gotName string
	handler := testHandler(&fakePastes{
		categoryFn: func(name string) error {
			gotName = name
			return nil
		},
	}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", strings.NewReader(`{"name":"Work"}`))
	rr := httptest.NewRecorder()
	handler.CreateCategoryHandler(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusCreated)
	}
	if gotName != "Work" {
		t.Fatalf("service got name %q, want %q", gotName, "Work")
	}
}

func TestCreateCategoryHandlerValidationError(t *testing.T) {
	handler := testHandler(&fakePastes{
		categoryFn: func(string) error {
			return errors.New("must not be blank")
		},
	}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", strings.NewReader(`{"name":""}`))
	rr := httptest.NewRecorder()
	handler.CreateCategoryHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestLegacyPasswordCheckHandler(t *testing.T) {
	users := &fakeUsers{valid: true}
	handler := testHandler(&fakePastes{}, users)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/legacy-password-check",
		strings.NewReader(`{"login":"bob","password":"secret"}`))
	rr := httptest.NewRecorder()
	handler.LegacyPasswordCheckHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if users.login != "bob" || users.password != "secret" {
		t.Fatalf("service got login=%q password=%q", users.login, users.password)
	}

	body := decodeBody(t, rr)
	if body["valid"] != true {
		t.Fatalf("valid = %v, want true", body["valid"])
	}
}

func TestLegacyPasswordCheckUserNotFound(t *testing.T) {
	handler := testHandler(&fakePastes{}, &fakeUsers{err: repository.ErrUserNotFound})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/legacy-password-check",
		strings.NewReader(`{"login":"bob","password":"secret"}`))
	rr := httptest.NewRecorder()
	handler.LegacyPasswordCheckHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestPublicConfigHandler(t *testing.T) {
	handler := testHandler(&fakePastes{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public-config", nil)
	rr := httptest.NewRecorder()
	handler.PublicConfigHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	body := decodeBody(t, rr)
	if body["demo_key"] != "demo-key-for-docs" {
		t.Fatalf("demo_key = %v", body["demo_key"])
	}
	if body["max_paste_size"] != float64(1024*1024) {
		t.Fatalf("max_paste_size = %v", body["max_paste_size"])
	}
}
