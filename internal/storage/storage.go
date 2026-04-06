package storage

type Storage struct {
	inMemory *InMemoryStorage
	db       *DBStorage
}

func New() *Storage {
	return &Storage{
		inMemory: NewInMemory(),
		db:       nil,
	}
}

func NewWithFile(filePath string) *Storage {
	return &Storage{
		inMemory: NewInMemoryWithFile(filePath),
		db:       nil,
	}
}

func NewWithDB(dsn string) (*Storage, error) {
	db, err := NewDBStorage(dsn)
	if err != nil {
		return nil, err
	}

	return &Storage{
		inMemory: nil,
		db:       db,
	}, nil
}

func (s *Storage) UpdateGauge(name string, value float64) error {
	if s.db != nil {
		return s.db.UpdateGauge(name, value)
	}
	s.inMemory.UpdateGauge(name, value)
	return nil
}

func (s *Storage) GetGauge(name string) (float64, bool, error) {
	if s.db != nil {
		return s.db.GetGauge(name)
	}
	val, ok := s.inMemory.GetGauge(name)
	return val, ok, nil
}

func (s *Storage) UpdateCounter(name string, delta int64) (int64, error) {
	if s.db != nil {
		return s.db.UpdateCounter(name, delta)
	}
	return s.inMemory.UpdateCounter(name, delta), nil
}

func (s *Storage) GetCounter(name string) (int64, bool, error) {
	if s.db != nil {
		return s.db.GetCounter(name)
	}
	val, ok := s.inMemory.GetCounter(name)
	return val, ok, nil
}

func (s *Storage) GetAll() (map[string]float64, map[string]int64, error) {
	if s.db != nil {
		return s.db.GetAll()
	}
	gauges, counters := s.inMemory.GetAll()
	return gauges, counters, nil
}

func (s *Storage) SaveToFile() error {
	if s.inMemory != nil {
		return s.inMemory.SaveToFile()
	}
	return nil
}

func (s *Storage) LoadFromFile() error {
	if s.inMemory != nil {
		return s.inMemory.LoadFromFile()
	}
	return nil
}

func (s *Storage) Ping() error {
	if s.db != nil {
		return s.db.Ping()
	}
	return nil
}

func (s *Storage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
