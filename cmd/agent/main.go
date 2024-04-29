package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"time"
)

const (
	PollInterval   int64 = 2
	ReportInterval int64 = 10
)

type Collector struct {
	gauge   map[string]float64
	counter map[string]int64
}

func NewCollector() *Collector {
	return &Collector{
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
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
	collectTicker := time.NewTicker(time.Duration(PollInterval) * time.Second)
	defer collectTicker.Stop()

	sendTicker := time.NewTicker(time.Duration(ReportInterval) * time.Second)
	defer sendTicker.Stop()

	for {
		select {

		case <-collectTicker.C:
			c.collectMetrics()
		case <-sendTicker.C:
			go c.sendMetrics()
		}
	}
}

func (c *Collector) sendMetrics() error {

	// Sending counter metrics

	for k, v := range c.counter {

		mvalue := fmt.Sprintf("%d", v)
		url := fmt.Sprintf("http://localhost:8080/update/counter/%s/%s", k, mvalue)

		req, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			return err
		}
		req.Header.Set(`Content-Type`, `text/plain`)

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		fmt.Printf("Sent counter metric: %s, Status code: %d\n", k, res.StatusCode)
	}

	// Sending gauge metrics

	for k, v := range c.gauge {

		mvalue := fmt.Sprintf("%.02f", v)
		url := fmt.Sprintf("http://localhost:8080/update/gauge/%s/%s", k, mvalue)

		req, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			return err
		}
		req.Header.Set(`Content-Type`, `text/plain`)

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		fmt.Printf("Sent counter metric: %s, Status code: %d\n", k, res.StatusCode)
	}

	return nil
}

func main() {
	collector := NewCollector()

	collector.StartTickers()

}
