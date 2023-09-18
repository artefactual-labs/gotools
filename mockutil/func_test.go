package mockutil_test

import (
	"fmt"
	"testing"

	"gotest.tools/v3/assert"

	"go.artefactual.dev/tools/mockutil"
)

type thing struct {
	age int
}

func TestFunc(t *testing.T) {
	t.Parallel()

	t.Run("Matches expected behavior", func(t *testing.T) {
		t.Parallel()

		// Given this function that is being passed to a mock in some code.
		fn := func(t *thing) {
			t.age = 10
		}

		// And a matcher that expects age to be updated to 10.
		m := mockutil.Func(
			"description",
			func(fn func(*thing)) error {
				t := &thing{}

				fn(t)

				if t.age != 10 {
					return fmt.Errorf("got %d; want %d", t.age, 10)
				}

				return nil
			},
		)

		// Then the matcher should report a match.
		assert.Equal(t, m.Matches(fn), true)
		assert.Equal(t, m.String(), "description")
		assert.Equal(t, m.Got(fn), "")
	})

	t.Run("Reports unexpected behavior", func(t *testing.T) {
		t.Parallel()

		// Given this function that is being passed to a mock in some code.
		fn := func(t *thing) {
			t.age = 10
		}

		// And a matcher that expects age to be updated to 1000.
		m := mockutil.Func(
			"description",
			func(fn func(*thing)) error {
				t := &thing{}

				fn(t)

				if t.age != 1000 {
					return fmt.Errorf("got %d; want %d", t.age, 1000)
				}

				return nil
			},
		)

		// Then the matcher should report that it didn't match.
		assert.Equal(t, m.Matches(fn), false)
		assert.Equal(t, m.String(), "description")
		assert.Equal(t, m.Got(fn), "got 10; want 1000")
	})
}
