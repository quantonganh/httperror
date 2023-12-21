package httperror

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
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

func RealIPHandler(key string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, err := GetIP(r)
			if err == nil {
				log := zerolog.Ctx(r.Context())
				log.UpdateContext(func(c zerolog.Context) zerolog.Context {
					return c.Str(key, ip)
				})
			}
			next.ServeHTTP(w, r)

		})
	}
}

func GetIP(r *http.Request) (string, error) {
	ip := r.Header.Get("X-Real-IP")
	netIP := net.ParseIP(ip)
	if netIP != nil {
		return ip, nil
	}

	ips := r.Header.Get("X-Forwarded-For")
	for _, ip := range strings.Split(ips, ",") {
		netIP := net.ParseIP(ip)
		if netIP != nil {
			return ip, nil
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", err
	}
	netIP = net.ParseIP(ip)
	if netIP != nil {
		return ip, nil
	}

	return "", fmt.Errorf("no valid IP found")
}
