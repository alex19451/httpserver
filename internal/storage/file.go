package storage

import (
	"encoding/json"
	"os"
)

type FileStorage struct {
	filePath string
}

func NewFileStorage(filePath string) *FileStorage {
	return &FileStorage{
		filePath: filePath,
	}
}

func (fs *FileStorage) Save(gauges map[string]float64, counters map[string]int64) error {
	data := struct {
		Gauges   map[string]float64 `json:"gauges"`
		Counters map[string]int64   `json:"counters"`
	}{
		Gauges:   gauges,
		Counters: counters,
	}

	jsonData, err := json.MarshalIndent(data, "", "   ")
	if err != nil {
		return err
	}

	return os.WriteFile(fs.filePath, jsonData, 0666)
}

func (fs *FileStorage) Load() (map[string]float64, map[string]int64, error) {
	jsonData, err := os.ReadFile(fs.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]float64), make(map[string]int64), nil
		}
		return nil, nil, err
	}

	var data struct {
		Gauges   map[string]float64 `json:"gauges"`
		Counters map[string]int64   `json:"counters"`
	}

	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, nil, err
	}

	if data.Gauges == nil {
		data.Gauges = make(map[string]float64)
	}
	if data.Counters == nil {
		data.Counters = make(map[string]int64)
	}

	return data.Gauges, data.Counters, nil
}
