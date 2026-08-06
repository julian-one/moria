package middleware

import (
	"net/http"
	"strings"
)

// GetClientIP returns the originating client IP for r, for request logging
// only. X-Forwarded-For is trusted because every reachable caller is
// in-cluster (docs/security.md).
func GetClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}
