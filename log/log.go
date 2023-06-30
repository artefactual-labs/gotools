// Package log provides simple functions to build an application logger based
// on the [logr.Logger] interface and the [zap] logging library.
//
// Use [New] to build the logger and [Sync] to flush buffered logs, e.g.;
//
//	logger := logr.New(os.Stderr)
//	defer log.Sync(logger)
//	logger.Info("Hello!", "count", 10)
//
// [New] accepts multiple functional options, e.g. use [WithLevel] to specify
// the verbosity of the logger:
//
//	logger := logr.New(os.Stderr, log.WithLevel(10))
//
// [logr.Logger]: https://github.com/go-logr/logr
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
// library.
func New(w io.Writer, opts ...option) logr.Logger {
	options := options{
		clock: zapcore.DefaultClock,
	}
	for _, o := range opts {
		o.apply(&options)
	}

	var encoder zapcore.Encoder
	{
		var config zapcore.EncoderConfig
		if options.debug {
			config = zap.NewDevelopmentEncoderConfig()
		} else {
			config = zap.NewProductionEncoderConfig()
		}
		config.EncodeName = nameEncoder(options.debug)
		config.EncodeTime = timeEncoder(options.debug)
		config.EncodeLevel = levelEncoder(options.debug)
		config.CallerKey = "caller"

		if options.debug {
			encoder = zapcore.NewConsoleEncoder(config)
		} else {
			encoder = zapcore.NewJSONEncoder(config)
		}
	}

	var writer zapcore.WriteSyncer = zapcore.Lock(zapcore.AddSync(w))

	var levelEnabler zapcore.LevelEnabler = zap.NewAtomicLevelAt(zapcore.Level(-options.level))

	logger := zap.New(
		zapcore.NewCore(encoder, writer, levelEnabler),
		zap.WithCaller(true),
		zap.AddStacktrace(zap.ErrorLevel),
		zap.WithClock(options.clock),
	).Named(options.name)

	return zapr.NewLogger(logger)
}

// Sync flushes buffered logs.
func Sync(logger logr.Logger) {
	if zl, ok := Underlying(logger); ok {
		zl.Core().Sync()
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
