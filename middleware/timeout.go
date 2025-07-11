package middleware

import (
	"net/http"
	"time"
)

// WriteTimeout sets the write deadline for writing the response. Use 0 to set
// an unlimited timeout.
func WriteTimeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rc := http.NewResponseController(w)
			var deadline time.Time
			if timeout != 0 {
				deadline = time.Now().Add(timeout)
			}
			_ = rc.SetWriteDeadline(deadline)
			h.ServeHTTP(w, r)
		})
	}
}

// ReadTimeout sets the read deadline for reading the request. A zero value
// means no timeout.
func ReadTimeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rc := http.NewResponseController(w)
			var deadline time.Time
			if timeout != 0 {
				deadline = time.Now().Add(timeout)
			}
			_ = rc.SetReadDeadline(deadline)
			h.ServeHTTP(w, r)
		})
	}
}
