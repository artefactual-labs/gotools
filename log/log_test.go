package log_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"go.artefactual.dev/tools/log"
	"gotest.tools/v3/assert"
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

		assertLogRecord(t, b, map[string]interface{}{
			"level":  "0",
			"caller": "log/log_test.go:25",
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

		assertLogRecord(t, b, map[string]interface{}{
			"logger": "name",
			"level":  "4",
			"caller": "log/log_test.go:42",
			"msg":    "Hello world!",
			"foo":    "bar",
		})
	})
}

func assertLogRecord(t *testing.T, b bytes.Buffer, keysAndValues map[string]interface{}) {
	t.Helper()

	entry := map[string]interface{}{}
	err := json.Unmarshal(b.Bytes(), &entry)
	assert.NilError(t, err)
	assert.DeepEqual(t, entry, keysAndValues, ignoreTimestampField)
}
