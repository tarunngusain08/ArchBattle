package http

import (
	stdhttp "net/http"
	"strings"
)

// InternalAuth enforces a shared-secret token so only the Core service (or trusted
// callers within the Docker network) can call the AI service endpoints.
func InternalAuth(secret string) func(stdhttp.Handler) stdhttp.Handler {
	return func(next stdhttp.Handler) stdhttp.Handler {
		return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			if strings.TrimSpace(secret) == "" {
				// If no secret is configured, allow all traffic (dev mode only).
				next.ServeHTTP(w, r)
				return
			}
			token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
			if token != secret {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(stdhttp.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
