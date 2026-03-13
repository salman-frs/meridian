package validate

import (
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"

	"github.com/salman-frs/meridian/internal/model"
)

func Run(cfg model.ConfigModel) []model.Finding {
	findings := []model.Finding{}
	sourcePath := cfg.PrimarySourcePath()
	if len(cfg.Pipelines) == 0 {
		findings = append(findings, model.Finding{
			Severity:    model.SeverityFail,
			Code:        "missing-pipelines",
			Message:     "config does not declare any service pipelines",
			Location:    model.SourceLocation{Path: sourcePath, Key: "service.pipelines"},
			Remediation: "add service.pipelines entries",
			NextStep:    "define at least one traces, metrics, or logs pipeline",
		})
	}
	if len(cfg.MissingEnvNames) > 0 {
		for _, ref := range cfg.EnvReferences {
			if ref.HasValue {
				continue
			}
			findings = append(findings, model.Finding{
				Severity:    model.SeverityFail,
				Code:        "missing-env",
				Message:     fmt.Sprintf("config references env var %q but no value was provided", ref.Name),
				Location:    model.SourceLocation{Path: ref.SourcePath, Key: ref.SourceKey},
				Remediation: "provide the variable through --env-file, --env, or the shell environment",
				NextStep:    fmt.Sprintf("re-run with --env %s=value or add %s to your env file", ref.Name, ref.Name),
			})
		}
	}

	findings = append(findings, validatePipelineReferences(cfg)...)
	findings = append(findings, validateUnusedComponents(cfg)...)
	findings = append(findings, validateEndpoints(cfg)...)
	return findings
}

func validatePipelineReferences(cfg model.ConfigModel) []model.Finding {
	findings := []model.Finding{}
	sourcePath := cfg.PrimarySourcePath()
	for name, pipeline := range cfg.Pipelines {
		location := model.SourceLocation{Path: sourcePath, Key: "service.pipelines." + name}
		if len(pipeline.Receivers) == 0 {
			findings = append(findings, model.Finding{
				Severity:    model.SeverityFail,
				Code:        "missing-receivers",
				Message:     fmt.Sprintf("pipeline %q has no receivers", name),
				Location:    location,
				Remediation: "add at least one receiver or remove the empty pipeline",
				NextStep:    "update the pipeline receiver list in the collector config",
			})
		}
		if len(pipeline.Exporters) == 0 {
			findings = append(findings, model.Finding{
				Severity:    model.SeverityFail,
				Code:        "missing-exporters",
				Message:     fmt.Sprintf("pipeline %q has no exporters", name),
				Location:    location,
				Remediation: "add at least one exporter or remove the empty pipeline",
				NextStep:    "update the pipeline exporter list in the collector config",
			})
		}
		checkRefs := func(kind string, refs []string, inventory map[string]model.Component) {
			for _, ref := range refs {
				if _, ok := inventory[ref]; ok {
					continue
				}
				altInventory := cfg.Connectors
				if kind == "exporter" {
					if _, ok := altInventory[ref]; ok {
						continue
					}
				}
				if kind == "receiver" {
					if _, ok := cfg.Connectors[ref]; ok {
						continue
					}
				}
				findings = append(findings, model.Finding{
					Severity:    model.SeverityFail,
					Code:        "undefined-" + kind,
					Message:     fmt.Sprintf("pipeline %q references undefined %s %q", name, kind, ref),
					Location:    location,
					Remediation: "declare the component or remove the broken reference",
					NextStep:    fmt.Sprintf("fix the %s list for pipeline %s", kind, name),
				})
			}
		}
		checkRefs("receiver", pipeline.Receivers, cfg.Receivers)
		checkRefs("processor", pipeline.Processors, cfg.Processors)
		checkRefs("exporter", pipeline.Exporters, cfg.Exporters)
	}
	return findings
}

func validateUnusedComponents(cfg model.ConfigModel) []model.Finding {
	sourcePath := cfg.PrimarySourcePath()
	used := map[string]map[string]struct{}{
		"receiver":  {},
		"processor": {},
		"exporter":  {},
		"connector": {},
	}
	for _, pipeline := range cfg.Pipelines {
		for _, name := range pipeline.Receivers {
			used["receiver"][name] = struct{}{}
			used["connector"][name] = struct{}{}
		}
		for _, name := range pipeline.Processors {
			used["processor"][name] = struct{}{}
		}
		for _, name := range pipeline.Exporters {
			used["exporter"][name] = struct{}{}
			used["connector"][name] = struct{}{}
		}
	}

	findings := []model.Finding{}
	appendUnused := func(kind string, inventory map[string]model.Component) {
		names := make([]string, 0, len(inventory))
		for name := range inventory {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if _, ok := used[kind][name]; ok {
				continue
			}
			findings = append(findings, model.Finding{
				Severity:    model.SeverityWarn,
				Code:        "unused-" + kind,
				Message:     fmt.Sprintf("%s %q is declared but not referenced by any pipeline", kind, name),
				Location:    model.SourceLocation{Path: sourcePath, Key: kind + "s." + name},
				Remediation: "remove the unused component or wire it into a pipeline",
				NextStep:    "review whether this component is still needed",
			})
		}
	}
	appendUnused("receiver", cfg.Receivers)
	appendUnused("processor", cfg.Processors)
	appendUnused("exporter", cfg.Exporters)
	appendUnused("connector", cfg.Connectors)
	return findings
}

func validateEndpoints(cfg model.ConfigModel) []model.Finding {
	findings := []model.Finding{}
	sourcePath := cfg.PrimarySourcePath()
	validateComponent := func(section string, components map[string]model.Component) {
		for name, component := range components {
			endpoint, ok := component.Config["endpoint"].(string)
			if !ok || endpoint == "" {
				continue
			}
			if strings.Contains(endpoint, "://") {
				if _, err := url.Parse(endpoint); err != nil {
					findings = append(findings, endpointFinding(sourcePath, section, name, endpoint))
				}
				continue
			}
			if _, _, err := net.SplitHostPort(endpoint); err != nil {
				findings = append(findings, endpointFinding(sourcePath, section, name, endpoint))
			}
		}
	}
	validateComponent("receiver", cfg.Receivers)
	validateComponent("exporter", cfg.Exporters)
	return findings
}

func endpointFinding(path, section, name, endpoint string) model.Finding {
	return model.Finding{
		Severity:    model.SeverityWarn,
		Code:        "invalid-endpoint",
		Message:     fmt.Sprintf("%s %q has an endpoint that Meridian could not parse: %s", section, name, endpoint),
		Location:    model.SourceLocation{Path: path, Key: section + "s." + name + ".endpoint"},
		Remediation: "use host:port or a valid URL format",
		NextStep:    "correct the endpoint syntax and rerun validate",
	}
}
