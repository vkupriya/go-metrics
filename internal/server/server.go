package server

import (
	"log"
	"net/http"

	"github.com/vkupriya/go-metrics/internal/server/handlers"
	"github.com/vkupriya/go-metrics/internal/server/storage"
	"go.uber.org/zap"
)

func Start() {
	c, err := NewConfig()
	if err != nil {
		log.Fatal(err)
	}
	zap.ReplaceGlobals(zap.Must(zap.NewDevelopment()))
	sugar := zap.L().Sugar()

	s := storage.NewMemStorage()

	mr := handlers.NewMetricResource(s)

	r := handlers.NewMetricRouter(mr)

	sugar.Infow(
		"Starting server",
		"addr", c.hostAddress,
	)
	if err := http.ListenAndServe(c.hostAddress, r); err != nil {
		sugar.Fatalw(err.Error(), "event", "start server")
	}
}
