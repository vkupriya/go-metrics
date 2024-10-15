package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"testing"

	"golang.org/x/sync/errgroup"

	"github.com/stretchr/testify/require"
	"github.com/vkupriya/go-metrics/internal/agent"
)

func TestServer(t *testing.T) {
	ctxroot, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	g, ctx := errgroup.WithContext(ctxroot)

	g.Go(func() error {
		if err := agent.Start(ctx); err != nil {
			return fmt.Errorf("server failed: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		require.Error(t, err)
	}
}
