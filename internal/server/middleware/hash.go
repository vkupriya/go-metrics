package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"

	"go.uber.org/zap"

	"github.com/vkupriya/go-metrics/internal/server/models"
)

type MiddlewareHash struct {
	config *models.Config
}

func NewMiddlewareHash(c *models.Config) *MiddlewareHash {
	return &MiddlewareHash{
		config: c,
	}
}

func (m *MiddlewareHash) HashCheck(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		logger := m.config.Logger

		var reqHash string

		if m.config.HashKey == "" {
			return
		}

		reqHash = r.Header.Get("HashSHA256")

		if reqHash == "" {
			return
		}

		sig, err := hex.DecodeString(reqHash)
		if err != nil {
			logger.Sugar().Debug("failed to decode hex string", zap.Error(err))
			http.Error(w, "", http.StatusBadRequest)
			return
		}

		b, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Sugar().Error("failed to read request body", zap.Error(err))
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewBuffer(b))
		mac := hmac.New(sha256.New, []byte(m.config.HashKey))
		mac.Write(b)
		if !hmac.Equal(sig, mac.Sum(nil)) {
			logger.Sugar().Debug("hmac signature does not match.")
			http.Error(w, "", http.StatusBadRequest)
			return
		}

		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(logFn)
}

func (m *MiddlewareHash) HashSend(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		logger := m.config.Logger

		var reqHash string

		if m.config.HashKey == "" {
			return
		}

		if reqHash == "" {
			return
		}

		b, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Sugar().Debug("failed to read request body", zap.Error(err))
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewBuffer(b))
		mac := hmac.New(sha256.New, []byte(m.config.HashKey))
		mac.Write(b)
		hdst := mac.Sum(nil)
		w.Header().Set(`HashSHA256`, hex.EncodeToString(hdst))

		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(logFn)
}
