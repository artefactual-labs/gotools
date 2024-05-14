// Package ref provides utility functions for handling pointers in Go.
package ref

import "golang.org/x/exp/constraints"

// New returns a pointer to x.
func New[T any](x T) *T {
	return &x
}

// NewNillable returns a pointer to x.
// It returns nil when x has the zero value of T
func NewNillable[T comparable](x T) *T {
	var z T
	if x == z {
		return nil
	}
	return &x
}

// Deref returns the value stored at the pointer x.
// It panics if x is nil.
func Deref[T any](x *T) T {
	return *x
}

// DerefZero returns the value stored at the pointer x.
// It returns the zero value of T when the pointer is nil.
func DerefZero[T any](x *T) T {
	if x == nil {
		var z T
		return z
	}
	return *x
}

// DerefDefault returns the value stored at the pointer x.
// It returns the provided default when the pointer is nil.
func DerefDefault[T any](x *T, d T) T {
	if x == nil {
		return d
	}
	return *x
}

// UnsignedPtr converts a signed integer into an unsigned integer.
func UnsignedPtr[S constraints.Signed, U constraints.Unsigned](s *S) *U {
	var ret *U
	if s != nil {
		u := U(*s)
		ret = &u
	}
	return ret
}
