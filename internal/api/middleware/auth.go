package middleware

import (
	"net/http"
	"strings"
)

func Auth(apiKey string, enabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !enabled {
				next.ServeHTTP(w, r)
				return
			}
			// Skip auth for health/observability endpoints
			if r.URL.Path == "/health" || r.URL.Path == "/ready" || r.URL.Path == "/pressure" {
				next.ServeHTTP(w, r)
				return
			}

			auth := r.Header.Get("Authorization")
			token := strings.TrimPrefix(auth, "Bearer ")
			if token == "" || token != apiKey {
				http.Error(w, `{"success":false,"error":{"code":"UNAUTHORIZED","message":"invalid api key"}}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
