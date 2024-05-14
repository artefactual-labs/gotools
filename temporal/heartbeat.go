package temporal

import (
	"context"
	"time"

	temporalsdk_activity "go.temporal.io/sdk/activity"
)

// maxInterval sets the heartbeat interval cap. Changes to this should only be
// done in tests.
var maxInterval = time.Second * 10

type AutoHeartbeat struct {
	ticker *time.Ticker
	done   chan struct{}
}

// StartAutoHeartbeat starts an auto-heartbeat helper for activities.
//
// The interval is chosen to be one-third of the configured timeout to ensure
// frequent heartbeats. It is capped at a maximum of 10s to prevent the interval
// from being too large.
//
//	func Activity(ctx context.Context) error {
//		h := temporal.StartAutoHeartbeat(ctx)
//		defer h.Stop()
//
//		// ... long running code.
//
//		return nil
//	}
//
// Temporal is planning to provide auto-heartbeating capabilities in the future,
// see https://github.com/temporalio/features/issues/229 for more.
func StartAutoHeartbeat(ctx context.Context) *AutoHeartbeat {
	heartbeatTimeout := temporalsdk_activity.GetInfo(ctx).HeartbeatTimeout
	if heartbeatTimeout == 0 {
		return nil
	}

	// No risk in having a very small interval since Temporal throttles it anyways.
	heartbeatInterval := heartbeatTimeout / 3
	if heartbeatInterval > maxInterval {
		heartbeatInterval = maxInterval
	}

	ticker := time.NewTicker(heartbeatInterval)
	done := make(chan struct{})
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				temporalsdk_activity.RecordHeartbeat(ctx)
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
