package grpcserver

import (
	// ...

	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	pb "github.com/vkupriya/go-metrics/internal/proto"
	ic "github.com/vkupriya/go-metrics/internal/server/grpc/interceptors"
	_ "google.golang.org/grpc/encoding/gzip"

	"google.golang.org/grpc"

	"github.com/vkupriya/go-metrics/internal/server/models"
)

type Storage interface {
	UpdateGaugeMetric(c *models.Config, name string, value float64) (float64, error)
	UpdateCounterMetric(c *models.Config, name string, value int64) (int64, error)
	UpdateBatch(c *models.Config, g models.Metrics, cr models.Metrics) error
	Close()
}

type MetricServer struct {
	pb.UnimplementedMetricsServer
	Store  Storage
	config *models.Config
}

func (m *MetricServer) UpdateMetric(ctx context.Context, in *pb.UpdateMetricRequest) (*pb.UpdateMetricResponse, error) {
	var response pb.UpdateMetricResponse

	modelMetric, err := protoToMetric(in.GetMetric())
	if err != nil {
		return nil, fmt.Errorf("failed to convert proto Metric into model Metric: %w", err)
	}
	switch modelMetric.MType {
	case "gauge":
		gaugeValue, err := m.Store.UpdateGaugeMetric(m.config, in.GetMetric().GetId(), in.GetMetric().GetGauge())
		if err != nil {
			response.Error = "failed to update gauge metric: " + in.GetMetric().GetId()
		}
		response.Metric = &pb.Metric{
			Id:    in.GetMetric().GetId(),
			Mtype: in.GetMetric().GetMtype(),
			Gauge: gaugeValue,
		}

		return &response, nil

	case "counter":
		counterValue, err := m.Store.UpdateCounterMetric(m.config, in.GetMetric().GetId(), in.GetMetric().GetDelta())
		if err != nil {
			response.Error = "failed to update counter metric: " + in.GetMetric().GetId()
		}
		response.Metric = &pb.Metric{
			Id:    in.GetMetric().GetId(),
			Mtype: in.GetMetric().GetMtype(),
			Delta: counterValue,
		}
	}
	return &response, nil
}

func (m *MetricServer) UpdateMetrics(ctx context.Context, in *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse,
	error) {
	logger := m.config.Logger
	var response pb.UpdateMetricsResponse

	var (
		gauge   models.Metrics
		counter models.Metrics
	)

	for _, metric := range in.GetMetric() {
		modelMetric, err := protoToMetric(metric)
		if err != nil {
			return nil, fmt.Errorf("failed to convert proto Metric into model Metric: %w", err)
		}
		switch modelMetric.MType {
		case "gauge":
			gauge = append(gauge, modelMetric)
		case "counter":
			counter = append(counter, modelMetric)
		}
	}

	err := m.Store.UpdateBatch(m.config, gauge, counter)
	if err != nil {
		logger.Sugar().Error("grpc: failed to update metric batch")
		return nil, fmt.Errorf("failed to update metric batch: %w", err)
	}

	return &response, nil
}

func protoToMetric(pm *pb.Metric) (models.Metric, error) {
	var mtype string
	switch pm.GetMtype() {
	case pb.Mtype_gauge:
		mtype = "gauge"
	case pb.Mtype_counter:
		mtype = "counter"
	default:
		return models.Metric{}, fmt.Errorf("unknown metric type: %s", pm.GetMtype())
	}

	return models.Metric{
		Delta: &pm.Delta,
		Value: &pm.Gauge,
		ID:    pm.GetId(),
		MType: mtype,
	}, nil
}

func Run(ctx context.Context, s Storage, c *models.Config) error {
	logger := c.Logger
	hostport := strings.Replace(c.Address, "http://", "", 1)
	grpcHost := strings.Split(hostport, ":")[0]
	if grpcHost == "" || grpcHost == "localhost" {
		grpcHost = "127.0.0.1"
	}
	grpcHost += ":3200"
	listen, err := net.Listen("tcp", grpcHost)
	if err != nil {
		return fmt.Errorf("failed to set up listener on port 3200: %w", err)
	}

	loggerOpts := []logging.Option{
		logging.WithLogOnEvents(logging.StartCall, logging.FinishCall),
	}

	interceptors := make([]grpc.ServerOption, 0)

	interceptors = append(interceptors, grpc.ChainUnaryInterceptor(
		ic.TrustedSubnetInterceptor(c.TrustedSubnet),
		logging.UnaryServerInterceptor(ic.InterceptorLogger(logger), loggerOpts...),
	))

	srv := grpc.NewServer(interceptors...)

	pb.RegisterMetricsServer(srv, &MetricServer{
		Store:  s,
		config: c,
	})

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		<-ctx.Done()

		log.Printf("got signal %v, attempting graceful shutdown", s)

		srv.GracefulStop()

		wg.Done()
	}()

	logger.Sugar().Infow("gRPC server is starting", "Address", grpcHost)

	if err := srv.Serve(listen); err != nil {
		logger.Sugar().Fatal(err)
		return fmt.Errorf("failed to run grpc server: %w", err)
	}
	return nil
}
