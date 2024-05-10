package server

import (
	"flag"
	"os"
)

var flagRunAddr string

func parseFlags() {
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "Metric server host address and port.")

	flag.Parse()

	if envRunAddr, ok := os.LookupEnv("ADDRESS"); ok {
		flagRunAddr = envRunAddr
	}
}
