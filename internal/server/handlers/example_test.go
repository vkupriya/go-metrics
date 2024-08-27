package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/vkupriya/go-metrics/internal/server/models"
	"github.com/vkupriya/go-metrics/internal/server/storage"
	"go.uber.org/zap"
)

func ExampleMetricResource_UpdateMetricJSON() {
	logConfig := zap.NewDevelopmentConfig()
	logger, err := logConfig.Build()
	if err != nil {
		fmt.Println("failed to initialize Logger: ", err)
	}

	cfg := &models.Config{
		Address:         "http://localhost:8080",
		StoreInterval:   300,
		FileStoragePath: "",
		RestoreMetrics:  false,
		Logger:          logger,
		PostgresDSN:     "",
		ContextTimeout:  3,
		HashKey:         "",
	}
	s, err := storage.NewMemStorage(cfg)
	if err != nil {
		fmt.Println(err)
	}
	mr := NewMetricResource(s, cfg)

	body := `{ "id": "PacketsIn",
  			"type": "counter",
  			"delta": 100287}`

	r := httptest.NewRequest(http.MethodPost, "/update/gauge/testSetGet17/571444.361", strings.NewReader(body))
	w := httptest.NewRecorder()
	mr.UpdateMetricJSON(w, r)
	res := w.Result()
	if err := res.Body.Close(); err != nil {
		fmt.Print("failed to close response body")
	}
	fmt.Println(res.StatusCode)
	// Output:
	// 200
}
