package generator

import (
	"context"
	"encoding/binary"
	"strconv"
	"time"

	"github.com/salman-frs/meridian/internal/model"
	collectlogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	collectmetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	collecttrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	logsv1 "go.opentelemetry.io/proto/otlp/logs/v1"
	metricsv1 "go.opentelemetry.io/proto/otlp/metrics/v1"
	resourcev1 "go.opentelemetry.io/proto/otlp/resource/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Generator struct {
	address string
	seed    int64
}

func New(address string, seed int64) *Generator {
	return &Generator{address: address, seed: seed}
}

func (g *Generator) Send(ctx context.Context, plan model.TestPlan) error {
	conn, err := grpc.NewClient(g.address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	if hasSignal(plan.Signals, model.SignalTraces) {
		if err := g.sendTraces(ctx, conn, plan.RunID); err != nil {
			return err
		}
	}
	if hasSignal(plan.Signals, model.SignalMetrics) {
		if err := g.sendMetrics(ctx, conn, plan.RunID); err != nil {
			return err
		}
	}
	if hasSignal(plan.Signals, model.SignalLogs) {
		if err := g.sendLogs(ctx, conn, plan.RunID); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) sendTraces(ctx context.Context, conn *grpc.ClientConn, runID string) error {
	client := collecttrace.NewTraceServiceClient(conn)
	spans := make([]*tracev1.Span, 0, 5)
	for i := 0; i < 5; i++ {
		traceID, spanID := ids(g.seed + int64(i*10))
		spans = append(spans, &tracev1.Span{
			Name:              "meridian.synthetic.span",
			TraceId:           traceID,
			SpanId:            spanID,
			StartTimeUnixNano: uint64(time.Now().Add(-time.Second).UnixNano()),
			EndTimeUnixNano:   uint64(time.Now().UnixNano()),
			Attributes: []*commonv1.KeyValue{
				attr("meridian.run_id", runID),
				attr("http.route", "/checkout"),
				attr("http.request.header.authorization", "Bearer meridian-secret"),
				attr("meridian.sequence", integerString(i+1)),
			},
		})
	}
	_, err := client.Export(ctx, &collecttrace.ExportTraceServiceRequest{
		ResourceSpans: []*tracev1.ResourceSpans{
			{
				Resource: &resourcev1.Resource{Attributes: resourceAttrs(runID)},
				ScopeSpans: []*tracev1.ScopeSpans{
					{
						Spans: spans,
					},
				},
			},
		},
	})
	return err
}

func (g *Generator) sendMetrics(ctx context.Context, conn *grpc.ClientConn, runID string) error {
	client := collectmetrics.NewMetricsServiceClient(conn)
	points := make([]*metricsv1.NumberDataPoint, 0, 3)
	for i := 0; i < 3; i++ {
		points = append(points, &metricsv1.NumberDataPoint{
			Attributes: []*commonv1.KeyValue{
				attr("meridian.run_id", runID),
				attr("meridian.sequence", integerString(i+1)),
			},
			Value: &metricsv1.NumberDataPoint_AsDouble{
				AsDouble: float64(i + 1),
			},
			TimeUnixNano:      uint64(time.Now().UnixNano()),
			StartTimeUnixNano: uint64(time.Now().Add(-time.Second).UnixNano()),
		})
	}
	_, err := client.Export(ctx, &collectmetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*metricsv1.ResourceMetrics{
			{
				Resource: &resourcev1.Resource{Attributes: resourceAttrs(runID)},
				ScopeMetrics: []*metricsv1.ScopeMetrics{
					{
						Metrics: []*metricsv1.Metric{
							{
								Name: "meridian.synthetic.metric",
								Data: &metricsv1.Metric_Sum{
									Sum: &metricsv1.Sum{
										DataPoints: points,
										AggregationTemporality: metricsv1.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
										IsMonotonic:            true,
									},
								},
							},
						},
					},
				},
			},
		},
	})
	return err
}

func (g *Generator) sendLogs(ctx context.Context, conn *grpc.ClientConn, runID string) error {
	client := collectlogs.NewLogsServiceClient(conn)
	records := make([]*logsv1.LogRecord, 0, 2)
	for i := 0; i < 2; i++ {
		records = append(records, &logsv1.LogRecord{
			TimeUnixNano: uint64(time.Now().UnixNano()),
			Body:         &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "meridian synthetic log"}},
			Attributes: []*commonv1.KeyValue{
				attr("meridian.run_id", runID),
				attr("meridian.sequence", integerString(i+1)),
			},
		})
	}
	_, err := client.Export(ctx, &collectlogs.ExportLogsServiceRequest{
		ResourceLogs: []*logsv1.ResourceLogs{
			{
				Resource: &resourcev1.Resource{Attributes: resourceAttrs(runID)},
				ScopeLogs: []*logsv1.ScopeLogs{
					{
						LogRecords: records,
					},
				},
			},
		},
	})
	return err
}

func resourceAttrs(runID string) []*commonv1.KeyValue {
	return []*commonv1.KeyValue{
		attr("service.name", "meridian"),
		attr("meridian.run_id", runID),
	}
}

func attr(key string, value string) *commonv1.KeyValue {
	return &commonv1.KeyValue{
		Key:   key,
		Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: value}},
	}
}

func integerString(value int) string {
	return strconv.Itoa(value)
}

func ids(seed int64) ([]byte, []byte) {
	traceID := make([]byte, 16)
	spanID := make([]byte, 8)
	binary.BigEndian.PutUint64(traceID[:8], uint64(seed))
	binary.BigEndian.PutUint64(traceID[8:], uint64(seed+1))
	binary.BigEndian.PutUint64(spanID, uint64(seed+2))
	return traceID, spanID
}

func hasSignal(signals []model.SignalType, signal model.SignalType) bool {
	for _, item := range signals {
		if item == signal {
			return true
		}
	}
	return false
}
