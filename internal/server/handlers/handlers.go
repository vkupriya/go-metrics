package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	logger "github.com/vkupriya/go-metrics/internal/server/middleware"
	"github.com/vkupriya/go-metrics/internal/server/models"
)

const tmpl string = `
	<!doctype html>

	<body>
		<ul>
		{{ range $key, $value := . }}
			<li><b>{{ $key }}</b>: {{ $value }}</li>
		{{ end }}
		</ul>
	</body>

	</html>
`

type Storage interface {
	UpdateGaugeMetric(name string, value float64) float64
	UpdateCounterMetric(name string, value int64) int64
	GetCounterMetric(name string) (int64, error)
	GetGaugeMetric(name string) (float64, error)
	GetAllValues() (map[string]float64, map[string]int64)
}

const (
	counter string = "counter"
	gauge   string = "gauge"
)

type MetricResource struct {
	store Storage
}

var sugar = zap.L().Sugar()

func NewMetricResource(store Storage) *MetricResource {
	return &MetricResource{store: store}
}

func NewMetricRouter(mr *MetricResource) chi.Router {
	r := chi.NewRouter()

	// r.Use(middleware.Logger)
	// r.Use(middleware.AllowContentType("text/plain"))
	r.Use(logger.Logging)

	r.Get("/", mr.GetAllMetrics)
	r.Get("/value/{metricType}/{metricName}", mr.GetMetric)
	r.Post("/value/", mr.GetMetricJSON)
	r.Post("/update/", mr.UpdateMetricJSON)
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
			mr.store.UpdateGaugeMetric(mname, mv)
			rw.WriteHeader(http.StatusOK)

		case mtype == counter:
			mv, err := strconv.ParseInt(mvalue, 10, 64)
			if err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			mr.store.UpdateCounterMetric(mname, mv)
			rw.WriteHeader(http.StatusOK)
		}
		return
	}
}

func (mr *MetricResource) UpdateMetricJSON(rw http.ResponseWriter, r *http.Request) {
	var req models.Metrics
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		sugar.Debug("cannot decode request JSON body", zap.Error(err))
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if req.MType != "counter" && req.MType != "gauge" {
		sugar.Debug("unsupported metric type", zap.String("type", req.MType))
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	mtype := req.MType
	mname := req.ID

	rw.Header().Set("Content-Type", "application/json")

	switch {
	case mtype == gauge:
		if req.Value == nil {
			sugar.Debug("request contains empty value for metric", zap.String("id", req.ID))
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		*req.Value = mr.store.UpdateGaugeMetric(mname, *req.Value)

	case mtype == counter:
		if req.Delta == nil {
			sugar.Debug("request contains empty value for metric", zap.String("id", req.ID))
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		*req.Delta = mr.store.UpdateCounterMetric(mname, *req.Delta)
	}
	enc := json.NewEncoder(rw)
	if err := enc.Encode(req); err != nil {
		sugar.Debug("error encoding JSON response", zap.Error(err))
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
		v, err := mr.store.GetGaugeMetric(mname)
		if err != nil {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		if _, err := rw.Write([]byte(strconv.FormatFloat(v, 'f', -1, 64))); err != nil {
			log.Printf("failed to write into response writer value for metric %s: %v", mname, err)
			http.Error(rw, "", http.StatusInternalServerError)
			return
		}

	case mtype == counter:
		v, err := mr.store.GetCounterMetric(mname)
		if err != nil {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		if _, err := rw.Write([]byte(strconv.FormatInt(v, 10))); err != nil {
			log.Printf("failed to write into response writer value for metric %s: %v", mname, err)
			http.Error(rw, "", http.StatusInternalServerError)
			return
		}
	}
}

func (mr *MetricResource) GetMetricJSON(rw http.ResponseWriter, r *http.Request) {
	var req models.Metrics
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		sugar.Debug("cannot decode request JSON body", zap.Error(err))
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if req.MType != "counter" && req.MType != "gauge" {
		sugar.Debug("unsupported metric type", zap.String("type", req.MType))
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	mtype := req.MType
	mname := req.ID
	fmt.Println(req)

	rw.Header().Set("Content-Type", "application/json")

	switch {
	case mtype == gauge:
		v, err := mr.store.GetGaugeMetric(mname)
		if err != nil {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		req.Value = &v

	case mtype == counter:
		v, err := mr.store.GetCounterMetric(mname)
		if err != nil {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		fmt.Println(v)
		req.Delta = &v
	}

	rw.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(rw)
	if err := enc.Encode(req); err != nil {
		sugar.Debug("error encoding JSON response", zap.Error(err))
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (mr *MetricResource) GetAllMetrics(rw http.ResponseWriter, r *http.Request) {
	gauge, counter := mr.store.GetAllValues()

	allMetrics := make(map[string]any)

	for name, value := range gauge {
		allMetrics[name] = value
	}

	for name, value := range counter {
		allMetrics[name] = value
	}

	t, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		log.Printf("failed to load http template: %v", err)
		http.Error(rw, "", http.StatusInternalServerError)
		return
	}

	if err := t.Execute(rw, allMetrics); err != nil {
		log.Printf("failed to execute http template: %v", err)
		http.Error(rw, "", http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
}
