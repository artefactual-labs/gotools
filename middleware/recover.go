package middleware

import (
	"bytes"
	"errors"
	"net/http"
	"runtime"
	"strings"

	"github.com/go-logr/logr"
)

// Recover from panics and logs the error.
func Recover(logger logr.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					// Don't recover if the request is aborted, as this would
					// prevent the client from detecting the error.
					if rec == http.ErrAbortHandler {
						panic(rec)
					}

					// Prepare the error message and log it.
					b := strings.Builder{}

					switch x := rec.(type) {
					case string:
						b.WriteString("panic: ")
						b.WriteString(x)
					case error:
						b.WriteString("panic: ")
						b.WriteString(x.Error())
					default:
						b.WriteString("unknown panic")
					}

					const size = 64 << 10 // 64KB
					buf := make([]byte, size)
					n := runtime.Stack(buf, false)
					lines := bytes.Split(buf[:n], []byte{'\n'})
					b.WriteByte('\n')
					for _, line := range lines {
						b.Write(line)
						b.WriteByte('\n')
					}

					logger.Error(errors.New(b.String()), "Panic error recovered.")

					// Skip write header on upgrade connection.
					if r.Header.Get("Connection") != "Upgrade" {
						w.WriteHeader(http.StatusInternalServerError)
					}
				}
			}()
			h.ServeHTTP(w, r)
		})
	}
}
