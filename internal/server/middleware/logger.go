package middleware

import (
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type (
	responseData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
		done         bool
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	if err != nil {
		return 0, fmt.Errorf("failed to write into http.ResponseWriter: %w", err)
	}
	r.responseData.size += size
	if !r.done {
		r.responseData.status = http.StatusOK
		r.done = true
	}
	return size, nil
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	if !r.done {
		r.responseData.status = statusCode
		r.done = true
	}
}

func Logging(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		logger, _ := zap.NewDevelopment()

		start := time.Now()

		responseData := &responseData{
			status: 0,
			size:   0,
		}

		lw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
			done:           false,
		}

		uri := r.RequestURI
		method := r.Method

		h.ServeHTTP(&lw, r)

		duration := time.Since(start)
		logger.Sugar().Infoln(
			"uri", uri,
			"method", method,
			"status", responseData.status,
			"duration", duration,
			"size", responseData.size,
		)
	}
	return http.HandlerFunc(logFn)
}
