package diffing

import (
	"testing"

	"github.com/salman-frs/meridian/internal/model"
)

func TestPipelineChangesClassifyWiringAndOrdering(t *testing.T) {
	t.Parallel()

	oldCfg := model.ConfigModel{
		Pipelines: map[string]model.PipelineModel{
			"traces": {
				Name:       "traces",
				Receivers:  []string{"otlp"},
				Processors: []string{"batch", "memory_limiter"},
				Exporters:  []string{"debug"},
			},
		},
	}
	newCfg := model.ConfigModel{
		Pipelines: map[string]model.PipelineModel{
			"traces": {
				Name:       "traces",
				Receivers:  []string{"otlp"},
				Processors: []string{"memory_limiter", "batch"},
				Exporters:  []string{"otlphttp"},
			},
		},
	}

	changes := pipelineChanges(oldCfg, newCfg)
	if len(changes) != 2 {
		t.Fatalf("pipelineChanges() len = %d, want 2", len(changes))
	}
	if changes[0].Kind != "pipeline-wiring-changed" {
		t.Fatalf("pipelineChanges()[0].Kind = %q", changes[0].Kind)
	}
	if changes[1].Kind != "processor-order-changed" {
		t.Fatalf("pipelineChanges()[1].Kind = %q", changes[1].Kind)
	}
}

func TestFilterBySeverity(t *testing.T) {
	t.Parallel()

	changes := []model.DiffChange{
		{Severity: model.SeverityInfo},
		{Severity: model.SeverityWarn},
		{Severity: model.SeverityFail},
	}

	got := filterBySeverity(changes, "medium")
	if len(got) != 2 {
		t.Fatalf("filterBySeverity() len = %d, want 2", len(got))
	}
}

func TestParseThresholdFallsBackToLow(t *testing.T) {
	t.Parallel()

	if got := parseThreshold("unexpected"); got != model.SeverityInfo {
		t.Fatalf("parseThreshold() = %q, want %q", got, model.SeverityInfo)
	}
}

func TestServiceTelemetryChanges(t *testing.T) {
	t.Parallel()

	oldCfg := model.ConfigModel{Raw: map[string]any{
		"service": map[string]any{
			"telemetry": map[string]any{"metrics": map[string]any{"level": "basic"}},
		},
	}}
	newCfg := model.ConfigModel{Raw: map[string]any{
		"service": map[string]any{
			"telemetry": map[string]any{"metrics": map[string]any{"level": "detailed"}},
		},
	}}

	changes := serviceTelemetryChanges(oldCfg, newCfg)
	if len(changes) != 1 || changes[0].Kind != "service-telemetry-changed" {
		t.Fatalf("serviceTelemetryChanges() = %#v", changes)
	}
}

func TestNestedComponentChanges(t *testing.T) {
	t.Parallel()

	oldCfg := model.ConfigModel{
		Exporters: map[string]model.Component{
			"otlp": {Name: "otlp", Config: map[string]any{"tls": map[string]any{"insecure": false}}},
		},
	}
	newCfg := model.ConfigModel{
		Exporters: map[string]model.Component{
			"otlp": {Name: "otlp", Config: map[string]any{"tls": map[string]any{"insecure": true}}},
		},
	}

	changes := nestedComponentChanges("tls", oldCfg, newCfg)
	if len(changes) != 1 || changes[0].Kind != "exporter-tls-changed" {
		t.Fatalf("nestedComponentChanges() = %#v", changes)
	}
}
