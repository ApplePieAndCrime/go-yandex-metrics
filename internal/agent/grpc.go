package internal_agent

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
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
	request, err := metricsToProtoRequest(metrics)
	if err != nil {
		return err
	}
	var lastErr error

	for attempt := 0; attempt <= len(retryIntervals); attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		ctx = metadata.AppendToOutgoingContext(ctx, realIPMetadataKey, hostIP())
		_, err = client.UpdateMetrics(ctx, request)
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

func metricsToProtoRequest(metrics *AgentMetrics) (*pb.UpdateMetricsRequest, error) {
	memMetrics := buildMemMetrics(metrics)
	protoMetrics := make([]*pb.Metric, 0, len(memMetrics))

	for name, metric := range memMetrics {
		builder := pb.Metric_builder{Id: name}
		switch metric.Type {
		case models.Counter:
			builder.Type = pb.Metric_COUNTER
			delta, err := strconv.ParseInt(metric.Value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parse counter %q: %w", name, err)
			}
			builder.Delta = delta
		case models.Gauge:
			builder.Type = pb.Metric_GAUGE
			value, err := strconv.ParseFloat(metric.Value, 64)
			if err != nil {
				return nil, fmt.Errorf("parse gauge %q: %w", name, err)
			}
			builder.Value = value
		default:
			return nil, fmt.Errorf("unsupported metric type %q", metric.Type)
		}
		protoMetrics = append(protoMetrics, builder.Build())
	}

	return pb.UpdateMetricsRequest_builder{Metrics: protoMetrics}.Build(), nil
}
