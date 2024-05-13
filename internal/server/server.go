package server

import (
	"log"
	"net/http"

	"github.com/vkupriya/go-metrics/internal/server/handlers"
	"github.com/vkupriya/go-metrics/internal/server/storage"
)

func Start() {
	c, err := NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	s := storage.NewMemStorage()

	mr := handlers.NewMetricResource(s)

	r := handlers.NewMetricRouter(mr)

	log.Fatal(http.ListenAndServe(c.hostAddress, r))
}
