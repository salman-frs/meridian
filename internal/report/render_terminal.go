package report

import (
	"fmt"
	"strings"

	"github.com/salman-frs/meridian/internal/model"
)

func RenderTerminal(result model.RunResult) string {
	lines := []string{
		fmt.Sprintf("RESULT: %s", result.Status),
		fmt.Sprintf("Run ID: %s", result.RunID),
		fmt.Sprintf("Config: %s", result.ConfigPath),
		fmt.Sprintf("Engine: %s", result.Engine),
		fmt.Sprintf("Runtime backend: %s", valueOrDefault(result.RuntimeBackend, "n/a")),
		fmt.Sprintf("Mode: %s", result.Mode),
		fmt.Sprintf("Collector image: %s", result.CollectorImage),
		fmt.Sprintf("Runtime config: %s", runtimeConfigSource(result)),
		fmt.Sprintf("Semantic validation: %s", semanticStatus(result.Semantic)),
		fmt.Sprintf("Contracts: %s", contractStatus(result.Contracts)),
		fmt.Sprintf("Artifacts: %s", result.Artifacts.RunDir),
	}
	if result.Message != "" {
		lines = append(lines, "", "What happened:", "- "+result.Message)
	}
	if len(result.Diff.Changes) > 0 {
		lines = append(lines, "", "What changed (risk highlights):")
		for _, change := range result.Diff.Changes {
			lines = append(lines, fmt.Sprintf("- [%s] %s", strings.ToUpper(string(change.Severity)), change.Message))
		}
	}
	if len(result.Findings) > 0 {
		lines = append(lines, "", "Validation:")
		for _, finding := range result.Findings {
			lines = append(lines, "- "+model.FormatFinding(finding))
		}
	}
	if result.Semantic.Enabled {
		lines = append(lines, "", "Semantic validation:")
		for _, stage := range result.Semantic.Stages {
			line := fmt.Sprintf("- %s: %s", stage.Name, stage.Status)
			if stage.Message != "" {
				line += " (" + stage.Message + ")"
			}
			lines = append(lines, line)
		}
	}
	if len(result.Contracts) > 0 {
		lines = append(lines, "", "Contract checks:")
		for _, contract := range result.Contracts {
			lines = append(lines, fmt.Sprintf("- %s: %s (%s observed %s expected %s)", contract.ID, contract.Status, contract.Message, contract.Observed, contract.Expected))
			if contract.Status == "FAIL" {
				lines = append(lines, "", "Contract diff:")
				for _, item := range contract.Diff {
					lines = append(lines, "- "+item)
				}
				lines = append(lines, "", "Likely causes:")
				for _, cause := range contract.LikelyCauses {
					lines = append(lines, "- "+cause)
				}
				lines = append(lines, "", "Next steps:")
				for _, step := range contract.NextSteps {
					lines = append(lines, "- "+step)
				}
				break
			}
		}
	}
	if len(result.Assertions) > 0 {
		lines = append(lines, "", "Runtime assertions:")
		for _, assertion := range result.Assertions {
			lines = append(lines, fmt.Sprintf("- %s: %s (%s observed %s expected %s)", assertion.ID, assertion.Status, assertion.Message, assertion.Observed, assertion.Expected))
			if assertion.Status == "FAIL" {
				lines = append(lines, "", "Likely causes:")
				for _, cause := range assertion.LikelyCauses {
					lines = append(lines, "- "+cause)
				}
				lines = append(lines, "", "Next steps:")
				for _, step := range assertion.NextSteps {
					lines = append(lines, "- "+step)
				}
				break
			}
		}
	}
	lines = append(lines, "", "Artifacts:")
	lines = append(lines, artifactPaths(result)...)
	return strings.Join(lines, "\n")
}

func artifactPaths(result model.RunResult) []string {
	paths := []string{
		"- " + result.Artifacts.ReportJSON,
		"- " + result.Artifacts.SummaryMD,
		"- " + result.Artifacts.GraphMMD,
	}
	if result.Artifacts.GraphSVG != "" {
		paths = append(paths, "- "+result.Artifacts.GraphSVG)
	}
	if result.Semantic.Enabled {
		if result.Artifacts.ComponentsJSON != "" {
			paths = append(paths, "- "+result.Artifacts.ComponentsJSON)
		}
		if strings.TrimSpace(result.Semantic.FinalConfig) != "" {
			paths = append(paths, "- "+result.Artifacts.FinalConfig)
		}
		if len(result.Semantic.Findings) > 0 {
			paths = append(paths, "- "+result.Artifacts.SemanticJSON)
		}
	}
	paths = append(paths,
		"- "+result.Artifacts.CollectorLog,
		"- "+result.Artifacts.PatchedConfig,
		"- "+result.Artifacts.ContractsJSON,
		"- "+result.Artifacts.ContractsMD,
		"- "+result.Artifacts.CaptureNormalizedJSON,
	)
	if len(result.Diff.Changes) > 0 {
		paths = append(paths, "- "+result.Artifacts.DiffMD)
	}
	return paths
}
