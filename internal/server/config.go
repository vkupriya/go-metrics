package server

import (
	"flag"
	"os"
)

type Config struct {
	hostAddress string
}

func NewConfig() (*Config, error) {
	h := flag.String("a", "localhost:8080", "Metric server host address and port.")

	flag.Parse()

	if envAddr, ok := os.LookupEnv("ADDRESS"); ok {
		h = &envAddr
	}
	return &Config{
		hostAddress: *h,
	}, nil
}
