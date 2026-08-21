package internal_agent

import (
	"context"
	"net"
	"testing"
	"time"

	pb "github.com/ApplePieAndCrime/go-yandex-metrics/internal/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type recordingMetricsClient struct {
	requests []*pb.UpdateMetricsRequest
	md       metadata.MD
	err      error
}

func (c *recordingMetricsClient) UpdateMetrics(ctx context.Context, request *pb.UpdateMetricsRequest, _ ...grpc.CallOption) (*pb.UpdateMetricsResponse, error) {
	c.requests = append(c.requests, request)
	c.md, _ = metadata.FromOutgoingContext(ctx)
	if c.err != nil {
		return nil, c.err
	}
	return pb.UpdateMetricsResponse_builder{}.Build(), nil
}

func TestSendAllMetricsGRPCSendsOneBatchWithRealIP(t *testing.T) {
	client := &recordingMetricsClient{}
	metrics := &AgentMetrics{PollCount: 7, RandomValue: 3.14, CPUutilization: []float64{25.5}}

	err := sendAllMetricsGRPCWithRetry(client, metrics, time.Second, func(time.Duration) {})
	require.NoError(t, err)
	require.Len(t, client.requests, 1)
	assert.Greater(t, len(client.requests[0].GetMetrics()), 2)

	byID := make(map[string]*pb.Metric)
	for _, metric := range client.requests[0].GetMetrics() {
		byID[metric.GetId()] = metric
	}
	assert.Equal(t, pb.Metric_COUNTER, byID["PollCount"].GetType())
	assert.Equal(t, int64(7), byID["PollCount"].GetDelta())
	assert.Equal(t, pb.Metric_GAUGE, byID["RandomValue"].GetType())
	assert.Equal(t, 3.14, byID["RandomValue"].GetValue())
	assert.Equal(t, 25.5, byID["CPUutilization1"].GetValue())

	ipValues := client.md.Get(realIPMetadataKey)
	require.Len(t, ipValues, 1)
	assert.NotNil(t, net.ParseIP(ipValues[0]))
}

func TestSendAllMetricsGRPCDoesNotRetryPermissionDenied(t *testing.T) {
	client := &recordingMetricsClient{err: status.Error(codes.PermissionDenied, "outside subnet")}

	err := sendAllMetricsGRPCWithRetry(client, &AgentMetrics{}, time.Second, func(time.Duration) {
		t.Fatal("unexpected retry")
	})

	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	assert.Len(t, client.requests, 1)
}
