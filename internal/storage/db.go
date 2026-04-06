package storage

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type DBStorage struct {
	db *sql.DB
}

func NewDBStorage(dsn string) (*DBStorage, error) {
	if dsn == "" {
		return nil, nil
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("create tables: %w", err)
	}

	return &DBStorage{db: db}, nil
}

func createTables(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS gauges (
		name TEXT PRIMARY KEY,
		value DOUBLE PRECISION NOT NULL,
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS counters (
		name TEXT PRIMARY KEY,
		value BIGINT NOT NULL,
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
	`

	_, err := db.Exec(query)
	return err
}

func (s *DBStorage) UpdateGauge(name string, value float64) error {
	query := `
	INSERT INTO gauges (name, value, updated_at)
	VALUES ($1, $2, NOW())
	ON CONFLICT (name) DO UPDATE
	SET value = $2, updated_at = NOW()
	`

	_, err := s.db.Exec(query, name, value)
	return err
}

func (s *DBStorage) GetGauge(name string) (float64, bool, error) {
	var value float64
	err := s.db.QueryRow("SELECT value FROM gauges WHERE name = $1", name).Scan(&value)
	if err == sql.ErrNoRows {
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
	SET value = counters.value + $2, updated_at = NOW()
	RETURNING value
	`

	var value int64
	err := s.db.QueryRow(query, name, delta).Scan(&value)
	return value, err
}

func (s *DBStorage) GetCounter(name string) (int64, bool, error) {
	var value int64
	err := s.db.QueryRow("SELECT value FROM counters WHERE name = $1", name).Scan(&value)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return value, true, nil
}

func (s *DBStorage) GetAll() (map[string]float64, map[string]int64, error) {
	gauges := make(map[string]float64)
	rows, err := s.db.Query("SELECT name, value FROM gauges")
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value float64
		if err := rows.Scan(&name, &value); err != nil {
			return nil, nil, err
		}
		gauges[name] = value
	}

	counters := make(map[string]int64)
	rows, err = s.db.Query("SELECT name, value FROM counters")
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value int64
		if err := rows.Scan(&name, &value); err != nil {
			return nil, nil, err
		}
		counters[name] = value
	}

	return gauges, counters, nil
}

func (s *DBStorage) Ping() error {
	return s.db.Ping()
}

func (s *DBStorage) Close() error {
	return s.db.Close()
}
