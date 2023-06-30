package log

import "go.uber.org/zap/zapcore"

type options struct {
	name  string
	level int
	debug bool
	clock zapcore.Clock
}

type option interface {
	apply(*options)
}

type nameOption string

func (o nameOption) apply(opts *options) {
	opts.name = string(o)
}

// WithName defines the name of the logger.
func WithName(name string) option {
	return nameOption(name)
}

type levelOption int

func (o levelOption) apply(opts *options) {
	opts.level = int(o)
}

// WithLevel defines the verbosity of the logger (0 is the least verbose).
func WithLevel(level int) option {
	return levelOption(level)
}

// WithVerbosity defines the V-level to 1.
func WithVerbosity() option {
	return levelOption(1)
}

type debugOption bool

func (o debugOption) apply(opts *options) {
	opts.debug = bool(o)
}

// WithDebug configures the logger for development environments.
func WithDebug(enabled bool) option {
	return debugOption(enabled)
}

type clockOption struct {
	clock zapcore.Clock
}

func (o clockOption) apply(opts *options) {
	opts.clock = o.clock
}

// WithClock configures the clock used by the logger to determine the current
// time for logged entries. Defaults to the system clock with time.Now.
func WithClock(clock zapcore.Clock) option {
	return clockOption{clock}
}
