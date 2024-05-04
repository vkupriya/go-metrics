package server

import (
	"log"
	"net/http"

	"github.com/vkupriya/go-metrics/internal/server/handlers"
	"github.com/vkupriya/go-metrics/internal/server/storage"
)

func Start() {
	parseFlags()

	s := storage.NewMemStorage()

	mr := handlers.NewMetricResource(s)

	r := handlers.NewMetricRouter(mr)

	log.Fatal(http.ListenAndServe(flagRunAddr, r))
}
