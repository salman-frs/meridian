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
