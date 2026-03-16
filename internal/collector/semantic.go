package collector

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/salman-frs/meridian/internal/configio"
	"github.com/salman-frs/meridian/internal/model"
	"github.com/salman-frs/meridian/internal/runtime"
	"gopkg.in/yaml.v3"
)

type Options struct {
	ConfigSources   []string
	ConfigModel     model.ConfigModel
	Env             map[string]string
	CollectorBinary string
	CollectorImage  string
	Engine          model.RuntimeEngine
	ResolvedEngine  runtime.ResolvedEngine
	RequireSemantic bool
}

type executionTarget struct {
	source         string
	target         string
	commandLabel   string
	runBinary      func(args ...string) ([]byte, error)
	runWithSources func(subcommand string, sources []string) ([]byte, error)
}

func Analyze(opts Options) (model.SemanticReport, error) {
	target, err := resolveExecutionTarget(opts)
	if err != nil {
		if opts.RequireSemantic {
			return model.SemanticReport{}, err
		}
		return skipped("unable to resolve collector semantic validation target", err.Error()), nil
	}

	report := model.SemanticReport{
		Enabled: true,
		Source:  target.source,
		Target:  target.target,
	}

	componentsOutput, err := target.runBinary("components")
	switch {
	case err == nil:
		report.Components = parseComponents(string(componentsOutput))
		report.RawComponents = strings.TrimSpace(string(componentsOutput))
		report.Stages = append(report.Stages, model.SemanticStage{Name: "components", Status: "PASS"})
		report.Findings = append(report.Findings, inventoryFindings(opts.ConfigModel, report.Components)...)
	case isUnsupportedCommand(err):
		report.Stages = append(report.Stages, model.SemanticStage{Name: "components", Status: "SKIP", Message: trimOutput(err.Error())})
		report.Findings = append(report.Findings, model.Finding{
			Severity:    model.SeverityWarn,
			Code:        "collector-components-skipped",
			Message:     "selected collector does not support the components command",
			Remediation: "use a newer collector build or provide --collector-binary for a distribution that supports inventory listing",
			NextStep:    "rerun semantic validation with a collector binary that supports components",
		})
	default:
		report.Stages = append(report.Stages, model.SemanticStage{Name: "components", Status: "FAIL", Message: trimOutput(err.Error())})
		report.Findings = append(report.Findings, model.Finding{
			Severity:    model.SeverityWarn,
			Code:        "collector-components-failed",
			Message:     trimOutput(err.Error()),
			Remediation: "verify the collector binary or image can execute the components command",
			NextStep:    "rerun with a reachable collector binary or a working container engine",
		})
	}

	validateOutput, err := target.runWithSources("validate", opts.ConfigSources)
	if err != nil {
		report.Status = "FAIL"
		report.Stages = append(report.Stages, model.SemanticStage{Name: "validate", Status: "FAIL", Message: trimOutput(err.Error())})
		report.Findings = append(report.Findings, model.Finding{
			Severity:    model.SeverityFail,
			Code:        "collector-validate-failed",
			Message:     trimOutput(err.Error()),
			Remediation: "fix the collector-native validation errors reported by the selected distribution",
			NextStep:    "run the collector validate command directly or inspect the semantic validation output in report.json",
		})
		return finalize(report), nil
	}
	validateMessage := strings.TrimSpace(string(validateOutput))
	if validateMessage == "" {
		validateMessage = "collector-native validation passed"
	}
	report.Stages = append(report.Stages, model.SemanticStage{Name: "validate", Status: "PASS", Message: validateMessage})

	finalConfig, err := target.runWithSources("print-config", opts.ConfigSources)
	switch {
	case err == nil:
		report.FinalConfig = string(finalConfig)
		report.Stages = append(report.Stages, model.SemanticStage{Name: "print-config", Status: "PASS"})
	case isUnsupportedCommand(err):
		report.Stages = append(report.Stages, model.SemanticStage{Name: "print-config", Status: "SKIP", Message: trimOutput(err.Error())})
		report.Findings = append(report.Findings, model.Finding{
			Severity:    model.SeverityInfo,
			Code:        "collector-print-config-skipped",
			Message:     "selected collector does not support print-config; semantic diff falls back to source config",
			Remediation: "use a collector build with print-config support if you want effective-config evidence",
			NextStep:    "rerun with a collector distribution that supports print-config",
		})
	default:
		report.Stages = append(report.Stages, model.SemanticStage{Name: "print-config", Status: "FAIL", Message: trimOutput(err.Error())})
		report.Findings = append(report.Findings, model.Finding{
			Severity:    model.SeverityWarn,
			Code:        "collector-print-config-failed",
			Message:     trimOutput(err.Error()),
			Remediation: "verify the collector can render the effective config for the selected sources",
			NextStep:    "inspect the collector command output and rerun semantic validation",
		})
	}

	return finalize(report), nil
}

