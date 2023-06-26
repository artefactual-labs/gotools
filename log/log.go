package log

import (
	"io"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New returns a new logger based on the logr interface and the zap logging library.
func New(w io.Writer, opts ...option) logr.Logger {
	options := options{}
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

	core := zapcore.NewCore(encoder, writer, levelEnabler)
	logger := zap.New(core).
		Named(options.name).
		WithOptions(zap.WithCaller(true))

	return zapr.NewLogger(logger)
}

// Sync flushes buffered logs.
func Sync(logger logr.Logger) {
	if logger, ok := logger.GetSink().(zapr.Underlier); ok {
		_ = logger.GetUnderlying().Core().Sync()
	}
}
