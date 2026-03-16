package collector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/salman-frs/meridian/internal/model"
)

func TestAnalyzeWithCollectorBinary(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "collector.yaml")
	if err := os.WriteFile(configPath, []byte("receivers:\n  otlp: {}\nprocessors:\n  batch: {}\nexporters:\n  debug: {}\nservice:\n  pipelines:\n    traces:\n      receivers: [otlp]\n      processors: [batch]\n      exporters: [debug]\n"), 0o755); err != nil {
		t.Fatalf("WriteFile(config) error = %v", err)
	}
	binaryPath := writeFakeCollector(t, dir)

	report, err := Analyze(Options{
		ConfigSources:   []string{configPath},
		ConfigModel:     model.ConfigModel{SourcePaths: []string{configPath}, Receivers: map[string]model.Component{"otlp": {Name: "otlp"}}, Processors: map[string]model.Component{"batch": {Name: "batch"}}, Exporters: map[string]model.Component{"debug": {Name: "debug"}}},
		CollectorBinary: binaryPath,
		RequireSemantic: true,
	})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if report.Status != "PASS" && report.Status != "INFO" {
		t.Fatalf("Analyze() status = %q", report.Status)
	}
	if len(report.Components) == 0 {
		t.Fatal("Analyze() components empty, want parsed inventory")
	}
	if !strings.Contains(report.FinalConfig, "service:") {
		t.Fatalf("Analyze() final config = %q, want rendered config", report.FinalConfig)
	}
}

func TestAnalyzeFlagsUnsupportedComponentFromInventory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "collector.yaml")
	if err := os.WriteFile(configPath, []byte("exporters:\n  notreal: {}\nservice:\n  pipelines:\n    traces:\n      receivers: []\n      exporters: [notreal]\n"), 0o755); err != nil {
		t.Fatalf("WriteFile(config) error = %v", err)
	}
	binaryPath := writeFakeCollector(t, dir)

	report, err := Analyze(Options{
		ConfigSources:   []string{configPath},
		ConfigModel:     model.ConfigModel{SourcePaths: []string{configPath}, Exporters: map[string]model.Component{"notreal": {Name: "notreal"}}},
		CollectorBinary: binaryPath,
		RequireSemantic: true,
	})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if report.Status != "FAIL" {
		t.Fatalf("Analyze() status = %q, want FAIL", report.Status)
	}
}

func writeFakeCollector(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "fake-otelcol.sh")
	script := `#!/bin/sh
set -eu
cmd="${1:-}"
shift || true
case "$cmd" in
  components)
    cat <<'EOF'
Receivers:
  otlp stable
Processors:
  batch stable
Exporters:
  debug stable
EOF
    ;;
  validate)
    echo "validation passed"
    ;;
  print-config)
    cat <<'EOF'
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug]
EOF
    ;;
  *)
    echo "unknown command: $cmd" >&2
    exit 1
    ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile(fake collector) error = %v", err)
	}
	return path
}
