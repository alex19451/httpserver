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

func (s *DBStorage) GetAll() (map[string]float64, map[string]int64, error) {
	var gauges map[string]float64
	var counters map[string]int64

	err := retry.DoWithRetry(func() error {
		g := make(map[string]float64)
		c := make(map[string]int64)

		rows, err := s.db.Query("SELECT name, value FROM gauges")
		if err != nil {
			return err
		}
		for rows.Next() {
			var name string
			var value float64
			if err := rows.Scan(&name, &value); err != nil {
				rows.Close()
				return err
			}
			g[name] = value
		}
		rows.Close()

		rows, err = s.db.Query("SELECT name, value FROM counters")
		if err != nil {
			return err
		}
		for rows.Next() {
			var name string
			var value int64
			if err := rows.Scan(&name, &value); err != nil {
				rows.Close()
				return err
			}
			c[name] = value
		}
		rows.Close()

		gauges = g
		counters = c
		return nil
	})

	return gauges, counters, err
}

func (s *DBStorage) BatchUpdate(metrics []models.Metrics) error {
	return retry.DoWithRetry(func() error {
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		for _, metric := range metrics {
			if metric.MType == "gauge" {
				if metric.Value == nil {
					return fmt.Errorf("value required for gauge %s", metric.ID)
				}
				query := `
					INSERT INTO gauges (name, value, updated_at)
					VALUES ($1, $2, NOW())
					ON CONFLICT (name) DO UPDATE
					SET value = EXCLUDED.value, updated_at = NOW()
				`
				if _, err := tx.Exec(query, metric.ID, *metric.Value); err != nil {
					return err
				}
			} else if metric.MType == "counter" {
				if metric.Delta == nil {
					return fmt.Errorf("delta required for counter %s", metric.ID)
				}
				query := `
					INSERT INTO counters (name, value, updated_at)
					VALUES ($1, $2, NOW())
					ON CONFLICT (name) DO UPDATE
					SET value = counters.value + EXCLUDED.value, updated_at = NOW()
				`
				if _, err := tx.Exec(query, metric.ID, *metric.Delta); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("invalid type %s", metric.MType)
			}
		}

		return tx.Commit()
	})
}

func (s *DBStorage) Ping() error {
	return retry.DoWithRetry(func() error {
		return s.db.Ping()
	})
}

func (s *DBStorage) Close() error {
	return s.db.Close()
}
