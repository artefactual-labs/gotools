package log_test

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"go.artefactual.dev/tools/log"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

// Custom cmp.Option to ignore the timestamp entry in log records.
var ignoreTimestampField = cmpopts.IgnoreMapEntries(func(k string, v any) bool {
	return k == "ts"
})

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("Creates logger without options provided", func(t *testing.T) {
		var b bytes.Buffer

		logger := log.New(&b)
		logger.Info("Hello world!", "foo", "bar")

		assertInfoRecord(t, b, map[string]interface{}{
			"level":  "0",
			"caller": "log/log_test.go:27",
			"msg":    "Hello world!",
			"foo":    "bar",
		})
	})

	t.Run("Creates logger with options provided", func(t *testing.T) {
		var b bytes.Buffer

		logger := log.New(&b,
			log.WithName("name"),
			log.WithLevel(10),
		)
		logger.V(4).Info("Hello world!", "foo", "bar")

		assertInfoRecord(t, b, map[string]interface{}{
			"logger": "name",
			"level":  "4",
			"caller": "log/log_test.go:44",
			"msg":    "Hello world!",
			"foo":    "bar",
		})
	})

	t.Run("Logs errors including stack qtrace", func(t *testing.T) {
		var b bytes.Buffer

		logger := log.New(&b)
		logger.Error(io.EOF, "End of file.", "foo", "bar")

		assertErrorRecord(t, b, map[string]interface{}{
			"level":  "2",
			"caller": "log/log_test.go:59",
			"error":  "EOF",
			"msg":    "End of file.",
			"foo":    "bar",
		})
	})
}

func assertInfoRecord(t *testing.T, b bytes.Buffer, keysAndValues map[string]interface{}) {
	t.Helper()

	entry := map[string]interface{}{}
	err := json.Unmarshal(b.Bytes(), &entry)
	assert.NilError(t, err)
	assert.DeepEqual(t, entry, keysAndValues, ignoreTimestampField)
}

func assertErrorRecord(t *testing.T, b bytes.Buffer, keysAndValues map[string]interface{}) {
	t.Helper()

	entry := map[string]interface{}{}
	err := json.Unmarshal(b.Bytes(), &entry)
	assert.NilError(t, err)

	// Must include a stack trace.
	st, ok := entry["stacktrace"]
	assert.Assert(t, ok, "Stack trace is missing.")
	assert.Assert(t, cmp.Contains(st, "testing.tRunner"))
	delete(entry, "stacktrace")

	assert.DeepEqual(t, entry, keysAndValues, ignoreTimestampField)
}
