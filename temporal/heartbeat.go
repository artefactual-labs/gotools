package temporal

import (
	"context"
	"time"

	temporalsdk_activity "go.temporal.io/sdk/activity"
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
	heartbeatTimeout := temporalsdk_activity.GetInfo(ctx).HeartbeatTimeout
	if heartbeatTimeout == 0 {
		return nil
	}

	// We want to heartbeat three times as often as timeout (this is throttled anyways).
	heartbeatInterval := heartbeatTimeout / 3
	// We don't want to heartbeat in intervals higher than 10 seconds.
	maxHeartBeatInterval := time.Second * 10
	if heartbeatInterval > maxHeartBeatInterval {
		heartbeatInterval = maxHeartBeatInterval
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
