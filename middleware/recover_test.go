package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/logr/funcr"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"go.artefactual.dev/tools/middleware"
)

func TestRescoverMiddleware(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	var logged string
	logger := funcr.New(
		func(prefix, args string) { logged = args },
		funcr.Options{},
	)

	handler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) { panic("opsie") })
	mw := middleware.Recover(logger)

	mw(handler).ServeHTTP(w, req)

	assert.Assert(t, cmp.Contains(logged, "\"msg\"=\"Panic error recovered.\""))
	assert.Assert(t, cmp.Contains(logged, "\"error\"=\"panic: opsie"))
}
