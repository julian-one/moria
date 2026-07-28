package middleware

import (
	"bufio"
	"errors"
	"net"
	"net/http"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader captures the status code and calls the underlying ResponseWriter's WriteHeader.
func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Flush implements the http.Flusher interface, allowing HTTP handlers to flush
// buffered data to the client if the underlying ResponseWriter supports it.
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack implements the http.Hijacker interface, allowing HTTP handlers to hijack
// the connection (needed for WebSocket upgrades).
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("underlying ResponseWriter does not implement http.Hijacker")
}