func ResolveFinalConfig(opts Options) (string, bool, error) {
	target, err := resolveExecutionTarget(opts)
	if err != nil {
		return "", false, nil
	}
	finalConfig, err := target.runWithSources("print-config", opts.ConfigSources)
	if err != nil {
		if isUnsupportedCommand(err) {
			return "", false, nil
		}
		return "", false, err
	}
	return string(finalConfig), true, nil
}

func MaterializeConfig(opts Options) (model.ConfigModel, string, bool, error) {
	finalConfig, ok, err := ResolveFinalConfig(opts)
	if err != nil || !ok {
		return model.ConfigModel{}, "", ok, err
	}
	cfg, err := LoadEffectiveConfig(finalConfig, primarySource(opts.ConfigSources)+"#print-config")
	if err != nil {
		return model.ConfigModel{}, "", true, err
	}
	cfg.SourcePaths = append([]string{}, opts.ConfigSources...)
	return cfg, finalConfig, true, nil
}

func skipped(reason string, detail string) model.SemanticReport {
	report := model.SemanticReport{
		Enabled:       false,
		Status:        "SKIP",
		SkippedReason: reason,
	}
	if detail != "" {
		report.Stages = append(report.Stages, model.SemanticStage{Name: "semantic", Status: "SKIP", Message: detail})
	}
	return report
}

func finalize(report model.SemanticReport) model.SemanticReport {
	if report.Status == "" {
		report.Status = "PASS"
	}
	for _, finding := range report.Findings {
		switch finding.Severity {
		case model.SeverityFail:
			report.Status = "FAIL"
			return report
		case model.SeverityWarn:
			if report.Status == "PASS" {
				report.Status = "WARN"
			}
		case model.SeverityInfo:
			if report.Status == "PASS" {
				report.Status = "INFO"
			}
		}
	}
	return report
}

func resolveExecutionTarget(opts Options) (executionTarget, error) {
	if len(opts.ConfigSources) == 0 {
		return executionTarget{}, errors.New("no config sources were provided")
	}
	if opts.CollectorBinary != "" {
		return binaryTarget(opts.CollectorBinary, opts.Env), nil
	}
	if opts.CollectorImage == "" {
		return executionTarget{}, errors.New("no collector target available")
	}
	resolved := opts.ResolvedEngine
	if resolved.Engine() == "" {
		var err error
		resolved, err = runtime.Resolve(opts.Engine)
		if err != nil {
			return executionTarget{}, err
		}
	}
	return imageTarget(resolved, opts.CollectorImage, opts.ConfigSources, opts.Env)
}

func binaryTarget(path string, env map[string]string) executionTarget {
	label := path
	return executionTarget{
		source:       "binary",
		target:       path,
		commandLabel: label,
		runBinary: func(args ...string) ([]byte, error) {
			return runCommand(commandWithEnv(exec.Command(path, args...), env))
		},
		runWithSources: func(subcommand string, sources []string) ([]byte, error) {
			args := []string{subcommand}
			for _, source := range sources {
				args = append(args, "--config", source)
			}
			return runCommand(commandWithEnv(exec.Command(path, args...), env))
		},
	}
}

func imageTarget(engine runtime.ResolvedEngine, image string, sources []string, env map[string]string) (executionTarget, error) {
	mountArgs, mappedSources, err := mapConfigSourcesForContainer(sources)
	if err != nil {
		return executionTarget{}, err
	}
	envArgs := containerEnvArgs(env)
	networkArgs := semanticContainerRunArgs(engine.Engine(), engine.RuntimeBackend())
	label := engine.CommandLabel()
	if label == "" {
		return executionTarget{}, errors.New("no container engine is available")
	}
	return executionTarget{
		source:       "image",
		target:       image,
		commandLabel: label,
		runBinary: func(args ...string) ([]byte, error) {
			runArgs := append([]string{"run", "--rm"}, networkArgs...)
			runArgs = append(runArgs, mountArgs...)
			runArgs = append(runArgs, envArgs...)
			runArgs = append(runArgs, image)
			runArgs = append(runArgs, args...)
			cmd := engine.Command(runArgs...)
			return runCommand(cmd)
		},
		runWithSources: func(subcommand string, _ []string) ([]byte, error) {
			runArgs := append([]string{"run", "--rm"}, networkArgs...)
			runArgs = append(runArgs, mountArgs...)
			runArgs = append(runArgs, envArgs...)
			runArgs = append(runArgs, image, subcommand)
			for _, source := range mappedSources {
				runArgs = append(runArgs, "--config", source)
			}
			cmd := engine.Command(runArgs...)
			return runCommand(cmd)
		},
	}, nil
}

