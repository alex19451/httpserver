package storage

type Storage struct {
	Gauges   map[string]float64
	Counters map[string]int64
}

func New() *Storage {
	return &Storage{
		Gauges:   make(map[string]float64),
		Counters: make(map[string]int64),
	}
}
