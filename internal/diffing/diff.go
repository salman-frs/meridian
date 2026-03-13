package diffing

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/salman-frs/meridian/internal/configio"
	"github.com/salman-frs/meridian/internal/model"
)

type Options struct {
	OldPath   string
	NewPath   string
	BaseRef   string
	HeadRef   string
	EnvFile   string
	EnvInline []string
	Threshold string
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
	changes := []model.DiffChange{}
	changes = append(changes, pipelineChanges(oldCfg, newCfg)...)
	changes = append(changes, componentChanges("receiver", oldCfg.Receivers, newCfg.Receivers)...)
	changes = append(changes, componentChanges("exporter", oldCfg.Exporters, newCfg.Exporters)...)
	changes = append(changes, envChanges(oldCfg, newCfg)...)
	changes = filterBySeverity(changes, opts.Threshold)
	return model.DiffResult{
		OldConfig: oldPath,
		NewConfig: newPath,
		Changes:   changes,
	}, nil
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
	}
	for name := range newItems {
		if _, ok := oldItems[name]; !ok {
			changes = append(changes, change(model.SeverityInfo, kind+"-added", fmt.Sprintf("%s %q was added", kind, name), "confirm it is intentional and referenced by a pipeline"))
		}
	}
	return changes
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
