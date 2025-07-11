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

func TestReadTimeout(t *testing.T) {
	t.Parallel()

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusRequestTimeout)
		}
	})

	for _, tt := range []struct {
		name     string
		timeout  time.Duration
		respCode int
	}{
		{
			name:     "Sets a read timeout",
			timeout:  time.Millisecond * 10,
			respCode: http.StatusRequestTimeout,
		},
		{
			name:     "Sets an unlimited read timeout",
			timeout:  0,
			respCode: http.StatusOK,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := httptest.NewServer(middleware.ReadTimeout(tt.timeout)(h))
			defer ts.Close()

			pr, pw := io.Pipe()
			defer pr.Close()
			go func() {
				time.Sleep(time.Millisecond * 100)
				pw.Write([]byte("body"))
				pw.Close()
			}()

			resp, err := ts.Client().Post(ts.URL, "application/octet-stream", pr)
			assert.NilError(t, err)
			assert.Equal(t, resp.StatusCode, tt.respCode)
		})
	}
}
