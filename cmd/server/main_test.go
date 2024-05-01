package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
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
			name: "positive test: update gauge metric",
			args: args{
				path:   "/update/gauge/test/20.0",
				method: http.MethodPost,
			},
			wantCode: 200,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMemStorage()
			mr := NewMetricResource(s)

			request := httptest.NewRequest(tt.args.method, tt.args.path, nil)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(mr.UpdateMetric)
			h(w, request)

			result := w.Result()

			assert.Equal(t, tt.wantCode, result.StatusCode)

		})
	}

}
