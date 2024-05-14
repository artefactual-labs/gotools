package ref_test

import (
	"testing"

	"gotest.tools/v3/assert"

	"go.artefactual.dev/tools/ref"
)

func TestNew(t *testing.T) {
	t.Parallel()

	s := "string"

	p1 := ref.New(s)
	assert.Equal(t, s, *p1)
	assert.Assert(t, &s != p1)

	p2 := ref.New(s)
	assert.Equal(t, s, *p2)
	assert.Assert(t, &s != p2)
}

func TestNewNillable(t *testing.T) {
	t.Parallel()

	s := "string"
	ptr := ref.NewNillable(s)
	assert.Equal(t, *ptr, s)
	assert.Assert(t, &s != ptr)

	s = ""
	ptr = ref.NewNillable(s)
	assert.Assert(t, ptr == nil)
}

func TestDeref(t *testing.T) {
	t.Parallel()

	t.Run("Returns the value referenced by the pointer", func(t *testing.T) {
		t.Parallel()

		s := "string"

		assert.Equal(t, s, ref.Deref(&s))
	})

	t.Run("Panics if a nil value is received", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("ref.Deref did not panic.")
			}
		}()
		ref.Deref[int](nil)
	})
}

func TestDerefZero(t *testing.T) {
	t.Parallel()

	t.Run("Returns the underlying value of the pointer", func(t *testing.T) {
		t.Parallel()

		s := "string"
		assert.Equal(t, ref.DerefZero(&s), "string")
	})

	t.Run("Returns the zero value if the pointer is nil", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, ref.DerefZero[string](nil), "")
	})
}

func TestDerefDefault(t *testing.T) {
	t.Parallel()

	t.Run("Returns the underlying value of the pointer", func(t *testing.T) {
		t.Parallel()

		s := "string"
		assert.Equal(t, ref.DerefDefault(&s, "default"), "string")
	})

	t.Run("Returns the default value if the pointer is nil", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, ref.DerefDefault[string](nil, "default"), "default")
	})
}

func TestUnsignedPtr(t *testing.T) {
	t.Parallel()

	t.Run("*int to *uint", func(t *testing.T) {
		s := int(4)
		u := uint(4)
		assert.DeepEqual(t, ref.UnsignedPtr[int, uint](&s), &u)
	})

	t.Run("*int (nil) to *uint (nil)", func(t *testing.T) {
		var s *int
		var u *uint
		assert.DeepEqual(t, ref.UnsignedPtr[int, uint](s), u)
	})

	t.Run("*int8 to *uint8", func(t *testing.T) {
		s := int8(1)
		u := uint8(1)
		assert.DeepEqual(t, ref.UnsignedPtr[int8, uint8](&s), &u)
	})

	t.Run("*int8 (nil) to *uint8 (nil)", func(t *testing.T) {
		var s *int8
		var u *uint8
		assert.DeepEqual(t, ref.UnsignedPtr[int8, uint8](s), u)
	})
}
