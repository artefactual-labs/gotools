package temporal

import (
	"context"
	"time"

	"go.temporal.io/sdk/activity"
)

type AutoHeartbeat struct {
	ticker *time.Ticker
	done   chan struct{}
}

// StartAutoHeartbeat is an auto-heartbeat helper that chooses the heartbeat
// frequency based on the activity hearbeat timeout configuration.
//
// Temporal is planning to provide auto-heartbeating capabilities in the future,
// see https://github.com/temporalio/features/issues/229 for more.
func StartAutoHeartbeat(ctx context.Context) *AutoHeartbeat {
	heartbeatTimeout := activity.GetInfo(ctx).HeartbeatTimeout
	if heartbeatTimeout == 0 {
		return nil
	}

	// We'll heartbeat twice as often as timeout (this is throttled anyways).
	ticker := time.NewTicker(heartbeatTimeout / 2)
	done := make(chan struct{})
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				activity.RecordHeartbeat(ctx)
			case <-done:
				return
			}
		}
	}()

	return &AutoHeartbeat{ticker, done}
}

func (m *AutoHeartbeat) Stop() {
	if m != nil {
		close(m.done)
	}
}
