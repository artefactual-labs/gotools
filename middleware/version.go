package middleware

import "net/http"

// VersionHeader sets a version header on the response. If name is empty, it
// defaults to "X-Version".
func VersionHeader(name, version string) func(http.Handler) http.Handler {
	if name == "" {
		name = "X-Version"
	}
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(name, version)
			h.ServeHTTP(w, r)
		})
	}
}
