package httperror

import (
	"net/http"
)

type middlewareFunc func(http.Handler) http.Handler

type Router struct {
	Mux         *http.ServeMux
	middlewares []middlewareFunc
}

func NewRouter() *Router {
	return &Router{
		Mux: http.NewServeMux(),
	}
}

func (r *Router) Add(path string, h Handler) {
	r.Mux.Handle(path, chainMiddlewares(Handler(h), r.middlewares...))
}

func (r *Router) Use(mwf ...middlewareFunc) {
	for _, fn := range mwf {
		r.middlewares = append(r.middlewares, fn)
	}
}

func chainMiddlewares(h http.Handler, middlewares ...middlewareFunc) http.Handler {
	if len(middlewares) == 0 {
		return h
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for i := len(middlewares) - 1; i >= 0; i-- {
			h = middlewares[i](h)
		}

		h.ServeHTTP(w, r)
	})
}
