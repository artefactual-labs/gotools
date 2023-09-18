package mockutil_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"go.artefactual.dev/tools/mockutil"
)

func TestEq(t *testing.T) {
	t.Parallel()

	type thing struct {
		A int
		B string
	}

	type timeThing struct {
		A int
		T time.Time
	}

	t.Run("Reports equal values", func(t *testing.T) {
		want := thing{A: 1}
		got := thing{A: 1}

		m := mockutil.Eq(want)
		assert.Equal(t, m.Matches(got), true)
	})

	t.Run("Reports unequal values", func(t *testing.T) {
		want := thing{A: 1}
		got := thing{A: 2}

		m := mockutil.Eq(want)
		assert.Equal(t, m.Matches(got), false)
		assert.Assert(t, cmp.Contains(m.String(), "{1 } (mockutil_test.thing)"))
		assert.Assert(t, cmp.Contains(m.Got(got), "Diff (-got +want):\nmockutil_test.thing{"))
	})

	t.Run("Honours go-cmp options", func(t *testing.T) {
		want := thing{
			A: 1,
			B: "x",
		}
		got := thing{
			A: 1,
			B: "y",
		}

		m := mockutil.Eq(
			want,
			cmpopts.IgnoreFields(thing{}, "B"), // We'll ignore "thing.B".
		)
		assert.Equal(t, m.Matches(got), true)
	})

	t.Run("Equates nearly same time", func(t *testing.T) {
		now := time.Now()
		want := timeThing{
			A: 1,
			T: now,
		}
		got := timeThing{
			A: 1,
			T: now.Add(time.Second),
		}

		m := mockutil.Eq(
			want,
			mockutil.EquateNearlySameTime(),
		)
		assert.Equal(t, m.Matches(got), true)

		got = timeThing{
			A: 1,
			T: now.Add(time.Second * 2),
		}

		m = mockutil.Eq(
			want,
			mockutil.EquateNearlySameTime(),
		)
		assert.Equal(t, m.Matches(got), false)
	})
}
