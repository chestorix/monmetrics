package utils

import (
	"errors"
	"net"
	"time"
)

var (
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
)

func Retry(attempts int, delays []time.Duration, fn func() error) error {
	var err error

	for i := 0; i < attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}

		if i == attempts-1 {
			break
		}

		if i < len(delays) {
			time.Sleep(delays[i])
		} else {
			time.Sleep(time.Second)
		}
	}

	return ErrMaxRetriesExceeded
}

func IsNetworkError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
}
