package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vkupriya/go-metrics/internal/server/config"
	mock_handlers "github.com/vkupriya/go-metrics/internal/server/handlers/mocks"
	"github.com/vkupriya/go-metrics/internal/server/models"
	"github.com/vkupriya/go-metrics/internal/server/storage"
)

func testRequest(t *testing.T, ts *httptest.Server, method, path string, body string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, ts.URL+path, strings.NewReader(body))
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	if err := resp.Body.Close(); err != nil {
		panic(err)
	}

	return resp
}

func TestNewStorePostgres(t *testing.T) {
	logConfig := zap.NewDevelopmentConfig()
	logger, err := logConfig.Build()
	if err != nil {
		t.Error("failed to initialize Logger: %w", err)
	}

	cfg := &models.Config{
		Address:         "http://localhost:8080",
		StoreInterval:   5,
		FileStoragePath: "/tmp/metrics-db.json",
		RestoreMetrics:  false,
		Logger:          logger,
		PostgresDSN:     "postgres://test:test@localhost:5432/metrics?sslmode=disable",
		ContextTimeout:  3,
		HashKey:         "",
	}
	_, err = NewStore(cfg)
	require.Error(t, err)
}

func TestNewStoreFileStore(t *testing.T) {
	logConfig := zap.NewDevelopmentConfig()
	logger, err := logConfig.Build()
	if err != nil {
		t.Error("failed to initialize Logger: %w", err)
	}

	cfg := &models.Config{
		Address:         "http://localhost:8080",
		StoreInterval:   5,
		FileStoragePath: "//metrics-db.json",
		RestoreMetrics:  false,
		Logger:          logger,
		PostgresDSN:     "",
		ContextTimeout:  3,
		HashKey:         "",
	}
	_, err = NewStore(cfg)
	require.Error(t, err)
}

func TestUpdateAndGetMetricsMemStore(t *testing.T) {
	logConfig := zap.NewDevelopmentConfig()
	logger, err := logConfig.Build()
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := config.NewConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Logger = logger
	cfg.FileStoragePath = ""
	cfg.HashKey = "kjsldkfjlskd"
	s, err := NewStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	mr := NewMetricResource(s, cfg)

	ts := httptest.NewServer(NewMetricRouter(mr))

	tests := []struct {
		name         string
		method       string
		path         string
		body         string
		expectedBody string
		expectedCode int
	}{
		{
			name:         "ping_memstore: OK",
			method:       http.MethodGet,
			path:         "/ping",
			body:         "",
			expectedCode: 200,
			expectedBody: "",
		},
		{
			name:         "get_metric_wrongURL: FAIL",
			method:       http.MethodGet,
			path:         "/value/",
			body:         "",
			expectedCode: 405,
			expectedBody: "",
		},
		{
			name:         "get_gauge_metric: FAIL",
			method:       http.MethodGet,
			path:         "/value/gauge/test",
			body:         "",
			expectedCode: 404,
			expectedBody: "",
		},
		{
			name:         "get_metric_wrongtype: FAIL",
			method:       http.MethodGet,
			path:         "/value/wrongtype/test",
			body:         "",
			expectedCode: 400,
			expectedBody: "",
		},
		{
			name:         "update_gauge_metric_wrongvalue: FAIL",
			method:       http.MethodPost,
			path:         "/update/gauge/test/string",
			body:         "",
			expectedCode: 400,
		},
		{
			name:         "update_gauge_metric: OK",
			method:       http.MethodPost,
			path:         "/update/gauge/test/20.0",
			body:         "",
			expectedCode: 200,
		},
		{
			name:         "update_gauge_metric_novalue: FAIL",
			method:       http.MethodPost,
			path:         `/update/gauge/test205/""`,
			body:         "",
			expectedCode: 400,
		},
		{
			name:         "update_gauge_metric_noname: FAIL",
			method:       http.MethodPost,
			path:         "/update/gauge/''",
			body:         "",
			expectedCode: 404,
		},
		{
			name:         "get_gauge_metric: OK",
			method:       http.MethodGet,
			path:         "/value/gauge/test",
			body:         "",
			expectedCode: 200,
			expectedBody: "20.0",
		},
		{
			name:         "get_gauge_metric_JSON: OK",
			method:       http.MethodPost,
			path:         "/value/",
			body:         `{ "id": "test", "type": "gauge"}`,
			expectedCode: 200,
			expectedBody: `{ "id": "test", "type": "gauge", "value": 20.0}`,
		},
		{
			name:         "update_gauge_metric_JSON_wrongvalue: FAIL",
			method:       http.MethodPost,
			path:         "/value/",
			body:         `{ "id": "test", "type": "gauge", "delta": 20.0}`,
			expectedCode: 500,
			expectedBody: `{ "id": "test", "type": "gauge", "value": 20.0}`,
		},
		{
			name:         "update_gauge_metric_JSON_wrongvalue: FAIL",
			method:       http.MethodPost,
			path:         "/value/",
			body:         `{ "id": "test", "type": "gauge", "delta": 20.0}`,
			expectedCode: 500,
			expectedBody: `{ "id": "test", "type": "gauge", "value": 20.0}`,
		},
		{
			name:         "get_counter_metric: FAIL",
			method:       http.MethodGet,
			path:         "/value/counter/test",
			body:         "",
			expectedCode: 404,
		},
		{
			name:         "update_counter_metric_wrongvalue: FAIL",
			method:       http.MethodPost,
			path:         "/update/counter/test/string",
			body:         "",
			expectedCode: 400,
		},
		{
			name:         "update_counter_metric: OK",
			method:       http.MethodPost,
			path:         "/update/counter/test/20",
			body:         "",
			expectedCode: 200,
		},
		{
			name:         "get_counter_metric: OK",
			method:       http.MethodGet,
			path:         "/value/counter/test",
			body:         "",
			expectedCode: 200,
			expectedBody: "20",
		},
		{
			name:         "update_batch_metric: OK",
			method:       http.MethodPost,
			path:         "/updates/",
			body:         `[{ "id": "test", "type": "gauge", "value": 20.0}, { "id": "test", "type": "counter", "delta": 20}]`,
			expectedCode: 200,
		},
		{
			name:         "get_all_metrics: OK",
			method:       http.MethodGet,
			path:         "/",
			body:         "",
			expectedCode: 200,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testRequest(t, ts, tt.method, tt.path, tt.body)
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			if err := resp.Body.Close(); err != nil {
				assert.Error(t, err)
			}
		})
	}
}

