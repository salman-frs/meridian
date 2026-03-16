package configio

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzLoadConfigSingleFile(f *testing.F) {
	f.Add("receivers:\n  otlp:\n    protocols:\n      grpc:\nservice:\n  pipelines:\n    traces:\n      receivers: [otlp]\n      exporters: [debug]\nexporters:\n  debug: {}\n")
	f.Add("service:\n  pipelines:\n    logs:\n      receivers: [otlp]\n      exporters: [debug]\n")

	f.Fuzz(func(t *testing.T, content string) {
		dir := t.TempDir()
		path := filepath.Join(dir, "collector.yaml")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		_, _ = LoadConfig(LoadOptions{ConfigPath: path})
	})
}
