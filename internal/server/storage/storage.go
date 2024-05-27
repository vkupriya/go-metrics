package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"

	"github.com/vkupriya/go-metrics/internal/server/models"
	"go.uber.org/zap"
)

var (
	StoreInterval   int64
	FileStoragePath string
	RestoreMetrics              = false
	SyncFileStore               = false
	FilePermissions fs.FileMode = 0600
	FileExists                  = false
)

type MemStorage struct {
	gauge   map[string]float64
	counter map[string]int64
}

func NewMemStorage(c *models.Config) (*MemStorage, error) {
	sugar := zap.L().Sugar()

	gauge := make(map[string]float64)
	counter := make(map[string]int64)

	StoreInterval = c.StoreInterval
	FileStoragePath = c.FileStoragePath
	RestoreMetrics = c.RestoreMetrics

	if c.StoreInterval == 0 {
		SyncFileStore = true
	}

	// Checking if file exists
	_, err := os.Stat(FileStoragePath)
	if err == nil {
		FileExists = true
	}

	file, err := os.OpenFile(FileStoragePath, os.O_RDWR|os.O_CREATE, FilePermissions)

	if err != nil {
		sugar.Error("File open error", zap.Error(err))
		return nil, fmt.Errorf("failed to create metrics db file %s", FileStoragePath)
	}

	defer func() {
		if err := file.Close(); err != nil {
			sugar.Error(err)
		}
	}()

	if RestoreMetrics && FileExists {
		var data struct {
			Gauge   *map[string]float64
			Counter *map[string]int64
		}

		err := json.NewDecoder(file).Decode(&data)
		if err != nil && !errors.Is(err, io.EOF) {
			zap.L().Error(`File decode error`, zap.Error(err))
		}

		if data.Gauge != nil {
			gauge = *data.Gauge
		}
		if data.Counter != nil {
			counter = *data.Counter
		}

		if len(gauge) > 0 || len(counter) > 0 {
			zap.L().Info(
				"MemStorage restored",
				zap.Int("Gauge", len(gauge)),
				zap.Int("Counter", len(counter)),
			)
		}
	}

	return &MemStorage{
		gauge:   gauge,
		counter: counter,
	}, nil
}

func (m *MemStorage) UpdateGaugeMetric(name string, value float64) float64 {
	sugar := zap.L().Sugar()

	m.gauge[name] = value
	if SyncFileStore {
		err := m.SaveMetricsToFile()
		if err != nil {
			sugar.Error("failed to save metrics to file", zap.Error(err))
		}
	}
	return m.gauge[name]
}

func (m *MemStorage) UpdateCounterMetric(name string, value int64) int64 {
	sugar := zap.L().Sugar()

	m.counter[name] += value
	if SyncFileStore {
		err := m.SaveMetricsToFile()
		if err != nil {
			sugar.Error("failed to save metrics to file", zap.Error(err))
		}
	}
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

func (m *MemStorage) SaveMetricsToFile() error {
	sugar := zap.L().Sugar()
	sugar.Info("Saving metrics to file db.")

	_, err := os.Stat(FileStoragePath)
	if err != nil {
		zap.L().Warn("File doesn't exist")
	}

	file, err := os.OpenFile(FileStoragePath, os.O_RDWR|os.O_CREATE, FilePermissions)
	if err != nil {
		sugar.Error(zap.Error(err))
		return fmt.Errorf("failed to open file %s: %w", FileStoragePath, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			sugar.Error(err)
		}
	}()

	data := make(map[string]any)
	data["gauge"] = m.gauge
	data["counter"] = m.counter

	if err := json.NewEncoder(file).Encode(data); err != nil {
		sugar.Error("File encode error", zap.Error(err))
		return fmt.Errorf("failed to json encode data: %w", err)
	}

	zap.L().Debug(`Metrics saved to file`)
	return nil
}

func (m *MemStorage) SaveToFileTicker() {
	if SyncFileStore {
		return
	}

	zap.L().Info(
		`MemStorage's tickers started`,
		zap.Int64(`StoreInterval`, StoreInterval),
	)

	saveTicker := time.NewTicker(time.Duration(StoreInterval) * time.Second)

	go func() {
		for range saveTicker.C {
			if err := m.SaveMetricsToFile(); err != nil {
				zap.L().Error("failed to save metrics to file using ticker", zap.Error(err))
				return
			}
		}
	}()
}
