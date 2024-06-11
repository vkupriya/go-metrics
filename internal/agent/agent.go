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

type Metric struct {
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
				return fmt.Errorf("error while sending metrics to server: %w", err)
			}
		}
	}
}

func (c *Collector) sendMetrics() error {
	// Sending counter metrics
	metrics := make([]Metric, 0)
	for k, v := range c.counter {
		mtype := "counter"
		delta := v
		metrics = append(metrics, Metric{ID: k, MType: mtype, Delta: &delta})
	}
	// Resetting PollCount to 0
	c.counter["PollCount"] = 0

	// Sending gauge metrics
	for k, v := range c.gauge {
		mtype := "gauge"
		value := v
		metrics = append(metrics, Metric{ID: k, MType: mtype, Value: &value})
	}
	if metrics != nil {
		var (
			retries    = 3
			retry      = 0
			retryDelay = 2
		)
		for retry <= retries {
			if err := metricPost(metrics, c.config.metricHost); err != nil {
				log.Print("failed http post metrics batch, retrying\n")
				if retry == retries {
					return fmt.Errorf("failed http post metrics batch: %w", err)
				}
			} else {
				break
			}
			time.Sleep(time.Duration(1+(retry*retryDelay)) * time.Second)
			retry++
		}
	}
	return nil
}

func metricPost(m []Metric, h string) error {
	const httpTimeout int = 30
	client := resty.New()
	client.SetTimeout(time.Duration(httpTimeout) * time.Second)

	url := fmt.Sprintf("http://%s/updates/", h)

	body, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("error encoding JSON response for metrics batch: %w", err)
	}
	var gz bytes.Buffer
	w := gzip.NewWriter(&gz)
	_, err = w.Write(body)
	if err != nil {
		return fmt.Errorf("failed to write into gzip.NewWriter metrics batch: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close gzip.NewWriter for metrics batch: %w", err)
	}

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(&gz).
		Post(url)

	if err != nil {
		return fmt.Errorf("error to do http post: %w", err)
	}

	fmt.Printf("Sent metrics batch Status code: %d\n", resp.StatusCode())

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
