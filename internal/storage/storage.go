package storage

import (
	"github.com/alex19451/httpserver/internal/models"
)

type Storage struct {
	inMemory      *InMemoryStorage
	file          *InMemoryStorage
	db            *DBStorage
	mode          string
	filePath      string
	storeInterval int
}

func New() *Storage {
	return &Storage{
		inMemory: NewInMemory(),
		mode:     "memory",
	}
}

func NewWithFile(filePath string) *Storage {
	return &Storage{
		file:     NewInMemoryWithFile(filePath),
		mode:     "file",
		filePath: filePath,
	}
}

func NewWithDB(dsn string) (*Storage, error) {
	db, err := NewDBStorage(dsn)
	if err != nil {
		return nil, err
	}
	return &Storage{
		db:   db,
		mode: "db",
	}, nil
}

func (s *Storage) SetStoreInterval(interval int) {
	s.storeInterval = interval
}

func (s *Storage) shouldSaveSync() bool {
	return s.storeInterval == 0 && s.mode == "file" && s.filePath != ""
}

func (s *Storage) UpdateGauge(name string, value float64) error {
	switch s.mode {
	case "db":
		return s.db.UpdateGauge(name, value)
	case "file":
		s.file.UpdateGauge(name, value)
		if s.shouldSaveSync() {
			return s.file.SaveToFile()
		}
		return nil
	default:
		s.inMemory.UpdateGauge(name, value)
		return nil
	}
}

func (s *Storage) GetGauge(name string) (float64, bool, error) {
	switch s.mode {
	case "db":
		return s.db.GetGauge(name)
	case "file":
		val, ok := s.file.GetGauge(name)
		return val, ok, nil
	default:
		val, ok := s.inMemory.GetGauge(name)
		return val, ok, nil
	}
}

func (s *Storage) UpdateCounter(name string, delta int64) (int64, error) {
	switch s.mode {
	case "db":
		return s.db.UpdateCounter(name, delta)
	case "file":
		result := s.file.UpdateCounter(name, delta)
		if s.shouldSaveSync() {
			return result, s.file.SaveToFile()
		}
		return result, nil
	default:
		return s.inMemory.UpdateCounter(name, delta), nil
	}
}

func (s *Storage) GetCounter(name string) (int64, bool, error) {
	switch s.mode {
	case "db":
		return s.db.GetCounter(name)
	case "file":
		val, ok := s.file.GetCounter(name)
		return val, ok, nil
	default:
		val, ok := s.inMemory.GetCounter(name)
		return val, ok, nil
	}
}

func (s *Storage) GetAll() (map[string]float64, map[string]int64, error) {
	switch s.mode {
	case "db":
		return s.db.GetAll()
	case "file":
		gauges, counters := s.file.GetAll()
		return gauges, counters, nil
	default:
		gauges, counters := s.inMemory.GetAll()
		return gauges, counters, nil
	}
}

func (s *Storage) SaveToFile() error {
	if s.mode == "file" {
		return s.file.SaveToFile()
	}
	return nil
}

func (s *Storage) LoadFromFile() error {
	if s.mode == "file" {
		return s.file.LoadFromFile()
	}
	return nil
}

func (s *Storage) Ping() error {
	if s.mode == "db" {
		return s.db.Ping()
	}
	return nil
}

func (s *Storage) Close() error {
	if s.mode == "db" {
		return s.db.Close()
	}
	return nil
}

func (s *Storage) IsDB() bool {
	return s.mode == "db"
}

func (s *Storage) BatchUpdate(metrics []models.Metrics) error {
	if s.mode == "db" {
		return s.db.BatchUpdate(metrics)
	}
	for _, metric := range metrics {
		if metric.MType == "gauge" {
			if err := s.UpdateGauge(metric.ID, *metric.Value); err != nil {
				return err
			}
		} else if metric.MType == "counter" {
			if _, err := s.UpdateCounter(metric.ID, *metric.Delta); err != nil {
				return err
			}
		}
	}
	return nil
}
