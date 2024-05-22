package storage

import (
	"fmt"
)

type MemStorage struct {
	gauge   map[string]float64
	counter map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
	}
}

func (m *MemStorage) UpdateGaugeMetric(name string, value float64) float64 {
	m.gauge[name] = value
	return m.gauge[name]
}

func (m *MemStorage) UpdateCounterMetric(name string, value int64) int64 {
	m.counter[name] += value
	fmt.Println("counter: ", name, m.counter[name])
	return m.counter[name]
}

func (m *MemStorage) GetCounterMetric(name string) (int64, error) {
	v, ok := m.counter[name]
	if ok {
		return v, nil
	}
	return v, fmt.Errorf("unknown metric %s ", name)
}

func (m *MemStorage) GetGaugeMetric(name string) (float64, error) {
	v, ok := m.gauge[name]
	if ok {
		return v, nil
	}
	return v, fmt.Errorf("unknown metric %s ", name)
}

func (m *MemStorage) GetAllValues() (map[string]float64, map[string]int64) {
	return m.gauge, m.counter
}
