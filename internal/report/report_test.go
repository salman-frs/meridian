package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/salman-frs/meridian/internal/model"
)

func TestRenderSummaryMarkdownIncludesTopFailureAndArtifacts(t *testing.T) {
	t.Parallel()

	result := sampleRunResult()

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

func TestRenderSummaryMarkdownGolden(t *testing.T) {
	t.Parallel()

	got := RenderSummaryMarkdown(sampleRunResult())
	got = strings.TrimSuffix(got, "\n")
	want := readGolden(t, "summary.md.golden")
	if got != want {
		t.Fatalf("RenderSummaryMarkdown() mismatch\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

func TestRenderTerminalGolden(t *testing.T) {
	t.Parallel()

	got := RenderTerminal(sampleRunResult())
	got = strings.TrimSuffix(got, "\n")
	want := readGolden(t, "terminal.txt.golden")
	if got != want {
		t.Fatalf("RenderTerminal() mismatch\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

func sampleRunResult() model.RunResult {
	return model.RunResult{
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
				Message:      "received at least one item",
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
		Graph: model.GraphModel{
			Nodes: []model.GraphNode{{ID: "pipeline-traces", Label: "traces"}},
		},
		Artifacts: model.ArtifactManifest{
			ReportJSON: "",
			SummaryMD:  "",
			GraphMMD:   "",
			GraphSVG:   "/tmp/graph.svg",
			DiffMD:     "",
		},
	}
}

func readGolden(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return strings.TrimSuffix(string(data), "\n")
}