func TestUpdateMetricFileStore(t *testing.T) {
	logConfig := zap.NewDevelopmentConfig()
	logger, err := logConfig.Build()
	if err != nil {
		t.Error("failed to initialize Logger: %w", err)
	}

	cfg := &models.Config{
		Address:         "http://localhost:8080",
		StoreInterval:   0,
		FileStoragePath: "/tmp/metrics-db.json",
		RestoreMetrics:  true,
		Logger:          logger,
		PostgresDSN:     "",
		ContextTimeout:  3,
		HashKey:         "",
	}
	s, err := NewStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	mr := NewMetricResource(s, cfg)

	ts := httptest.NewServer(NewMetricRouter(mr))

	tests := []struct {
		name         string
		method       string
		path         string
		body         string
		expectedBody string
		expectedCode int
	}{
		{
			name:         "get_gauge_metric: FAIL",
			method:       http.MethodGet,
			path:         "/value/gauge/test25",
			body:         "",
			expectedCode: 404,
			expectedBody: "",
		},
		{
			name:         "update_gauge_metric: OK",
			method:       http.MethodPost,
			path:         "/update/gauge/test/20.0",
			body:         "",
			expectedCode: 200,
		},
		{
			name:         "get_gauge_metric: OK",
			method:       http.MethodGet,
			path:         "/value/gauge/test",
			body:         "",
			expectedCode: 200,
			expectedBody: "20.0",
		},
		{
			name:         "get_gauge_metric_JSON: OK",
			method:       http.MethodPost,
			path:         "/value/",
			body:         `{ "id": "test", "type": "gauge"}`,
			expectedCode: 200,
			expectedBody: `{ "id": "test", "type": "gauge", "value": 20.0}`,
		},
		{
			name:         "get_gauge_metric_JSON_no_type: FAIL",
			method:       http.MethodPost,
			path:         "/value/",
			body:         `{ "id": "test"}`,
			expectedCode: 400,
		},
		{
			name:         "get_gauge_metric_incorrect_JSON: OK",
			method:       http.MethodPost,
			path:         "/value/",
			body:         `{ "id": "test"`,
			expectedCode: 500,
		},
		{
			name:         "update_gauge_metric_JSON_novalue: FAIL",
			method:       http.MethodPost,
			path:         "/update/",
			body:         `{ "id": "test", "type"gauge"}`,
			expectedCode: 500,
		},
		{
			name:         "get_counter_metric: FAIL",
			method:       http.MethodGet,
			path:         "/value/counter/test45",
			body:         "",
			expectedCode: 404,
		},
		{
			name:         "update_counter_metric: OK",
			method:       http.MethodPost,
			path:         "/update/counter/test/20",
			body:         "",
			expectedCode: 200,
		},
		{
			name:         "get_counter_metric: OK",
			method:       http.MethodGet,
			path:         "/value/counter/test",
			body:         "",
			expectedCode: 200,
			expectedBody: "20",
		},
		{
			name:         "get_counter_metric_JSON: OK",
			method:       http.MethodPost,
			path:         "/value/",
			body:         `{ "id": "test", "type": "counter", "delta": 20}`,
			expectedCode: 200,
		},
		{
			name:         "update_batch_metric: OK",
			method:       http.MethodPost,
			path:         "/updates/",
			body:         `[{ "id": "test", "type": "gauge", "value": 20.0}, { "id": "test", "type": "counter", "delta": 20}]`,
			expectedCode: 200,
		},
		{
			name:         "update_batch_metric_wrong_JSON: FAIL",
			method:       http.MethodPost,
			path:         "/updates/",
			body:         `[{ "id": "test", "type": "gauge", "value": 20.0}, { "id": "test", "type": "counter", "delta": 20}`,
			expectedCode: 500,
		},
		{
			name:         "get_all_metrics: OK",
			method:       http.MethodGet,
			path:         "/",
			body:         "",
			expectedCode: 200,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testRequest(t, ts, tt.method, tt.path, tt.body)
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			if err := resp.Body.Close(); err != nil {
				assert.Error(t, err)
			}
		})
	}
}

