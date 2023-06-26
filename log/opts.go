package log

type options struct {
	name  string
	level int
	debug bool
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
