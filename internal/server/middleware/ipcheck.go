package middleware

import (
	"net"
	"net/http"

	"github.com/vkupriya/go-metrics/internal/server/models"
)

type MiddlewareIPCheck struct {
	config *models.Config
}

func NewMiddlewareIPCheck(c *models.Config) *MiddlewareIPCheck {
	return &MiddlewareIPCheck{
		config: c,
	}
}

func (i *MiddlewareIPCheck) IPCheckHandle(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := i.config.Logger
		if i.config.TrustedSubnet == nil {
			h.ServeHTTP(w, r)
			return
		}

		ipStr := r.Header.Get("X-Real-IP")
		ip := net.ParseIP(ipStr)

		if !i.config.TrustedSubnet.Contains(ip) {
			logger.Sugar().Error("agent source IP is not trusted")
			http.Error(w, "agent source IP is not trusted", http.StatusBadRequest)
			return
		}

		h.ServeHTTP(w, r)
	})
}
