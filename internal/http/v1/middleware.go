package v1

import (
	"errors"
	"expvar"
	"fmt"
	"golang.org/x/time/rate"
	"net"
	"net/http"
	"pasteAPI/internal/auth"
	"pasteAPI/internal/repository"
	"pasteAPI/internal/repository/models"
	"pasteAPI/pkg/helpers"
	"pasteAPI/pkg/validator"
	"strings"
	"sync"
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
		h.service.Logger.WithFields(map[string]interface{}{
			"request_method": r.Method,
			"request_url":    r.URL.Path,
		}).Debug("new request")
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) RateLimit(next http.Handler) http.Handler {
	// TODO: redis cache
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      = sync.Mutex{}
		clients = make(map[string]*client)
	)

	go func() {
		for {
			time.Sleep(time.Minute)

			mu.Lock()

			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.service.Config.Limiter.Enabled {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				h.ServerErrorResponse(w, r, err)
				return
			}
			mu.Lock()
			if _, found := clients[ip]; !found {
				clients[ip] = &client{limiter: rate.NewLimiter(
					rate.Limit(h.service.Config.Limiter.RPS),
					h.service.Config.Limiter.Burst,
				)}
			}

			clients[ip].lastSeen = time.Now()
			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				h.RateLimitExceededResponse(w, r)
				return
			}

			mu.Unlock()
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
		}

		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			h.InvalidAuthenticationTokenResponse(w, r)
			return
		}
		token := headerParts[1]

		v := validator.New()
		if repository.ValidateTokenPlaintext(v, token); !v.Valid() {
			h.InvalidAuthenticationTokenResponse(w, r)
			return
		}

		user, err := h.models.Users.GetForToken(repository.ScopeAuthentication, token)
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

		allowed, err := h.models.Permissions.GetWritePermission(user.ID, uint16(pasteId))
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
