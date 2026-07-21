// Package log provides simple functions to build an application logger based
// on the logr.Logger interface and the zap logging library.
//
// Use [New] to build the logger and [Sync] to flush buffered logs, e.g.;
//
//	logger := log.New(os.Stderr)
//	defer log.Sync(logger)
//	logger.Info("Hello!", "count", 10)
//
// [New] accepts multiple functional options, e.g. use [WithLevel] to specify
// the verbosity and [WithFormat] to select the record format:
//
//	logger := log.New(os.Stderr,
//		log.WithLevel(10),
//		log.WithFormat(log.FormatAuto),
//	)
//
// Visit the [logr] and [zap] projects for more details.
//
// [logr]: https://github.com/go-logr/logr
// [zap]: https://github.com/uber-go/zap
package log

import (
	"io"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New returns a new logger based on the logr interface and the zap logging
// library. Log records use [FormatJSON] by default.
func New(w io.Writer, opts ...option) logr.Logger {
	options := defaults()
	for _, o := range opts {
		o.apply(&options)
	}
	format := options.format
	if format == FormatAuto {
		format = resolveFormat(format, writerIsTerminal(w))
	}

	var encoder zapcore.Encoder
	{
		var config zapcore.EncoderConfig
		if format == FormatText {
			config = zap.NewDevelopmentEncoderConfig()
		} else {
			config = zap.NewProductionEncoderConfig()
		}
		config.EncodeName = nameEncoder(format)
		config.EncodeTime = timeEncoder(format)
		config.EncodeLevel = levelEncoder(format)
		config.CallerKey = "caller"

		if format == FormatText {
			encoder = zapcore.NewConsoleEncoder(config)
		} else {
			encoder = zapcore.NewJSONEncoder(config)
		}
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.Lock(zapcore.AddSync(w)),
		zap.NewAtomicLevelAt(zapcore.Level(-options.level)),
	)

	zapOpts := []zap.Option{
		zap.WithCaller(true),
		zap.WithClock(options.clock),
	}
	if options.addStack {
		zapOpts = append(zapOpts, zap.AddStacktrace(zap.ErrorLevel))
	}

	logger := zap.New(core, zapOpts...).Named(options.name)

	return zapr.NewLogger(logger)
}

// Sync flushes buffered logs.
func Sync(logger logr.Logger) {
	if zl, ok := Underlying(logger); ok {
		_ = zl.Core().Sync()
	}
}

// Underlying returns the zap logger used as a logr sink.
func Underlying(logger logr.Logger) (*zap.Logger, bool) {
	zl, ok := logger.GetSink().(zapr.Underlier)
	if !ok {
		return nil, false
	}

	return zl.GetUnderlying(), true
}
