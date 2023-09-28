package temporal_test

import (
	"context"
	"testing"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
	"gotest.tools/v3/assert"

	"go.artefactual.dev/tools/temporal"
)

func TestAutoHeartbeat(t *testing.T) {
	t.Parallel()

	// Heartbeat listener.
	received := make(chan struct{}, 10)
	var calls []string
	l := func(info *activity.Info, details converter.EncodedValues) {
		calls = append(calls, info.ActivityID)
		received <- struct{}{}
	}

	a := func(ctx context.Context) error {
		h := temporal.StartAutoHeartbeat(ctx)
		defer h.Stop()

		// Wait until we receive at least one heartbeat.
		select {
		case <-received:
		case <-time.After(time.Second):
		}

		return nil
	}

	w := func(ctx workflow.Context) error {
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Second,
			HeartbeatTimeout:    time.Millisecond,
		})
		return workflow.ExecuteActivity(ctx, a).Get(ctx, nil)
	}

	ts := &testsuite.WorkflowTestSuite{}
	env := ts.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(w)
	env.RegisterActivity(a)
	env.SetOnActivityHeartbeatListener(l)
	env.ExecuteWorkflow(w)

	assert.Equal(t, env.IsWorkflowCompleted(), true)
	assert.NilError(t, env.GetWorkflowError())
	assert.Assert(t, len(calls) > 0)
}
