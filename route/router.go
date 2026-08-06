package route

import (
	"net/http"
	"slices"

	"moria/internal/middleware"
)

type router struct {
	mux         *http.ServeMux
	middlewares []middleware.Middleware
}

func (r *router) Use(m ...middleware.Middleware) {
	r.middlewares = append(r.middlewares, m...)
}

func (r *router) Group(fn func(*router)) {
	fn(&router{mux: r.mux, middlewares: slices.Clone(r.middlewares)})
}

func (r *router) Handle(pattern string, h http.Handler) {
	for _, mw := range slices.Backward(r.middlewares) {
		h = mw(h)
	}
	r.mux.Handle(pattern, h)
}
