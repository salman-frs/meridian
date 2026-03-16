package generator

import (
	"context"
	"encoding/binary"
	"fmt"
	"slices"
	"strconv"
	"strings"
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

const (
	FixturePassThrough     = "pass-through"
	FixtureRedaction       = "redaction"
	FixtureFilterDrop      = "filter-drop"
	FixtureRoutingCopy     = "routing-copy"
	FixtureRoutingMove     = "routing-move"
	FixtureMetricTransform = "metric-transform"
)

var knownFixtures = []string{
	FixturePassThrough,
	FixtureRedaction,
	FixtureFilterDrop,
	FixtureRoutingCopy,
	FixtureRoutingMove,
	FixtureMetricTransform,
}

type Generator struct {
	address string
	seed    int64
}

func New(address string, seed int64) *Generator {
	return &Generator{address: address, seed: seed}
}

func KnownFixtures() []string {
	return slices.Clone(knownFixtures)
}

func IsKnownFixture(name string) bool {
	return slices.Contains(knownFixtures, name)
}

func (g *Generator) Send(ctx context.Context, plan model.TestPlan) error {
	conn, err := grpc.NewClient(g.address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	if len(plan.Fixtures) == 0 {
		return g.sendLegacy(ctx, conn, plan)
	}
	for _, fixture := range plan.Fixtures {
		if err := g.sendFixture(ctx, conn, plan, fixture); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) sendLegacy(ctx context.Context, conn *grpc.ClientConn, plan model.TestPlan) error {
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

func (g *Generator) sendFixture(ctx context.Context, conn *grpc.ClientConn, plan model.TestPlan, fixture string) error {
	switch fixture {
	case FixturePassThrough:
		return g.sendPassThroughFixture(ctx, conn, plan)
	case FixtureRedaction:
		return g.sendRedactionFixture(ctx, conn, plan)
	case FixtureFilterDrop:
		return g.sendFilterDropFixture(ctx, conn, plan)
	case FixtureRoutingCopy:
		return g.sendRoutingFixture(ctx, conn, plan, FixtureRoutingCopy)
	case FixtureRoutingMove:
		return g.sendRoutingFixture(ctx, conn, plan, FixtureRoutingMove)
	case FixtureMetricTransform:
		return g.sendMetricTransformFixture(ctx, conn, plan)
	default:
		return fmt.Errorf("unknown fixture %q", fixture)
	}
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
	return sendTraceBatch(ctx, client, resourceAttrs(runID, ""), spans)
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
	return sendMetricBatch(ctx, client, resourceAttrs(runID, ""), "meridian.synthetic.metric", points)
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
	return sendLogBatch(ctx, client, resourceAttrs(runID, ""), records)
}

func (g *Generator) sendPassThroughFixture(ctx context.Context, conn *grpc.ClientConn, plan model.TestPlan) error {
	if hasSignal(plan.Signals, model.SignalTraces) {
		if err := sendFixtureTrace(ctx, conn, g.seed, plan.RunID, FixturePassThrough, "meridian.synthetic.pass_through", map[string]string{
			"http.route": "/browse",
		}); err != nil {
			return err
		}
	}
	if hasSignal(plan.Signals, model.SignalMetrics) {
		if err := sendFixtureMetric(ctx, conn, plan.RunID, FixturePassThrough, "meridian.synthetic.metric.pass_through", 1, map[string]string{
			"fixture.mode": "pass-through",
		}); err != nil {
			return err
		}
	}
	if hasSignal(plan.Signals, model.SignalLogs) {
		if err := sendFixtureLog(ctx, conn, plan.RunID, FixturePassThrough, "meridian synthetic log pass-through", map[string]string{
			"fixture.mode": "pass-through",
		}); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) sendRedactionFixture(ctx context.Context, conn *grpc.ClientConn, plan model.TestPlan) error {
	if !hasSignal(plan.Signals, model.SignalTraces) {
		return nil
	}
	return sendFixtureTrace(ctx, conn, g.seed+100, plan.RunID, FixtureRedaction, "meridian.synthetic.redaction", map[string]string{
		"http.route":                        "/checkout",
		"http.request.header.authorization": "Bearer meridian-secret",
	})
}

func (g *Generator) sendFilterDropFixture(ctx context.Context, conn *grpc.ClientConn, plan model.TestPlan) error {
	if !hasSignal(plan.Signals, model.SignalTraces) {
		return nil
	}
	return sendFixtureTrace(ctx, conn, g.seed+200, plan.RunID, FixtureFilterDrop, "meridian.synthetic.filter_drop", map[string]string{
		"http.route": "/checkout",
	})
}

func (g *Generator) sendRoutingFixture(ctx context.Context, conn *grpc.ClientConn, plan model.TestPlan, fixture string) error {
	attrs := map[string]string{
		"routing.key":     "gold",
		"routing.fixture": fixture,
	}
	if hasSignal(plan.Signals, model.SignalTraces) {
		if err := sendFixtureTrace(ctx, conn, g.seed+300, plan.RunID, fixture, "meridian.synthetic."+sanitizeFixtureName(fixture), attrs); err != nil {
			return err
		}
	}
	if hasSignal(plan.Signals, model.SignalLogs) {
		if err := sendFixtureLog(ctx, conn, plan.RunID, fixture, "meridian synthetic "+fixture+" log", attrs); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) sendMetricTransformFixture(ctx context.Context, conn *grpc.ClientConn, plan model.TestPlan) error {
	if !hasSignal(plan.Signals, model.SignalMetrics) {
		return nil
	}
	return sendFixtureMetric(ctx, conn, plan.RunID, FixtureMetricTransform, "meridian.synthetic.metric.raw", 2, map[string]string{
		"metric.fixture": "transform",
	})
}

func sendFixtureTrace(ctx context.Context, conn *grpc.ClientConn, seed int64, runID string, fixture string, spanName string, attrsMap map[string]string) error {
	client := collecttrace.NewTraceServiceClient(conn)
	traceID, spanID := ids(seed)
	attributes := []*commonv1.KeyValue{
		attr("meridian.run_id", runID),
		attr("meridian.fixture", fixture),
	}
	for key, value := range attrsMap {
		attributes = append(attributes, attr(key, value))
	}
	return sendTraceBatch(ctx, client, resourceAttrs(runID, fixture), []*tracev1.Span{
		{
			Name:              spanName,
			TraceId:           traceID,
			SpanId:            spanID,
			StartTimeUnixNano: uint64(time.Now().Add(-time.Second).UnixNano()),
			EndTimeUnixNano:   uint64(time.Now().UnixNano()),
			Attributes:        attributes,
		},
	})
}

func sendFixtureMetric(ctx context.Context, conn *grpc.ClientConn, runID string, fixture string, metricName string, value float64, attrsMap map[string]string) error {
	client := collectmetrics.NewMetricsServiceClient(conn)
	attributes := []*commonv1.KeyValue{
		attr("meridian.run_id", runID),
		attr("meridian.fixture", fixture),
	}
	for key, item := range attrsMap {
		attributes = append(attributes, attr(key, item))
	}
	return sendMetricBatch(ctx, client, resourceAttrs(runID, fixture), metricName, []*metricsv1.NumberDataPoint{
		{
			Attributes: attributes,
			Value: &metricsv1.NumberDataPoint_AsDouble{
				AsDouble: value,
			},
			TimeUnixNano:      uint64(time.Now().UnixNano()),
			StartTimeUnixNano: uint64(time.Now().Add(-time.Second).UnixNano()),
		},
	})
}

func sendFixtureLog(ctx context.Context, conn *grpc.ClientConn, runID string, fixture string, body string, attrsMap map[string]string) error {
	client := collectlogs.NewLogsServiceClient(conn)
	attributes := []*commonv1.KeyValue{
		attr("meridian.run_id", runID),
		attr("meridian.fixture", fixture),
	}
	for key, item := range attrsMap {
		attributes = append(attributes, attr(key, item))
	}
	return sendLogBatch(ctx, client, resourceAttrs(runID, fixture), []*logsv1.LogRecord{
		{
			TimeUnixNano: uint64(time.Now().UnixNano()),
			Body:         &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: body}},
			Attributes:   attributes,
		},
	})
}

func sendTraceBatch(ctx context.Context, client collecttrace.TraceServiceClient, resource []*commonv1.KeyValue, spans []*tracev1.Span) error {
	_, err := client.Export(ctx, &collecttrace.ExportTraceServiceRequest{
		ResourceSpans: []*tracev1.ResourceSpans{
			{
				Resource: &resourcev1.Resource{Attributes: resource},
				ScopeSpans: []*tracev1.ScopeSpans{
					{Spans: spans},
				},
			},
		},
	})
	return err
}

func sendMetricBatch(ctx context.Context, client collectmetrics.MetricsServiceClient, resource []*commonv1.KeyValue, metricName string, points []*metricsv1.NumberDataPoint) error {
	_, err := client.Export(ctx, &collectmetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*metricsv1.ResourceMetrics{
			{
				Resource: &resourcev1.Resource{Attributes: resource},
				ScopeMetrics: []*metricsv1.ScopeMetrics{
					{
						Metrics: []*metricsv1.Metric{
							{
								Name: metricName,
								Data: &metricsv1.Metric_Sum{
									Sum: &metricsv1.Sum{
										DataPoints:             points,
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

func sendLogBatch(ctx context.Context, client collectlogs.LogsServiceClient, resource []*commonv1.KeyValue, records []*logsv1.LogRecord) error {
	_, err := client.Export(ctx, &collectlogs.ExportLogsServiceRequest{
		ResourceLogs: []*logsv1.ResourceLogs{
			{
				Resource: &resourcev1.Resource{Attributes: resource},
				ScopeLogs: []*logsv1.ScopeLogs{
					{LogRecords: records},
				},
			},
		},
	})
	return err
}

func resourceAttrs(runID string, fixture string) []*commonv1.KeyValue {
	items := []*commonv1.KeyValue{
		attr("service.name", "meridian"),
		attr("meridian.run_id", runID),
	}
	if fixture != "" {
		items = append(items, attr("meridian.fixture", fixture))
	}
	return items
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

func sanitizeFixtureName(value string) string {
	return strings.ReplaceAll(value, "-", "_")
}
