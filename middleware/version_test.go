package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"

	"go.artefactual.dev/tools/middleware"
)

func TestVersionHeaderMiddleware(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	var continued bool
	handler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) { continued = true })
	mw := middleware.VersionHeader("", "v1.2.3")

	mw(handler).ServeHTTP(w, req)
	resp := w.Result()

	assert.Equal(t, resp.Header.Get("X-Version"), "v1.2.3")
	assert.Equal(t, continued, true)
}
