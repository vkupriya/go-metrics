package handlers

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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

const (
	counter string = "counter"
	gauge   string = "gauge"
)

type MetricResource struct {
	store storage.Storage
}

func NewMetricResource(store storage.Storage) *MetricResource {
	return &MetricResource{store: store}
}

func NewMetricRouter(mr *MetricResource) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.AllowContentType("text/plain"))

	r.Get("/", mr.GetAllMetrics)
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
		} else {
			if _, err := io.WriteString(rw, fmt.Sprintf("%g", v)); err != nil {
				panic(err)
			}
			rw.WriteHeader(http.StatusOK)
			return
		}
	case mtype == counter:
		v, err := mr.store.GetCounterMetric(mname)
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

func (mr *MetricResource) GetAllMetrics(rw http.ResponseWriter, r *http.Request) {
	gauge, counter := mr.store.GetAllValues()

	allMetrics := make(map[string]any)

	for name, value := range gauge {
		allMetrics[name] = value
		fmt.Println("name: ", name)
	}

	for name, value := range counter {
		allMetrics[name] = value
	}

	t, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	if err := t.Execute(rw, allMetrics); err != nil {
		panic(err)
	}

	rw.WriteHeader(http.StatusOK)
}
