package temporal

import (
	"go.temporal.io/sdk/temporal"
)

func NonRetryableError(err error) error {
	return temporal.NewNonRetryableApplicationError(err.Error(), "", nil, nil)
}
