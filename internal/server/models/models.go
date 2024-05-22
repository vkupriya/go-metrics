package models

type Metrics struct {
	Delta *int64   `json:"delta,omitempty"` // value of counter metric
	Value *float64 `json:"value,omitempty"` // value of gauge metric
	ID    string   `json:"id"`              // metric name
	MType string   `json:"type"`            // metric type: counter or gauge
}
