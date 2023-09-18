package mockutil_test

import (
	"context"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"go.artefactual.dev/tools/mockutil"
)

func TestContext(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		ctx     func(t *testing.T) any
		matches bool
	}{
		"Matches background context": {
			ctx: func(t *testing.T) any {
				return context.Background()
			},
			matches: true,
		},
		"Matches TODO context": {
			ctx: func(t *testing.T) any {
				return context.TODO()
			},
			matches: true,
		},
		"Matches timed context": {
			ctx: func(t *testing.T) any {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				t.Cleanup(func() { cancel() })
				return ctx
			},
			matches: true,
		},
		"Matches valued context": {
			ctx: func(t *testing.T) any {
				type ctxKey string
				ctx := context.WithValue(context.Background(), ctxKey("k"), "v")
				return ctx
			},
			matches: true,
		},
		"Rejects other values": {
			ctx: func(t *testing.T) any {
				return struct{}{}
			},
			matches: false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			m := mockutil.Context()
			ctx := tc.ctx(t)

			assert.Equal(t, m.Matches(ctx), tc.matches)
		})
	}
}
