package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"time"

	"github.com/go-resty/resty/v2"
)

type Collector struct {
	gauge   map[string]float64
	counter map[string]int64
	config  Config
}

type Metrics struct {
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
}

func NewCollector(c Config) *Collector {
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

func (c *Collector) StartTickers() error {
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
				log.Printf("error while sending metrics to server: %v", err)
			}
		}
	}
}

func (c *Collector) sendMetrics() error {
	// Sending counter metrics
	for k, v := range c.counter {
		mtype := "counter"
		if err := metricPost(Metrics{ID: k, MType: mtype, Delta: &v}, c.config.metricHost); err != nil {
			return fmt.Errorf("failed http post for %s metric %s: %w", mtype, k, err)
		}
	}
	// Resetting PollCount to 0
	c.counter["PollCount"] = 0

	// Sending gauge metrics
	for k, v := range c.gauge {
		mtype := "gauge"
		if err := metricPost(Metrics{ID: k, MType: mtype, Value: &v}, c.config.metricHost); err != nil {
			return fmt.Errorf("failed http post for %s metric %s: %w", mtype, k, err)
		}
	}
	return nil
}

func metricPost(m Metrics, h string) error {
	const httpTimeout int = 30
	client := resty.New()
	client.SetTimeout(time.Duration(httpTimeout) * time.Second)

	url := fmt.Sprintf("http://%s/update/", h)

	body, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("error encoding JSON response for %s for metric %s: %w", m.MType, m.ID, err)
	}
	var gz bytes.Buffer
	w := gzip.NewWriter(&gz)
	w.Write(body)
	w.Close()

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(&gz).
		Post(url)

	if err != nil {
		return fmt.Errorf("error to do http post: %w", err)
	}

	fmt.Printf("Sent %s metric: %s, Status code: %d\n", m.MType, m.ID, resp.StatusCode())

	return nil
}

func Start() error {
	c, err := NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	collector := NewCollector(*c)
	if err := collector.StartTickers(); err != nil {
		return err
	}

	return nil
}
