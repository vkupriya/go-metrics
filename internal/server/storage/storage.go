// Package storage provides interface to upstream services for storing metrics.
// It defines and support 3 storage types: MemStorage, FileStorage and PostgresStorage.
package storage

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

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
	pool *pgxpool.Pool
}

func NewPostgresStorage(dsn string) (*PostgresStorage, error) {
	if err := runMigrations(dsn); err != nil {
		return nil, fmt.Errorf("failed to run DB migrations: %w", err)
	}
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the DSN: %w", err)
	}

	ctx := context.Background()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a connection pool: %w", err)
	}

	return &PostgresStorage{
		pool: pool,
	}, nil
}

//go:embed migrations/*.sql
var migrationsDir embed.FS

func runMigrations(dsn string) error {
	d, err := iofs.New(migrationsDir, "migrations")
	if err != nil {
		return fmt.Errorf("failed to return an iofs driver: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, dsn)
	if err != nil {
		return fmt.Errorf("failed to get a new migrate instance: %w", err)
	}
	if err := m.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("failed to apply migrations to the DB: %w", err)
		}
	}
	return nil
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
	FileExists := false

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
			logger.Sugar().Error(zap.Error(err))
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
		},
	}

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

func (m *MemStorage) GetCounterMetric(c *models.Config, name string) (int64, bool, error) {
	v, ok := m.counter[name]
	if ok {
		return v, true, nil
	}
	return v, false, fmt.Errorf("unknown metric %s ", name)
}

func (m *MemStorage) GetGaugeMetric(c *models.Config, name string) (float64, bool, error) {
	v, ok := m.gauge[name]
	if ok {
		return v, true, nil
	}
	return v, false, fmt.Errorf("unknown metric %s ", name)
}

func (m *MemStorage) GetAllMetrics(c *models.Config) (map[string]float64, map[string]int64, error) {
	return m.gauge, m.counter, nil
}

func (m *MemStorage) UpdateBatch(c *models.Config, g models.Metrics, cr models.Metrics) error {
	if g != nil || cr != nil {
		for _, i := range g {
			m.gauge[i.ID] = *i.Value
		}
		for _, i := range cr {
			m.counter[i.ID] += *i.Delta
		}
	}
	return nil
}

func (m *MemStorage) PingStore(c *models.Config) error {
	return nil
}

func (m *MemStorage) Close() {
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

func (f *FileStorage) GetCounterMetric(c *models.Config, name string) (int64, bool, error) {
	v, ok := f.counter[name]
	if ok {
		return v, true, nil
	}
	return v, false, fmt.Errorf("unknown counter metric %s ", name)
}

func (f *FileStorage) GetGaugeMetric(c *models.Config, name string) (float64, bool, error) {
	v, ok := f.gauge[name]
	if ok {
		return v, true, nil
	}
	return v, false, fmt.Errorf("unknown gauge metric %s ", name)
}

func (f *FileStorage) GetAllMetrics(c *models.Config) (map[string]float64, map[string]int64, error) {
	return f.gauge, f.counter, nil
}

func (f *FileStorage) UpdateBatch(c *models.Config, g models.Metrics, cr models.Metrics) error {
	if g != nil || cr != nil {
		for _, i := range g {
			f.gauge[i.ID] = *i.Value
		}
		for _, i := range cr {
			f.counter[i.ID] += *i.Delta
		}
		if c.StoreInterval == 0 {
			err := f.SaveMetrics(c)
			if err != nil {
				return fmt.Errorf("failed to save metrics to file: %w", err)
			}
		}
	}
	return nil
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
			logger.Sugar().Error(zap.Error(err))
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
		`SaveMetricsTicker started`,
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

func (f *FileStorage) PingStore(c *models.Config) error {
	return nil
}

func (f *FileStorage) Close() {
}

func (p *PostgresStorage) UpdateGaugeMetric(c *models.Config, name string, value float64) (float64, error) {
	db := p.pool
	mtype := "gauge"

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.ContextTimeout)*time.Second)
	defer cancel()

	querySQL := "INSERT INTO gauge (name, value) VALUES($1, $2) ON CONFLICT (name) DO UPDATE SET value = $2"

	_, err := db.Exec(ctx, querySQL, name, value)
	if err != nil {
		return value, fmt.Errorf("failed to insert/update %s metric into Postgres DB: %w", mtype, err)
	}
	return value, nil
}

func (p *PostgresStorage) UpdateCounterMetric(c *models.Config, name string, value int64) (int64, error) {
	db := p.pool

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.ContextTimeout)*time.Second)
	defer cancel()

	v, e, err := p.GetCounterMetric(c, name)
	if err != nil {
		return value, fmt.Errorf("failed query: %w", err)
	}

	if !e {
		if err := retryOnConnErr(func() error {
			_, err := db.Exec(ctx, "INSERT INTO counter (name, value) VALUES($1, $2)", name, value)
			if err != nil {
				return fmt.Errorf("error inserting into db: %w", err)
			}
			return nil
		}); err != nil {
			return value, fmt.Errorf("failed to insert counter metric '%s': %w", name, err)
		}
	} else {
		value += v
		if err := retryOnConnErr(func() error {
			_, err := db.Exec(ctx, "UPDATE counter SET value = $1 WHERE name = $2", value, name)
			if err != nil {
				return fmt.Errorf("error updating db: %w", err)
			}
			return nil
		}); err != nil {
			return value, fmt.Errorf("failed to update counter metric '%s': %w", name, err)
		}
	}

	return value, nil
}

