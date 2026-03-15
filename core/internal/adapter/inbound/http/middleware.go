package http

import (
	"context"
	"log/slog"
	stdhttp "net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	domainauth "github.com/radhakrishna/archbattle/core/internal/domain/auth"
	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)


type Middleware struct {
	auth   *domainauth.Service
	logger *slog.Logger
}

func NewMiddleware(auth *domainauth.Service, logger *slog.Logger) *Middleware {
	if logger == nil {
		logger = slog.Default()
	}
	return &Middleware{auth: auth, logger: logger}
}

func (m *Middleware) RequestLogger(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		startedAt := time.Now()
		next.ServeHTTP(ww, r)
		m.logger.Info("http_request", "method", r.Method, "path", r.URL.Path, "status", ww.Status(), "duration_ms", time.Since(startedAt).Milliseconds())
	})
}

func (m *Middleware) Authenticated(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if token == "" {
			writeJSON(w, stdhttp.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
			return
		}
		session, err := m.auth.Authenticate(r.Context(), token)
		if err != nil {
			writeJSON(w, stdhttp.StatusUnauthorized, map[string]string{"error": "invalid auth token"})
			return
		}
		ctx := context.WithValue(r.Context(), shared.UserIDContextKey, session.UserID)
		ctx = context.WithValue(ctx, shared.UsernameContextKey, session.Username)
		ctx = context.WithValue(ctx, shared.TierContextKey, session.Tier)
		ctx = context.WithValue(ctx, shared.RoleContextKey, session.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AdminOnly requires the authenticated user to have the "admin" role.
// Must be used after Authenticated middleware.
func (m *Middleware) AdminOnly(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		role, _ := r.Context().Value(shared.RoleContextKey).(string)
		if role != domainauth.RoleAdmin {
			writeJSON(w, stdhttp.StatusForbidden, map[string]string{"error": "admin access required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func CurrentUserID(ctx context.Context) (uuid.UUID, bool) {
	value := ctx.Value(shared.UserIDContextKey)
	userID, ok := value.(uuid.UUID)
	return userID, ok
}
