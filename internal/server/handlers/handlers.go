package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	mw "github.com/vkupriya/go-metrics/internal/server/middleware"
	"github.com/vkupriya/go-metrics/internal/server/models"
	"github.com/vkupriya/go-metrics/internal/server/storage"
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

// Storage interface implements CRUD operations with metrics store.
type Storage interface {
	UpdateGaugeMetric(c *models.Config, name string, value float64) (float64, error)
	UpdateCounterMetric(c *models.Config, name string, value int64) (int64, error)
	GetCounterMetric(c *models.Config, name string) (int64, bool, error)
	GetGaugeMetric(c *models.Config, name string) (float64, bool, error)
	GetAllMetrics(c *models.Config) (map[string]float64, map[string]int64, error)
	UpdateBatch(c *models.Config, g models.Metrics, cr models.Metrics) error
	PingStore(c *models.Config) error
}

const (
	counter string = "counter"
	gauge   string = "gauge"
)

type MetricResource struct {
	store  Storage
	config *models.Config
}

// NewMetricResource initializes MetricResource type.
func NewMetricResource(store Storage, cfg *models.Config) *MetricResource {
	return &MetricResource{
		store:  store,
		config: cfg,
	}
}

// NewStore instantiates metric store based on configuration parameters.
// Options: Memory Store, File Store and PostgresDB Store.
func NewStore(c *models.Config) (Storage, error) {
	if c.PostgresDSN != "" {
		db, err := storage.NewPostgresStorage(c)
		if err != nil {
			return db, fmt.Errorf("failed to initialize PostgresDB: %w", err)
		}
		return db, nil
	}

	if c.FileStoragePath != "" {
		fs, err := storage.NewFileStorage(c)
		if err != nil {
			return fs, fmt.Errorf("failed to initialize FileStorage: %w", err)
		}
		return fs, nil
	}
	ms, err := storage.NewMemStorage(c)
	if err != nil {
		return ms, fmt.Errorf("failed to initialize MemStorage: %w", err)
	}
	return ms, nil
}

// NewMetricRouter intitializes chi router.
func NewMetricRouter(mr *MetricResource) chi.Router {
	r := chi.NewRouter()

	ml := mw.NewMiddlewareLogger(mr.config)
	mh := mw.NewMiddlewareHash(mr.config)
	mg := mw.NewMiddlewareGzip(mr.config)

	r.Use(ml.Logging)

	r.Group(func(r chi.Router) {
		r.Use(mh.HashSend)
		r.Use(mg.GzipHandle)
		r.Get("/", mr.GetAllMetrics)
		r.Get("/ping", mr.PingStore)
		r.Get("/value/{metricType}/{metricName}", mr.GetMetric)
	})

	r.Group(func(r chi.Router) {
		r.Use(mh.HashCheck)
		r.Use(mg.GzipHandle)
		r.Post("/value/", mr.GetMetricJSON)
		r.Post("/update/", mr.UpdateMetricJSON)
		r.Post("/update/{metricType}/{metricName}/{metricValue}", mr.UpdateMetric)
		r.Post("/updates/", mr.UpdateBatchJSON)
	})

	r.Mount("/debug", middleware.Profiler())

	return r
}

// UpdateMetric is an endpoint to update individual metric of gauge or counter type via url.
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
				logger.Sugar().Error("failed to update gauge metric", zap.Error(err))
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
				logger.Sugar().Error("failed to update counter metric", zap.Error(err))
			}
			rw.WriteHeader(http.StatusOK)
		}
		return
	}
}

// UpdateMetricJSON endpoint to update individual metric of gauge or counter type via JSON body.
func (mr *MetricResource) UpdateMetricJSON(rw http.ResponseWriter, r *http.Request) {
	var req models.Metric
	logger := mr.config.Logger

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		logger.Sugar().Debug("cannot decode request JSON body", zap.Error(err))
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if req.MType != "counter" && req.MType != "gauge" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	mtype := req.MType
	mname := req.ID

	rw.Header().Set("Content-Type", "application/json")

	switch {
	case mtype == gauge:
		if req.Value == nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		rv, err := mr.store.UpdateGaugeMetric(mr.config, mname, *req.Value)
		if err != nil {
			logger.Sugar().Error("failed to update gauge metric", zap.Error(err))
		}
		*req.Value = rv

	case mtype == counter:
		if req.Delta == nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		rd, err := mr.store.UpdateCounterMetric(mr.config, mname, *req.Delta)
		if err != nil {
			logger.Sugar().Error("failed to update counter metric", zap.Error(err))
		}
		*req.Delta = rd
	}
	enc := json.NewEncoder(rw)
	if err := enc.Encode(req); err != nil {
		logger.Sugar().Debug("error encoding JSON response", zap.Error(err))
		return
	}
}

