package capture

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"sync"
	"time"

	"github.com/salman-frs/meridian/internal/model"
	collectlogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	collectmetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	collecttrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	logsv1 "go.opentelemetry.io/proto/otlp/logs/v1"
	metricsv1 "go.opentelemetry.io/proto/otlp/metrics/v1"
	"google.golang.org/grpc"
)

type InMemorySink struct {
	runID       string
	capturesDir string
	sampleLimit int
	mu          sync.Mutex
	traces      signalState
	metrics     signalState
	logs        signalState
	errors      []string
	server      *grpc.Server
	listener    net.Listener
	address     string
}

type signalState struct {
	count     int
	samples   []map[string]any
	entries   []model.NormalizedTelemetry
	firstSeen time.Time
	lastSeen  time.Time
	truncated bool
}

func NewInMemorySink(runID string, capturesDir string, sampleLimit int) *InMemorySink {
	if sampleLimit <= 0 {
		sampleLimit = 5
	}
	return &InMemorySink{
		runID:       runID,
		capturesDir: capturesDir,
		sampleLimit: sampleLimit,
	}
}

func (s *InMemorySink) Start(port int) (string, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return "", err
	}
	s.listener = listener
	s.address = listener.Addr().String()
	s.server = grpc.NewServer()
	collecttrace.RegisterTraceServiceServer(s.server, &traceServer{sink: s})
	collectmetrics.RegisterMetricsServiceServer(s.server, &metricsServer{sink: s})
	collectlogs.RegisterLogsServiceServer(s.server, &logsServer{sink: s})
	go func() {
		_ = s.server.Serve(listener)
	}()
	return s.address, nil
}

func (s *InMemorySink) Stop() error {
	if s.server != nil {
		s.server.GracefulStop()
	}
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

type traceServer struct {
	collecttrace.UnimplementedTraceServiceServer
	sink *InMemorySink
}

type metricsServer struct {
	collectmetrics.UnimplementedMetricsServiceServer
	sink *InMemorySink
}

type logsServer struct {
	collectlogs.UnimplementedLogsServiceServer
	sink *InMemorySink
}

func (t *traceServer) Export(ctx context.Context, req *collecttrace.ExportTraceServiceRequest) (*collecttrace.ExportTraceServiceResponse, error) {
	s := t.sink
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, resourceSpans := range req.ResourceSpans {
		resourceAttrs := attrs(resourceSpans.Resource.Attributes)
		for _, scopeSpans := range resourceSpans.ScopeSpans {
			for _, span := range scopeSpans.Spans {
				entry := map[string]any{
					"trace_id":   fmt.Sprintf("%x", span.TraceId),
					"span_id":    fmt.Sprintf("%x", span.SpanId),
					"span_name":  span.Name,
					"resource":   resourceAttrs,
					"attributes": attrs(span.Attributes),
					"run_id":     findRunID(resourceAttrs, attrs(span.Attributes)),
					"fixture":    findFixture(resourceAttrs, attrs(span.Attributes)),
				}
				s.record(model.SignalTraces, &s.traces, entry)
			}
		}
	}
	return &collecttrace.ExportTraceServiceResponse{}, nil
}

func (m *metricsServer) Export(ctx context.Context, req *collectmetrics.ExportMetricsServiceRequest) (*collectmetrics.ExportMetricsServiceResponse, error) {
	s := m.sink
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, resourceMetrics := range req.ResourceMetrics {
		resourceAttrs := attrs(resourceMetrics.Resource.Attributes)
		for _, scopeMetrics := range resourceMetrics.ScopeMetrics {
			for _, metric := range scopeMetrics.Metrics {
				s.record(model.SignalMetrics, &s.metrics, metricEntry(resourceAttrs, metric))
			}
		}
	}
	return &collectmetrics.ExportMetricsServiceResponse{}, nil
}

func (l *logsServer) Export(ctx context.Context, req *collectlogs.ExportLogsServiceRequest) (*collectlogs.ExportLogsServiceResponse, error) {
	s := l.sink
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, resourceLogs := range req.ResourceLogs {
		resourceAttrs := attrs(resourceLogs.Resource.Attributes)
		for _, scopeLogs := range resourceLogs.ScopeLogs {
			for _, logRecord := range scopeLogs.LogRecords {
				s.record(model.SignalLogs, &s.logs, logEntry(resourceAttrs, logRecord))
			}
		}
	}
	return &collectlogs.ExportLogsServiceResponse{}, nil
}

func (s *InMemorySink) Snapshot() []model.SignalCapture {
	s.mu.Lock()
	defer s.mu.Unlock()
	return []model.SignalCapture{
		{
			Signal:          model.SignalTraces,
			Count:           s.traces.count,
			Samples:         append([]map[string]any{}, s.traces.samples...),
			Errors:          append([]string{}, s.errors...),
			FirstReceivedAt: s.traces.firstSeen,
			LastReceivedAt:  s.traces.lastSeen,
			Truncated:       s.traces.truncated,
		},
		{
			Signal:          model.SignalMetrics,
			Count:           s.metrics.count,
			Samples:         append([]map[string]any{}, s.metrics.samples...),
			Errors:          append([]string{}, s.errors...),
			FirstReceivedAt: s.metrics.firstSeen,
			LastReceivedAt:  s.metrics.lastSeen,
			Truncated:       s.metrics.truncated,
		},
		{
			Signal:          model.SignalLogs,
			Count:           s.logs.count,
			Samples:         append([]map[string]any{}, s.logs.samples...),
			Errors:          append([]string{}, s.errors...),
			FirstReceivedAt: s.logs.firstSeen,
			LastReceivedAt:  s.logs.lastSeen,
			Truncated:       s.logs.truncated,
		},
	}
}

func (s *InMemorySink) Persist() error {
	for _, capture := range s.Snapshot() {
		path := filepath.Join(s.capturesDir, string(capture.Signal)+".json")
		if err := model.WriteJSON(path, capture); err != nil {
			return err
		}
	}
	return nil
}

func (s *InMemorySink) PersistNormalized(path string) error {
	return model.WriteJSON(path, s.Normalized())
}

func (s *InMemorySink) GetRunID() string {
	return s.runID
}

func (s *InMemorySink) Normalized() []model.NormalizedTelemetry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]model.NormalizedTelemetry, 0, len(s.traces.entries)+len(s.metrics.entries)+len(s.logs.entries))
	out = append(out, s.traces.entries...)
	out = append(out, s.metrics.entries...)
	out = append(out, s.logs.entries...)
	return out
}

