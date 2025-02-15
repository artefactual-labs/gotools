package middleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"go.artefactual.dev/tools/middleware"
)

func TestWriteTimeout(t *testing.T) {
	t.Parallel()

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Microsecond * 100)
		w.Write([]byte("Hi there!"))
	})

	t.Run("Sets a write timeout", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(middleware.WriteTimeout(time.Microsecond)(h))
		defer ts.Close()

		_, err := ts.Client().Get(ts.URL)
		assert.ErrorIs(t, err, io.EOF)
	})

	t.Run("Sets an unlimited write timeout", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(middleware.WriteTimeout(0)(h))
		defer ts.Close()

		resp, err := ts.Client().Get(ts.URL)
		assert.NilError(t, err)

		blob, err := io.ReadAll(resp.Body)
		assert.NilError(t, err)
		assert.Equal(t, string(blob), "Hi there!")
	})
}
