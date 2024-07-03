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
	reportInterval int64
	pollInterval   int64
	httpTimeout    int64
}

func NewConfig() (*Config, error) {
	const (
		pollIntDefault   int64 = 2
		reportIntDefault int64 = 10
		httpTimeout      int64 = 30
	)

	metricHost := flag.String("a", "localhost:8080", "Address and port of the metric server.")
	reportInterval := flag.Int64("r", reportIntDefault, "Metrics report interval in seconds.")
	pollInterval := flag.Int64("p", pollIntDefault, "Metric collection interval in seconds")
	key := flag.String("k", "", "Hash key")
	flag.Parse()

	if envAddr, ok := os.LookupEnv("ADDRESS"); ok {
		metricHost = &envAddr
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
		Logger:         logger,
		HashKey:        *key,
	}, nil
}
