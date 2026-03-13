package patch

import (
	"strings"
	"testing"
	"time"

	"github.com/salman-frs/meridian/internal/model"
)

func TestBuildUsesEngineSpecificCaptureEndpoint(t *testing.T) {
	cfg := model.ConfigModel{
		Raw: map[string]any{
			"receivers": map[string]any{},
			"exporters": map[string]any{
				"otlp": map[string]any{"endpoint": "example.com:4317"},
			},
			"service": map[string]any{
				"pipelines": map[string]any{
					"traces": map[string]any{
						"receivers":  []string{"otlp"},
						"processors": []string{"batch"},
						"exporters":  []string{"otlp"},
					},
				},
			},
		},
		Pipelines: map[string]model.PipelineModel{
			"traces": {
				Name:       "traces",
				Signal:     model.SignalTraces,
				Receivers:  []string{"otlp"},
				Processors: []string{"batch"},
				Exporters:  []string{"otlp"},
			},
		},
	}

	patched, plan, err := Build(cfg, Options{
		RunID:           "run-123",
		Engine:          model.RuntimeEngineContainerd,
		Mode:            model.RuntimeModeSafe,
		CollectorImage:  "otel/test:latest",
		Timeout:         30 * time.Second,
		StartupTimeout:  10 * time.Second,
		InjectTimeout:   5 * time.Second,
		CaptureTimeout:  10 * time.Second,
		InjectionPort:   4317,
		CapturePort:     4318,
		CaptureEndpoint: "127.0.0.1:4318",
		CaptureSamples:  5,
	})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if plan.Engine != model.RuntimeEngineContainerd {
		t.Fatalf("plan.Engine = %q, want %q", plan.Engine, model.RuntimeEngineContainerd)
	}
	if plan.CaptureEndpoint != "127.0.0.1:4318" {
		t.Fatalf("plan.CaptureEndpoint = %q, want 127.0.0.1:4318", plan.CaptureEndpoint)
	}
	if !strings.Contains(patched.CanonicalYAML, "endpoint: 127.0.0.1:4318") {
		t.Fatalf("patched config did not include engine-specific capture endpoint:\n%s", patched.CanonicalYAML)
	}
}
