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
	FilePermissions fs.FileMode = 0o600
	FileExists                  = false
)

type MemStorage struct {
	gauge   map[string]float64
	counter map[string]int64
}

func NewMemStorage(c *models.Config) (*MemStorage, error) {
	logger := c.Logger

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
		logger.Error("File open error", zap.Error(err))
		return nil, fmt.Errorf("failed to create metrics db file %s", FileStoragePath)
	}

	defer func() {
		if err := file.Close(); err != nil {
			logger.Sugar().Error(err)
		}
	}()

	if RestoreMetrics && FileExists {
		var data struct {
			Gauge   *map[string]float64 `json:"gauge"`
			Counter *map[string]int64   `json:"counter"`
		}

		err := json.NewDecoder(file).Decode(&data)
		if err != nil && !errors.Is(err, io.EOF) {
			logger.Sugar().Error(`File decode error`, zap.Error(err))
		}

		if data.Gauge != nil {
			gauge = *data.Gauge
		}
		if data.Counter != nil {
			counter = *data.Counter
		}

		if len(gauge) > 0 || len(counter) > 0 {
			logger.Sugar().Infow(
				"MemStorage restored",
				zap.Int("Gauge", len(gauge)),
				zap.Int("Counter", len(counter)),
			)
		}
	}

	mr := &MemStorage{
		gauge:   gauge,
		counter: counter,
	}
	if StoreInterval != 0 {
		go mr.SaveMetricsTicker(c)
	}
	return mr, nil
}

func (m *MemStorage) UpdateGaugeMetric(c *models.Config, name string, value float64) (float64, error) {
	m.gauge[name] = value
	if SyncFileStore {
		err := m.SaveMetrics(c)
		if err != nil {
			return 0, fmt.Errorf("failed to save metrics to file: %w", err)
		}
	}
	return m.gauge[name], nil
}

func (m *MemStorage) UpdateCounterMetric(c *models.Config, name string, value int64) (int64, error) {
	m.counter[name] += value
	if SyncFileStore {
		err := m.SaveMetrics(c)
		if err != nil {
			return 0, fmt.Errorf("failed to save metrics to file: %w", err)
		}
	}
	return m.counter[name], nil
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

func (m *MemStorage) SaveMetrics(c *models.Config) error {
	logger := c.Logger
	logger.Sugar().Info("Saving metrics to file db.")

	_, err := os.Stat(FileStoragePath)
	if err != nil {
		zap.L().Warn("File doesn't exist")
	}

	file, err := os.OpenFile(FileStoragePath, os.O_RDWR|os.O_CREATE, FilePermissions)
	if err != nil {
		logger.Sugar().Error(zap.Error(err))
		return fmt.Errorf("failed to open file %s: %w", FileStoragePath, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Sugar().Error(err)
		}
	}()

	data := make(map[string]any)
	data["gauge"] = m.gauge
	data["counter"] = m.counter

	if err := json.NewEncoder(file).Encode(data); err != nil {
		logger.Sugar().Error("File encode error", zap.Error(err))
		return fmt.Errorf("failed to json encode data: %w", err)
	}

	zap.L().Debug(`Metrics saved to file`)
	return nil
}

func (m *MemStorage) SaveMetricsTicker(c *models.Config) {
	if SyncFileStore {
		return
	}

	logger := c.Logger
	logger.Sugar().Infow(
		`MemStorage's ticker started`,
		zap.Int64(`StoreInterval`, StoreInterval),
	)

	saveTicker := time.NewTicker(time.Duration(StoreInterval) * time.Second)

	go func() {
		for range saveTicker.C {
			if err := m.SaveMetrics(c); err != nil {
				logger.Sugar().Error("failed to save metrics to file using ticker", zap.Error(err))
				return
			}
		}
	}()
}