func TestUpdateMetricFileStoreTicker(t *testing.T) {
	logConfig := zap.NewDevelopmentConfig()
	logger, err := logConfig.Build()
	if err != nil {
		t.Error("failed to initialize Logger: %w", err)
	}

	cfg := &models.Config{
		Address:         "http://localhost:8080",
		StoreInterval:   5,
		FileStoragePath: "/tmp/metrics-db.json",
		RestoreMetrics:  false,
		Logger:          logger,
		PostgresDSN:     "",
		ContextTimeout:  3,
		HashKey:         "",
	}
	s, err := NewStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	mr := NewMetricResource(s, cfg)

	ts := httptest.NewServer(NewMetricRouter(mr))

	tests := []struct {
		name         string
		method       string
		path         string
		body         string
		expectedBody string
		expectedCode int
	}{
		{
			name:         "ping_filestore: OK",
			method:       http.MethodGet,
			path:         "/ping",
			body:         "",
			expectedCode: 200,
			expectedBody: "",
		},
		{
			name:         "get_gauge_metric: FAIL",
			method:       http.MethodGet,
			path:         "/value/gauge/test",
			body:         "",
			expectedCode: 404,
			expectedBody: "",
		},
		{
			name:         "update_gauge_metric: OK",
			method:       http.MethodPost,
			path:         "/update/gauge/test/20.0",
			body:         "",
			expectedCode: 200,
		},
		{
			name:         "get_gauge_metric: OK",
			method:       http.MethodGet,
			path:         "/value/gauge/test",
			body:         "",
			expectedCode: 200,
			expectedBody: "20.0",
		},
		{
			name:         "get_gauge_metric_incorrect_JSON: OK",
			method:       http.MethodPost,
			path:         "/value/",
			body:         `{ "id": "test"`,
			expectedCode: 500,
		},
		{
			name:         "update_gauge_metric_JSON_novalue: FAIL",
			method:       http.MethodPost,
			path:         "/update/",
			body:         `{ "id": "test", "type"gauge"}`,
			expectedCode: 500,
		},
		{
			name:         "get_counter_metric: FAIL",
			method:       http.MethodGet,
			path:         "/value/counter/test",
			body:         "",
			expectedCode: 404,
		},
		{
			name:         "update_counter_metric: OK",
			method:       http.MethodPost,
			path:         "/update/counter/test/20",
			body:         "",
			expectedCode: 200,
		},
		{
			name:         "get_counter_metric: OK",
			method:       http.MethodGet,
			path:         "/value/counter/test",
			body:         "",
			expectedCode: 200,
			expectedBody: "20",
		},
		{
			name:         "update_batch_metric_wrong_JSON: FAIL",
			method:       http.MethodPost,
			path:         "/updates/",
			body:         `[{ "id": "test", "type": "gauge", "value": 20.0}, { "id": "test", "type": "counter", "delta": 20}`,
			expectedCode: 500,
		},
		{
			name:         "get_all_metrics: OK",
			method:       http.MethodGet,
			path:         "/",
			body:         "",
			expectedCode: 200,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testRequest(t, ts, tt.method, tt.path, "")
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			if err := resp.Body.Close(); err != nil {
				assert.Error(t, err)
			}
		})
	}
}
func TestUpdateMetric(t *testing.T) {
	logConfig := zap.NewDevelopmentConfig()
	logger, err := logConfig.Build()
	if err != nil {
		t.Error("failed to initialize Logger: %w", err)
	}
	var f = 54.555
	var i int64 = 555

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

	tests := []struct {
		mockStore    func(*gomock.Controller) *mock_handlers.MockStorage
		name         string
		method       string
		path         string
		expectedBody string
		expectedCode int
	}{
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				s.EXPECT().UpdateGaugeMetric(gomock.Any(), gomock.Any(), gomock.Any()).Return(f, nil).AnyTimes()
				return s
			},
			name:         "update_gauge_metric:OK",
			method:       http.MethodPost,
			path:         "/update/gauge/test/54.555",
			expectedCode: 200,
			expectedBody: "",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				s.EXPECT().UpdateCounterMetric(gomock.Any(), gomock.Any(), gomock.Any()).Return(i, nil).AnyTimes()
				return s
			},
			name:         "update_counter_metric:OK",
			method:       http.MethodPost,
			path:         "/update/counter/test/555",
			expectedCode: 200,
			expectedBody: "",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				return s
			},
			name:         "invalid_metric_type:FAIL",
			method:       http.MethodPost,
			path:         "/update/timeseries/test/555",
			expectedCode: 400,
			expectedBody: "",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				return s
			},
			name:         "missing_metric_name:FAIL",
			method:       http.MethodPost,
			path:         "/update/timeseries",
			expectedCode: 404,
			expectedBody: "",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				return s
			},
			name:         "wrong_counter_metric_value_type:FAIL",
			method:       http.MethodPost,
			path:         "/update/counter/test/20.0",
			expectedCode: 400,
			expectedBody: "",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				return s
			},
			name:         "wrong_gauge_metric_value_type:FAIL",
			method:       http.MethodPost,
			path:         "/update/counter/test/string",
			expectedCode: 400,
			expectedBody: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			s := tt.mockStore(ctrl)

			mr := NewMetricResource(s, cfg)

			ts := httptest.NewServer(NewMetricRouter(mr))
			resp := testRequest(t, ts, tt.method, tt.path, "")
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			if err := resp.Body.Close(); err != nil {
				assert.Error(t, err)
			}
		})
	}
}

