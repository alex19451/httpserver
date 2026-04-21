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
	backoffs := []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second, 10 * time.Second}

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
			ticker := time.NewTicker(backoff)
			<-ticker.C
			ticker.Stop()
		}
	}

	return fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
}
