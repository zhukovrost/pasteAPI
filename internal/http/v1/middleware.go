package v1

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"github.com/zhukovrost/pasteAPI/internal/auth"
	"github.com/zhukovrost/pasteAPI/internal/repository"
	"github.com/zhukovrost/pasteAPI/internal/repository/models"
	"github.com/zhukovrost/pasteAPI/pkg/helpers"
	"github.com/zhukovrost/pasteAPI/pkg/validator"
	"net"
	"net/http"
	"strings"
	"time"
)

func (h *Handler) RecoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				h.ServerErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) DebugRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.service.Deps.Logger.WithFields(map[string]interface{}{
			"request_method": r.Method,
			"request_url":    r.URL.Path,
			"origin":         r.Header.Get("Origin"),
		}).Debug("new request")
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.service.Config.Limiter.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			h.ServerErrorResponse(w, r, err)
			return
		}

		redisKey := fmt.Sprintf("client:%s", ip)
		ctx, cancel := context.WithTimeout(context.Background(), h.service.Deps.Redis.Timeout)
		defer cancel()

		// Increment the counter for this IP and set expiration if it's a new key
		pipe := h.service.Deps.Redis.Cache.TxPipeline()
		incr := pipe.Incr(ctx, redisKey)
		pipe.Expire(ctx, redisKey, time.Second)
		_, err = pipe.Exec(ctx)
		if err != nil {
			h.ServerErrorResponse(w, r, err)
			return
		}

		requests, err := incr.Result()
		if err != nil {
			h.ServerErrorResponse(w, r, err)
			return
		}

		if requests > int64(h.service.Config.Limiter.Burst) {
			h.RateLimitExceededResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h *Handler) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")
		authorizationHeader := r.Header.Get("Authorization")

		if authorizationHeader == "" {
			r = auth.ContextSetUser(r, models.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		} else {
			h.service.Deps.Logger.Debugf("Authorization header token is: %s", authorizationHeader)
		}

		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			h.InvalidAuthenticationTokenResponse(w, r)
			return
		}
		token := headerParts[1]

		v := validator.New()
		if models.ValidateTokenPlaintext(v, token); !v.Valid() {
			h.InvalidAuthenticationTokenResponse(w, r)
			return
		}

		user, err := h.service.Models.Users.GetForToken(repository.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrRecordNotFound):
				h.InvalidAuthenticationTokenResponse(w, r)
			default:
				h.ServerErrorResponse(w, r, err)
			}
			return
		}

		r = auth.ContextSetUser(r, user)
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) RequireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.ContextGetUser(r)
		if user.IsAnonymous() {
			h.AuthenticationRequiredResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) RequireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.ContextGetUser(r)
		if !user.Activated {
			h.InactiveAccountResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
	return h.RequireAuthenticatedUser(fn)
}

func (h *Handler) RequireAllowedToWriteUser(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.ContextGetUser(r)
		pasteId, err := helpers.ReadIDParam(r)
		if err != nil {
			h.BadRequestResponse(w, r, err)
			return
		}

		allowed, err := h.service.Models.Permissions.GetWritePermission(user.ID, uint16(pasteId))
		if err != nil {
			h.ServerErrorResponse(w, r, err)
			return
		}

		if !allowed {
			h.ForbiddenResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
	return h.RequireActivatedUser(fn)
}

func (h *Handler) EnableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		w.Header().Add("Vary", "Access-Control-Request-Method")
		origin := r.Header.Get("Origin")
		if origin != "" {
			for i := range h.service.Config.CORS.TrustedOrigins {
				if origin == h.service.Config.CORS.TrustedOrigins[i] {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
						w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
						w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

						w.WriteHeader(http.StatusOK)
						return
					}
					break
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) Metrics(next http.Handler) http.Handler {
	totalRequestsReceived := expvar.NewInt("total_requests_received")
	totalResponsesSent := expvar.NewInt("total_responses_sent")
	totalProcessingTimeMicroseconds := expvar.NewInt("total_processing_time_μs")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		totalRequestsReceived.Add(1)

		next.ServeHTTP(w, r)

		totalResponsesSent.Add(1)
		duration := time.Since(start).Microseconds()
		totalProcessingTimeMicroseconds.Add(duration)
	})
}
