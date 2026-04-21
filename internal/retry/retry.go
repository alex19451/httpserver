package retry

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"syscall"
	"time"
)

var (
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
)

type RetriableError struct {
	Err error
}

func (e *RetriableError) Error() string {
	return e.Err.Error()
}

func (e *RetriableError) Unwrap() error {
	return e.Err
}

func IsRetriable(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, sql.ErrNoRows) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	if errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ETIMEDOUT) {
		return true
	}

	return false
}

func DoWithRetry(fn func() error) error {
	backoffs := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

	var lastErr error

	for i, backoff := range backoffs {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		if errors.Is(lastErr, sql.ErrNoRows) {
			return lastErr
		}

		if !IsRetriable(lastErr) {
			return lastErr
		}

		if i < len(backoffs)-1 {
			time.Sleep(backoff)
		}
	}

	return fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
}
