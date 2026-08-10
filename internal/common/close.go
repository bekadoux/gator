package common

import (
	"errors"
	"fmt"
	"io"
)

// For deferring close
func CloseWithError(errp *error, c io.Closer, what string) {
	if closeErr := c.Close(); closeErr != nil {
		*errp = errors.Join(*errp, fmt.Errorf("close %s: %w", what, closeErr))
	}
}
