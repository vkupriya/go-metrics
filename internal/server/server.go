// Package server initializes and starts Metric Server.
package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/vkupriya/go-metrics/internal/server/config"
	grpcserver "github.com/vkupriya/go-metrics/internal/server/grpc"
	"github.com/vkupriya/go-metrics/internal/server/handlers"
	"github.com/vkupriya/go-metrics/internal/server/models"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const TimeoutShutdown time.Duration = 10 * time.Second
const TimeoutServerShutdown time.Duration = 5 * time.Second

func NewServer(c *models.Config, gr chi.Router) *http.Server {
	return &http.Server{
		Addr:    c.Address,
		Handler: gr,
	}
}

func Start(logger *zap.Logger) error {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal(zap.Error(err))
	}
	cfg.Logger = logger

	rootCtx, cancelCtx := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelCtx()

	g, ctx := errgroup.WithContext(rootCtx)

	_ = context.AfterFunc(ctx, func() {
		ctx, cancelCtx := context.WithTimeout(context.Background(), TimeoutShutdown)
		defer cancelCtx()

		<-ctx.Done()
		logger.Sugar().Error("failed to gracefully shutdown the service")
	})

	s, err := handlers.NewStore(cfg)
	if err != nil {
		logger.Sugar().Fatal(err)
	}

	mr := handlers.NewMetricResource(s, cfg)

	r := handlers.NewMetricRouter(mr)
	srv := NewServer(cfg, r)

	logger.Sugar().Infow(
		"Starting server",
		"addr", cfg.Address,
	)

	g.Go(func() error {
		defer logger.Sugar().Info("closed GRPC server")

		if err := grpcserver.Run(ctx, s, cfg); err != nil {
			return fmt.Errorf("failed to run grpc server: %w", err)
		}

		return nil
	})

	g.Go(func() error {
		defer logger.Sugar().Info("closed store")

		<-ctx.Done()

		mr.Store.Close()
		return nil
	})

	g.Go(func() (err error) {
		defer func() {
			errRec := recover()
			if errRec != nil {
				switch x := errRec.(type) {
				case string:
					err = errors.New(x)
					logger.Sugar().Error("a panic occured", zap.Error(err))
				case error:
					err = fmt.Errorf("a panic occurred: %w", x)
					logger.Sugar().Error(zap.Error(err))
				default:
					err = errors.New("unknown panic")
					logger.Sugar().Error(zap.Error(err))
				}
			}
		}()
		if err = srv.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return fmt.Errorf("listen and server has failed: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		defer logger.Sugar().Info("server has been shutdown")
		<-ctx.Done()

		shutdownTimeoutCtx, cancelShutdownTimeoutCtx := context.WithTimeout(context.Background(), TimeoutServerShutdown)
		defer cancelShutdownTimeoutCtx()
		if err := srv.Shutdown(shutdownTimeoutCtx); err != nil {
			return fmt.Errorf("an error occurred during server shutdown: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("go routines stopped with error: %w", err)
	}
	return nil
}
