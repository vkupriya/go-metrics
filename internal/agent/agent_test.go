package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAbs(t *testing.T) {
	c := Config{}
	collector := NewCollector(c)
	collector.collectMetrics()

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
