package middleware

import (
	"net/http"
	"strings"
)

// GetClientIP returns the originating client IP for r.
//
// Trusts X-Forwarded-For because Traefik (the only public ingress in stark)
// strips externally-supplied XFF and rewrites it from the real TCP source.
// If citadel is ever exposed without that ingress in front, this becomes
// spoofable — see stark/traefik/service-patch.yaml.
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
