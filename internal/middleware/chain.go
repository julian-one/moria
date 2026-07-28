package middleware

import (
	"net/http"
	"slices"
)

// Middleware is the standard middleware signature
type Middleware func(http.Handler) http.Handler

// Chain holds a sequence of middleware to apply in order
type Chain struct {
	middlewares []Middleware
}

// New creates a new middleware chain
func New(m Middleware) Chain {
	return Chain{middlewares: []Middleware{m}}
}

// Append adds a middleware to the chain, returning a new chain.
func (c Chain) Append(m Middleware) Chain {
	return Chain{
		middlewares: append(slices.Clone(c.middlewares), m),
	}
}

// Wrap applies the chain to a handler.
// The first middleware in the chain is the outermost (executed first).
func (c Chain) Wrap(h http.Handler) http.Handler {
	for _, mw := range slices.Backward(c.middlewares) {
		h = mw(h)
	}
	return h
}
