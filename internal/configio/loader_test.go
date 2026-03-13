package configio

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigDirectoryMergesFilesAndTracksEnv(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "00-receivers.yaml"), "receivers:\n  otlp:\n    protocols:\n      grpc:\n        endpoint: ${OTLP_ENDPOINT}\n")
	writeFile(t, filepath.Join(dir, "10-service.yaml"), "service:\n  pipelines:\n    traces:\n      receivers: [otlp]\n      exporters: [debug]\n")
	writeFile(t, filepath.Join(dir, "20-exporters.yaml"), "exporters:\n  debug: {}\n")

	cfg, err := LoadConfig(LoadOptions{
		ConfigDir: dir,
		EnvInline: []string{"OTLP_ENDPOINT=0.0.0.0:4317"},
	})
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(cfg.SourcePaths) != 3 {
		t.Fatalf("LoadConfig() source paths = %d, want 3", len(cfg.SourcePaths))
	}
	if got := cfg.Receivers["otlp"].Config["protocols"].(map[string]any)["grpc"].(map[string]any)["endpoint"]; got != "0.0.0.0:4317" {
		t.Fatalf("LoadConfig() endpoint = %#v, want interpolated value", got)
	}
	if len(cfg.MissingEnvNames) != 0 {
		t.Fatalf("LoadConfig() missing env = %#v, want none", cfg.MissingEnvNames)
	}
}

func TestLoadConfigTracksMissingEnvOnce(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "collector.yaml"), "exporters:\n  otlp:\n    endpoint: ${MISSING}\nservice:\n  pipelines:\n    traces:\n      receivers: [otlp]\n      exporters: [otlp]\n")

	cfg, err := LoadConfig(LoadOptions{ConfigPath: filepath.Join(dir, "collector.yaml")})
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if len(cfg.MissingEnvNames) != 1 || cfg.MissingEnvNames[0] != "MISSING" {
		t.Fatalf("LoadConfig() missing env = %#v, want [MISSING]", cfg.MissingEnvNames)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}
