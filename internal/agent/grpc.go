package internal_agent

import (
	"context"
	"errors"
	"strconv"
	"time"

	pb "github.com/ApplePieAndCrime/go-yandex-metrics/internal/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const realIPMetadataKey = "x-real-ip"

func sendUnsentMetricsGRPC(
	client pb.MetricsClient,
	metrics *SafeMetrics,
	timeout time.Duration,
) (bool, error) {
	metrics.sendMu.Lock()
	defer metrics.sendMu.Unlock()

	snapshot, revision := metrics.snapshotWithRevision()
	if !metrics.hasUnsent(revision) {
		return false, nil
	}

	if err := sendAllMetricsGRPCWithRetry(client, &snapshot, timeout, time.Sleep); err != nil {
		return false, err
	}

	metrics.markSent(revision)
	return true, nil
}

func sendAllMetricsGRPCWithRetry(
	client pb.MetricsClient,
	metrics *AgentMetrics,
	timeout time.Duration,
	sleep func(time.Duration),
) error {
	request := metricsToProtoRequest(metrics)
	var lastErr error

	for attempt := 0; attempt <= len(retryIntervals); attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		ctx = metadata.AppendToOutgoingContext(ctx, realIPMetadataKey, hostIP())
		_, err := client.UpdateMetrics(ctx, request)
		cancel()
		if err == nil {
			return nil
		}

		lastErr = err
		if !isRetriableGRPCError(err) || attempt == len(retryIntervals) {
			return err
		}
		sleep(retryIntervals[attempt])
	}

	return lastErr
}

func isRetriableGRPCError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	switch status.Code(err) {
	case codes.Unavailable, codes.ResourceExhausted, codes.Aborted, codes.DeadlineExceeded, codes.Internal:
		return true
	default:
		return false
	}
}

func metricsToProtoRequest(metrics *AgentMetrics) *pb.UpdateMetricsRequest {
	memMetrics := buildMemMetrics(metrics)
	request := &pb.UpdateMetricsRequest{Metrics: make([]*pb.Metric, 0, len(memMetrics))}

	for name, metric := range memMetrics {
		protoMetric := &pb.Metric{Id: name}
		switch metric.Type {
		case "counter":
			protoMetric.Type = pb.Metric_COUNTER
			delta, err := strconv.ParseInt(metric.Value, 10, 64)
			if err != nil {
				continue
			}
			protoMetric.Delta = delta
		default:
			protoMetric.Type = pb.Metric_GAUGE
			value, err := strconv.ParseFloat(metric.Value, 64)
			if err != nil {
				continue
			}
			protoMetric.Value = value
		}
		request.Metrics = append(request.Metrics, protoMetric)
	}

	return request
}
