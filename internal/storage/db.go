package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/alex19451/httpserver/internal/models"
	"github.com/alex19451/httpserver/internal/retry"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type DBStorage struct {
	db *sql.DB
}

func NewDBStorage(dsn string) (*DBStorage, error) {
	var db *sql.DB
	var err error

	err = retry.DoWithRetry(func() error {
		db, err = sql.Open("pgx", dsn)
		if err != nil {
			return err
		}

		if err := db.Ping(); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("create tables: %w", err)
	}

	return &DBStorage{db: db}, nil
}

func createTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS gauges (
			name VARCHAR(255) PRIMARY KEY,
			value DOUBLE PRECISION NOT NULL,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS counters (
			name VARCHAR(255) PRIMARY KEY,
			value BIGINT NOT NULL,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
	}

	var errs []error
	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (s *DBStorage) UpdateGauge(name string, value float64) error {
	query := `
		INSERT INTO gauges (name, value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (name) DO UPDATE
		SET value = EXCLUDED.value, updated_at = NOW()
	`

	return retry.DoWithRetry(func() error {
		_, err := s.db.Exec(query, name, value)
		return err
	})
}

func (s *DBStorage) GetGauge(name string) (float64, bool, error) {
	var value float64

	err := retry.DoWithRetry(func() error {
		return s.db.QueryRow("SELECT value FROM gauges WHERE name = $1", name).Scan(&value)
	})

	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return value, true, nil
}

func (s *DBStorage) UpdateCounter(name string, delta int64) (int64, error) {
	query := `
		INSERT INTO counters (name, value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (name) DO UPDATE
		SET value = counters.value + EXCLUDED.value, updated_at = NOW()
		RETURNING value
	`

	var value int64
	err := retry.DoWithRetry(func() error {
		return s.db.QueryRow(query, name, delta).Scan(&value)
	})

	return value, err
}

func (s *DBStorage) GetCounter(name string) (int64, bool, error) {
	var value int64

	err := retry.DoWithRetry(func() error {
		return s.db.QueryRow("SELECT value FROM counters WHERE name = $1", name).Scan(&value)
	})

	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return value, true, nil
}

func (s *DBStorage) getAllGauges() (map[string]float64, error) {
	rows, err := s.db.Query("SELECT name, value FROM gauges")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]float64)
	for rows.Next() {
		var name string
		var value float64
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		result[name] = value
	}
	return result, rows.Err()
}

func (s *DBStorage) getAllCounters() (map[string]int64, error) {
	rows, err := s.db.Query("SELECT name, value FROM counters")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var name string
		var value int64
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		result[name] = value
	}
	return result, rows.Err()
}

func (s *DBStorage) GetAll() (map[string]float64, map[string]int64, error) {
	var gauges map[string]float64
	var counters map[string]int64

	err := retry.DoWithRetry(func() error {
		var err1, err2 error
		gauges, err1 = s.getAllGauges()
		counters, err2 = s.getAllCounters()

		if err1 != nil {
			return err1
		}
		if err2 != nil {
			return err2
		}
		return nil
	})

	return gauges, counters, err
}

func (s *DBStorage) BatchUpdate(metrics []models.Metrics) error {
	for _, metric := range metrics {
		if metric.MType == "gauge" {
			if metric.Value == nil {
				return fmt.Errorf("value required for gauge %s", metric.ID)
			}
			if err := s.UpdateGauge(metric.ID, *metric.Value); err != nil {
				return err
			}
		} else if metric.MType == "counter" {
			if metric.Delta == nil {
				return fmt.Errorf("delta required for counter %s", metric.ID)
			}
			if _, err := s.UpdateCounter(metric.ID, *metric.Delta); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("invalid type %s", metric.MType)
		}
	}
	return nil
}

func (s *DBStorage) Ping() error {
	return retry.DoWithRetry(func() error {
		return s.db.Ping()
	})
}

func (s *DBStorage) Close() error {
	return s.db.Close()
}
