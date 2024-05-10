package agent

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	pollIntDefault   int64 = 2
	reportIntDefault int64 = 10
)

var (
	metricHost     = flag.String("a", "localhost:8080", "Address and port of the metric server.")
	reportInterval = flag.Int64("r", reportIntDefault, "Metrics report interval in seconds.")
	pollInterval   = flag.Int64("p", pollIntDefault, "Metric collection interval in seconds")
)

type config struct {
	metricHost     string
	reportInterval int64
	pollInterval   int64
}

type Collector struct {
	gauge   map[string]float64
	counter map[string]int64
	config  config
}

func NewCollector(c config) *Collector {
	return &Collector{
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
		config:  c,
	}
}

func (c *Collector) collectMetrics() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	c.gauge[`Alloc`] = float64(memStats.Alloc)
	c.gauge[`BuckHashSys`] = float64(memStats.BuckHashSys)
	c.gauge[`Frees`] = float64(memStats.Frees)
	c.gauge[`GCCPUFraction`] = float64(memStats.GCCPUFraction)
	c.gauge[`GCSys`] = float64(memStats.GCSys)
	c.gauge[`HeapAlloc`] = float64(memStats.HeapAlloc)
	c.gauge[`HeapIdle`] = float64(memStats.HeapIdle)
	c.gauge[`HeapInuse`] = float64(memStats.HeapInuse)
	c.gauge[`HeapReleased`] = float64(memStats.HeapReleased)
	c.gauge[`HeapObjects`] = float64(memStats.HeapObjects)
	c.gauge[`HeapSys`] = float64(memStats.HeapSys)
	c.gauge[`LastGC`] = float64(memStats.LastGC)
	c.gauge[`Lookups`] = float64(memStats.Lookups)
	c.gauge[`MCacheInuse`] = float64(memStats.MCacheInuse)
	c.gauge[`MCacheSys`] = float64(memStats.MCacheSys)
	c.gauge[`MSpanInuse`] = float64(memStats.MSpanInuse)
	c.gauge[`MSpanSys`] = float64(memStats.MSpanSys)
	c.gauge[`Mallocs`] = float64(memStats.Mallocs)
	c.gauge[`NextGC`] = float64(memStats.NextGC)
	c.gauge[`NumForcedGC`] = float64(memStats.NumForcedGC)
	c.gauge[`NumGC`] = float64(memStats.NumGC)
	c.gauge[`OtherSys`] = float64(memStats.OtherSys)
	c.gauge[`PauseTotalNs`] = float64(memStats.PauseTotalNs)
	c.gauge[`StackInuse`] = float64(memStats.StackInuse)
	c.gauge[`StackSys`] = float64(memStats.StackSys)
	c.gauge[`Sys`] = float64(memStats.Sys)
	c.gauge[`TotalAlloc`] = float64(memStats.TotalAlloc)
	c.gauge[`RandomValue`] = rand.Float64()

	c.counter[`PollCount`]++
}

func (c *Collector) StartTickers() {
	// Start tickers

	collectTicker := time.NewTicker(time.Duration(c.config.pollInterval) * time.Second)
	defer collectTicker.Stop()

	sendTicker := time.NewTicker(time.Duration(c.config.reportInterval) * time.Second)
	defer sendTicker.Stop()

	for {
		select {
		case <-collectTicker.C:
			c.collectMetrics()
		case <-sendTicker.C:
			if err := c.sendMetrics(); err != nil {
				fmt.Println(err)
			}
		}
	}
}

func (c *Collector) sendMetrics() error {
	// Sending counter metrics
	for k, v := range c.counter {
		mvalue := strconv.FormatInt(v, 10)
		mtype := "counter"

		if err := metricPost(mtype, k, mvalue, c.config.metricHost); err != nil {
			return fmt.Errorf("failed http post for %s metric %s: %w", mtype, k, err)
		}
	}
	// Resetting PollCount to 0
	c.counter["PollCount"] = 0

	// Sending gauge metrics
	for k, v := range c.gauge {
		mvalue := fmt.Sprintf("%.02f", v)
		mtype := "gauge"

		if err := metricPost(mtype, k, mvalue, c.config.metricHost); err != nil {
			return fmt.Errorf("failed http post for %s metric %s: %w", mtype, k, err)
		}
	}
	return nil
}

func metricPost(t string, m string, v string, h string) error {
	const httpTimeout int = 30
	client := resty.New()
	client.SetTimeout(time.Duration(httpTimeout) * time.Second)

	url := fmt.Sprintf("http://%s/update/%s/%s/%s", h, t, m, v)

	resp, err := client.R().
		SetHeader("Content-Type", "text/plain").
		Post(url)

	if err != nil {
		return fmt.Errorf("error to do http post: %w", err)
	}

	fmt.Printf("Sent %s metric: %s, Status code: %d\n", t, m, resp.StatusCode())

	return nil
}

func Start() {
	flag.Parse()
	c := config{
		metricHost:     *metricHost,
		pollInterval:   *pollInterval,
		reportInterval: *reportInterval,
	}

	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		c.metricHost = envAddr
	}

	if envPoll := os.Getenv("POLL_INTERVAL"); envPoll != "" {
		envPollInt, err := strconv.ParseInt(envPoll, 10, 64)
		if err != nil {
			panic(err)
		}
		c.pollInterval = envPollInt
	}

	if envReport := os.Getenv("REPORT_INTERVAL"); envReport != "" {
		envReportInt, err := strconv.ParseInt(envReport, 10, 64)
		if err != nil {
			panic(err)
		}
		c.reportInterval = envReportInt
	}

	collector := NewCollector(c)
	collector.StartTickers()
}
