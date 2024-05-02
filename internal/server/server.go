package server

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/vkupriya/go-metrics/internal/server/handlers"
	"github.com/vkupriya/go-metrics/internal/server/storage"
)

func Start() {
	s := storage.NewMemStorage()

	mr := handlers.NewMetricResource(s)

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.AllowContentType("text/plain"))

	r.Post("/update/{metricType}/{metricName}/{metricValue}", mr.UpdateMetric)

	log.Fatal(http.ListenAndServe(":8080", r))

}