func TestPingStore(t *testing.T) {
	logConfig := zap.NewDevelopmentConfig()
	logger, err := logConfig.Build()
	if err != nil {
		t.Error("failed to initialize Logger: %w", err)
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

	tests := []struct {
		mockStore    func(*gomock.Controller) *mock_handlers.MockStorage
		name         string
		method       string
		path         string
		expectedBody string
		expectedCode int
	}{
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				s.EXPECT().PingStore(gomock.Any()).Return(nil).AnyTimes()
				return s
			},
			name:         "ping_store:OK",
			method:       http.MethodGet,
			path:         "/ping",
			expectedCode: 200,
			expectedBody: "",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				s.EXPECT().PingStore(gomock.Any()).Return(errors.New("failed to ping store.")).AnyTimes()
				return s
			},
			name:         "ping_store:FAIL",
			method:       http.MethodGet,
			path:         "/ping",
			expectedCode: 500,
			expectedBody: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			s := tt.mockStore(ctrl)

			mr := NewMetricResource(s, cfg)

			ts := httptest.NewServer(NewMetricRouter(mr))
			resp := testRequest(t, ts, tt.method, tt.path, "")
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			if err := resp.Body.Close(); err != nil {
				assert.Error(t, err)
			}
		})
	}
}

//nolint:dupl // handlers unit tests following same pattern
func TestUpdateMetricJSON(t *testing.T) {
	logConfig := zap.NewDevelopmentConfig()
	logger, err := logConfig.Build()
	if err != nil {
		t.Error("failed to initialize Logger: %w", err)
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

	var f = 100287.253
	var i int64 = 100287

	tests := []struct {
		mockStore    func(*gomock.Controller) *mock_handlers.MockStorage
		name         string
		method       string
		path         string
		body         string
		expectedBody string
		expectedCode int
	}{
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				s.EXPECT().UpdateGaugeMetric(gomock.Any(), gomock.Any(), gomock.Any()).Return(f, nil).AnyTimes()
				return s
			},
			name:         "update_gauge_metric:OK",
			method:       http.MethodPost,
			path:         "/update/",
			body:         `{ "id": "PacketsIn", "type": "gauge", "value": 100287.253}`,
			expectedCode: 200,
			expectedBody: "",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				s.EXPECT().UpdateGaugeMetric(gomock.Any(), gomock.Any(), gomock.Any()).Return(f, errors.New("error")).AnyTimes()
				return s
			},
			name:         "update_gauge_metric:FAIL",
			method:       http.MethodPost,
			path:         "/update/",
			body:         `{ "id": "PacketsIn", "type": "gauge", "value": 100287.253}`,
			expectedCode: 500,
			expectedBody: "",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				s.EXPECT().UpdateCounterMetric(gomock.Any(), gomock.Any(), gomock.Any()).Return(i, nil).AnyTimes()
				return s
			},
			name:         "update_counter_metric:OK",
			method:       http.MethodPost,
			path:         "/update/",
			body:         `{ "id": "PacketsIn", "type": "counter", "delta": 100287}`,
			expectedCode: 200,
			expectedBody: "",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				s.EXPECT().UpdateCounterMetric(gomock.Any(), gomock.Any(), gomock.Any()).Return(i, errors.New("error")).AnyTimes()
				return s
			},
			name:         "update_counter_metric:FAIL",
			method:       http.MethodPost,
			path:         "/update/",
			body:         `{ "id": "PacketsIn", "type": "counter", "delta": 100287}`,
			expectedCode: 500,
			expectedBody: "",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				return s
			},
			name:         "update_metric_wrong_type:FAIL",
			method:       http.MethodPost,
			path:         "/update/",
			body:         `{ "id": "PacketsIn", "type": "wrongtype", "delta": 100287}`,
			expectedCode: 400,
			expectedBody: "",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				return s
			},
			name:         "update_counter_metric_novalue:FAIL",
			method:       http.MethodPost,
			path:         "/update/",
			body:         `{ "id": "PacketsIn", "type": "counter"}`,
			expectedCode: 400,
			expectedBody: "",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				return s
			},
			name:         "update_gauge_metric_novalue:FAIL",
			method:       http.MethodPost,
			path:         "/update/",
			body:         `{ "id": "PacketsIn", "type": "gauge"}`,
			expectedCode: 400,
			expectedBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := gomock.NewController(t)
			s := tt.mockStore(c)

			mr := NewMetricResource(s, cfg)
			r := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			mr.UpdateMetricJSON(w, r)
			res := w.Result()
			assert.Equal(t, tt.expectedCode, res.StatusCode)
			if err := res.Body.Close(); err != nil {
				assert.Error(t, err)
			}
		})
	}
}

