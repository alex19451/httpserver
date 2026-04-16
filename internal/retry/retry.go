package retry

import (
	"errors"
	"net"
	"syscall"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
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

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.ConnectionException,
			pgerrcode.ConnectionDoesNotExist,
			pgerrcode.ConnectionFailure,
			pgerrcode.SQLClientUnableToEstablishSQLConnection,
			pgerrcode.SQLServerRejectedEstablishmentOfSQLConnection,
			pgerrcode.TransactionResolutionUnknown,
			pgerrcode.ProtocolViolation:
			return true
		}
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

	var errs []error

	for i, backoff := range backoffs {
		err := fn()
		if err == nil {
			return nil
		}

		errs = append(errs, err)

		if !IsRetriable(err) {
			return errors.Join(errs...)
		}

		if i < len(backoffs)-1 {
			time.Sleep(backoff)
		}
	}

	errs = append(errs, ErrMaxRetriesExceeded)
	return errors.Join(errs...)
}
