package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAbs(t *testing.T) {
	c := Config{}
	collector := NewCollector(&c)
	collector.collectMetrics()
	collector.collectPsutilMetrics()

	tests := []struct {
		name    string
		mname   string
		mtype   string
		present bool
	}{
		{
			name:    "Test#1 Success - gauge runtime metric 'Alloc'.",
			mname:   "Alloc",
			mtype:   "gauge",
			present: true,
		},
		{
			name:    "Test#2 Failure - gauge runtime metric 'Unknown'.",
			mname:   "Unknown",
			mtype:   "gauge",
			present: false,
		},
		{
			name:    "Test#3 Success - counter metric 'PollCount'.",
			mname:   "PollCount",
			mtype:   "counter",
			present: true,
		},
		{
			name:    "Test#4 Success - gauge metric 'TotalMemory'.",
			mname:   "TotalMemory",
			mtype:   "gauge",
			present: true,
		},
		{
			name:    "Test#5 Success - counter metric 'FreeMemory'.",
			mname:   "FreeMemory",
			mtype:   "gauge",
			present: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			switch {
			case test.mtype == "gauge":
				_, ok := collector.gauge[test.mname]
				assert.Equal(t, test.present, ok)
			case test.mtype == "counter":
				_, ok := collector.counter[test.mname]
				assert.Equal(t, test.present, ok)
			}
		})
	}
}

func TestSendMetrics(t *testing.T) {
	c := Config{}
	c.SecretKey, _ = generateRandom(32)
	collector := NewCollector(&c)

	var f = 27873.01
	metrics := []Metric{
		{
			Value: &f,
			ID:    "testgauge01",
			MType: "gauge",
		},
		{
			Value: &f,
			ID:    "testgauge02",
			MType: "gauge",
		},
	}

	t.Run("test01", func(t *testing.T) {
		err := collector.metricPost(metrics, "localhost:8080")
		require.Error(t, err)
	})
}

func TestConfig(t *testing.T) {
	t.Setenv("ADDRESS", "localhost:8443")
	t.Setenv("RATE_LIMIT", "5")
	t.Setenv("KEY", "ksjdflksjdf")
	t.Setenv("POLL_INTERVAL", "10")
	t.Setenv("REPORT_INTERVAL", "20")
	t.Setenv("CONFIG", "../../cmd/agent/config.json")

	t.Run("test01", func(t *testing.T) {
		c, err := NewConfig()
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, c.MetricHost, "localhost:8443")
		assert.Equal(t, c.rateLimit, 5)
		assert.Equal(t, c.HashKey, "ksjdflksjdf")
		assert.Equal(t, c.PollInterval, int64(10))
		assert.Equal(t, c.ReportInterval, int64(20))
	})
}

func TestGenerateRandom(t *testing.T) {
	res, err := generateRandom(32)
	if err != nil {
		t.Error("failed to generate random sequence.")
	}
	if len(res) != int(32) {
		t.Error("incorrect length of random sequence.")
	}
}
