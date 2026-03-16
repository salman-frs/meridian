package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/salman-frs/meridian/internal/model"
)

func TestHelpGoldens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		file string
	}{
		{name: "root", args: []string{"--help"}, file: "root_help.txt.golden"},
		{name: "validate", args: []string{"validate", "--help"}, file: "validate_help.txt.golden"},
		{name: "graph", args: []string{"graph", "--help"}, file: "graph_help.txt.golden"},
		{name: "ci", args: []string{"ci", "--help"}, file: "ci_help.txt.golden"},
		{name: "debug", args: []string{"debug", "--help"}, file: "debug_help.txt.golden"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stdout, stderr, err := executeRootCommand(t, tt.args...)
			if err != nil {
				t.Fatalf("executeRootCommand(%v) error = %v", tt.args, err)
			}
			if stderr != "" {
				t.Fatalf("executeRootCommand(%v) stderr = %q, want empty", tt.args, stderr)
			}
			if got, want := strings.TrimSpace(stdout), readAppGolden(t, tt.file); got != want {
				t.Fatalf("help mismatch for %s\nwant:\n%s\n\ngot:\n%s", tt.name, want, got)
			}
		})
	}
}

func TestGraphDefaultOutputIsMermaidOnly(t *testing.T) {
	t.Parallel()

	stdout, stderr, err := executeRootCommand(t, "graph", "-c", exampleConfigPath(t, "basic", "collector.yaml"))
	if err != nil {
		t.Fatalf("executeRootCommand(graph) error = %v", err)
	}
	if stderr != "" {
		t.Fatalf("graph stderr = %q, want empty", stderr)
	}
	if strings.Contains(stdout, "PIPELINE | RECEIVERS") {
		t.Fatalf("graph default output unexpectedly included table:\n%s", stdout)
	}
	if !strings.Contains(stdout, "flowchart LR") {
		t.Fatalf("graph default output = %q, want Mermaid graph", stdout)
	}
}

func TestGraphViewTablePrintsOnlyTable(t *testing.T) {
	t.Parallel()

	stdout, stderr, err := executeRootCommand(t, "graph", "-c", exampleConfigPath(t, "basic", "collector.yaml"), "--view", "table")
	if err != nil {
		t.Fatalf("executeRootCommand(graph --view table) error = %v", err)
	}
	if stderr != "" {
		t.Fatalf("graph stderr = %q, want empty", stderr)
	}
	if strings.Contains(stdout, "flowchart LR") {
		t.Fatalf("graph table output unexpectedly included Mermaid graph:\n%s", stdout)
	}
	if !strings.Contains(stdout, "PIPELINE | RECEIVERS | PROCESSORS | EXPORTERS") {
		t.Fatalf("graph table output = %q, want table header", stdout)
	}
}

func TestGraphAsciiAliasStillParses(t *testing.T) {
	t.Parallel()

	stdout, stderr, err := executeRootCommand(t, "graph", "-c", exampleConfigPath(t, "basic", "collector.yaml"), "--view", "ascii")
	if err != nil {
		t.Fatalf("executeRootCommand(graph --view ascii) error = %v", err)
	}
	if !strings.Contains(stderr, "deprecated") {
		t.Fatalf("graph --view ascii stderr = %q, want deprecation warning", stderr)
	}
	if !strings.Contains(stdout, "PIPELINE | RECEIVERS | PROCESSORS | EXPORTERS") {
		t.Fatalf("graph --view ascii output = %q, want table output", stdout)
	}
}

func TestDeprecatedFlagsStillParseDuringHelp(t *testing.T) {
	t.Parallel()

	tests := [][]string{
		{"validate", "--rules", "default", "--help"},
		{"check", "--output", t.TempDir(), "--help"},
	}

	for _, args := range tests {
		stdout, _, err := executeRootCommand(t, args...)
		if err != nil {
			t.Fatalf("executeRootCommand(%v) error = %v", args, err)
		}
		if !strings.Contains(stdout, "Usage:") {
			t.Fatalf("executeRootCommand(%v) stdout = %q, want help output", args, stdout)
		}
	}
}

