package mockutil

import (
	"context"
	"reflect"

	"go.uber.org/mock/gomock"
)

var ctxType = reflect.TypeOf((*context.Context)(nil)).Elem()

func Context() gomock.Matcher {
	return gomock.AssignableToTypeOf(ctxType)
}
