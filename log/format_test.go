package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestFormat(t *testing.T) {
	t.Parallel()

	t.Run("Encodes JSON explicitly", func(t *testing.T) {
		t.Parallel()

		output := logRecord(WithFormat(FormatJSON))

		assert.Assert(t, json.Valid([]byte(output)))
	})

	t.Run("Encodes human-readable text explicitly", func(t *testing.T) {
		t.Parallel()

		output := logRecord(WithFormat(FormatText))

		assert.Assert(t, !json.Valid([]byte(output)))
		assert.Assert(t, cmp.Contains(output, "V(0)"))
		assert.Assert(t, cmp.Contains(output, "Hello world!"))
	})

	t.Run("Uses JSON for an automatic non-terminal writer", func(t *testing.T) {
		t.Parallel()

		output := logRecord(WithFormat(FormatAuto))

		assert.Assert(t, json.Valid([]byte(output)))
	})

	t.Run("Rejects an invalid format", func(t *testing.T) {
		t.Parallel()

		defer func() {
			r := recover()
			assert.Assert(t, r != nil)
			assert.Assert(t, strings.Contains(fmt.Sprint(r), "invalid format"))
		}()

		WithFormat(Format(255))
	})
}

func TestResolveFormat(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		format   Format
		terminal bool
		want     Format
	}{
		"Keeps JSON for a terminal": {
			format:   FormatJSON,
			terminal: true,
			want:     FormatJSON,
		},
		"Keeps text for a non-terminal": {
			format: FormatText,
			want:   FormatText,
		},
		"Resolves auto to text for a terminal": {
			format:   FormatAuto,
			terminal: true,
			want:     FormatText,
		},
		"Resolves auto to JSON for a non-terminal": {
			format: FormatAuto,
			want:   FormatJSON,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, resolveFormat(tt.format, tt.terminal), tt.want)
		})
	}
}

func TestWithDebug(t *testing.T) {
	t.Parallel()

	t.Run("Aliases enabled to text", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t,
			logRecord(WithDebug(true)),
			logRecord(WithFormat(FormatText)),
		)
	})

	t.Run("Aliases disabled to JSON", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t,
			logRecord(WithDebug(false)),
			logRecord(WithFormat(FormatJSON)),
		)
	})

	t.Run("Uses the last format option", func(t *testing.T) {
		t.Parallel()

		text := logRecord(WithFormat(FormatJSON), WithDebug(true))
		jsonOutput := logRecord(WithDebug(true), WithFormat(FormatJSON))

		assert.Assert(t, !json.Valid([]byte(text)))
		assert.Assert(t, json.Valid([]byte(jsonOutput)))
	})
}

func TestWithLevelForAllFormats(t *testing.T) {
	t.Parallel()

	formats := map[string]Format{
		"JSON": FormatJSON,
		"Text": FormatText,
		"Auto": FormatAuto,
	}
	for name, format := range formats {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var b bytes.Buffer
			logger := New(&b, WithFormat(format), WithLevel(0))
			logger.V(1).Info("Not logged.")
			assert.Equal(t, b.String(), "")

			logger = New(&b, WithFormat(format), WithLevel(1))
			logger.V(1).Info("Logged.")
			assert.Assert(t, b.Len() > 0)
		})
	}
}

func logRecord(opts ...option) string {
	var b bytes.Buffer
	opts = append(opts, WithClock(fixedClock{}))
	logger := New(&b, opts...)
	logger.Info("Hello world!", "foo", "bar")

	return b.String()
}

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return time.Unix(626572800, 0)
}

func (fixedClock) NewTicker(time.Duration) *time.Ticker {
	return nil
}
