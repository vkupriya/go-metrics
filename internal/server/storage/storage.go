package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
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

type PostgresStorage struct {
	*sql.DB
}

func NewPostgresStorage(c *models.Config) (*PostgresStorage, error) {
	db, err := sql.Open("pgx", c.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to create PG DB connection pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	defer cancel()

	tx, err := db.BeginTx(ctx, nil)

	if err != nil {
		return nil, fmt.Errorf("failed to start a transaction: %w", err)
	}

	defer func() {
		if err := tx.Rollback(); err != nil {
			if !errors.Is(err, sql.ErrTxDone) {
				log.Printf("failed to rollback the transaction: %v", err)
			}
		}
	}()

	createSchema := []string{
		`CREATE TABLE IF NOT EXISTS gauge(
			id INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
			name VARCHAR(255) UNIQUE NOT NULL,
			value DOUBLE PRECISION
		)`,

		`CREATE TABLE IF NOT EXISTS counter(
			id INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
			name VARCHAR(255) UNIQUE NOT NULL,
			value BIGINT
		)`,
	}

	for _, table := range createSchema {
		if _, err := tx.ExecContext(ctx, table); err != nil {
			return nil, fmt.Errorf("failed to execute statement `%s`: %w", table, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit PostgresDB transaction: %w", err)
	}

	return &PostgresStorage{
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

func (m *MemStorage) GetCounterMetric(name string) (int64, bool, error) {
	v, ok := m.counter[name]
	if ok {
		return v, true, nil
	}
	return v, false, fmt.Errorf("unknown metric %s ", name)
}

func (m *MemStorage) GetGaugeMetric(name string) (float64, bool, error) {
	v, ok := m.gauge[name]
	if ok {
		return v, true, nil
	}
	return v, false, fmt.Errorf("unknown metric %s ", name)
}

func (m *MemStorage) GetAllMetrics(c *models.Config) (map[string]float64, map[string]int64, error) {
	return m.gauge, m.counter, nil
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

func (f *FileStorage) GetCounterMetric(name string) (int64, bool, error) {
	v, ok := f.counter[name]
	if ok {
		return v, true, nil
	}
	return v, false, fmt.Errorf("unknown counter metric %s ", name)
}

func (f *FileStorage) GetGaugeMetric(name string) (float64, bool, error) {
	v, ok := f.gauge[name]
	if ok {
		return v, true, nil
	}
	return v, false, fmt.Errorf("unknown gauge metric %s ", name)
}

func (f *FileStorage) GetAllMetrics(c *models.Config) (map[string]float64, map[string]int64, error) {
	return f.gauge, f.counter, nil
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

func (p *PostgresStorage) UpdateGaugeMetric(c *models.Config, name string, value float64) (float64, error) {
	db := p.DB

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, e, err := p.GetGaugeMetric(name)

	if err != nil {
		return value, err
	}

	// If metric doesn't exist then Insert, otherwise Update
	if !e {
		_, err = db.ExecContext(ctx, "INSERT INTO gauge (name, value) VALUES($1, $2)", name, value)

		if err != nil {
			return value, fmt.Errorf("failed to insert gauge metric into Postgres DB: %w", err)
		}
	} else {
		_, err = db.ExecContext(ctx, "UPDATE gauge SET value = $1 WHERE name = $2", value, name)
		if err != nil {
			return value, fmt.Errorf("failed to update gauge metric %s in Postgres DB: %w", name, err)
		}
	}

	return value, nil
}

func (p *PostgresStorage) UpdateCounterMetric(c *models.Config, name string, value int64) (int64, error) {
	db := p.DB

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, e, err := p.GetCounterMetric(name)

	if err != nil {
		return value, err
	}

	// If metric doesn't exist then Insert, otherwise Update
	if !e {
		_, err = db.ExecContext(ctx, "INSERT INTO counter (name, value) VALUES($1, $2)", name, value)

		if err != nil {
			return value, fmt.Errorf("failed to insert counter metric into Postgres DB: %w", err)
		}
	} else {
		_, err = db.ExecContext(ctx, "UPDATE counter SET value = $1 WHERE name = $2", value, name)
		if err != nil {
			return value, fmt.Errorf("failed to update counter metric %s in Postgres DB: %w", name, err)
		}
	}

	return value, nil
}

func (p *PostgresStorage) GetCounterMetric(name string) (int64, bool, error) {
	db := p.DB

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var i int64

	row := db.QueryRowContext(ctx, "SELECT value FROM counter WHERE name=$1", name)
	err := row.Scan(&i)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return i, false, nil
		} else {
			return i, false, fmt.Errorf("failed to query counter table in Postgres DB: %w", err)
		}
	}
	return i, true, nil
}

func (p *PostgresStorage) GetGaugeMetric(name string) (float64, bool, error) {
	db := p.DB

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var f float64

	row := db.QueryRowContext(ctx, "SELECT value FROM gauge WHERE name=$1", name)
	err := row.Scan(&f)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return f, false, nil
		} else {
			return f, false, fmt.Errorf("failed to query gauge table in Postgres DB: %w", err)
		}
	}
	return f, true, nil
}

func (p *PostgresStorage) GetAllMetrics(c *models.Config) (map[string]float64, map[string]int64, error) {
	logger := c.Logger
	gaugeAll := make(map[string]float64)
	counterAll := make(map[string]int64)

	db := p.DB

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, "SELECT name, value FROM gauge")
	if err != nil {
		return nil, nil, fmt.Errorf("gauge table query error: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			logger.Sugar().Error("failed to close rows cursor: %w", err)
		}
	}()
	for rows.Next() {
		var gauge models.GaugeModel
		if err := rows.Scan(
			&gauge.Name,
			&gauge.Value,
		); err != nil {
			return nil, nil, fmt.Errorf("failed to scan row in gauge table: %w", err)
		}
		gaugeAll[gauge.Name] = gauge.Value
	}
	if err := rows.Err(); err != nil {
		logger.Sugar().Error("errors reading rows: %w", err)
	}
	rows, err = db.QueryContext(ctx, "SELECT name, value FROM counter")
	if err != nil {
		return nil, nil, fmt.Errorf("counter table query error: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			logger.Sugar().Error("failed to close rows cursor: %w", err)
		}
	}()

	for rows.Next() {
		var counter models.CounterModel
		if err := rows.Scan(
			&counter.Name,
			&counter.Value,
		); err != nil {
			return nil, nil, fmt.Errorf("failed to scan row in gauge table: %w", err)
		}
		counterAll[counter.Name] = counter.Value
	}
	if err := rows.Err(); err != nil {
		logger.Sugar().Error("errors reading rows: %w", err)
	}

	return gaugeAll, counterAll, nil
}
