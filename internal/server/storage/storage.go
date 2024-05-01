package storage

type Storage interface {
	UpdateGaugeMetric(name string, value float64) float64
	UpdateCounterMetric(name string, value int64) int64
}

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
	return m.counter[name]
}