func (p *PostgresStorage) GetCounterMetric(c *models.Config, name string) (int64, bool, error) {
	db := p.pool
	var i int64

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.ContextTimeout)*time.Second)
	defer cancel()

	if err := retryOnConnErr(func() error {
		return db.QueryRow(ctx, "SELECT value FROM counter WHERE name=$1", name).Scan(&i)
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return i, false, nil
		}
		return i, false, fmt.Errorf("failed to query counter table in Postgres DB: %w", err)
	}
	return i, true, nil
}

func (p *PostgresStorage) GetGaugeMetric(c *models.Config, name string) (float64, bool, error) {
	db := p.pool

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.ContextTimeout)*time.Second)
	defer cancel()
	var f float64

	if err := retryOnConnErr(func() error {
		return db.QueryRow(ctx, "SELECT value FROM gauge WHERE name=$1", name).Scan(&f)
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return f, false, nil
		}
		return f, false, fmt.Errorf("failed to query gauge table in Postgres DB: %w", err)
	}
	return f, true, nil
}

func (p *PostgresStorage) GetAllMetrics(c *models.Config) (map[string]float64, map[string]int64, error) {
	logger := c.Logger
	gaugeAll := make(map[string]float64)
	counterAll := make(map[string]int64)

	db := p.pool

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.ContextTimeout)*time.Second)
	defer cancel()

	rows, err := db.Query(ctx, "SELECT name, value FROM gauge")
	if err != nil {
		return nil, nil, fmt.Errorf("gauge table query error: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var gauge models.GaugeModel
		if err = rows.Scan(
			&gauge.Name,
			&gauge.Value,
		); err != nil {
			return nil, nil, fmt.Errorf("failed to scan row in gauge table: %w", err)
		}
		gaugeAll[gauge.Name] = gauge.Value
	}
	if err = rows.Err(); err != nil {
		logger.Sugar().Error("errors reading rows.", zap.Error(err))
	}
	rows, err = db.Query(ctx, "SELECT name, value FROM counter")
	if err != nil {
		return nil, nil, fmt.Errorf("counter table query error: %w", err)
	}
	defer rows.Close()

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
		logger.Sugar().Error("errors reading rows.", zap.Error(err))
	}

	return gaugeAll, counterAll, nil
}

func (p *PostgresStorage) UpdateBatch(c *models.Config, g models.Metrics, cr models.Metrics) error {
	logger := c.Logger
	db := p.pool

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.ContextTimeout)*time.Second)
	defer cancel()

	// processing counter metrics
	for _, i := range cr {
		v, e, err := p.GetCounterMetric(c, i.ID)
		if err != nil {
			return fmt.Errorf("failed query: %w", err)
		}
		if !e {
			_, err := db.Exec(ctx, "INSERT INTO counter (name, value) VALUES($1, $2)", i.ID, i.Delta)
			if err != nil {
				return fmt.Errorf("failed to insert counter metric '%s': %w", i.ID, err)
			}
		} else {
			v += *i.Delta
			_, err := db.Exec(ctx, "UPDATE counter SET value = $1 WHERE name = $2", v, i.ID)
			if err != nil {
				return fmt.Errorf("failed to update counter metric '%s': %w", i.ID, err)
			}
		}
	}

	if g != nil {
		// processing gauge metrics
		tx, err := db.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}
		querySQL := "INSERT INTO gauge (name, value) VALUES($1, $2) ON CONFLICT (name) DO UPDATE SET value = $2"
		for _, i := range g {
			_, err := tx.Exec(ctx, querySQL, i.ID, i.Value)
			if err != nil {
				logger.Sugar().Error(zap.Error(err))
				if err := tx.Rollback(ctx); err != nil {
					return fmt.Errorf("failed to rollback transaction: %w", err)
				}
			}
		}
		if err := retryOnConnErr(func() error {
			return tx.Commit(ctx)
		}); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}
	return nil
}

func (p *PostgresStorage) PingStore(c *models.Config) error {
	logger := c.Logger
	db := p.pool

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := retryOnConnErr(func() error {
		return db.Ping(ctx)
	}); err != nil {
		logger.Sugar().Debug("failed to connect to DB.", zap.Error(err))
		return fmt.Errorf("%w", err)
	}
	return nil
}

func (p *PostgresStorage) Close() {
	p.pool.Close()
}

func retryOnConnErr(f func() error) error {
	var (
		PgErr   *pgconn.PgError
		retries = 3
		retry   = 0
		backoff = 2
	)

	for retry <= retries {
		if err := f(); err != nil {
			if errors.As(err, &PgErr) && pgerrcode.IsConnectionException(PgErr.Code) {
				if retry == retries {
					return fmt.Errorf("failed after maximum retries: %w", err)
				}
			} else {
				return fmt.Errorf("failed to perform operation: %w", err)
			}
		} else {
			break
		}
		time.Sleep(time.Duration(1+(retry*backoff)) * time.Second)
		retry++
	}
	return nil
}
