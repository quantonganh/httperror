package httperror

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"golang.org/x/time/rate"
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
	ip := r.Header.Get("X-Real-Ip")
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

type Message struct {
	Body string `json:"body"`
}

// https://blog.logrocket.com/rate-limiting-go-application/#per-client-rate-limiting
func PerClientRateLimiter(interval time.Duration) func(next func(w http.ResponseWriter, r *http.Request)) http.Handler {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		clients = make(map[string]*client)
		mu      sync.Mutex
	)

	go func() {
		for {
			time.Sleep(5 * interval)
			mu.Lock()
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 10*interval {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next func(w http.ResponseWriter, r *http.Request)) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, err := GetIP(r)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			mu.Lock()
			if _, found := clients[ip]; !found {
				clients[ip] = &client{
					limiter: rate.NewLimiter(rate.Every(interval), 1),
				}
			}
			clients[ip].lastSeen = time.Now()
			if !clients[ip].limiter.Allow() {
				mu.Unlock()

				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			mu.Unlock()
			next(w, r)
		})
	}
}
