// Package server initializes and starts Metric Server.
package server

import (
	"log"
	"net/http"

	"github.com/vkupriya/go-metrics/internal/server/config"
	"github.com/vkupriya/go-metrics/internal/server/handlers"

	"go.uber.org/zap"
)

func Start(logger *zap.Logger) {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal(zap.Error(err))
	}
	cfg.Logger = logger

	s, err := handlers.NewStore(cfg)
	if err != nil {
		logger.Sugar().Fatal(err)
	}

	mr := handlers.NewMetricResource(s, cfg)

	r := handlers.NewMetricRouter(mr)

	logger.Sugar().Infow(
		"Starting server",
		"addr", cfg.Address,
	)

	if err := http.ListenAndServe(cfg.Address, r); err != nil {
		logger.Sugar().Fatalw(err.Error(), "event", "start server")
	}
}