func (s *InMemorySink) record(signal model.SignalType, state *signalState, entry map[string]any) {
	now := time.Now().UTC()
	state.count++
	if state.firstSeen.IsZero() {
		state.firstSeen = now
	}
	state.lastSeen = now
	entry["received_at"] = now.Format(time.RFC3339Nano)
	state.entries = append(state.entries, normalizeEntry(signal, entry, now))
	if len(state.samples) < s.sampleLimit {
		state.samples = append(state.samples, entry)
		return
	}
	state.truncated = true
}

func attrs(items []*commonv1.KeyValue) map[string]any {
	out := map[string]any{}
	for _, item := range items {
		out[item.Key] = anyValue(item.Value)
	}
	return out
}

func anyValue(value *commonv1.AnyValue) any {
	if value == nil {
		return nil
	}
	switch typed := value.Value.(type) {
	case *commonv1.AnyValue_StringValue:
		return typed.StringValue
	case *commonv1.AnyValue_BoolValue:
		return typed.BoolValue
	case *commonv1.AnyValue_IntValue:
		return typed.IntValue
	case *commonv1.AnyValue_DoubleValue:
		return typed.DoubleValue
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func metricEntry(resource map[string]any, metric *metricsv1.Metric) map[string]any {
	entry := map[string]any{
		"resource":    resource,
		"metric_name": metric.Name,
		"run_id":      resource["meridian.run_id"],
		"fixture":     findFixture(resource),
	}
	if gauge := metric.GetGauge(); gauge != nil && len(gauge.DataPoints) > 0 {
		entry["attributes"] = attrs(gauge.DataPoints[0].Attributes)
		value := metricPointValue(gauge.DataPoints[0])
		if value != nil {
			entry["metric_value"] = *value
		}
	}
	if sum := metric.GetSum(); sum != nil && len(sum.DataPoints) > 0 {
		entry["attributes"] = attrs(sum.DataPoints[0].Attributes)
		value := metricPointValue(sum.DataPoints[0])
		if value != nil {
			entry["metric_value"] = *value
		}
		entry["fixture"] = findFixture(resource, attrs(sum.DataPoints[0].Attributes))
	}
	if gauge := metric.GetGauge(); gauge != nil && len(gauge.DataPoints) > 0 {
		entry["fixture"] = findFixture(resource, attrs(gauge.DataPoints[0].Attributes))
	}
	return entry
}

func logEntry(resource map[string]any, record *logsv1.LogRecord) map[string]any {
	return map[string]any{
		"resource":   resource,
		"body":       anyValue(record.Body),
		"attributes": attrs(record.Attributes),
		"run_id":     findRunID(resource, attrs(record.Attributes)),
		"fixture":    findFixture(resource, attrs(record.Attributes)),
	}
}

func findRunID(maps ...map[string]any) any {
	for _, m := range maps {
		if value, ok := m["meridian.run_id"]; ok {
			return value
		}
	}
	return nil
}

func findFixture(maps ...map[string]any) any {
	for _, m := range maps {
		if value, ok := m["meridian.fixture"]; ok {
			return value
		}
	}
	return nil
}

func metricPointValue(point *metricsv1.NumberDataPoint) *float64 {
	switch point.Value.(type) {
	case *metricsv1.NumberDataPoint_AsDouble:
		value := point.GetAsDouble()
		return &value
	case *metricsv1.NumberDataPoint_AsInt:
		value := float64(point.GetAsInt())
		return &value
	default:
		return nil
	}
}

func normalizeEntry(signal model.SignalType, entry map[string]any, receivedAt time.Time) model.NormalizedTelemetry {
	item := model.NormalizedTelemetry{
		Signal:     signal,
		RunID:      toString(entry["run_id"]),
		Fixture:    toString(entry["fixture"]),
		SpanName:   toString(entry["span_name"]),
		MetricName: toString(entry["metric_name"]),
		Body:       toString(entry["body"]),
		Resource:   mapValue(entry["resource"]),
		Attributes: mapValue(entry["attributes"]),
		ReceivedAt: receivedAt,
	}
	if value, ok := entry["metric_value"].(float64); ok {
		item.MetricValue = &value
	}
	return item
}

func mapValue(v any) map[string]any {
	out, _ := v.(map[string]any)
	if out == nil {
		return nil
	}
	return out
}

func toString(v any) string {
	switch typed := v.(type) {
	case string:
		return typed
	case nil:
		return ""
	default:
		return fmt.Sprint(typed)
	}
}