func mapConfigSourcesForContainer(sources []string) ([]string, []string, error) {
	mountArgs := []string{}
	mappedSources := make([]string, 0, len(sources))
	seenMounts := map[string]struct{}{}
	for idx, source := range sources {
		hostPath, ok := localSourcePath(source)
		if !ok {
			mappedSources = append(mappedSources, source)
			continue
		}
		absPath, err := filepath.Abs(hostPath)
		if err != nil {
			return nil, nil, err
		}
		containerPath := fmt.Sprintf("/meridian/config/%d-%s", idx, filepath.Base(absPath))
		mount := absPath + ":" + containerPath + ":ro"
		if _, ok := seenMounts[mount]; !ok {
			mountArgs = append(mountArgs, "-v", mount)
			seenMounts[mount] = struct{}{}
		}
		if strings.HasPrefix(source, "file:") {
			mappedSources = append(mappedSources, "file:"+containerPath)
			continue
		}
		mappedSources = append(mappedSources, containerPath)
	}
	return mountArgs, mappedSources, nil
}

func localSourcePath(source string) (string, bool) {
	if !strings.Contains(source, ":") || strings.HasPrefix(source, "/") || strings.HasPrefix(source, ".") {
		return source, true
	}
	if strings.HasPrefix(source, "file:") {
		return strings.TrimPrefix(source, "file:"), true
	}
	return "", false
}

func runCommand(cmd *exec.Cmd) ([]byte, error) {
	if cmd == nil {
		return nil, errors.New("collector command is unavailable")
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	output := strings.TrimSpace(stdout.String())
	errOutput := strings.TrimSpace(stderr.String())
	if err == nil {
		if output == "" {
			return []byte(errOutput), nil
		}
		return []byte(output), nil
	}
	if output != "" && errOutput != "" {
		return nil, fmt.Errorf("%s\n%s", output, errOutput)
	}
	if output != "" {
		return nil, errors.New(output)
	}
	if errOutput != "" {
		return nil, errors.New(errOutput)
	}
	return nil, err
}

func commandWithEnv(cmd *exec.Cmd, env map[string]string) *exec.Cmd {
	if cmd == nil {
		return nil
	}
	if len(env) == 0 {
		return cmd
	}
	cmd.Env = append(os.Environ(), envPairs(env)...)
	return cmd
}

func containerEnvArgs(env map[string]string) []string {
	pairs := envPairs(env)
	args := make([]string, 0, len(pairs)*2)
	for _, pair := range pairs {
		args = append(args, "-e", pair)
	}
	return args
}

func envPairs(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, key+"="+env[key])
	}
	return pairs
}

func semanticContainerRunArgs(engine model.RuntimeEngine, backend string) []string {
	if engine == model.RuntimeEngineContainerd && backend == "nerdctl" {
		return []string{"--network", "host"}
	}
	return nil
}

func primarySource(sources []string) string {
	if len(sources) == 0 {
		return "config"
	}
	return sources[0]
}

func parseComponents(output string) []model.CollectorComponent {
	if components := parseStructuredComponents(output); len(components) > 0 {
		return components
	}

	kind := ""
	components := []model.CollectorComponent{}
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasSuffix(trimmed, ":") && !strings.Contains(trimmed, " ") {
			kind = normalizeComponentKind(strings.TrimSuffix(strings.ToLower(trimmed), ":"))
			continue
		}
		if kind == "" {
			continue
		}
		name := trimmed
		detail := ""
		fields := strings.Fields(trimmed)
		if len(fields) > 1 {
			name = fields[0]
			detail = strings.TrimSpace(strings.TrimPrefix(trimmed, name))
		}
		components = append(components, model.CollectorComponent{
			Kind:      kind,
			Name:      strings.Trim(name, "-"),
			Stability: detectStability(detail),
			Raw:       trimmed,
		})
	}
	return components
}