func TestCIJSONStdoutRemainsPureJSON(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	summaryPath := filepath.Join(tempDir, "summary.md")
	jsonPath := filepath.Join(tempDir, "report.json")
	commentPath := filepath.Join(tempDir, "pr-comment.md")

	restore := stubRunService(t, stubRunServiceFunc(func(global *GlobalOptions, runtimeOpts *RuntimeOptions, includeDiff bool) (model.RunResult, error) {
		return sampleRunResultForCommandTests(tempDir), nil
	}))
	defer restore()

	stdout, stderr, err := executeRootCommand(
		t,
		"ci",
		"--format", "json",
		"--summary-file", summaryPath,
		"--json-file", jsonPath,
		"--pr-comment-file", commentPath,
	)
	if err != nil && ExitCode(err) != 2 {
		t.Fatalf("executeRootCommand(ci --format json) error = %v", err)
	}
	if !json.Valid([]byte(stdout)) {
		t.Fatalf("ci stdout = %q, want valid JSON only", stdout)
	}
	if !strings.Contains(stderr, "::error") {
		t.Fatalf("ci stderr = %q, want annotations on stderr", stderr)
	}
	for _, path := range []string{summaryPath, jsonPath, commentPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected output file %s: %v", path, err)
		}
	}
}

func TestDebugSummaryResolvesLatestFromCustomOutputRoot(t *testing.T) {
	t.Parallel()

	outputRoot := t.TempDir()
	runDir := filepath.Join(outputRoot, "runs", "run-123")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(runDir) error = %v", err)
	}
	summary := "# Stored summary\n"
	if err := os.WriteFile(filepath.Join(runDir, "summary.md"), []byte(summary), 0o644); err != nil {
		t.Fatalf("WriteFile(summary.md) error = %v", err)
	}
	if err := os.Symlink(runDir, filepath.Join(outputRoot, "runs", "latest")); err != nil {
		t.Fatalf("Symlink(latest) error = %v", err)
	}

	stdout, stderr, err := executeRootCommand(t, "debug", "--output", outputRoot, "summary")
	if err != nil {
		t.Fatalf("executeRootCommand(debug summary) error = %v", err)
	}
	if stderr != "" {
		t.Fatalf("debug summary stderr = %q, want empty", stderr)
	}
	if stdout != summary {
		t.Fatalf("debug summary stdout = %q, want %q", stdout, summary)
	}
}

type stubRunServiceFunc func(global *GlobalOptions, runtimeOpts *RuntimeOptions, includeDiff bool) (model.RunResult, error)

func (f stubRunServiceFunc) Execute(global *GlobalOptions, runtimeOpts *RuntimeOptions, includeDiff bool) (model.RunResult, error) {
	return f(global, runtimeOpts, includeDiff)
}

func stubRunService(t *testing.T, service interface{ Execute(*GlobalOptions, *RuntimeOptions, bool) (model.RunResult, error) }) func() {
	t.Helper()
	originalFactory := runServiceFactory
	originalExecute := executeRunService
	executeRunService = func(_ RunService, global *GlobalOptions, runtimeOpts *RuntimeOptions, includeDiff bool) (model.RunResult, error) {
		return service.Execute(global, runtimeOpts, includeDiff)
	}
	return func() {
		runServiceFactory = originalFactory
		executeRunService = originalExecute
	}
}

func executeRootCommand(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	cmd := NewRootCommand()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func readAppGolden(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", name, err)
	}
	return strings.TrimSpace(string(data))
}

func exampleConfigPath(t *testing.T, parts ...string) string {
	t.Helper()
	pathParts := append([]string{"..", "..", "examples"}, parts...)
	return filepath.Join(pathParts...)
}

func sampleRunResultForCommandTests(outputRoot string) model.RunResult {
	return model.RunResult{
		RunID:          "fake-run",
		ConfigPath:     "collector.yaml",
		Status:         "FAIL",
		Engine:         model.RuntimeEngineDocker,
		RuntimeBackend: "docker",
		Mode:           model.RuntimeModeSafe,
		CollectorImage: "otel/test:latest",
		StartedAt:      time.Unix(0, 0).UTC(),
		FinishedAt:     time.Unix(1, 0).UTC(),
		Timings:        map[string]string{"total": "1s"},
		Semantic:       model.SemanticReport{Target: "otel/test:latest"},
		Artifacts:      model.NewArtifactManifest(outputRoot, "fake-run"),
		Findings: []model.Finding{
			{Severity: model.SeverityFail, Code: "collector-validate-failed", Message: "collector-native validation failed"},
		},
	}
}
