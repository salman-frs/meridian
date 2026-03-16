package diffing

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/salman-frs/meridian/internal/collector"
	"github.com/salman-frs/meridian/internal/configio"
	"github.com/salman-frs/meridian/internal/model"
)

type Options struct {
	OldPath         string
	NewPath         string
	BaseRef         string
	HeadRef         string
	EnvFile         string
	EnvInline       []string
	Env             map[string]string
	Threshold       string
	CollectorBinary string
	CollectorImage  string
	Engine          model.RuntimeEngine
}

func Run(opts Options) (model.DiffResult, error) {
	oldPath := opts.OldPath
	newPath := opts.NewPath
	if opts.BaseRef != "" || opts.HeadRef != "" {
		if opts.BaseRef == "" || opts.HeadRef == "" || newPath == "" {
			return model.DiffResult{}, fmt.Errorf("provide both --base and --head together with --new")
		}
		var err error
		oldPath, err = materializeRef(opts.BaseRef, newPath)
		if err != nil {
			return model.DiffResult{}, err
		}
		newPath, err = materializeRef(opts.HeadRef, newPath)
		if err != nil {
			return model.DiffResult{}, err
		}
	}
	if oldPath == "" || newPath == "" {
		return model.DiffResult{}, fmt.Errorf("provide --old and --new, or --base/--head with --new")
	}

	oldCfg, err := configio.LoadConfig(configio.LoadOptions{ConfigPath: oldPath, EnvFile: opts.EnvFile, EnvInline: opts.EnvInline})
	if err != nil {
		return model.DiffResult{}, err
	}
	newCfg, err := configio.LoadConfig(configio.LoadOptions{ConfigPath: newPath, EnvFile: opts.EnvFile, EnvInline: opts.EnvInline})
	if err != nil {
		return model.DiffResult{}, err
	}

	result := model.DiffResult{
		OldConfig: oldPath,
		NewConfig: newPath,
	}

	compareOld := oldCfg
	compareNew := newCfg
	if oldEffective, newEffective, ok := effectiveConfigs(opts, oldPath, newPath, oldCfg, newCfg); ok {
		compareOld = oldEffective
		compareNew = newEffective
		result.ComparedEffective = true
		result.OldEffectiveConfig = oldPath
		result.NewEffectiveConfig = newPath
	}

	changes := []model.DiffChange{}
	changes = append(changes, pipelineChanges(oldCfg, newCfg)...)
	changes = append(changes, componentChanges("receiver", compareOld.Receivers, compareNew.Receivers)...)
	changes = append(changes, componentChanges("processor", compareOld.Processors, compareNew.Processors)...)
	changes = append(changes, componentChanges("exporter", compareOld.Exporters, compareNew.Exporters)...)
	changes = append(changes, componentChanges("connector", compareOld.Connectors, compareNew.Connectors)...)
	changes = append(changes, componentChanges("extension", compareOld.Extensions, compareNew.Extensions)...)
	changes = append(changes, envChanges(oldCfg, newCfg)...)
	changes = append(changes, nestedComponentChanges("auth", compareOld, compareNew)...)
	changes = append(changes, nestedComponentChanges("tls", compareOld, compareNew)...)
	changes = append(changes, serviceTelemetryChanges(compareOld, compareNew)...)
	result.Changes = filterBySeverity(changes, opts.Threshold)
	return result, nil
}

func Empty() model.DiffResult {
	return model.DiffResult{Changes: []model.DiffChange{}}
}