func TestGetMetric(t *testing.T) {
	logConfig := zap.NewDevelopmentConfig()
	logger, err := logConfig.Build()
	if err != nil {
		t.Error("failed to initialize Logger: %w", err)
	}
	var f = 54.555
	var i int64 = 555

	gauge := map[string]float64{
		"Test01": 3535.31,
		"Test02": 32384927.61,
	}

	counter := map[string]int64{
		"Test03": 53528,
		"Test04": 3241,
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

	tests := []struct {
		mockStore    func(*gomock.Controller) *mock_handlers.MockStorage
		name         string
		method       string
		body         string
		path         string
		expectedBody string
		expectedCode int
	}{
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				s.EXPECT().GetGaugeMetric(gomock.Any(), gomock.Any()).Return(f, true, nil).AnyTimes()
				return s
			},
			name:         "get_gauge_metric:OK",
			method:       http.MethodGet,
			path:         "/value/gauge/test",
			body:         "",
			expectedCode: 200,
			expectedBody: "54.555",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				s.EXPECT().GetGaugeMetric(gomock.Any(), gomock.Any()).Return(f, true, nil).AnyTimes()
				return s
			},
			name:         "get_gauge_metric:OK",
			method:       http.MethodGet,
			path:         "/value/gauge/test",
			body:         "",
			expectedCode: 200,
			expectedBody: "54.555",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				s.EXPECT().GetCounterMetric(gomock.Any(), gomock.Any()).Return(i, true, nil).AnyTimes()
				return s
			},
			name:         "get_counter_metric:OK",
			method:       http.MethodGet,
			path:         "/value/counter/test",
			body:         "",
			expectedCode: 200,
			expectedBody: "555",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				s.EXPECT().GetGaugeMetric(gomock.Any(), gomock.Any()).Return(
					0.0, false, errors.New("unknown gauge metric")).AnyTimes()
				return s
			},
			name:         "get_gauge_metric:FAIL",
			method:       http.MethodGet,
			path:         "/value/gauge/test",
			body:         "",
			expectedCode: 404,
			expectedBody: "",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				s.EXPECT().GetCounterMetric(gomock.Any(), gomock.Any()).Return(
					int64(0), false, errors.New("unknown counter metric")).AnyTimes()
				return s
			},
			name:         "get_counter_metric:FAIL",
			method:       http.MethodGet,
			path:         "/value/counter/test",
			body:         "",
			expectedCode: 404,
			expectedBody: "",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				s.EXPECT().GetAllMetrics(gomock.Any()).Return(
					gauge, counter, nil).AnyTimes()
				return s
			},
			name:         "get_all_metrics:OK",
			method:       http.MethodGet,
			path:         "/",
			body:         "",
			expectedCode: 200,
			expectedBody: "",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				s.EXPECT().GetAllMetrics(gomock.Any()).Return(
					nil, nil, errors.New("failed to get all metrics.")).AnyTimes()
				return s
			},
			name:         "get_all_metrics:FAIL",
			method:       http.MethodGet,
			path:         "/",
			body:         "",
			expectedCode: 500,
			expectedBody: "",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				s.EXPECT().GetCounterMetric(gomock.Any(), gomock.Any()).Return(
					int64(100287), true, nil).AnyTimes()
				return s
			},
			name:         "get_counter_metric_JSON:OK",
			method:       http.MethodPost,
			path:         "/value/",
			body:         `{ "id": "PacketsIn", "type": "counter"}`,
			expectedCode: 200,
			expectedBody: `{"id": "PacketsIn", "type": "counter", "delta": 100287}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			s := tt.mockStore(ctrl)

			mr := NewMetricResource(s, cfg)

			ts := httptest.NewServer(NewMetricRouter(mr))
			resp := testRequest(t, ts, tt.method, tt.path, tt.body)
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			if err := resp.Body.Close(); err != nil {
				assert.Error(t, err)
			}
		})
	}
}

//nolint:dupl // handlers unit tests following same pattern
func TestUpdateBatchJSON(t *testing.T) {
	logConfig := zap.NewDevelopmentConfig()
	logger, err := logConfig.Build()
	if err != nil {
		t.Error("failed to initialize Logger: %w", err)
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

	body := `[{"delta":4,"id":"PollCount","type":"counter"}, 
	{"value":240632,"id":"HeapAlloc","type":"gauge"},
	{"value":1757776,"id":"GCSys","type":"gauge"},
	{"value":2621440,"id":"HeapIdle","type":"gauge"},
	{"value":3702784,"id":"HeapSys","type":"gauge"},
	{"value":0,"id":"NumForcedGC","type":"gauge"},
	{"value":0,"id":"NumGC","type":"gauge"},
	{"value":491520,"id":"StackSys","type":"gauge"},
	{"value":941952,"id":"OtherSys","type":"gauge"},
	{"value":5023.92,"id":"CPUutilization1","type":"gauge"},
	{"value":1187.16,"id":"CPUutilization3","type":"gauge"},
	{"value":374.08,"id":"CPUutilization4","type":"gauge"},
	{"value":238.16,"id":"CPUutilization5","type":"gauge"},
	{"value":68.21,"id":"CPUutilization7","type":"gauge"},
	{"value":240632,"id":"Alloc","type":"gauge"},
	{"value":0,"id":"Lookups","type":"gauge"},
	{"value":5278.46,"id":"CPUutilization0","type":"gauge"},
	{"value":108.03,"id":"CPUutilization6","type":"gauge"},
	{"value":7696,"id":"BuckHashSys","type":"gauge"},
	{"value":1081344,"id":"HeapInuse","type":"gauge"},
	{"value":710,"id":"HeapObjects","type":"gauge"},
	{"value":44000,"id":"MSpanInuse","type":"gauge"},
	{"value":240632,"id":"TotalAlloc","type":"gauge"},
	{"value":17179869184,"id":"TotalMemory","type":"gauge"},
	{"value":2621440,"id":"HeapReleased","type":"gauge"},
	{"value":0,"id":"LastGC","type":"gauge"},
	{"value":0,"id":"PauseTotalNs","type":"gauge"},
	{"value":156581888,"id":"FreeMemory","type":"gauge"},
	{"value":46,"id":"Frees","type":"gauge"},
	{"value":48960,"id":"MSpanSys","type":"gauge"},
	{"value":756,"id":"Mallocs","type":"gauge"},
	{"value":4194304,"id":"NextGC","type":"gauge"},
	{"value":491520,"id":"StackInuse","type":"gauge"},
	{"value":1814.73,"id":"CPUutilization2","type":"gauge"},
	{"value":0,"id":"GCCPUFraction","type":"gauge"},
	{"value":9600,"id":"MCacheInuse","type":"gauge"},
	{"value":15600,"id":"MCacheSys","type":"gauge"},
	{"value":6966288,"id":"Sys","type":"gauge"},
	{"value":0.08536959506538425,"id":"RandomValue","type":"gauge"}]`

	tests := []struct {
		mockStore    func(*gomock.Controller) *mock_handlers.MockStorage
		name         string
		method       string
		path         string
		body         string
		expectedBody string
		expectedCode int
	}{
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				s.EXPECT().UpdateBatch(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				return s
			},
			name:         "update_batch_metrics:OK",
			method:       http.MethodPost,
			path:         "/updates/",
			body:         body,
			expectedCode: 200,
			expectedBody: "",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				s.EXPECT().UpdateBatch(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("error")).AnyTimes()
				return s
			},
			name:         "update_batch_metrics:OK",
			method:       http.MethodPost,
			path:         "/updates/",
			body:         body,
			expectedCode: 500,
			expectedBody: "",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				return s
			},
			name:         "update_batch_metrics_gauge_novalue:FAIL",
			method:       http.MethodPost,
			path:         "/updates/",
			body:         `[{"id":"HeapAlloc","type":"gauge"}]`,
			expectedCode: 400,
			expectedBody: "",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				return s
			},
			name:         "update_batch_metrics_counter_nodelta:FAIL",
			method:       http.MethodPost,
			path:         "/updates/",
			body:         `[{"id":"PollCount","type":"counter"}]`,
			expectedCode: 400,
			expectedBody: "",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				return s
			},
			name:         "update_batch_metrics_badjson:FAIL",
			method:       http.MethodPost,
			path:         "/updates/",
			body:         `[{"id":"PollCount","type":"counter"}`,
			expectedCode: 500,
			expectedBody: "",
		},
		{
			mockStore: func(c *gomock.Controller) *mock_handlers.MockStorage {
				s := mock_handlers.NewMockStorage(c)
				return s
			},
			name:         "update_batch_metrics_badtype:FAIL",
			method:       http.MethodPost,
			path:         "/updates/",
			body:         `[{"delta":4,"id":"PollCount","type":"wrongtype"}]`,
			expectedCode: 400,
			expectedBody: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			s := tc.mockStore(ctrl)

			mr := NewMetricResource(s, cfg)
			r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			w := httptest.NewRecorder()
			mr.UpdateBatchJSON(w, r)
			resp := w.Result()
			assert.Equal(t, tc.expectedCode, resp.StatusCode)
			if err := resp.Body.Close(); err != nil {
				assert.Error(t, err)
			}
		})
	}
}

func BenchmarkGetAllMetrics(b *testing.B) {
	cfg := &models.Config{
		Address:         "http://localhost:8080",
		StoreInterval:   300,
		FileStoragePath: "",
		RestoreMetrics:  false,
		Logger:          nil,
		PostgresDSN:     "",
		ContextTimeout:  3,
		HashKey:         "",
	}
	s, err := storage.NewMemStorage(cfg)
	if err != nil {
		b.Fatal(err)
	}
	mr := NewMetricResource(s, cfg)

	r := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	w := httptest.NewRecorder()

	for i := 0; i < b.N; i++ {
		mr.GetAllMetrics(w, r)
		res := w.Result()
		if err := res.Body.Close(); err != nil {
			b.Error("failed to close response body")
		}
	}
}

func BenchmarkUpdateBatch(b *testing.B) {
	body := `[{"delta":4,"id":"PollCount","type":"counter"}, 
	{"value":240632,"id":"HeapAlloc","type":"gauge"},
	{"value":1757776,"id":"GCSys","type":"gauge"},
	{"value":2621440,"id":"HeapIdle","type":"gauge"},
	{"value":3702784,"id":"HeapSys","type":"gauge"},
	{"value":0,"id":"NumForcedGC","type":"gauge"},
	{"value":0,"id":"NumGC","type":"gauge"},
	{"value":491520,"id":"StackSys","type":"gauge"},
	{"value":941952,"id":"OtherSys","type":"gauge"},
	{"value":5023.92,"id":"CPUutilization1","type":"gauge"},
	{"value":1187.16,"id":"CPUutilization3","type":"gauge"},
	{"value":374.08,"id":"CPUutilization4","type":"gauge"},
	{"value":238.16,"id":"CPUutilization5","type":"gauge"},
	{"value":68.21,"id":"CPUutilization7","type":"gauge"},
	{"value":240632,"id":"Alloc","type":"gauge"},
	{"value":0,"id":"Lookups","type":"gauge"},
	{"value":5278.46,"id":"CPUutilization0","type":"gauge"},
	{"value":108.03,"id":"CPUutilization6","type":"gauge"},
	{"value":7696,"id":"BuckHashSys","type":"gauge"},
	{"value":1081344,"id":"HeapInuse","type":"gauge"},
	{"value":710,"id":"HeapObjects","type":"gauge"},
	{"value":44000,"id":"MSpanInuse","type":"gauge"},
	{"value":240632,"id":"TotalAlloc","type":"gauge"},
	{"value":17179869184,"id":"TotalMemory","type":"gauge"},
	{"value":2621440,"id":"HeapReleased","type":"gauge"},
	{"value":0,"id":"LastGC","type":"gauge"},
	{"value":0,"id":"PauseTotalNs","type":"gauge"},
	{"value":156581888,"id":"FreeMemory","type":"gauge"},
	{"value":46,"id":"Frees","type":"gauge"},
	{"value":48960,"id":"MSpanSys","type":"gauge"},
	{"value":756,"id":"Mallocs","type":"gauge"},
	{"value":4194304,"id":"NextGC","type":"gauge"},
	{"value":491520,"id":"StackInuse","type":"gauge"},
	{"value":1814.73,"id":"CPUutilization2","type":"gauge"},
	{"value":0,"id":"GCCPUFraction","type":"gauge"},
	{"value":9600,"id":"MCacheInuse","type":"gauge"},
	{"value":15600,"id":"MCacheSys","type":"gauge"},
	{"value":6966288,"id":"Sys","type":"gauge"},
	{"value":0.08536959506538425,"id":"RandomValue","type":"gauge"}]`

	logConfig := zap.NewDevelopmentConfig()
	logger, err := logConfig.Build()
	if err != nil {
		b.Error("failed to initialize Logger: %w", err)
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
		b.Fatal(err)
	}
	mr := NewMetricResource(s, cfg)

	for i := 0; i < b.N; i++ {
		r := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader(body))
		w := httptest.NewRecorder()
		mr.UpdateBatchJSON(w, r)
		res := w.Result()
		if err := res.Body.Close(); err != nil {
			b.Error("failed to close response body")
		}
	}
}
