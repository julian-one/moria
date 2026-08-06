package middleware

import "net/http"

// BodyLimit caps the request body with http.MaxBytesReader, so oversized
// bodies fail at the handler's decoder instead of being read in full.
func BodyLimit(maxBytes int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
