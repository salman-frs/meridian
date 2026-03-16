package report

import (
	"fmt"
	"strings"
	"time"

	"github.com/salman-frs/meridian/internal/model"
)

func RenderSummaryMarkdown(result model.RunResult) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Meridian: %s\n\n", emojiStatus(result.Status)))
	b.WriteString(fmt.Sprintf("**Config:** `%s`  \n", result.ConfigPath))
	b.WriteString(fmt.Sprintf("**Engine:** `%s`  \n", result.Engine))
	b.WriteString(fmt.Sprintf("**Runtime backend:** `%s`  \n", valueOrDefault(result.RuntimeBackend, "n/a")))
	b.WriteString(fmt.Sprintf("**Mode:** `%s`  \n", result.Mode))
	b.WriteString(fmt.Sprintf("**Collector image:** `%s`  \n", result.CollectorImage))
	b.WriteString(fmt.Sprintf("**Run:** `%s`\n\n", result.StartedAt.Format(time.RFC3339)))
	b.WriteString("### Summary\n")
	b.WriteString(fmt.Sprintf("- Validate: %s\n", findingsStatus(result.Findings)))
	b.WriteString(fmt.Sprintf("- Semantic: %s\n", semanticStatus(result.Semantic)))
	b.WriteString(fmt.Sprintf("- Graph: %s\n", graphStatus(result)))
	if len(result.Assertions) == 0 {
		b.WriteString("- Runtime tests: not executed\n")
	} else {
		for _, line := range assertionSummaryLines(result.Assertions) {
			b.WriteString(fmt.Sprintf("- %s\n", line))
		}
	}
	if len(result.Diff.Changes) > 0 {
		b.WriteString("\n### What changed (risk highlights)\n")
		if result.Diff.ComparedEffective {
			b.WriteString("- Effective config diff was available and used for classification.\n")
		}
		for _, change := range topDiffChanges(result.Diff.Changes, 5) {
			b.WriteString(fmt.Sprintf("- **%s:** %s\n", strings.ToUpper(string(change.Severity)), change.Message))
		}
	}
	if result.Semantic.Enabled {
		b.WriteString("\n### Semantic validation\n")
		for _, stage := range result.Semantic.Stages {
			line := fmt.Sprintf("- %s: %s", stage.Name, stage.Status)
			if stage.Message != "" {
				line += " (" + stage.Message + ")"
			}
			b.WriteString(line + "\n")
		}
	}
	if failure := topFailure(result); failure != "" {
		b.WriteString("\n### Top failure\n")
		b.WriteString(failure)
	}
	b.WriteString("\n### Artifacts\n")
	b.WriteString("Download the `meridian-artifacts` workflow artifact for the full bundle.\n")
	for _, item := range ciArtifactNames(result) {
		b.WriteString(fmt.Sprintf("- `%s`\n", item))
	}
	return b.String()
}

func RenderPRComment(result model.RunResult) string {
	var b strings.Builder
	b.WriteString("<!-- meridian-comment -->\n")
	b.WriteString(RenderSummaryMarkdown(result))
	return b.String()
}

func emojiStatus(status string) string {
	if status == "PASS" {
		return "PASS ✅"
	}
	return "FAIL ❌"
}

func valueOrDefault(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func findingsStatus(findings []model.Finding) string {
	failCount := 0
	warnCount := 0
	for _, finding := range findings {
		switch finding.Severity {
		case model.SeverityFail:
			failCount++
		case model.SeverityWarn:
			warnCount++
		}
	}
	if failCount == 0 && warnCount == 0 {
		return "PASS"
	}
	if failCount > 0 {
		return fmt.Sprintf("FAIL (%d fail, %d warn)", failCount, warnCount)
	}
	return fmt.Sprintf("WARN (%d warn)", warnCount)
}

func graphStatus(result model.RunResult) string {
	if len(result.Graph.Nodes) == 0 {
		return "FAIL"
	}
	return "PASS"
}

func assertionSummaryLines(assertions []model.AssertionResult) []string {
	bySignal := map[model.SignalType]string{}
	for _, signal := range []model.SignalType{model.SignalTraces, model.SignalMetrics, model.SignalLogs} {
		status := "not-run"
		observed := "0"
		for _, assertion := range assertions {
			if assertion.Signal != signal || !strings.HasSuffix(assertion.ID, "-received") {
				continue
			}
			status = assertion.Status
			observed = assertion.Observed
			break
		}
		if status == "not-run" {
			continue
		}
		bySignal[signal] = fmt.Sprintf("%s: %s (received %s)", signal, status, observed)
	}
	lines := []string{}
	for _, signal := range []model.SignalType{model.SignalTraces, model.SignalMetrics, model.SignalLogs} {
		if line, ok := bySignal[signal]; ok {
			lines = append(lines, line)
		}
	}
	return lines
}

func topDiffChanges(changes []model.DiffChange, limit int) []model.DiffChange {
	if len(changes) <= limit {
		return changes
	}
	return changes[:limit]
}

func topFailure(result model.RunResult) string {
	for _, assertion := range result.Assertions {
		if assertion.Status != "FAIL" {
			continue
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("- **%s**: %s\n", assertion.ID, assertion.Message))
		for _, cause := range assertion.LikelyCauses {
			b.WriteString(fmt.Sprintf("- Likely cause: %s\n", cause))
		}
		for _, step := range assertion.NextSteps {
			b.WriteString(fmt.Sprintf("- Next step: %s\n", step))
		}
		return b.String()
	}
	if result.Message == "" {
		return ""
	}
	return "- " + result.Message + "\n"
}

func ciArtifactNames(result model.RunResult) []string {
	names := []string{
		"report.json",
		"summary.md",
		"graph.mmd",
		"collector.log",
		"config.patched.yaml",
	}
	if result.Artifacts.GraphSVG != "" {
		names = append(names, "graph.svg")
	}
	if result.Semantic.Enabled {
		if result.Artifacts.ComponentsJSON != "" {
			names = append(names, "collector-components.json")
		}
		if len(result.Semantic.Findings) > 0 {
			names = append(names, "semantic-findings.json")
		}
		if strings.TrimSpace(result.Semantic.FinalConfig) != "" {
			names = append(names, "config.final.yaml")
		}
	}
	if len(result.Diff.Changes) > 0 {
		names = append(names, "diff.md")
	}
	names = append(names, "captures/")
	return names
}

func semanticStatus(report model.SemanticReport) string {
	if !report.Enabled {
		if report.SkippedReason == "" {
			return "SKIP"
		}
		return "SKIP (" + report.SkippedReason + ")"
	}
	if report.Status == "" {
		return "PASS"
	}
	return report.Status
}
