package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vkupriya/go-metrics/internal/server/storage"
)

func TestUpdateMetricHandler(t *testing.T) {
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
			s := storage.NewMemStorage()
			mr := NewMetricResource(s)

			request := httptest.NewRequest(tt.args.method, tt.args.path, nil)

			w := httptest.NewRecorder()
			h := http.HandlerFunc(mr.UpdateMetric)
			h(w, request)

			result := w.Result()
			assert.Equal(t, tt.wantCode, result.StatusCode)

			if err := result.Body.Close(); err != nil {
				panic(err)
			}
		})
	}
}
