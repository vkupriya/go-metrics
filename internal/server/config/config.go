// Package config - initialises metric server configuration through flags and env vars, and default settings.
// It instantiatiates zap logger.
package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/vkupriya/go-metrics/internal/server/models"
)

type ConfigFile struct {
	Address         string `json:"address,omitempty"`
	CryptoKeyFile   string `json:"crypto_key,omitempty"`
	FileStoragePath string `json:"store_file,omitempty"`
	PostgresDSN     string `json:"database_dsn,omitempty"`
	TrustedSubnet   string `json:"trusted_subnet,omitempty"`
	RestoreMetrics  bool   `json:"restore,omitempty"`
	StoreInterval   int64  `json:"store_interval,omitempty"`
}

const (
	defaultStoreInterval  int64 = 300
	defaultContextTimeout int64 = 3
)

func NewConfig() (*models.Config, error) {
	var err error
	var trustedSubnet *net.IPNet

	a := flag.String("a", "localhost:8080", "Metric server host address and port.")
	i := flag.Int64("i", defaultStoreInterval, "Store interval in seconds, 0 sets it to synchronous.")
	p := flag.String("f", "/tmp/metrics-db.json", "File storage path.")
	r := flag.Bool("r", true, "Restore in memory DB at start up.")
	d := flag.String("d", "", "PostgreSQL DSN")
	k := flag.String("k", "", "Key for HMAC signature.")
	cr := flag.String("cr", "", "Path to assymetric crypto private key.")
	t := flag.String("t", "", "Accepting metrics from Trusted IP CIDR only.")
	configFile := flag.String("c", "", "Path to json config file.")
	flag.Parse()

	cfg := ConfigFile{}
	privatePEM := make([]byte, 0)
	secretKey := make([]byte, 0)

	if envConfig, ok := os.LookupEnv("CONFIG"); ok {
		configFile = &envConfig
	}

	if *configFile != "" {
		fcontent, err := os.ReadFile(*configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open config file %s: %w", *configFile, err)
		}

		err = json.Unmarshal(fcontent, &cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal config file: %w", err)
		}
	}

	if cfg.Address != "" {
		a = &cfg.Address
	}

	if envAddr, ok := os.LookupEnv("ADDRESS"); ok {
		a = &envAddr
	}

	if cfg.PostgresDSN != "" && *d == "" {
		d = &cfg.PostgresDSN
	}

	if envDSN, ok := os.LookupEnv("DATABASE_DSN"); ok {
		d = &envDSN
	}

	if cfg.StoreInterval != 0 {
		i = &cfg.StoreInterval
	}

	if envStoreInterval, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		envStoreInterval, err := strconv.ParseInt(envStoreInterval, 10, 64)
		if err != nil {
			return nil, errors.New("failed to convert env var STORE_INTERVAL to integer")
		}
		i = &envStoreInterval
	}

	if cfg.FileStoragePath != "" {
		p = &cfg.FileStoragePath
	}

	if envFileStoragePath, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		p = &envFileStoragePath
	}

	if cfg.RestoreMetrics != *r {
		r = &cfg.RestoreMetrics
	}

	if envRestore, ok := os.LookupEnv("RESTORE"); ok {
		envRestore, err := strconv.ParseBool(envRestore)
		if err != nil {
			return nil, errors.New("failed to convert env var RESTORE to bool")
		}
		r = &envRestore
	}

	if envKey, ok := os.LookupEnv("KEY"); ok {
		k = &envKey
	}

	if cfg.CryptoKeyFile != "" {
		cr = &cfg.CryptoKeyFile
	}

	if envCryptoKey, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		cr = &envCryptoKey
	}

	if *cr != "" {
		privatePEM, err = os.ReadFile(*cr)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key file %s: %w", *cr, err)
		}
	}

	if cfg.TrustedSubnet != "" {
		t = &cfg.TrustedSubnet
	}

	if envTrustedSubnet, ok := os.LookupEnv("TRUSTED_SUBNET"); ok {
		t = &envTrustedSubnet
	}

	if *t != "" {
		_, trustedSubnet, err = net.ParseCIDR(*t)
		if err != nil {
			fmt.Println("Error: ", err)
			return nil, errors.New("trusted subnet is incorrect format, expected 1.2.3.4/24")
		}
	}
	fmt.Println("TrustedSubnet: ", trustedSubnet)
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
		TrustedSubnet:   trustedSubnet,
	}, nil
}
