package report

import (
	"strings"
	"testing"
	"time"

	"github.com/salman-frs/meridian/internal/model"
)

func TestRenderSummaryMarkdownIncludesTopFailureAndArtifacts(t *testing.T) {
	t.Parallel()

	result := model.RunResult{
		Status:         "FAIL",
		ConfigPath:     "collector.yaml",
		Engine:         model.RuntimeEngineDocker,
		RuntimeBackend: "docker",
		Mode:           model.RuntimeModeSafe,
		CollectorImage: "otel/test:latest",
		StartedAt:      time.Unix(0, 0).UTC(),
		Assertions: []model.AssertionResult{
			{
				ID:           "traces-received",
				Signal:       model.SignalTraces,
				Status:       "FAIL",
				Observed:     "0",
				Expected:     ">=1",
				LikelyCauses: []string{"pipeline wiring"},
				NextSteps:    []string{"open collector.log"},
			},
		},
		Diff: model.DiffResult{
			Changes: []model.DiffChange{{Severity: model.SeverityFail, Message: "pipeline wiring changed"}},
		},
		Artifacts: model.ArtifactManifest{
			GraphSVG: "/tmp/graph.svg",
		},
	}

	out := RenderSummaryMarkdown(result)
	for _, fragment := range []string{
		"### Top failure",
		"pipeline wiring",
		"`graph.svg`",
		"**Collector image:** `otel/test:latest`",
	} {
		if !strings.Contains(out, fragment) {
			t.Fatalf("RenderSummaryMarkdown() missing %q in:\n%s", fragment, out)
		}
	}
}
