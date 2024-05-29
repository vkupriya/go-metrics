package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vkupriya/go-metrics/internal/server/config"
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
