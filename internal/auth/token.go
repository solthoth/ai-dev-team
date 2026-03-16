package auth

import "net/http"

func RequireToken(expected string, next http.Handler) http.Handler {
	if expected == "" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Agent-Token") != expected {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}