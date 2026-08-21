package grpcserver

import (
	"context"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

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
	_, err = pb.NewMetricsClient(connection).UpdateMetrics(ctx, &pb.UpdateMetricsRequest{Metrics: []*pb.Metric{
		{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 42.5},
	}})
	require.NoError(t, err)
}

func TestUpdateMetricsStoresBatch(t *testing.T) {
	storage := repository.NewMemoryStorage()
	services := service.NewService(storage)
	server := NewMetricsServer(services)

	_, err := server.UpdateMetrics(context.Background(), &pb.UpdateMetricsRequest{Metrics: []*pb.Metric{
		{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 42.5},
		{Id: "PollCount", Type: pb.Metric_COUNTER, Delta: 3},
	}})
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
