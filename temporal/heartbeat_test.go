package temporal_test

import (
	"context"
	"testing"
	"time"

	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_converter "go.temporal.io/sdk/converter"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"
	"gotest.tools/v3/assert"

	"go.artefactual.dev/tools/temporal"
)

func TestAutoHeartbeat(t *testing.T) {
	t.Parallel()

	// Heartbeat listener.
	received := make(chan struct{}, 10)
	var calls []string
	l := func(info *temporalsdk_activity.Info, details temporalsdk_converter.EncodedValues) {
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

	w := func(ctx temporalsdk_workflow.Context) error {
		ctx = temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
			StartToCloseTimeout: time.Second,
			HeartbeatTimeout:    time.Millisecond,
		})
		return temporalsdk_workflow.ExecuteActivity(ctx, a).Get(ctx, nil)
	}

	ts := &temporalsdk_testsuite.WorkflowTestSuite{}
	env := ts.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(w)
	env.RegisterActivity(a)
	env.SetOnActivityHeartbeatListener(l)
	env.ExecuteWorkflow(w)

	assert.Equal(t, env.IsWorkflowCompleted(), true)
	assert.NilError(t, env.GetWorkflowError())
	assert.Assert(t, len(calls) > 0)
}
