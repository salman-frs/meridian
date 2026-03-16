package app

import (
	"strings"
	"testing"

	"github.com/salman-frs/meridian/internal/model"
)

func TestSelectRuntimeConfigUsesSourceConfigWhenSourcesAreMaterializable(t *testing.T) {
	t.Parallel()

	inputs := runInputs{
		configSources: []string{"yaml:receivers:\n  otlp: {}\nservice:\n  pipelines:\n    traces:\n      receivers: [otlp]\n      exporters: [debug]\n"},
		sourceConfig: model.ConfigModel{
			SourcePaths: []string{"yaml:receivers:\n  otlp: {}\nservice:\n  pipelines:\n    traces:\n      receivers: [otlp]\n      exporters: [debug]\n"},
			Raw:         map[string]any{"service": map[string]any{}},
			Pipelines: map[string]model.PipelineModel{
				"metrics": {Name: "metrics"},
			},
		},
	}
	semantic := model.SemanticReport{
		FinalConfig: "receivers:\n  otlp: {}\nexporters:\n  debug: {}\nservice:\n  pipelines:\n    traces:\n      receivers: [otlp]\n      exporters: [debug]\n",
	}

	cfg, source, err := selectRuntimeConfig(inputs, semantic)
	if err != nil {
		t.Fatalf("selectRuntimeConfig() error = %v", err)
	}
	if _, ok := cfg.Pipelines["metrics"]; !ok {
		t.Fatalf("selectRuntimeConfig() = %#v, want source config pipelines", cfg.Pipelines)
	}
	if source != "source-merged config" {
		t.Fatalf("selectRuntimeConfig() source = %q, want source-merged config", source)
	}
	if got := cfg.SourcePaths[0]; !strings.HasPrefix(got, "yaml:receivers:") {
		t.Fatalf("selectRuntimeConfig() source path = %q, want original source", got)
	}
}

func TestSelectRuntimeConfigRejectsNonLocalSourcesWithoutPrintConfig(t *testing.T) {
	t.Parallel()

	_, _, err := selectRuntimeConfig(runInputs{
		configSources: []string{"env:MERIDIAN_CONFIG"},
		sourceConfig:  model.ConfigModel{SourcePaths: []string{"env:MERIDIAN_CONFIG"}},
	}, model.SemanticReport{})
	if err == nil || !strings.Contains(err.Error(), "print-config") {
		t.Fatalf("selectRuntimeConfig() error = %v, want print-config failure", err)
	}
}

func TestSelectRuntimeConfigFallsBackToSourceConfigForLocalOnly(t *testing.T) {
	t.Parallel()

	sourceConfig := model.ConfigModel{
		SourcePaths: []string{"collector.yaml"},
		Raw:         map[string]any{"service": map[string]any{}},
		Pipelines: map[string]model.PipelineModel{
			"traces": {Name: "traces"},
		},
	}
	cfg, sourceLabel, err := selectRuntimeConfig(runInputs{
		configSources: []string{"collector.yaml"},
		sourceConfig:  sourceConfig,
	}, model.SemanticReport{})
	if err != nil {
		t.Fatalf("selectRuntimeConfig() error = %v", err)
	}
	if got := len(cfg.Pipelines); got != 1 {
		t.Fatalf("selectRuntimeConfig() pipelines = %d, want 1", got)
	}
	if sourceLabel != "repo-local config" {
		t.Fatalf("selectRuntimeConfig() source = %q, want repo-local config", sourceLabel)
	}
}

func TestSelectRuntimeConfigUsesRenderedConfigWhenSourcesAreNotMaterializable(t *testing.T) {
	t.Parallel()

	source := model.ConfigModel{
		SourcePaths: []string{"collector.yaml", "env:MERIDIAN_CONFIG"},
		Raw:         map[string]any{"service": map[string]any{}},
		Pipelines: map[string]model.PipelineModel{
			"metrics": {Name: "metrics"},
		},
	}
	semantic := model.SemanticReport{
		FinalConfig: "receivers:\n  otlp: {}\nexporters:\n  debug: {}\nservice:\n  pipelines:\n    traces:\n      receivers: [otlp]\n      exporters: [debug]\n",
	}
	cfg, sourceLabel, err := selectRuntimeConfig(runInputs{
		configSources: []string{"collector.yaml", "env:MERIDIAN_CONFIG"},
		sourceConfig:  source,
	}, semantic)
	if err != nil {
		t.Fatalf("selectRuntimeConfig() error = %v", err)
	}
	if _, ok := cfg.Pipelines["traces"]; !ok {
		t.Fatalf("selectRuntimeConfig() = %#v, want rendered traces pipeline", cfg.Pipelines)
	}
	if sourceLabel != "collector-rendered effective config" {
		t.Fatalf("selectRuntimeConfig() source = %q, want collector-rendered effective config", sourceLabel)
	}
}
