package main

import (
	"log"

	"github.com/vkupriya/go-metrics/internal/agent"
)

func main() {
	if err := agent.Start(); err != nil {
		log.Fatal(err)
	}
}
