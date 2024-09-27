// Package middleware provides custom middlewares: Logger, Gzip and Hash for metric REST API service.
package middleware

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"io"
	"net/http"

	"github.com/vkupriya/go-metrics/internal/server/models"
	"go.uber.org/zap"
)

type MiddlewareDecrypt struct {
	config *models.Config
}

func NewMiddlewareDecrypt(c *models.Config) *MiddlewareDecrypt {
	return &MiddlewareDecrypt{
		config: c,
	}
}

func (d *MiddlewareDecrypt) DecryptHandle(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := d.config.Logger
		if len(d.config.SecretKey) == 0 {
			h.ServeHTTP(w, r)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Sugar().Error("failed to read request body", zap.Error(err))
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		body, _ = hex.DecodeString(string(body))

		block, err := aes.NewCipher(d.config.SecretKey)
		if err != nil {
			logger.Sugar().Error("failed to create new cypher block", zap.Error(err))
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		aesgcm, err := cipher.NewGCM(block)
		if err != nil {
			logger.Sugar().Error("failed to create new GCM block", zap.Error(err))
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		nonce, body := body[:aesgcm.NonceSize()], body[aesgcm.NonceSize():]

		srcBody, err := aesgcm.Open(nil, nonce, body, nil)
		if err != nil {
			logger.Sugar().Error("failed to decrypt body", zap.Error(err))
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		r.Body = io.NopCloser(bytes.NewBuffer(srcBody))

		h.ServeHTTP(w, r)
	})
}
