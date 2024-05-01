package server

import (
	"net/http"

	"github.com/vkupriya/go-metrics/internal/server/handlers"
	"github.com/vkupriya/go-metrics/internal/server/storage"
)

func Start() {
	s := storage.NewMemStorage()

	mr := handlers.NewMetricResource(s)

	mux := http.NewServeMux()

	mux.HandleFunc("/update/", mr.UpdateMetric)

	err := http.ListenAndServe(":8080", mux)

	if err != nil {
		panic(err)
	}
}
