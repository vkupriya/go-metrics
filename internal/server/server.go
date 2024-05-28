package server

import (
	"log"
	"net/http"

	"github.com/vkupriya/go-metrics/internal/server/config"
	"github.com/vkupriya/go-metrics/internal/server/handlers"
	"github.com/vkupriya/go-metrics/internal/server/storage"
	"go.uber.org/zap"
)

func Start() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal(zap.Error(err))
	}

	s, err := storage.NewMemStorage(cfg)
	if err != nil {
		log.Fatal(zap.Error(err))
	}
	logger := cfg.Logger

	mr := handlers.NewMetricResource(s, cfg)
	if cfg.StoreInterval != 0 {
		logger.Info("starting ticker to save metrics to file")
		s.SaveMetricsTicker(cfg)
	}

	r := handlers.NewMetricRouter(mr)

	logger.Sugar().Infow(
		"Starting server",
		"addr", cfg.Address,
	)
	if err := http.ListenAndServe(cfg.Address, r); err != nil {
		logger.Sugar().Fatalw(err.Error(), "event", "start server")
	}
}
