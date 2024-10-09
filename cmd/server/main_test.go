package main

import (
	"fmt"
	"syscall"
	"testing"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/vkupriya/go-metrics/internal/server"
)

func TestServer(t *testing.T) {
	logConfig := zap.NewDevelopmentConfig()
	logger, err := logConfig.Build()
	if err != nil {
		t.Error(err)
	}

	g := errgroup.Group{}

	g.Go(func() error {
		if err := server.Start(logger); err != nil {
			return fmt.Errorf("server failed: %w", err)
		}
		return nil
	})
	time.Sleep(3 * time.Second)

	// sending Kill event
	_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)

	if err := g.Wait(); err != nil {
		t.Error("failed to run collector/sender go routines: %w", err)
	}
}
