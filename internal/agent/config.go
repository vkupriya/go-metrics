package agent

import (
	"errors"
	"flag"
	"os"
	"strconv"
)

const (
	pollIntDefault   int64 = 2
	reportIntDefault int64 = 10
)

type Config struct {
	metricHost     string
	reportInterval int64
	pollInterval   int64
	httpTimeout    int64
}

func NewConfig() (*Config, error) {
	// default httpTimeout = 30 sec
	const httpTimeout int64 = 30

	metricHost := flag.String("a", "localhost:8080", "Address and port of the metric server.")
	reportInterval := flag.Int64("r", reportIntDefault, "Metrics report interval in seconds.")
	pollInterval := flag.Int64("p", pollIntDefault, "Metric collection interval in seconds")
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

	return &Config{
		metricHost:     *metricHost,
		reportInterval: *reportInterval,
		pollInterval:   *pollInterval,
		httpTimeout:    httpTimeout,
	}, nil
}
