package httperror

import "net/http"

func NewServer(handler http.Handler, addr string) *http.Server {
	return &http.Server{
		Handler: handler,
		Addr:    addr,
	}
}
