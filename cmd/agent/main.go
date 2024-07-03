package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/vkupriya/go-metrics/internal/agent"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := agent.Start(ctx); err != nil {
		log.Printf("error running agent: %v", err)
	}
	log.Println("agent stopped.")
}
