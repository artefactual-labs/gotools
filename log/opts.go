package log

import (
	"fmt"

	"go.uber.org/zap/zapcore"
)

// Format determines how log records are encoded.
type Format uint8

const (
	// FormatJSON encodes each log record as JSON.
	FormatJSON Format = iota
	// FormatText encodes log records as human-readable text.
	FormatText
	// FormatAuto uses text for terminal writers and JSON otherwise.
	FormatAuto
)

type options struct {
	name     string
	level    int
	format   Format
	clock    zapcore.Clock
	addStack bool
}

func defaults() options {
	return options{
		format:   FormatJSON,
		clock:    zapcore.DefaultClock,
		addStack: true,
	}
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

type formatOption Format

func (o formatOption) apply(opts *options) {
	opts.format = Format(o)
}

// WithFormat configures the log record format. It panics if format is not one
// of the formats defined by this package.
func WithFormat(format Format) option {
	switch format {
	case FormatJSON, FormatText, FormatAuto:
		return formatOption(format)
	default:
		panic(fmt.Sprintf("log: invalid format %d", format))
	}
}

// WithDebug is an alias for WithFormat with [FormatText] when enabled and
// [FormatJSON] otherwise. It does not change the logger's verbosity.
func WithDebug(enabled bool) option {
	if enabled {
		return WithFormat(FormatText)
	}

	return WithFormat(FormatJSON)
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

type addStackOption bool

func (o addStackOption) apply(opts *options) {
	opts.addStack = bool(o)
}

// WithStacktrace configures the logger to record error stacktraces.
// It is enabled by default.
func WithStacktrace(enabled bool) option {
	return addStackOption(enabled)
}
