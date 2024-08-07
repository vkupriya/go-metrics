package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/go-resty/resty/v2"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

type Collector struct {
	gauge        map[string]float64
	counter      map[string]int64
	config       Config
	gaugeMutex   sync.Mutex
	counterMutex sync.Mutex
}

type Metric struct {
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter

}

func NewCollector(cfg Config) *Collector {
	return &Collector{
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
		config:  cfg,
	}
}

func (c *Collector) collectMetrics() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	c.gaugeMutex.Lock()
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
	c.gaugeMutex.Unlock()

	c.counterMutex.Lock()
	c.counter[`PollCount`]++
	c.counterMutex.Unlock()
}

func (c *Collector) collectPsutilMetrics() {
	v, _ := mem.VirtualMemory()

	cp, _ := cpu.Times(true)

	c.gaugeMutex.Lock()
	c.gauge[`TotalMemory`] = float64(v.Total)
	c.gauge[`FreeMemory`] = float64(v.Free)
	c.gaugeMutex.Unlock()

	for i := range len(cp) {
		c.gauge[`CPUutilization`+strconv.Itoa(i)] = float64(cp[i].System)
	}
}

func (c *Collector) startSender(ctx context.Context, ch chan []Metric) {
	sendTicker := time.NewTicker(time.Duration(c.config.reportInterval) * time.Second)
	defer sendTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(ch)
			return
		case <-sendTicker.C:
			c.dispatcher(ch)
		}
	}
}

func (c *Collector) startCollector(ctx context.Context) {
	collectTicker := time.NewTicker(time.Duration(c.config.pollInterval) * time.Second)
	defer collectTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-collectTicker.C:
			c.collectMetrics()
			c.collectPsutilMetrics()
		}
	}
}

func (c *Collector) StartTickers(ctx context.Context) error {
	// Start tickers
	inputCh := make(chan []Metric, c.config.rateLimit)

	eg, egCtx := errgroup.WithContext(ctx)

	go c.startCollector(ctx)

	go c.startSender(ctx, inputCh)

	for w := 1; w <= c.config.rateLimit; w++ {
		eg.Go(func() error {
			if err := c.sendMetrics(egCtx, inputCh); err != nil {
				return fmt.Errorf("failed to send metrics: %w", err)
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("failed to run collector/sender go routines: %w", err)
	}
	return nil
}

func (c *Collector) dispatcher(ch chan []Metric) {
	logger := c.config.Logger
	c.counterMutex.Lock()
	metrics := make([]Metric, 0)
	for k, v := range c.counter {
		mtype := "counter"
		delta := v
		metrics = append(metrics, Metric{ID: k, MType: mtype, Delta: &delta})
	}
	c.counterMutex.Unlock()

	// Sending gauge metrics
	c.gaugeMutex.Lock()
	for k, v := range c.gauge {
		mtype := "gauge"
		value := v
		metrics = append(metrics, Metric{ID: k, MType: mtype, Value: &value})
	}
	c.gaugeMutex.Unlock()
	logger.Sugar().Debug("Posting metrics to channel")

	ch <- metrics
}

func (c *Collector) sendMetrics(ctx context.Context, ch chan []Metric) error {
	logger := c.config.Logger
	// Sending counter metrics
	const (
		retries    = 3
		retryDelay = 2
	)
	var retry int
	var metrics []Metric

	for {
		select {
		case <-ctx.Done():
			return nil
		case metrics = <-ch:
			retry = 0
			for retry <= retries {
				if retry == retries {
					return fmt.Errorf("failed to send metrics after %d", retries)
				}

				if err := c.metricPost(metrics, c.config.metricHost); err != nil {
					logger.Sugar().Errorf("failed http post metrics batch, retrying: %v\n", err)
				} else {
					break
				}
				time.Sleep(time.Duration(1+(retry*retryDelay)) * time.Second)
				retry++
			}
			// Resetting PollCount to 0 on successful Post
			c.counterMutex.Lock()
			c.counter["PollCount"] = 0
			c.counterMutex.Unlock()
		}
	}
}

func (c *Collector) metricPost(m []Metric, h string) error {
	logger := c.config.Logger
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

	if c.config.HashKey != "" {
		c.hashHeader(client, gz.Bytes())
	}
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(&gz).
		Post(url)

	if err != nil {
		return fmt.Errorf("error to do http post: %w", err)
	}

	logger.Sugar().Infof("sent metrics batch Status code: %d\n", resp.StatusCode())

	return nil
}

func (c *Collector) hashHeader(req *resty.Client, body []byte) {
	h := hmac.New(sha256.New, []byte(c.config.HashKey))
	h.Write(body)
	hdst := h.Sum(nil)

	req.Header.Set(`HashSHA256`, hex.EncodeToString(hdst))
}

func Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	c, err := NewConfig()
	if err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	collector := NewCollector(*c)
	if err := collector.StartTickers(ctx); err != nil {
		fmt.Println("Error in Start Tickers")
		return fmt.Errorf("failed to run tickers: %w", err)
	}

	return nil
}
