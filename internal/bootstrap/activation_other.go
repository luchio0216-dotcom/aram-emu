//go:build !windows

package bootstrap

func activationRetryable(error) bool { return false }
