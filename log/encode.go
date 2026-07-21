package log

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"go.uber.org/zap/zapcore"
)

type color uint8

const (
	black color = iota + 30
	red
	green
	yellow
	blue
	magenta
	cyan
	white
)

func (c color) Add(s string) string {
	if c == 0 {
		return s
	}
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", uint8(c), s)
}

func nameEncoder(format Format) func(loggerName string, enc zapcore.PrimitiveArrayEncoder) {
	return func(loggerName string, enc zapcore.PrimitiveArrayEncoder) {
		var c color
		if format == FormatText {
			c = green
		}

		enc.AppendString(c.Add(loggerName))
	}
}

func timeEncoder(format Format) func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		if format == FormatJSON {
			zapcore.EpochTimeEncoder(t, enc)
			return
		}

		const layout = "2006-01-02T15:04:05.000Z0700" // ISO8601TimeEncoder.
		formatted := t.Format(layout)
		enc.AppendString(yellow.Add(formatted))
	}
}

func levelEncoder(format Format) func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	return func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		vl := math.Abs(float64(l))
		level := strconv.FormatFloat(vl, 'f', 0, 64)

		if format == FormatJSON {
			enc.AppendString(level)
			return
		}

		var c color
		switch {
		case vl == 2:
			c = red
		case vl >= 0:
			c = cyan
		default:
			c = white
		}

		enc.AppendString(
			c.Add(fmt.Sprintf("V(%s)", level)),
		)
	}
}
