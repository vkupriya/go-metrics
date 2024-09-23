// Package models - defines format of key data structures for metric server.
package models

import (
	"go.uber.org/zap"
)

type Config struct {
	Logger          *zap.Logger
	HashKey         string
	Address         string
	FileStoragePath string
	PostgresDSN     string
	CryptoKey       []byte
	SecretKey       []byte
	StoreInterval   int64
	RestoreMetrics  bool
	ContextTimeout  int64
}

type Metrics []Metric

type Metric struct {
	Delta *int64   `json:"delta,omitempty"` // value of counter metric
	Value *float64 `json:"value,omitempty"` // value of gauge metric
	ID    string   `json:"id"`              // metric name
	MType string   `json:"type"`            // metric type: counter or gauge
}

type CounterModel struct {
	Name  string
	Value int64
}

type GaugeModel struct {
	Name  string
	Value float64
}
