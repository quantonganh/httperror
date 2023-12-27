package httperror

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/hlog"
)

type Handler func(w http.ResponseWriter, r *http.Request) error

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h(w, r)
	if err == nil {
		return
	}

	var e Error
	if !errors.As(err, &e) {
		hlog.FromRequest(r).Err(err).Msg("")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(e.Status)
	w.Write([]byte(fmt.Sprintf("message: %s", e.Error())))
}
