package mockutil

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type eqMatcher struct {
	want interface{}
	opts []cmp.Option
}

// Eq is simiar to gomock.Eq but uses go-cmp, accepts options and returns a
// human-readable report of the differences between the compared values.
func Eq(want interface{}, opts ...cmp.Option) eqMatcher {
	return eqMatcher{
		want: want,
		opts: opts,
	}
}

func (eq eqMatcher) Matches(got interface{}) bool {
	return cmp.Equal(got, eq.want, eq.opts...)
}

func (eq eqMatcher) Got(got interface{}) string {
	diff := strings.TrimSpace(cmp.Diff(got, eq.want, eq.opts...))
	return fmt.Sprintf("%v (%T)\nDiff (-got +want):\n%s", got, got, diff)
}

func (eq eqMatcher) String() string {
	return fmt.Sprintf("%v (%T)\n", eq.want, eq.want)
}

// EquateNearlySameTime returns a cmp.Comparer option that determines two
// non-zero [time.Time] values to be equal if they are within one second of one
// another. It uses [cmpopts.EquateApproxtime].
func EquateNearlySameTime() cmp.Option {
	return cmpopts.EquateApproxTime(time.Second)
}
