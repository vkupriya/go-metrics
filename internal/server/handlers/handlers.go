package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"

	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	mw "github.com/vkupriya/go-metrics/internal/server/middleware"
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
	UpdateGaugeMetric(c *models.Config, name string, value float64) (float64, error)
	UpdateCounterMetric(c *models.Config, name string, value int64) (int64, error)
	GetCounterMetric(name string) (int64, error)
	GetGaugeMetric(name string) (float64, error)
	GetAllValues() (map[string]float64, map[string]int64)
}

const (
	counter string = "counter"
	gauge   string = "gauge"
)

type MetricResource struct {
	store  Storage
	config *models.Config
}

func NewMetricResource(store Storage, cfg *models.Config) *MetricResource {
	return &MetricResource{
		store:  store,
		config: cfg}
}

func NewMetricRouter(mr *MetricResource) chi.Router {
	r := chi.NewRouter()

	r.Use(mw.Logging)
	r.Use(mw.Compress)

	r.Get("/", mr.GetAllMetrics)
	r.Get("/value/{metricType}/{metricName}", mr.GetMetric)
	r.Post("/value/", mr.GetMetricJSON)
	r.Post("/update/", mr.UpdateMetricJSON)
	r.Post("/update/{metricType}/{metricName}/{metricValue}", mr.UpdateMetric)

	return r
}

func (mr *MetricResource) UpdateMetric(rw http.ResponseWriter, r *http.Request) {
	logger := mr.config.Logger

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
			_, err = mr.store.UpdateGaugeMetric(mr.config, mname, mv)
			if err != nil {
				logger.Sugar().Error("failed to update gauge metric", err)
			}
			rw.WriteHeader(http.StatusOK)

		case mtype == counter:
			mv, err := strconv.ParseInt(mvalue, 10, 64)
			if err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			_, err = mr.store.UpdateCounterMetric(mr.config, mname, mv)
			if err != nil {
				logger.Sugar().Error("failed to update counter metric", err)
			}
			rw.WriteHeader(http.StatusOK)
		}
		return
	}
}

func (mr *MetricResource) UpdateMetricJSON(rw http.ResponseWriter, r *http.Request) {
	var req models.Metrics
	logger := mr.config.Logger

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		logger.Sugar().Debug("cannot decode request JSON body", zap.Error(err))
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if req.MType != "counter" && req.MType != "gauge" {
		logger.Sugar().Debug("unsupported metric type", zap.String("type", req.MType))
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	mtype := req.MType
	mname := req.ID

	rw.Header().Set("Content-Type", "application/json")

	switch {
	case mtype == gauge:
		if req.Value == nil {
			logger.Sugar().Debug("request contains empty value for metric", zap.String("id", req.ID))
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		rv, err := mr.store.UpdateGaugeMetric(mr.config, mname, *req.Value)
		if err != nil {
			logger.Sugar().Error("failed to update gauge metric", err)
		}
		*req.Value = rv

	case mtype == counter:
		if req.Delta == nil {
			logger.Sugar().Debug("request contains empty value for metric", zap.String("id", req.ID))
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		rd, err := mr.store.UpdateCounterMetric(mr.config, mname, *req.Delta)
		if err != nil {
			logger.Sugar().Error("failed to update counter metric", err)
		}
		*req.Delta = rd
	}
	enc := json.NewEncoder(rw)
	if err := enc.Encode(req); err != nil {
		logger.Sugar().Debug("error encoding JSON response", zap.Error(err))
		return
	}
}

func (mr *MetricResource) GetMetric(rw http.ResponseWriter, r *http.Request) {
	logger := mr.config.Logger

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
			logger.Sugar().Errorf("failed to write into response writer value for metric %s: %v", mname, err)
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
			logger.Sugar().Errorf("failed to write into response writer value for metric %s: %v", mname, err)
			http.Error(rw, "", http.StatusInternalServerError)
			return
		}
	}
}

func (mr *MetricResource) GetMetricJSON(rw http.ResponseWriter, r *http.Request) {
	var req models.Metrics
	logger := mr.config.Logger

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		logger.Sugar().Error("cannot decode request JSON body", zap.Error(err))
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if req.MType != "counter" && req.MType != "gauge" {
		logger.Sugar().Error("unsupported metric type", zap.String("type", req.MType))
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	mtype := req.MType
	mname := req.ID

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
		logger.Sugar().Debug("error encoding JSON response", zap.Error(err))
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (mr *MetricResource) GetAllMetrics(rw http.ResponseWriter, r *http.Request) {
	logger := mr.config.Logger

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
		logger.Sugar().Errorf("failed to load http template: %v", err)
		http.Error(rw, "", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "text/html")
	if err := t.Execute(rw, allMetrics); err != nil {
		logger.Sugar().Errorf("failed to execute http template: %v", err)
		http.Error(rw, "", http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
}
