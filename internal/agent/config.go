package agent

import (
	"errors"
	"flag"
	"fmt"

	"os"
	"strconv"

	"go.uber.org/zap"
)

type Config struct {
	Logger         *zap.Logger
	metricHost     string
	HashKey        string
	CryptoKey      []byte
	SecretKey      []byte
	reportInterval int64
	pollInterval   int64
	httpTimeout    int64
	rateLimit      int
}

func NewConfig() (*Config, error) {
	const (
		pollIntDefault   int64 = 2
		reportIntDefault int64 = 10
		httpTimeout      int64 = 30
		rateLimitDefault int   = 3
	)
	var certPEM []byte
	var secretKey []byte
	var err error

	metricHost := flag.String("a", "localhost:8080", "Address and port of the metric server.")
	reportInterval := flag.Int64("r", reportIntDefault, "Metrics report interval in seconds.")
	pollInterval := flag.Int64("p", pollIntDefault, "Metric collection interval in seconds")
	rateLimit := flag.Int("l", rateLimitDefault, "Rate Limit for concurrent server requests.")
	key := flag.String("k", "", "Hash key")
	c := flag.String("c", "", "Path to public key for asymmetric encryption.")
	flag.Parse()

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

	if envPoll, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		envPollInt, err := strconv.ParseInt(envPoll, 10, 64)
		if err != nil {
			return nil, errors.New("failed to convert POLL_INTERVAL to integer")
		}
		pollInterval = &envPollInt
	}

	if envReport, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		envReportInt, err := strconv.ParseInt(envReport, 10, 64)
		if err != nil {
			return nil, errors.New("failed to convert REPORT_INTERVAL to integer")
		}
		reportInterval = &envReportInt
	}

	if envKey, ok := os.LookupEnv("KEY"); ok {
		key = &envKey
	}

	if envCryptoKey, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		c = &envCryptoKey
	}

	if *c != "" {
		certPEM, err = os.ReadFile(*c)
		if err != nil {
			return nil, fmt.Errorf("failed to read public key file %s: %w", *c, err)
		}
	}

	logConfig := zap.NewDevelopmentConfig()
	logger, err := logConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Logger: %w", err)
	}

	return &Config{
		metricHost:     *metricHost,
		reportInterval: *reportInterval,
		pollInterval:   *pollInterval,
		httpTimeout:    httpTimeout,
		rateLimit:      *rateLimit,
		Logger:         logger,
		HashKey:        *key,
		CryptoKey:      certPEM,
		SecretKey:      secretKey,
	}, nil
}
