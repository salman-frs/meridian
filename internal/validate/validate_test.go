package validate

import (
	"testing"

	"github.com/salman-frs/meridian/internal/model"
)

func TestRunFindsBrokenPipelineReferencesAndUnusedComponents(t *testing.T) {
	t.Parallel()

	cfg := model.ConfigModel{
		SourcePaths: []string{"collector.yaml"},
		Receivers: map[string]model.Component{
			"otlp":   {Name: "otlp", Kind: "receiver", Config: map[string]any{"endpoint": "broken"}},
			"unused": {Name: "unused", Kind: "receiver", Config: map[string]any{}},
		},
		Exporters: map[string]model.Component{
			"debug": {Name: "debug", Kind: "exporter", Config: map[string]any{}},
		},
		Pipelines: map[string]model.PipelineModel{
			"traces": {
				Name:      "traces",
				Signal:    model.SignalTraces,
				Receivers: []string{"missing"},
			},
		},
	}

	findings := Run(cfg)
	if len(findings) < 3 {
		t.Fatalf("Run() findings = %#v, want multiple findings", findings)
	}
	assertContainsCode(t, findings, "missing-exporters")
	assertContainsCode(t, findings, "undefined-receiver")
	assertContainsCode(t, findings, "unused-receiver")
	assertContainsCode(t, findings, "invalid-endpoint")
}

func assertContainsCode(t *testing.T, findings []model.Finding, code string) {
	t.Helper()
	for _, finding := range findings {
		if finding.Code == code {
			return
		}
	}
	t.Fatalf("missing finding code %q in %#v", code, findings)
}
