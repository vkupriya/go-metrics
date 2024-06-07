package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"

	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/vkupriya/go-metrics/internal/server/models"
	"go.uber.org/zap"
)

type MemStorage struct {
	gauge   map[string]float64
	counter map[string]int64
}

type FileStorage struct {
	*MemStorage
}

type PostgresDB struct {
	*sql.DB
}

func NewPostgresDB(c *models.Config) (*PostgresDB, error) {
	db, err := sql.Open("pgx", c.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to create PG DB connection pool: %w", err)
	}

	createSchema := []string{
		`CREATE TABLE IF NOT EXISTS gauge(
			id INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
			name varchar(40) UNIQUE NOT NULL,
			value bigint
		)`,

		`CREATE TABLE IF NOT EXISTS counter(
			id INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
			name varchar(40) UNIQUE NOT NULL,
			value bigint
		)`,
	}
	ctx := context.Background()
	for _, table := range createSchema {
		if _, err := db.ExecContext(ctx, table); err != nil {
			return nil, fmt.Errorf("failed to execute statement `%s`: %w", table, err)
		}
	}

	return &PostgresDB{
		db,
	}, nil
}

func NewMemStorage(c *models.Config) (*MemStorage, error) {
	return &MemStorage{
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
	}, nil
}

func NewFileStorage(c *models.Config) (*FileStorage, error) {
	logger := c.Logger

	var FilePermissions fs.FileMode = 0o600
	var FileExists = false

	gauge := make(map[string]float64)
	counter := make(map[string]int64)

	// Checking if file exists
	_, err := os.Stat(c.FileStoragePath)
	if err == nil {
		FileExists = true
	}

	file, err := os.OpenFile(c.FileStoragePath, os.O_RDWR|os.O_CREATE, FilePermissions)

	if err != nil {
		logger.Error("File open error", zap.Error(err))
		return nil, fmt.Errorf("failed to create metrics db file %s", c.FileStoragePath)
	}

	defer func() {
		if err := file.Close(); err != nil {
			logger.Sugar().Error(err)
		}
	}()

	if c.RestoreMetrics && FileExists {
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

	f := &FileStorage{
		&MemStorage{
			gauge:   gauge,
			counter: counter,
		}}

	if c.StoreInterval != 0 {
		go f.SaveMetricsTicker(c)
	}
	return f, nil
}

func (m *MemStorage) UpdateGaugeMetric(c *models.Config, name string, value float64) (float64, error) {
	m.gauge[name] = value
	return m.gauge[name], nil
}

func (m *MemStorage) UpdateCounterMetric(c *models.Config, name string, value int64) (int64, error) {
	m.counter[name] += value
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

func (f *FileStorage) UpdateGaugeMetric(c *models.Config, name string, value float64) (float64, error) {
	f.gauge[name] = value
	if c.StoreInterval == 0 {
		err := f.SaveMetrics(c)
		if err != nil {
			return 0, fmt.Errorf("failed to save metrics to file: %w", err)
		}
	}
	return f.gauge[name], nil
}

func (f *FileStorage) UpdateCounterMetric(c *models.Config, name string, value int64) (int64, error) {
	f.counter[name] += value
	if c.StoreInterval == 0 {
		err := f.SaveMetrics(c)
		if err != nil {
			return 0, fmt.Errorf("failed to save metrics to file: %w", err)
		}
	}
	return f.counter[name], nil
}

func (f *FileStorage) GetCounterMetric(name string) (int64, error) {
	v, ok := f.counter[name]
	if ok {
		return v, nil
	}
	return v, fmt.Errorf("unknown counter metric %s ", name)
}

func (f *FileStorage) GetGaugeMetric(name string) (float64, error) {
	v, ok := f.gauge[name]
	if ok {
		return v, nil
	}
	return v, fmt.Errorf("unknown gauge metric %s ", name)
}

func (f *FileStorage) GetAllValues() (map[string]float64, map[string]int64) {
	return f.gauge, f.counter
}

func (f *FileStorage) SaveMetrics(c *models.Config) error {
	logger := c.Logger
	logger.Sugar().Info("Saving metrics to file db.")
	var FilePermissions fs.FileMode = 0o600

	_, err := os.Stat(c.FileStoragePath)
	if err != nil {
		zap.L().Warn("File doesn't exist")
	}

	file, err := os.OpenFile(c.FileStoragePath, os.O_RDWR|os.O_CREATE, FilePermissions)
	if err != nil {
		logger.Sugar().Error(zap.Error(err))
		return fmt.Errorf("failed to open file %s: %w", c.FileStoragePath, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Sugar().Error(err)
		}
	}()

	data := make(map[string]any)
	data["gauge"] = f.gauge
	data["counter"] = f.counter

	if err := json.NewEncoder(file).Encode(data); err != nil {
		logger.Sugar().Error("File encode error", zap.Error(err))
		return fmt.Errorf("failed to json encode data: %w", err)
	}

	zap.L().Debug(`Metrics saved to file`)
	return nil
}

func (f *FileStorage) SaveMetricsTicker(c *models.Config) {
	if c.StoreInterval == 0 {
		return
	}

	logger := c.Logger
	logger.Sugar().Infow(
		`MemStorage's ticker started`,
		zap.Int64(`StoreInterval`, c.StoreInterval),
	)

	saveTicker := time.NewTicker(time.Duration(c.StoreInterval) * time.Second)

	go func() {
		for range saveTicker.C {
			if err := f.SaveMetrics(c); err != nil {
				logger.Sugar().Error("failed to save metrics to file using ticker", zap.Error(err))
				return
			}
		}
	}()
}

func (m *PostgresDB) UpdateGaugeMetric(c *models.Config, name string, value float64) (float64, error) {
	f := 000.1
	return f, nil
}

func (m *PostgresDB) UpdateCounterMetric(c *models.Config, name string, value int64) (int64, error) {
	var i int64 = 1
	return i, nil
}

func (m *PostgresDB) GetCounterMetric(name string) (int64, error) {
	var i int64 = 1
	return i, nil
}

func (m *PostgresDB) GetGaugeMetric(name string) (float64, error) {
	f := 000.1
	return f, nil
}

func (m *PostgresDB) GetAllValues() (map[string]float64, map[string]int64) {
	f := make(map[string]float64)
	i := make(map[string]int64)

	return f, i
}
