package log_test

import (
	"os"
	"time"

	"go.artefactual.dev/tools/log"
)

func ExampleNew() {
	logger := log.New(os.Stderr)
	defer log.Sync(logger)

	logger.Info("Test", "count", 1)
}

func ExampleWithClock() {
	logger := log.New(os.Stdout,
		log.WithName("example"),
		log.WithLevel(2),
		log.WithClock(&fakeClock{}),
	)
	defer log.Sync(logger)

	logger.Info("Test.", "key", "val")
	logger.V(4).Info("This should be ignored as per the level configured.")

	// output: {"level":"0","ts":626572800,"logger":"example","caller":"log/log_example_test.go:25","msg":"Test.","key":"val"}
}

type fakeClock struct{}

func (c *fakeClock) Now() time.Time {
	return time.Unix(626572800, 0)
}

// NewTicker returns a time.Ticker that ticks at the specified frequency.
func (c *fakeClock) NewTicker(d time.Duration) *time.Ticker {
	return nil
}
