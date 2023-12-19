package httperror

import "net/http"

type Router struct {
	Handler *http.ServeMux
}

func NewRouter() *Router {
	return &Router{
		Handler: http.NewServeMux(),
	}
}

func (r *Router) Add(path string, h Handler) {
	r.Handler.Handle(path, Handler(h))
}
