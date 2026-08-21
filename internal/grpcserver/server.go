// Пакет grpcserver реализует gRPC-сервис Metrics.
package grpcserver

import (
	"context"
	"net"

	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
	pb "github.com/ApplePieAndCrime/go-yandex-metrics/internal/proto"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const realIPMetadataKey = "x-real-ip"

// MetricsServer принимает батчи метрик и сохраняет их через прикладной сервис.
type MetricsServer struct {
	pb.UnimplementedMetricsServer
	service *service.Service
	logger  *zap.SugaredLogger
}

// NewMetricsServer создаёт gRPC-сервис Metrics.
func NewMetricsServer(service *service.Service, loggers ...*zap.SugaredLogger) *MetricsServer {
	logger := zap.NewNop().Sugar()
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	}
	return &MetricsServer{service: service, logger: logger}
}

// UpdateMetrics проверяет и сохраняет все метрики из запроса.
func (s *MetricsServer) UpdateMetrics(ctx context.Context, request *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	if request == nil || len(request.GetMetrics()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "metrics batch is empty")
	}

	metrics := make([]models.Metrics, 0, len(request.GetMetrics()))
	for _, metric := range request.GetMetrics() {
		modelMetric, err := metricToModel(metric)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, modelMetric)
	}

	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	if _, err := s.service.CreateOrUpdateMetricsBatch(ctx, metrics); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, status.FromContextError(ctxErr).Err()
		}
		s.logger.Errorw("failed to save metrics batch", "error", err, "metrics_count", len(metrics))
		return nil, status.Error(codes.Internal, "failed to save metrics")
	}

	return pb.UpdateMetricsResponse_builder{}.Build(), nil
}

func metricToModel(metric *pb.Metric) (models.Metrics, error) {
	if metric == nil || metric.GetId() == "" {
		return models.Metrics{}, status.Error(codes.InvalidArgument, "metric id is required")
	}

	switch metric.GetType() {
	case pb.Metric_GAUGE:
		value := metric.GetValue()
		return models.Metrics{ID: metric.GetId(), MType: models.Gauge, Value: &value}, nil
	case pb.Metric_COUNTER:
		delta := metric.GetDelta()
		return models.Metrics{ID: metric.GetId(), MType: models.Counter, Delta: &delta}, nil
	default:
		return models.Metrics{}, status.Errorf(codes.InvalidArgument, "unsupported metric type: %d", metric.GetType())
	}
}

// TrustedSubnetInterceptor разрешает вызовы, только если x-real-ip принадлежит trustedSubnet.
// Значение nil отключает ограничение аналогично HTTP middleware.
func TrustedSubnetInterceptor(trustedSubnet *net.IPNet) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if trustedSubnet != nil {
			md, ok := metadata.FromIncomingContext(ctx)
			if !ok {
				return nil, status.Error(codes.PermissionDenied, "agent IP is missing")
			}

			values := md.Get(realIPMetadataKey)
			if len(values) == 0 {
				return nil, status.Error(codes.PermissionDenied, "agent IP is missing")
			}

			agentIP := net.ParseIP(values[0])
			if agentIP == nil || !trustedSubnet.Contains(agentIP) {
				return nil, status.Error(codes.PermissionDenied, "agent IP is outside the trusted subnet")
			}
		}

		return handler(ctx, req)
	}
}
