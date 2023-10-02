package log_test

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"go.artefactual.dev/tools/log"
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
			"caller": "log/log_test.go:28",
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
			"caller": "log/log_test.go:45",
			"msg":    "Hello world!",
			"foo":    "bar",
		})
	})

	t.Run("Logs errors including stacktrace", func(t *testing.T) {
		var b bytes.Buffer

		logger := log.New(&b)
		logger.Error(io.EOF, "End of file.", "foo", "bar")

		assertErrorRecordWithStacktrace(t, b, map[string]interface{}{
			"level":  "2",
			"caller": "log/log_test.go:60",
			"error":  "EOF",
			"msg":    "End of file.",
			"foo":    "bar",
		})
	})

	t.Run("Logs errors without stacktrace", func(t *testing.T) {
		var b bytes.Buffer

		logger := log.New(&b, log.WithStacktrace(false))
		logger.Error(io.EOF, "End of file.", "foo", "bar")

		assertErrorRecordWithoutStacktrace(t, b, map[string]interface{}{
			"level":  "2",
			"caller": "log/log_test.go:75",
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

func assertErrorRecordWithoutStacktrace(t *testing.T, b bytes.Buffer, keysAndValues map[string]interface{}) {
	t.Helper()

	entry := map[string]interface{}{}
	err := json.Unmarshal(b.Bytes(), &entry)
	assert.NilError(t, err)

	// Must not include a stacktrace.
	_, ok := entry["stacktrace"]
	assert.Assert(t, !ok, "Stacktrace is included.")

	assert.DeepEqual(t, entry, keysAndValues, ignoreTimestampField)
}

func assertErrorRecordWithStacktrace(t *testing.T, b bytes.Buffer, keysAndValues map[string]interface{}) {
	t.Helper()

	entry := map[string]interface{}{}
	err := json.Unmarshal(b.Bytes(), &entry)
	assert.NilError(t, err)

	// Must include a stacktrace.
	st, ok := entry["stacktrace"]
	assert.Assert(t, ok, "Stacktrace is missing.")
	assert.Assert(t, cmp.Contains(st, "testing.tRunner"))
	delete(entry, "stacktrace")

	assert.DeepEqual(t, entry, keysAndValues, ignoreTimestampField)
}