func pipelineChanges(oldCfg, newCfg model.ConfigModel) []model.DiffChange {
	changes := []model.DiffChange{}
	for name, oldPipe := range oldCfg.Pipelines {
		newPipe, ok := newCfg.Pipelines[name]
		if !ok {
			changes = append(changes, change(model.SeverityWarn, "pipeline-removed", fmt.Sprintf("pipeline %q was removed", name), "confirm the downstream signal is intentionally disabled"))
			continue
		}
		if !slices.Equal(oldPipe.Receivers, newPipe.Receivers) || !slices.Equal(oldPipe.Exporters, newPipe.Exporters) {
			changes = append(changes, change(model.SeverityFail, "pipeline-wiring-changed", fmt.Sprintf("pipeline %q changed receiver or exporter wiring", name), "review the graph and confirm traffic still reaches the intended destination"))
		}
		if !slices.Equal(oldPipe.Processors, newPipe.Processors) {
			kind := "processor-chain-changed"
			message := fmt.Sprintf("pipeline %q changed processors", name)
			hint := "processor changes can affect mutation, filtering, and batching behavior"
			if slices.Equal(sorted(oldPipe.Processors), sorted(newPipe.Processors)) {
				kind = "processor-order-changed"
				message = fmt.Sprintf("pipeline %q changed processor ordering", name)
				hint = "processor order can change mutation, filtering, and batching behavior"
			}
			changes = append(changes, change(model.SeverityWarn, kind, message, hint))
		}
	}
	for name := range newCfg.Pipelines {
		if _, ok := oldCfg.Pipelines[name]; !ok {
			changes = append(changes, change(model.SeverityInfo, "pipeline-added", fmt.Sprintf("pipeline %q was added", name), "ensure the new pipeline is covered by runtime tests"))
		}
	}
	return changes
}

func sorted(items []string) []string {
	cloned := slices.Clone(items)
	slices.Sort(cloned)
	return cloned
}

func componentChanges(kind string, oldItems, newItems map[string]model.Component) []model.DiffChange {
	changes := []model.DiffChange{}
	for name, oldComponent := range oldItems {
		newComponent, ok := newItems[name]
		if !ok {
			changes = append(changes, change(model.SeverityWarn, kind+"-removed", fmt.Sprintf("%s %q was removed", kind, name), "review whether any pipeline or env reference still depends on it"))
			continue
		}
		oldEndpoint, _ := oldComponent.Config["endpoint"].(string)
		newEndpoint, _ := newComponent.Config["endpoint"].(string)
		if oldEndpoint != "" && newEndpoint != "" && oldEndpoint != newEndpoint {
			changes = append(changes, change(model.SeverityFail, kind+"-endpoint-changed", fmt.Sprintf("%s %q changed endpoint from %s to %s", kind, name, oldEndpoint, newEndpoint), "endpoint changes are high risk and should be validated with runtime tests"))
		}
		if !reflect.DeepEqual(oldComponent.Config, newComponent.Config) {
			changes = append(changes, change(model.SeverityWarn, kind+"-config-changed", fmt.Sprintf("%s %q changed configuration", kind, name), "review the component config diff because behavior may have changed without a wiring change"))
		}
	}
	for name := range newItems {
		if _, ok := oldItems[name]; !ok {
			changes = append(changes, change(model.SeverityInfo, kind+"-added", fmt.Sprintf("%s %q was added", kind, name), "confirm it is intentional and referenced by a pipeline"))
		}
	}
	return dedupeChanges(changes)
}

func envChanges(oldCfg, newCfg model.ConfigModel) []model.DiffChange {
	oldSet := map[string]struct{}{}
	for _, ref := range oldCfg.EnvReferences {
		oldSet[ref.Name] = struct{}{}
	}
	newSet := map[string]struct{}{}
	for _, ref := range newCfg.EnvReferences {
		newSet[ref.Name] = struct{}{}
	}
	changes := []model.DiffChange{}
	for name := range newSet {
		if _, ok := oldSet[name]; ok {
			continue
		}
		changes = append(changes, change(model.SeverityWarn, "env-reference-added", fmt.Sprintf("new config now references env var %q", name), "make sure CI and local runs provide the variable"))
	}
	for name := range oldSet {
		if _, ok := newSet[name]; ok {
			continue
		}
		changes = append(changes, change(model.SeverityInfo, "env-reference-removed", fmt.Sprintf("new config no longer references env var %q", name), "confirm the environment dependency was intentionally removed"))
	}
	return changes
}

func materializeRef(ref string, path string) (string, error) {
	cmd := exec.Command("git", "show", ref+":"+path)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to read %s at %s: %w", path, ref, err)
	}
	pattern := strings.ReplaceAll(filepath.Base(path), ".", "_") + "." + strings.ReplaceAll(ref, "/", "_") + ".*.yaml"
	file, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	if _, err := file.Write(output); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return file.Name(), nil
}

func change(severity model.Severity, kind, message, hint string) model.DiffChange {
	return model.DiffChange{
		Severity:   severity,
		Kind:       kind,
		Message:    message,
		ReviewHint: hint,
	}
}

