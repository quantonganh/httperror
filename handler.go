package httperror

import (
	"errors"
	"fmt"
	"net/http"
)

type Handler func(w http.ResponseWriter, r *http.Request) error

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h(w, r)
	if err == nil {
		return
	}

	var e Error
	if !errors.As(err, &e) {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(e.Status)
	w.Write([]byte(fmt.Sprint("message: %s", e.Error())))
}
