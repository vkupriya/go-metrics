package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	t.Setenv("DATABASE_DSN", "postgres://sysmetrics:metrics@localhost:5432/metrics?sslmode=disable")
	t.Setenv("ADDRESS", "localhost:8443")
	t.Setenv("STORE_INTERVAL", "300")
	t.Setenv("KEY", "ksjdflksjdf")
	t.Setenv("FILE_STORAGE_PATH", "/tmp/store.txt")
	t.Setenv("RESTORE", "false")
	t.Setenv("CRYPTO_KEY", "../../../keys/private.pem")
	t.Run("test01", func(t *testing.T) {
		t.Setenv("DATABASE_DSN", "postgres://sysmetrics:metrics@localhost:5432/metrics?sslmode=disable")
		c, err := NewConfig()
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, c.PostgresDSN, "postgres://sysmetrics:metrics@localhost:5432/metrics?sslmode=disable")
		assert.Equal(t, c.Address, "localhost:8443")
		assert.Equal(t, c.StoreInterval, int64(300))
		assert.Equal(t, c.HashKey, "ksjdflksjdf")
		assert.Equal(t, c.FileStoragePath, "/tmp/store.txt")
		assert.Equal(t, c.RestoreMetrics, false)
	})
}
