package mockutil

import (
	"context"
	"reflect"

	"go.uber.org/mock/gomock"
)

var ctxType = reflect.TypeFor[context.Context]()

func Context() gomock.Matcher {
	return gomock.AssignableToTypeOf(ctxType)
}
