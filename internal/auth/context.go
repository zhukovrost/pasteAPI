package auth

import (
	"context"
	"net/http"
	"pasteAPI/internal/repository/models"
)

type contextKey string

const userContextKey = contextKey("user")

func ContextSetUser(r *http.Request, user *models.User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

func ContextGetUser(r *http.Request) *models.User {
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok {
		panic("missing user value in request context")
	}
	return user
}
