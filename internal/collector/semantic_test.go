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
		Env:             map[string]string{"MERIDIAN_TEST_ENV": "from-env"},
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
	if !strings.Contains(report.FinalConfig, "from-env") {
		t.Fatalf("Analyze() final config = %q, want env propagated", report.FinalConfig)
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

func TestMaterializeConfigParsesRenderedOutput(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "collector.yaml")
	if err := os.WriteFile(configPath, []byte("service: {}\n"), 0o755); err != nil {
		t.Fatalf("WriteFile(config) error = %v", err)
	}
	binaryPath := writeFakeCollector(t, dir)

	cfg, finalConfig, ok, err := MaterializeConfig(Options{
		ConfigSources:   []string{configPath},
		ConfigModel:     model.ConfigModel{SourcePaths: []string{configPath}},
		Env:             map[string]string{"MERIDIAN_TEST_ENV": "from-env"},
		CollectorBinary: binaryPath,
	})
	if err != nil {
		t.Fatalf("MaterializeConfig() error = %v", err)
	}
	if !ok {
		t.Fatal("MaterializeConfig() ok = false, want true")
	}
	if !strings.Contains(finalConfig, "service:") {
		t.Fatalf("MaterializeConfig() final config = %q, want rendered yaml", finalConfig)
	}
	if len(cfg.Pipelines) != 1 {
		t.Fatalf("MaterializeConfig() pipelines = %d, want 1", len(cfg.Pipelines))
	}
}

func TestResolveFinalConfigEnablesPrintConfigFeatureGate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "collector.yaml")
	if err := os.WriteFile(configPath, []byte("service: {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(config) error = %v", err)
	}
	logPath := filepath.Join(dir, "print-config.log")
	binaryPath := writeFakeCollector(t, dir)

	finalConfig, ok, err := ResolveFinalConfig(Options{
		ConfigSources:   []string{configPath},
		ConfigModel:     model.ConfigModel{SourcePaths: []string{configPath}},
		Env:             map[string]string{"MERIDIAN_PRINT_CONFIG_LOG": logPath, "MERIDIAN_TEST_ENV": "from-env"},
		CollectorBinary: binaryPath,
	})
	if err != nil {
		t.Fatalf("ResolveFinalConfig() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveFinalConfig() ok = false, want true")
	}
	if !strings.Contains(finalConfig, "service:") {
		t.Fatalf("ResolveFinalConfig() final config = %q, want rendered config", finalConfig)
	}
	loggedArgs, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile(logPath) error = %v", err)
	}
	if !strings.Contains(string(loggedArgs), "--feature-gates="+printConfigFeatureGate+" print-config --config") {
		t.Fatalf("print-config args = %q, want feature gate", string(loggedArgs))
	}
}

func TestAnalyzeSkipsPrintConfigWhenFeatureGateUnsupported(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "collector.yaml")
	if err := os.WriteFile(configPath, []byte("service: {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(config) error = %v", err)
	}
	binaryPath := writeFakeCollectorWithoutPrintConfigSupport(t, dir)

	report, err := Analyze(Options{
		ConfigSources:   []string{configPath},
		ConfigModel:     model.ConfigModel{SourcePaths: []string{configPath}},
		CollectorBinary: binaryPath,
		RequireSemantic: true,
	})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	stage := report.Stages[len(report.Stages)-1]
	if stage.Name != "print-config" || stage.Status != "SKIP" {
		t.Fatalf("print-config stage = %#v, want SKIP", stage)
	}
	if got := report.Findings[len(report.Findings)-1].Code; got != "collector-print-config-skipped" {
		t.Fatalf("last finding code = %q, want collector-print-config-skipped", got)
	}
	if report.FinalConfig != "" {
		t.Fatalf("Analyze() final config = %q, want empty when unsupported", report.FinalConfig)
	}
}

func TestSemanticContainerRunArgs(t *testing.T) {
	t.Parallel()

	if got := semanticContainerRunArgs(model.RuntimeEngineContainerd, "nerdctl"); len(got) != 2 || got[0] != "--network" || got[1] != "host" {
		t.Fatalf("semanticContainerRunArgs(containerd, nerdctl) = %#v, want host networking args", got)
	}
	if got := semanticContainerRunArgs(model.RuntimeEngineDocker, "docker"); got != nil {
		t.Fatalf("semanticContainerRunArgs(docker, docker) = %#v, want nil", got)
	}
	if got := semanticContainerRunArgs(model.RuntimeEngineContainerd, "lima nerdctl"); got != nil {
		t.Fatalf("semanticContainerRunArgs(containerd, lima nerdctl) = %#v, want nil", got)
	}
}

func TestParseComponentsStructuredOutput(t *testing.T) {
	t.Parallel()

	output := `buildinfo:
  command: otelcol-contrib
receivers:
  - name: otlp
    module: go.opentelemetry.io/collector/receiver/otlpreceiver v0.147.0
    stability:
      logs: Beta
      metrics: Beta
      traces: Beta
processors:
  - name: batch
    module: go.opentelemetry.io/collector/processor/batchprocessor v0.147.0
    stability:
      traces: Beta
exporters:
  - name: debug
    module: go.opentelemetry.io/collector/exporter/debugexporter v0.147.0
    stability:
      logs: Alpha
      metrics: Alpha
      traces: Alpha
connectors:
  - name: routing
    module: github.com/open-telemetry/opentelemetry-collector-contrib/connector/routingconnector v0.147.0
    stability:
      traces-to-traces: Alpha
extensions:
  - name: health_check
    module: github.com/open-telemetry/opentelemetry-collector-contrib/extension/healthcheckextension v0.147.0
    stability:
      extension: Alpha
providers:
  - scheme: yaml
    module: go.opentelemetry.io/collector/confmap/provider/yamlprovider v1.53.0
`

	components := parseComponents(output)
	if len(components) != 5 {
		t.Fatalf("parseComponents() count = %d, want 5", len(components))
	}
	if got := components[0]; got.Kind != "receiver" || got.Name != "otlp" || got.Stability != "beta" {
		t.Fatalf("parseComponents() receiver = %#v, want otlp beta receiver", got)
	}
	if got := components[2]; got.Kind != "exporter" || got.Name != "debug" || got.Stability != "alpha" {
		t.Fatalf("parseComponents() exporter = %#v, want debug alpha exporter", got)
	}
}

func writeFakeCollector(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "fake-otelcol.sh")
	script := `#!/bin/sh
set -eu
PRINT_CONFIG_LOG="${MERIDIAN_PRINT_CONFIG_LOG:-}"
cmd="${1:-}"
if [ -n "$PRINT_CONFIG_LOG" ]; then
  printf '%s\n' "$*" >> "$PRINT_CONFIG_LOG"
fi
while [ "${cmd#--feature-gates=}" != "$cmd" ]; do
  shift || true
  cmd="${1:-}"
done
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
    cat <<EOF
service:
  telemetry:
    logs:
      level: ${MERIDIAN_TEST_ENV}
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

func writeFakeCollectorWithoutPrintConfigSupport(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "fake-otelcol-no-print-config.sh")
	script := `#!/bin/sh
set -eu
cmd="${1:-}"
if [ "${cmd#--feature-gates=}" != "$cmd" ]; then
  echo "unknown flag: ${cmd%%=*}" >&2
  exit 1
fi
shift || true
case "$cmd" in
  components)
    echo "Receivers:"
    ;;
  validate)
    echo "validation passed"
    ;;
  *)
    echo "unknown command: $cmd" >&2
    exit 1
    ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile(fake collector no print-config) error = %v", err)
	}
	return path
}
