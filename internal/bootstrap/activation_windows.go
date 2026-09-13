package bootstrap

import (
	"errors"
	"syscall"
)

func activationRetryable(err error) bool {
	return errors.Is(err, syscall.Errno(5)) || errors.Is(err, syscall.Errno(32))
}
