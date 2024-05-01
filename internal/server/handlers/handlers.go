package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/vkupriya/go-metrics/internal/server/storage"
)

const (
	pos2 int = 2
	pos3 int = 3
	pos4 int = 4
)

type MetricResource struct {
	storage storage.Storage
}

func NewMetricResource(storage storage.Storage) *MetricResource {
	return &MetricResource{storage: storage}
}

func (mr *MetricResource) UpdateMetric(rw http.ResponseWriter, r *http.Request) {
	var (
		mtype  string
		mname  string
		mvalue string
	)

	url := r.URL.RequestURI()

	urlParams := strings.Split(url, "/")

	for i, v := range urlParams {
		switch {
		case i == pos2:
			mtype = v
		case i == pos3:
			mname = v
		case i == pos4:
			mvalue = v
		}
	}

	if r.Method != http.MethodPost {
		rw.WriteHeader(http.StatusBadRequest)
		return
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
			fmt.Printf("Updated gauge metric %s with value %f\n", mname, res)
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
