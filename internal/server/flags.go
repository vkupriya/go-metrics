package server

import (
	"flag"
)

var flagRunAddr string

func parseFlags() {
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "Host address and port to run metric server.")

	flag.Parse()
}
