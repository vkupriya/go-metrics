package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vkupriya/go-metrics/internal/server/config"
	"github.com/vkupriya/go-metrics/internal/server/models"
	"github.com/vkupriya/go-metrics/internal/server/storage"
)

func testRequest(t *testing.T, ts *httptest.Server, method, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, ts.URL+path, http.NoBody)
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	if err := resp.Body.Close(); err != nil {
		panic(err)
	}

	return resp
}

func TestUpdateMetric(t *testing.T) {
	cfg, err := config.NewConfig()
	if err != nil {
		t.Fatal(err)
	}
	s, err := storage.NewMemStorage(cfg)
	if err != nil {
		t.Fatal(err)
	}
	mr := NewMetricResource(s, cfg)

	ts := httptest.NewServer(NewMetricRouter(mr))

	type args struct {
		path   string
		method string
	}
	tests := []struct {
		name     string
		args     args
		wantCode int
	}{
		{
			name: "positive test #1: update gauge metric",
			args: args{
				path:   "/update/gauge/test/20.0",
				method: http.MethodPost,
			},
			wantCode: 200,
		},
		{
			name: "positive test #2: update counter metric",
			args: args{
				path:   "/update/counter/test/20",
				method: http.MethodPost,
			},
			wantCode: 200,
		},
		{
			name: "negative test #3: invalid metric type",
			args: args{
				path:   "/update/timeseries/test/20",
				method: http.MethodPost,
			},
			wantCode: 400,
		},
		{
			name: "negative test #4: missing metric name",
			args: args{
				path:   "/update/counter",
				method: http.MethodPost,
			},
			wantCode: 404,
		},
		{
			name: "negative test #5: wrong counter metric value type",
			args: args{
				path:   "/update/counter/test/20.01",
				method: http.MethodPost,
			},
			wantCode: 400,
		},
		{
			name: "negative test #6: wrong gauge metric value type",
			args: args{
				path:   "/update/gauge/test/string",
				method: http.MethodPost,
			},
			wantCode: 400,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testRequest(t, ts, tt.args.method, tt.args.path)
			assert.Equal(t, tt.wantCode, resp.StatusCode)
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
