package middleware

import (
	"net/http"
	"slices"
)

type Middleware func(http.Handler) http.Handler

type Chain struct {
	middlewares []Middleware
}

func New(m ...Middleware) Chain {
	return Chain{middlewares: slices.Clone(m)}
}

func (c Chain) Append(m ...Middleware) Chain {
	return Chain{
		middlewares: append(slices.Clone(c.middlewares), m...),
	}
}

func (c Chain) Wrap(h http.Handler) http.Handler {
	for _, mw := range slices.Backward(c.middlewares) {
		h = mw(h)
	}
	return h
}