// GetMetric endpoint returns gauge or counter metric value via URL parameters.
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
		v, _, err := mr.store.GetGaugeMetric(mr.config, mname)
		if err != nil {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		if _, err := rw.Write([]byte(strconv.FormatFloat(v, 'f', -1, 64))); err != nil {
			logger.Sugar().Errorf("failed to write into response writer value for metric %s", mname, zap.Error(err))
			http.Error(rw, "", http.StatusInternalServerError)
			return
		}

	case mtype == counter:
		v, _, err := mr.store.GetCounterMetric(mr.config, mname)
		if err != nil {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		if _, err := rw.Write([]byte(strconv.FormatInt(v, 10))); err != nil {
			logger.Sugar().Errorf("failed to write into response writer value for metric %s: %v", mname, zap.Error(err))
			http.Error(rw, "", http.StatusInternalServerError)
			return
		}
	}
}

// GetMetricJSON endpoint returns requested gauge or counter metric value in JSON.
func (mr *MetricResource) GetMetricJSON(rw http.ResponseWriter, r *http.Request) {
	var req models.Metric
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
		v, _, err := mr.store.GetGaugeMetric(mr.config, mname)
		if err != nil {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		req.Value = &v

	case mtype == counter:
		v, _, err := mr.store.GetCounterMetric(mr.config, mname)
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

// GetAllMetrics returns all stored metrics as HTML page.
func (mr *MetricResource) GetAllMetrics(rw http.ResponseWriter, r *http.Request) {
	logger := mr.config.Logger

	gauge, counter, err := mr.store.GetAllMetrics(mr.config)
	if err != nil {
		logger.Sugar().Debug("failed to get all metrics", zap.Error(err))
		http.Error(rw, "", http.StatusInternalServerError)
		return
	}

	allMetrics := make(map[string]any)

	for name, value := range gauge {
		allMetrics[name] = value
	}

	for name, value := range counter {
		allMetrics[name] = value
	}

	t, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		logger.Sugar().Errorf("failed to load http template.", zap.Error(err))
		http.Error(rw, "", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "text/html")
	if err := t.Execute(rw, allMetrics); err != nil {
		logger.Sugar().Errorf("failed to execute http template.", zap.Error(err))
		http.Error(rw, "", http.StatusInternalServerError)
		return
	}
}

// PingStore endpoint returns 200 OK if metric store is available, otherwise status code 500.
func (mr *MetricResource) PingStore(rw http.ResponseWriter, r *http.Request) {
	logger := mr.config.Logger
	if err := mr.store.PingStore(mr.config); err != nil {
		logger.Sugar().Errorf("failed to connect to store.", zap.Error(err))
		http.Error(rw, "", http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusOK)
}

// UpdateBatchJSON endpoint updates all metrics in a batch.
func (mr *MetricResource) UpdateBatchJSON(rw http.ResponseWriter, r *http.Request) {
	const NumberOfMetrics int64 = 64
	req := make(models.Metrics, NumberOfMetrics)
	logger := mr.config.Logger

	var (
		gauge   models.Metrics
		counter models.Metrics
	)

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		logger.Sugar().Debugf("cannot decode request JSON body", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, metric := range req {
		switch metric.MType {
		case "gauge":
			if metric.Value != nil {
				gauge = append(gauge, metric)
			} else {
				logger.Sugar().Errorf("Missing value for gauge metric '%s'.", metric.ID)
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
		case "counter":
			if metric.Delta != nil {
				counter = append(counter, metric)
			} else {
				logger.Sugar().Errorf("Missing delta for counter metric '%s'.", metric.ID)
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
		default:
			logger.Sugar().Errorf("wrong metric type '%s'", metric.MType)
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	err := mr.store.UpdateBatch(mr.config, gauge, counter)
	if err != nil {
		logger.Sugar().Error(zap.Error(err))
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusOK)
}