func parseStructuredComponents(output string) []model.CollectorComponent {
	var raw map[string]any
	if err := yaml.Unmarshal([]byte(output), &raw); err != nil {
		return nil
	}
	kinds := []struct {
		section string
		kind    string
	}{
		{section: "receivers", kind: "receiver"},
		{section: "processors", kind: "processor"},
		{section: "exporters", kind: "exporter"},
		{section: "connectors", kind: "connector"},
		{section: "extensions", kind: "extension"},
	}
	components := []model.CollectorComponent{}
	for _, kind := range kinds {
		items, _ := raw[kind.section].([]any)
		for _, item := range items {
			componentMap, _ := item.(map[string]any)
			name := componentValue(componentMap["name"])
			if name == "" {
				continue
			}
			components = append(components, model.CollectorComponent{
				Kind:      kind.kind,
				Name:      name,
				Stability: componentStability(componentMap["stability"]),
				Raw:       strings.TrimSpace(componentSummary(componentMap)),
			})
		}
	}
	return components
}

func componentValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func componentSummary(component map[string]any) string {
	name := componentValue(component["name"])
	module := componentValue(component["module"])
	if module == "" {
		return name
	}
	return fmt.Sprintf("%s (%s)", name, module)
}

func componentStability(value any) string {
	switch typed := value.(type) {
	case string:
		return detectStability(typed)
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		best := ""
		for _, key := range keys {
			stability := detectStability(componentValue(typed[key]))
			if stability == "" {
				continue
			}
			if best == "" || stabilityRank(stability) > stabilityRank(best) {
				best = stability
			}
		}
		return best
	default:
		return ""
	}
}

func stabilityRank(value string) int {
	switch value {
	case "stable":
		return 5
	case "beta":
		return 4
	case "alpha":
		return 3
	case "development":
		return 2
	case "unmaintained":
		return 1
	default:
		return 0
	}
}

func detectStability(detail string) string {
	lower := strings.ToLower(detail)
	switch {
	case strings.Contains(lower, "stable"):
		return "stable"
	case strings.Contains(lower, "beta"):
		return "beta"
	case strings.Contains(lower, "alpha"):
		return "alpha"
	case strings.Contains(lower, "development"):
		return "development"
	case strings.Contains(lower, "unmaintained"):
		return "unmaintained"
	default:
		return ""
	}
}

func normalizeComponentKind(kind string) string {
	switch kind {
	case "receivers":
		return "receiver"
	case "processors":
		return "processor"
	case "exporters":
		return "exporter"
	case "connectors":
		return "connector"
	case "extensions":
		return "extension"
	default:
		return kind
	}
}

func inventoryFindings(cfg model.ConfigModel, components []model.CollectorComponent) []model.Finding {
	if len(components) == 0 {
		return nil
	}
	index := map[string]map[string]model.CollectorComponent{}
	for _, component := range components {
		if index[component.Kind] == nil {
			index[component.Kind] = map[string]model.CollectorComponent{}
		}
		index[component.Kind][component.Name] = component
	}
	findings := []model.Finding{}
	appendFindings := func(kind string, items map[string]model.Component) {
		for name := range items {
			available := index[kind]
			component, ok := available[name]
			if !ok {
				findings = append(findings, model.Finding{
					Severity:    model.SeverityFail,
					Code:        "collector-component-unsupported",
					Message:     fmt.Sprintf("%s %q is not reported by the selected collector distribution", kind, name),
					Location:    model.SourceLocation{Path: cfg.PrimarySourcePath(), Key: kind + "s." + name},
					Remediation: "use a collector distribution that includes the component or change the config to supported components",
					NextStep:    "rerun with a matching collector binary or image for this config",
				})
				continue
			}
			if slices.Contains([]string{"alpha", "development"}, component.Stability) {
				findings = append(findings, model.Finding{
					Severity:    model.SeverityWarn,
					Code:        "collector-component-unstable",
					Message:     fmt.Sprintf("%s %q is reported as %s by the selected collector distribution", kind, name, component.Stability),
					Location:    model.SourceLocation{Path: cfg.PrimarySourcePath(), Key: kind + "s." + name},
					Remediation: "confirm the selected component stability is acceptable for this environment",
					NextStep:    "review the collector components output and consider a more stable distribution if needed",
				})
			}
		}
	}
	appendFindings("receiver", cfg.Receivers)
	appendFindings("processor", cfg.Processors)
	appendFindings("exporter", cfg.Exporters)
	appendFindings("connector", cfg.Connectors)
	appendFindings("extension", cfg.Extensions)
	return findings
}

func isUnsupportedCommand(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "unknown command") || strings.Contains(text, "unknown shorthand flag") || strings.Contains(text, "not a meridian") || strings.Contains(text, "no help topic")
}

func trimOutput(text string) string {
	return strings.TrimSpace(text)
}

func LoadEffectiveConfig(text string, source string) (model.ConfigModel, error) {
	return configio.LoadConfigText(source, text)
}
