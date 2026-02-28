package storage

type Storage struct {
	Gauges   map[string]float64
	Counters map[string]int64
	file     *FileStorage
}

func New() *Storage {
	return &Storage{
		Gauges:   make(map[string]float64),
		Counters: make(map[string]int64),
	}
}

func NewWithFile(filePath string) *Storage {
	return &Storage{
		Gauges:   make(map[string]float64),
		Counters: make(map[string]int64),
		file:     NewFileStorage(filePath),
	}
}

func (s *Storage) UpdateGauge(name string, value float64) {
	s.Gauges[name] = value
}

func (s *Storage) GetGauge(name string) (float64, bool) {
	val, ok := s.Gauges[name]
	return val, ok
}

func (s *Storage) UpdateCounter(name string, delta int64) int64 {
	s.Counters[name] += delta
	return s.Counters[name]
}

func (s *Storage) GetCounter(name string) (int64, bool) {
	val, ok := s.Counters[name]
	return val, ok
}

func (s *Storage) GetAll() (map[string]float64, map[string]int64) {
	gauges := make(map[string]float64, len(s.Gauges))
	for k, v := range s.Gauges {
		gauges[k] = v
	}

	counters := make(map[string]int64, len(s.Counters))
	for k, v := range s.Counters {
		counters[k] = v
	}

	return gauges, counters
}

func (s *Storage) SaveToFile() error {
	if s.file == nil {
		return nil
	}
	return s.file.Save(s.Gauges, s.Counters)
}

func (s *Storage) LoadFromFile() error {
	if s.file == nil {
		return nil
	}
	gauges, counters, err := s.file.Load()
	if err != nil {
		return err
	}

	s.Gauges = gauges
	s.Counters = counters
	return nil
}
