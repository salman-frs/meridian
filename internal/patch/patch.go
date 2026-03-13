package patch

import (
	"fmt"
	"slices"
	"time"

	"github.com/salman-frs/meridian/internal/model"
)

const (
	injectedReceiverName = "otlp/meridian_in"
	captureExporterName  = "otlp/meridian_capture"
)

type Options struct {
	RunID          string
	Mode           model.RuntimeMode
	CollectorImage string
	Timeout        time.Duration
	StartupTimeout time.Duration
	InjectTimeout  time.Duration
	CaptureTimeout time.Duration
	PipelineArgs   []string
	InjectionPort  int
	CapturePort    int
	CaptureSamples int
}

func Build(cfg model.ConfigModel, opts Options) (model.ConfigModel, model.TestPlan, error) {
	selected := selectPipelines(cfg, opts.PipelineArgs)
	if len(selected) == 0 {
		selected = cfg.PipelineNames()
	}
	signals := []model.SignalType{}
	seenSignals := map[model.SignalType]struct{}{}
	for _, name := range selected {
		signal := cfg.Pipelines[name].Signal
		if _, ok := seenSignals[signal]; ok {
			continue
		}
		seenSignals[signal] = struct{}{}
		signals = append(signals, signal)
	}

	raw := deepCloneMap(cfg.Raw)
	receivers := ensureSection(raw, "receivers")
	receivers[injectedReceiverName] = map[string]any{
		"protocols": map[string]any{
			"grpc": map[string]any{"endpoint": fmt.Sprintf("0.0.0.0:%d", opts.InjectionPort)},
		},
	}
	exporters := ensureSection(raw, "exporters")
	exporters[captureExporterName] = map[string]any{
		"endpoint": fmt.Sprintf("host.docker.internal:%d", opts.CapturePort),
		"compression": "none",
		"tls": map[string]any{
			"insecure": true,
		},
		"headers": map[string]any{
			"x-meridian-run-id": opts.RunID,
		},
	}
	service := ensureSection(raw, "service")
	pipelines := ensureSection(service, "pipelines")
	for name, value := range pipelines {
		pipelineMap, _ := value.(map[string]any)
		if pipelineMap == nil {
			continue
		}
		if !slices.Contains(selected, name) {
			continue
		}
		pipelineMap["receivers"] = []string{injectedReceiverName}
		switch opts.Mode {
		case model.RuntimeModeTee:
			exporterList := stringSlice(pipelineMap["exporters"])
			if !slices.Contains(exporterList, captureExporterName) {
				exporterList = append(exporterList, captureExporterName)
			}
			pipelineMap["exporters"] = exporterList
		case model.RuntimeModeLive:
			exporterList := stringSlice(pipelineMap["exporters"])
			if !slices.Contains(exporterList, captureExporterName) {
				exporterList = append(exporterList, captureExporterName)
			}
			pipelineMap["exporters"] = exporterList
		default:
			exporterList := keepConnectorExporters(stringSlice(pipelineMap["exporters"]), cfg.Connectors)
			exporterList = append(exporterList, captureExporterName)
			pipelineMap["exporters"] = exporterList
		}
		pipelines[name] = pipelineMap
	}

	patched := model.ConfigModel{
		SourcePaths: cfg.SourcePaths,
		Raw:         raw,
		Receivers:   cfg.Receivers,
		Processors:  cfg.Processors,
		Exporters:   cfg.Exporters,
		Connectors:  cfg.Connectors,
		Extensions:  cfg.Extensions,
		Pipelines:   cfg.Pipelines,
	}
	patchedYAML, err := model.MarshalYAML(raw)
	if err != nil {
		return model.ConfigModel{}, model.TestPlan{}, err
	}
	patched.CanonicalYAML = patchedYAML
	plan := model.TestPlan{
		RunID:             opts.RunID,
		Mode:              opts.Mode,
		CollectorImage:    opts.CollectorImage,
		Pipelines:         selected,
		Signals:           signals,
		InjectedReceiver:  injectedReceiverName,
		CaptureEndpoint:   fmt.Sprintf("host.docker.internal:%d", opts.CapturePort),
		InjectionEndpoint: fmt.Sprintf("127.0.0.1:%d", opts.InjectionPort),
		InjectionPort:     opts.InjectionPort,
		CapturePort:       opts.CapturePort,
		Timeout:           opts.Timeout.String(),
		StartupTimeout:    opts.StartupTimeout.String(),
		InjectTimeout:     opts.InjectTimeout.String(),
		CaptureTimeout:    opts.CaptureTimeout.String(),
		CaptureSamples:    opts.CaptureSamples,
	}
	return patched, plan, nil
}

func selectPipelines(cfg model.ConfigModel, requested []string) []string {
	if len(requested) == 0 {
		return cfg.PipelineNames()
	}
	selected := []string{}
	for _, candidate := range requested {
		for name, pipeline := range cfg.Pipelines {
			if name == candidate || string(pipeline.Signal) == candidate {
				selected = append(selected, name)
			}
		}
	}
	return slices.Compact(selected)
}

func deepCloneMap(input map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range input {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = deepCloneMap(typed)
		case []any:
			cloned := make([]any, 0, len(typed))
			for _, item := range typed {
				if itemMap, ok := item.(map[string]any); ok {
					cloned = append(cloned, deepCloneMap(itemMap))
					continue
				}
				cloned = append(cloned, item)
			}
			out[key] = cloned
		case []string:
			out[key] = slices.Clone(typed)
		default:
			out[key] = value
		}
	}
	return out
}

func ensureSection(root map[string]any, key string) map[string]any {
	section, _ := root[key].(map[string]any)
	if section == nil {
		section = map[string]any{}
		root[key] = section
	}
	return section
}

func stringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return slices.Clone(typed)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, fmt.Sprint(item))
		}
		return out
	default:
		return nil
	}
}

func keepConnectorExporters(exporters []string, connectors map[string]model.Component) []string {
	out := []string{}
	for _, exporter := range exporters {
		if _, ok := connectors[exporter]; ok {
			out = append(out, exporter)
		}
	}
	return out
}