func filterBySeverity(changes []model.DiffChange, threshold string) []model.DiffChange {
	thresholdSeverity := parseThreshold(threshold)
	if thresholdSeverity == "" {
		return changes
	}
	filtered := make([]model.DiffChange, 0, len(changes))
	for _, item := range changes {
		if model.SeverityRank(item.Severity) < model.SeverityRank(thresholdSeverity) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func parseThreshold(value string) model.Severity {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "low":
		return model.SeverityInfo
	case "medium":
		return model.SeverityWarn
	case "high":
		return model.SeverityFail
	default:
		return model.SeverityInfo
	}
}

func nestedComponentChanges(key string, oldCfg, newCfg model.ConfigModel) []model.DiffChange {
	changes := []model.DiffChange{}
	for _, section := range []struct {
		kind string
		data map[string]model.Component
		old  map[string]model.Component
	}{
		{kind: "receiver", data: newCfg.Receivers, old: oldCfg.Receivers},
		{kind: "processor", data: newCfg.Processors, old: oldCfg.Processors},
		{kind: "exporter", data: newCfg.Exporters, old: oldCfg.Exporters},
		{kind: "connector", data: newCfg.Connectors, old: oldCfg.Connectors},
		{kind: "extension", data: newCfg.Extensions, old: oldCfg.Extensions},
	} {
		for name, newComponent := range section.data {
			oldComponent, ok := section.old[name]
			if !ok {
				continue
			}
			if reflect.DeepEqual(oldComponent.Config[key], newComponent.Config[key]) {
				continue
			}
			changes = append(changes, change(model.SeverityWarn, section.kind+"-"+key+"-changed", fmt.Sprintf("%s %q changed %s settings", section.kind, name, key), key+" settings can change connectivity, auth, or transport behavior"))
		}
	}
	return dedupeChanges(changes)
}

func serviceTelemetryChanges(oldCfg, newCfg model.ConfigModel) []model.DiffChange {
	oldService, _ := oldCfg.Raw["service"].(map[string]any)
	newService, _ := newCfg.Raw["service"].(map[string]any)
	if reflect.DeepEqual(oldService["telemetry"], newService["telemetry"]) {
		return nil
	}
	return []model.DiffChange{
		change(model.SeverityWarn, "service-telemetry-changed", "service.telemetry changed", "collector internal telemetry settings changed; review metrics and logs coverage for troubleshooting impact"),
	}
}

func dedupeChanges(changes []model.DiffChange) []model.DiffChange {
	seen := map[string]struct{}{}
	out := make([]model.DiffChange, 0, len(changes))
	for _, item := range changes {
		key := string(item.Severity) + "|" + item.Kind + "|" + item.Message
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func effectiveConfigs(opts Options, oldPath, newPath string, oldCfg, newCfg model.ConfigModel) (model.ConfigModel, model.ConfigModel, bool) {
	if opts.CollectorBinary == "" && opts.CollectorImage == "" {
		return model.ConfigModel{}, model.ConfigModel{}, false
	}
	oldFinal, okOld, errOld := collector.ResolveFinalConfig(collector.Options{
		ConfigSources:   []string{oldPath},
		ConfigModel:     oldCfg,
		Env:             opts.Env,
		CollectorBinary: opts.CollectorBinary,
		CollectorImage:  opts.CollectorImage,
		Engine:          opts.Engine,
	})
	newFinal, okNew, errNew := collector.ResolveFinalConfig(collector.Options{
		ConfigSources:   []string{newPath},
		ConfigModel:     newCfg,
		Env:             opts.Env,
		CollectorBinary: opts.CollectorBinary,
		CollectorImage:  opts.CollectorImage,
		Engine:          opts.Engine,
	})
	if errOld != nil || errNew != nil || !okOld || !okNew {
		return model.ConfigModel{}, model.ConfigModel{}, false
	}
	oldEffective, err := collector.LoadEffectiveConfig(oldFinal, oldPath+"#print-config")
	if err != nil {
		return model.ConfigModel{}, model.ConfigModel{}, false
	}
	newEffective, err := collector.LoadEffectiveConfig(newFinal, newPath+"#print-config")
	if err != nil {
		return model.ConfigModel{}, model.ConfigModel{}, false
	}
	return oldEffective, newEffective, true
}
