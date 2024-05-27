package config

import (
	"errors"
	"flag"
	"os"
	"strconv"

	"github.com/vkupriya/go-metrics/internal/server/models"
)

const (
	defaultStoreInterval int64 = 300
)

func NewConfig() (*models.Config, error) {
	a := flag.String("a", "localhost:8080", "Metric server host address and port.")
	i := flag.Int64("i", defaultStoreInterval, "Store interval in seconds, 0 sets it to synchronous.")
	p := flag.String("f", "/tmp/metrics-db.json", "File storage path.")
	r := flag.Bool("r", true, "Restore in memory DB at start up.")

	flag.Parse()

	if envAddr, ok := os.LookupEnv("ADDRESS"); ok {
		a = &envAddr
	}

	if envStoreInterval, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		envStoreInterval, err := strconv.ParseInt(envStoreInterval, 10, 64)
		if err != nil {
			return nil, errors.New("failed to convert env var STORE_INTERVAL to integer")
		}
		i = &envStoreInterval
	}

	if envFileStoragePath, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		p = &envFileStoragePath
	}

	if envRestore, ok := os.LookupEnv("RESTORE"); ok {
		envRestore, err := strconv.ParseBool(envRestore)
		if err != nil {
			return nil, errors.New("failed to convert env var RESTORE to bool")
		}
		r = &envRestore
	}

	return &models.Config{
		Address:         *a,
		StoreInterval:   *i,
		FileStoragePath: *p,
		RestoreMetrics:  *r,
	}, nil
}
