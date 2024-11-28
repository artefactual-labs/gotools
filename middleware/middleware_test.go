package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-logr/logr/funcr"
	"github.com/gorilla/mux"
	"gotest.tools/v3/assert"

	"go.artefactual.dev/tools/middleware"
)

func TestMiddlewares(t *testing.T) {
	t.Parallel()

	logged := ""
	logger := funcr.New(func(prefix, args string) { logged += prefix + args }, funcr.Options{})

	panicker := func(w http.ResponseWriter, r *http.Request) { panic("opsie") }
	router := mux.NewRouter()
	router.HandleFunc("/", panicker).Methods("GET")
	router.Use(
		middleware.Recover(logger),
		middleware.WriteTimeout(0),
		middleware.VersionHeader("", "v1.2.3"),
	)

	rw := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	router.ServeHTTP(rw, req)

	// Recover logs the panic error and handles the response.
	assert.Assert(t, strings.Contains(logged, "Panic error recovered."))

	// VersionHeader injects a header into the response.
	assert.Equal(t, rw.Header().Get("X-Version"), "v1.2.3")
}
