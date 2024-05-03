package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/vkupriya/go-metrics/internal/server/storage"
)

const (
	counter string = "counter"
	gauge   string = "gauge"
)

type MetricResource struct {
	storage storage.Storage
}

func NewMetricResource(storage storage.Storage) *MetricResource {
	return &MetricResource{storage: storage}
}

func NewMetricRouter(mr *MetricResource) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.AllowContentType("text/plain"))
	r.Get("/value/{metricType}/{metricName}", mr.GetMetric)
	r.Post("/update/{metricType}/{metricName}/{metricValue}", mr.UpdateMetric)

	return r
}

func (mr *MetricResource) UpdateMetric(rw http.ResponseWriter, r *http.Request) {
	mtype := chi.URLParam(r, "metricType")
	mname := chi.URLParam(r, "metricName")
	mvalue := chi.URLParam(r, "metricValue")

	if mtype != gauge && mtype != counter {
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
		case mtype == gauge:
			mv, err := strconv.ParseFloat(mvalue, 64)
			if err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			mr.storage.UpdateGaugeMetric(mname, mv)
			rw.WriteHeader(http.StatusOK)

		case mtype == counter:
			mv, err := strconv.ParseInt(mvalue, 10, 64)
			if err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			mr.storage.UpdateCounterMetric(mname, mv)
			rw.WriteHeader(http.StatusOK)
		}
		return
	}
}

func (mr *MetricResource) GetMetric(rw http.ResponseWriter, r *http.Request) {
	mtype := chi.URLParam(r, "metricType")
	mname := chi.URLParam(r, "metricName")

	if mtype != gauge && mtype != counter {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	switch {
	case mtype == gauge:
		v, err := mr.storage.GetGaugeMetric(mname)
		if err != nil {
			rw.WriteHeader(http.StatusNotFound)
			return
		} else {
			if _, err := io.WriteString(rw, fmt.Sprintf("%.03f", v)); err != nil {
				panic(err)
			}
			rw.WriteHeader(http.StatusOK)
			return
		}
	case mtype == counter:
		v, err := mr.storage.GetCounterMetric(mname)
		if err != nil {
			rw.WriteHeader(http.StatusNotFound)
			return
		} else {
			if _, err := io.WriteString(rw, fmt.Sprintf("%d", v)); err != nil {
				panic(err)
			}
			rw.WriteHeader(http.StatusOK)
			return
		}
	}
}
