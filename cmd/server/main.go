package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type Storage interface {
	UpdateGaugeMetric(name string, value float64) float64
	UpdateCounterMetric(name string, value int64) int64
}

type MemStorage struct {
	gauge   map[string]float64
	counter map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
	}
}

type MetricResource struct {
	storage Storage
}

func NewMetricResource(storage Storage) *MetricResource {
	return &MetricResource{storage: storage}
}

func (m *MemStorage) UpdateGaugeMetric(name string, value float64) float64 {

	m.gauge[name] = value
	return m.gauge[name]

}

func (m *MemStorage) UpdateCounterMetric(name string, value int64) int64 {

	m.counter[name] += value
	return m.counter[name]

}

func (mr *MetricResource) UpdateMetric(rw http.ResponseWriter, r *http.Request) {

	var (
		mtype  string
		mname  string
		mvalue string
	)

	if r.Method != "POST" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	if r.Header.Get("Content-Type") != "text/plain" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	url := r.URL.RequestURI()

	urlParams := strings.Split(url, "/")

	for i, v := range urlParams {
		switch {
		case i == 2:
			mtype = v
		case i == 3:
			mname = v
		case i == 4:
			mvalue = v

		}
	}
	if mtype != "gauge" && mtype != "counter" {
		rw.WriteHeader(http.StatusBadRequest)
		return

	}

	if mname == "" {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	if mvalue == "" {
		rw.WriteHeader(http.StatusBadRequest)
	}

	if mtype != "" && mname != "" && mvalue != "" {

		switch {
		case mtype == "gauge":
			mv, err := strconv.ParseFloat(mvalue, 64)
			if err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			res := mr.storage.UpdateGaugeMetric(mname, mv)
			fmt.Printf("Updated gauge metric %s with value %f", mname, res)
			rw.WriteHeader(http.StatusOK)

		case mtype == "counter":
			mv, err := strconv.ParseInt(mvalue, 10, 64)
			if err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			res := mr.storage.UpdateCounterMetric(mname, mv)
			fmt.Printf("Updated counter metric %s, new value is %d\n", mname, res)
			rw.WriteHeader(http.StatusOK)
		}
		return
	}
}

func main() {

	s := NewMemStorage()
	mr := NewMetricResource(s)

	mux := http.NewServeMux()

	mux.HandleFunc("/update/", mr.UpdateMetric)

	err := http.ListenAndServe(":8080", mux)

	if err != nil {
		panic(err)
	}

}
