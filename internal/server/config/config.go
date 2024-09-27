// Package config - initialises metric server configuration through flags and env vars, and default settings.
// It instantiatiates zap logger.
package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/vkupriya/go-metrics/internal/server/models"
)

type ConfigFile struct {
	Address         string `json:"address,omitempty"`
	CryptoKeyFile   string `json:"crypto_key,omitempty"`
	FileStoragePath string `json:"store_file,omitempty"`
	PostgresDSN     string `json:"database_dsn,omitempty"`
	RestoreMetrics  bool   `json:"restore,omitempty"`
	StoreInterval   int64  `json:"store_interval,omitempty"`
}

const (
	defaultStoreInterval  int64 = 300
	defaultContextTimeout int64 = 3
)

var privatePEM []byte
var secretKey []byte
var err error

func NewConfig() (*models.Config, error) {
	a := flag.String("a", "localhost:8080", "Metric server host address and port.")
	i := flag.Int64("i", defaultStoreInterval, "Store interval in seconds, 0 sets it to synchronous.")
	p := flag.String("f", "/tmp/metrics-db.json", "File storage path.")
	r := flag.Bool("r", true, "Restore in memory DB at start up.")
	d := flag.String("d", "", "PostgreSQL DSN")
	k := flag.String("k", "", "Key for HMAC signature ")
	cr := flag.String("cr", "", "Path to assymetric crypto private key.")
	configFile := flag.String("c", "", "Path to json config file.")
	flag.Parse()

	cfg := ConfigFile{}
	privatePEM = make([]byte, 0)
	secretKey = make([]byte, 0)

	if envConfig, ok := os.LookupEnv("CONFIG"); ok {
		configFile = &envConfig
	}

	if *configFile != "" {
		fmt.Printf("Opening config file: %s", *configFile)
		fcontent, err := os.ReadFile(*configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open config file %s: %w", *configFile, err)
		}

		err = json.Unmarshal(fcontent, &cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal config file: %w", err)
		}
	}

	if envAddr, ok := os.LookupEnv("ADDRESS"); ok {
		a = &envAddr
	} else if cfg.Address != "" {
		a = &cfg.Address
	}

	if envDSN, ok := os.LookupEnv("DATABASE_DSN"); ok {
		d = &envDSN
	} else if cfg.PostgresDSN != "" && *d == "" {
		d = &cfg.PostgresDSN
	}

	if envStoreInterval, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		envStoreInterval, err := strconv.ParseInt(envStoreInterval, 10, 64)
		if err != nil {
			return nil, errors.New("failed to convert env var STORE_INTERVAL to integer")
		}
		i = &envStoreInterval
	} else if cfg.StoreInterval != 0 {
		i = &cfg.StoreInterval
	}

	if envFileStoragePath, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		p = &envFileStoragePath
	} else if cfg.FileStoragePath != "" {
		p = &cfg.FileStoragePath
	}

	if envRestore, ok := os.LookupEnv("RESTORE"); ok {
		envRestore, err := strconv.ParseBool(envRestore)
		if err != nil {
			return nil, errors.New("failed to convert env var RESTORE to bool")
		}
		r = &envRestore
	} else if cfg.RestoreMetrics != *r {
		r = &cfg.RestoreMetrics
	}

	if envKey, ok := os.LookupEnv("KEY"); ok {
		k = &envKey
	}

	if envCryptoKey, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		cr = &envCryptoKey
	} else if cfg.CryptoKeyFile != "" {
		cr = &cfg.CryptoKeyFile
	}

	if *cr != "" {
		privatePEM, err = os.ReadFile(*cr)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key file %s: %w", *cr, err)
		}
	}

	return &models.Config{
		Address:         *a,
		StoreInterval:   *i,
		FileStoragePath: *p,
		RestoreMetrics:  *r,
		PostgresDSN:     *d,
		ContextTimeout:  defaultContextTimeout,
		HashKey:         *k,
		CryptoKey:       privatePEM,
		SecretKey:       secretKey,
	}, nil
}
