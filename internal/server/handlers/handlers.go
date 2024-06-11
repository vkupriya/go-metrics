package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
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

type Storage interface {
	UpdateGaugeMetric(c *models.Config, name string, value float64) (float64, error)
	UpdateCounterMetric(c *models.Config, name string, value int64) (int64, error)
	GetCounterMetric(c *models.Config, name string) (int64, bool, error)
	GetGaugeMetric(c *models.Config, name string) (float64, bool, error)
	GetAllMetrics(c *models.Config) (map[string]float64, map[string]int64, error)
	UpdateBatch(c *models.Config, g models.Metrics, cr models.Metrics) error
}

const (
	counter string = "counter"
	gauge   string = "gauge"
)

type MetricResource struct {
	store  Storage
	config *models.Config
}

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

func NewMetricResource(store Storage, cfg *models.Config) *MetricResource {
	return &MetricResource{
		store:  store,
		config: cfg}
}

func NewMetricRouter(mr *MetricResource) chi.Router {
	r := chi.NewRouter()

	ml := mw.NewMiddlewareLogger(mr.config)
	mg := mw.NewMiddlewareGzip(mr.config)

	r.Use(ml.Logging)
	r.Use(mg.GzipHandle)

	r.Get("/", mr.GetAllMetrics)
	r.Get("/ping", mr.GetPostgresStatus)
	r.Get("/value/{metricType}/{metricName}", mr.GetMetric)
	r.Post("/value/", mr.GetMetricJSON)
	r.Post("/update/", mr.UpdateMetricJSON)
	r.Post("/update/{metricType}/{metricName}/{metricValue}", mr.UpdateMetric)
	r.Post("/updates/", mr.UpdateBatchJSON)

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
			logger.Sugar().Error("failed to update gauge metric", err)
		}
		*req.Value = rv

	case mtype == counter:
		if req.Delta == nil {
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
		v, _, err := mr.store.GetGaugeMetric(mr.config, mname)
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
		v, _, err := mr.store.GetCounterMetric(mr.config, mname)
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

func (mr *MetricResource) GetAllMetrics(rw http.ResponseWriter, r *http.Request) {
	logger := mr.config.Logger

	gauge, counter, err := mr.store.GetAllMetrics(mr.config)
	if err != nil {
		logger.Sugar().Errorf("failed to get all metrics: %w", err)
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

func (mr *MetricResource) GetPostgresStatus(rw http.ResponseWriter, r *http.Request) {
	logger := mr.config.Logger

	db, err := sql.Open("pgx", mr.config.PostgresDSN)
	if err != nil {
		logger.Sugar().Errorf("failed to create PG DB connection pool: %v", err)
		http.Error(rw, "", http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Sugar().Errorf("failed to close PG DB connection pool: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		logger.Sugar().Errorf("failed to connect to DB: %v", err)
		http.Error(rw, "", http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
}

func (mr *MetricResource) UpdateBatchJSON(rw http.ResponseWriter, r *http.Request) {
	var req models.Metrics
	logger := mr.config.Logger

	var (
		gauge   models.Metrics
		counter models.Metrics
	)

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		logger.Sugar().Debug("cannot decode request JSON body", zap.Error(err))
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, metric := range req {
		switch metric.MType {
		case "gauge":
			gauge = append(gauge, metric)
		case "counter":
			counter = append(counter, metric)
		default:
			logger.Sugar().Errorf("wrong metric type '%s'", metric.MType)
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	err := mr.store.UpdateBatch(mr.config, gauge, counter)
	if err != nil {
		logger.Sugar().Error(err)
	}
	rw.WriteHeader(http.StatusOK)
}
