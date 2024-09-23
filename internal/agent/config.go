package agent

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"

	"os"
	"strconv"

	"go.uber.org/zap"
)

type Config struct {
	Logger         *zap.Logger
	MetricHost     string `json:"address,omitempty"`
	HashKey        string
	CryptoKey      []byte `json:"crypto_key,omitempty"`
	SecretKey      []byte
	ReportInterval int64 `json:"report_interval,omitempty"`
	PollInterval   int64 `json:"poll_interval,omitempty"`
	httpTimeout    int64
	rateLimit      int
}

type ConfigFile struct {
	MetricHost     string `json:"address,omitempty"`
	CryptoKeyFile  string `json:"crypto_key,omitempty"`
	ReportInterval int64  `json:"report_interval,omitempty"`
	PollInterval   int64  `json:"poll_interval,omitempty"`
}

func NewConfig() (*Config, error) {
	const (
		pollIntDefault   int64 = 1
		reportIntDefault int64 = 10
		httpTimeout      int64 = 30
		rateLimitDefault int   = 3
	)
	var certPEM []byte
	var secretKey []byte
	var err error
	cfg := ConfigFile{}

	metricHost := flag.String("a", "localhost:8080", "Address and port of the metric server.")
	reportInterval := flag.Int64("r", pollIntDefault, "Metrics report interval in seconds.")
	pollInterval := flag.Int64("p", pollIntDefault, "Metric collection interval in seconds")
	rateLimit := flag.Int("l", rateLimitDefault, "Rate Limit for concurrent server requests.")
	hashKey := flag.String("k", "", "Hash key")
	cryptoKey := flag.String("crypto", "", "Path to public key for asymmetric encryption.")
	configFile := flag.String("c", "", "Path to json config file.")
	flag.Parse()

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

	if cfg.MetricHost != "" {
		metricHost = &cfg.MetricHost
	}

	if envAddr, ok := os.LookupEnv("ADDRESS"); ok {
		metricHost = &envAddr
	}

	if envRateLimit, ok := os.LookupEnv("RATE_LIMIT"); ok {
		envRateLimit, err := strconv.Atoi(envRateLimit)
		if err != nil {
			return nil, errors.New("failed to convert RATE_LIMIT to integer")
		}
		rateLimit = &envRateLimit
	}

	if cfg.PollInterval != 0 {
		pollInterval = &cfg.PollInterval
	}

	if envPoll, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		envPollInt, err := strconv.ParseInt(envPoll, 10, 64)
		if err != nil {
			return nil, errors.New("failed to convert POLL_INTERVAL to integer")
		}
		pollInterval = &envPollInt
	}

	if cfg.ReportInterval != 0 {
		reportInterval = &cfg.ReportInterval
	}

	if envReport, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		envReportInt, err := strconv.ParseInt(envReport, 10, 64)
		if err != nil {
			return nil, errors.New("failed to convert REPORT_INTERVAL to integer")
		}
		reportInterval = &envReportInt
	}

	if envKey, ok := os.LookupEnv("KEY"); ok {
		hashKey = &envKey
	}

	if cfg.CryptoKeyFile != "" {
		cryptoKey = &cfg.CryptoKeyFile
	}

	if envCryptoKey, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		cryptoKey = &envCryptoKey
	}

	if *cryptoKey != "" {
		certPEM, err = os.ReadFile(*cryptoKey)
		if err != nil {
			return nil, fmt.Errorf("failed to read public key file %s: %w", *cryptoKey, err)
		}
	}

	logConfig := zap.NewDevelopmentConfig()
	logger, err := logConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Logger: %w", err)
	}

	return &Config{
		MetricHost:     *metricHost,
		ReportInterval: *reportInterval,
		PollInterval:   *pollInterval,
		httpTimeout:    httpTimeout,
		rateLimit:      *rateLimit,
		Logger:         logger,
		HashKey:        *hashKey,
		CryptoKey:      certPEM,
		SecretKey:      secretKey,
	}, nil
}
