package grpcserver

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"testing"
	"time"

	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
	pb "github.com/ApplePieAndCrime/go-yandex-metrics/internal/proto"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/repository"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/service"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/tlsutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type cancelWaitingStorage struct {
	repository.Storage
	started chan struct{}
}

type failingBatchStorage struct {
	repository.Storage
	err error
}

func (s *failingBatchStorage) SaveMetricsBatch(context.Context, []models.Metrics) ([]models.Metrics, error) {
	return nil, s.err
}

func (s *cancelWaitingStorage) SaveMetricsBatch(ctx context.Context, _ []models.Metrics) ([]models.Metrics, error) {
	close(s.started)
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestUpdateMetricsPropagatesCanceledContext(t *testing.T) {
	storage := &cancelWaitingStorage{
		Storage: repository.NewMemoryStorage(),
		started: make(chan struct{}),
	}
	server := NewMetricsServer(service.NewService(storage))
	request := pb.UpdateMetricsRequest_builder{Metrics: []*pb.Metric{
		pb.Metric_builder{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 42.5}.Build(),
	}}.Build()

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := server.UpdateMetrics(ctx, request)
		result <- err
	}()

	<-storage.started
	cancel()
	assert.Equal(t, codes.Canceled, status.Code(<-result))
}

func TestUpdateMetricsHidesAndLogsInternalError(t *testing.T) {
	internalErr := errors.New("database password=secret")
	storage := &failingBatchStorage{
		Storage: repository.NewMemoryStorage(),
		err:     internalErr,
	}
	core, observedLogs := observer.New(zap.ErrorLevel)
	server := NewMetricsServer(service.NewService(storage), zap.New(core).Sugar())
	request := pb.UpdateMetricsRequest_builder{Metrics: []*pb.Metric{
		pb.Metric_builder{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 42.5}.Build(),
	}}.Build()

	_, err := server.UpdateMetrics(context.Background(), request)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Equal(t, "failed to save metrics", status.Convert(err).Message())
	assert.NotContains(t, err.Error(), "password")

	logs := observedLogs.FilterMessage("failed to save metrics batch").All()
	require.Len(t, logs, 1)
	assert.Equal(t, internalErr.Error(), logs[0].ContextMap()["error"])
}

func TestUpdateMetricsOverTLS(t *testing.T) {
	directory := t.TempDir()
	certFile := filepath.Join(directory, "server.crt")
	keyFile := filepath.Join(directory, "server.key")
	require.NoError(t, tlsutil.GenerateSelfSigned(tlsutil.CertificateOptions{
		CertFile: certFile,
		KeyFile:  keyFile,
		Hosts:    []string{"localhost"},
		ValidFor: time.Hour,
	}))

	serverCredentials, err := tlsutil.LoadServerCredentials(certFile, keyFile)
	require.NoError(t, err)
	clientCredentials, err := tlsutil.LoadClientCredentials(certFile, "")
	require.NoError(t, err)

	storage := repository.NewMemoryStorage()
	services := service.NewService(storage)
	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer(grpc.Creds(serverCredentials))
	pb.RegisterMetricsServer(grpcServer, NewMetricsServer(services))
	go func() {
		_ = grpcServer.Serve(listener)
	}()
	t.Cleanup(grpcServer.Stop)

	connection, err := grpc.NewClient(
		"passthrough:///localhost",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(clientCredentials),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = pb.NewMetricsClient(connection).UpdateMetrics(ctx, pb.UpdateMetricsRequest_builder{Metrics: []*pb.Metric{
		pb.Metric_builder{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 42.5}.Build(),
	}}.Build())
	require.NoError(t, err)
}

func TestUpdateMetricsStoresBatch(t *testing.T) {
	storage := repository.NewMemoryStorage()
	services := service.NewService(storage)
	server := NewMetricsServer(services)

	_, err := server.UpdateMetrics(context.Background(), pb.UpdateMetricsRequest_builder{Metrics: []*pb.Metric{
		pb.Metric_builder{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 42.5}.Build(),
		pb.Metric_builder{Id: "PollCount", Type: pb.Metric_COUNTER, Delta: 3}.Build(),
	}}.Build())
	require.NoError(t, err)

	gauge, exists, err := services.GetMetricsByID("Alloc", models.Gauge)
	require.NoError(t, err)
	require.True(t, exists)
	assert.Equal(t, 42.5, *gauge.Value)

	counter, exists, err := services.GetMetricsByID("PollCount", models.Counter)
	require.NoError(t, err)
	require.True(t, exists)
	assert.Equal(t, int64(3), *counter.Delta)
}

func TestTrustedSubnetInterceptor(t *testing.T) {
	_, subnet, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)
	interceptor := TrustedSubnetInterceptor(subnet)
	handler := func(context.Context, any) (any, error) { return "ok", nil }

	tests := []struct {
		name string
		ip   string
		code codes.Code
	}{
		{name: "trusted", ip: "192.168.1.42", code: codes.OK},
		{name: "untrusted", ip: "10.0.0.1", code: codes.PermissionDenied},
		{name: "invalid", ip: "not-an-ip", code: codes.PermissionDenied},
		{name: "missing", code: codes.PermissionDenied},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.ip != "" {
				ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(realIPMetadataKey, tt.ip))
			}
			_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)
			assert.Equal(t, tt.code, status.Code(err))
		})
	}
}

func TestNilTrustedSubnetAllowsRequestWithoutMetadata(t *testing.T) {
	interceptor := TrustedSubnetInterceptor(nil)
	result, err := interceptor(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{},
		func(context.Context, any) (any, error) { return "ok", nil },
	)
	require.NoError(t, err)
	assert.Equal(t, "ok", result)
}
