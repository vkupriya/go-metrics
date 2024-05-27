package server

import (
	"net/http"

	"github.com/vkupriya/go-metrics/internal/server/config"
	"github.com/vkupriya/go-metrics/internal/server/handlers"
	"github.com/vkupriya/go-metrics/internal/server/storage"
	"go.uber.org/zap"
)

func Start() {
	zap.ReplaceGlobals(zap.Must(zap.NewDevelopment()))
	sugar := zap.L().Sugar()

	cfg, err := config.NewConfig()
	if err != nil {
		sugar.Fatal(zap.Error(err))
	}

	s, err := storage.NewMemStorage(cfg)
	if err != nil {
		sugar.Fatal(zap.Error(err))
	}

	mr := handlers.NewMetricResource(s, cfg)
	if cfg.StoreInterval != 0 {
		sugar.Info("starting ticker to save metrics to file")
		s.SaveToFileTicker()
	}

	r := handlers.NewMetricRouter(mr)

	sugar.Infow(
		"Starting server",
		"addr", cfg.Address,
	)
	if err := http.ListenAndServe(cfg.Address, r); err != nil {
		sugar.Fatalw(err.Error(), "event", "start server")
	}
}
