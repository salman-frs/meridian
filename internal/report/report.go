package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/salman-frs/meridian/internal/model"
)

func WriteBundle(result model.RunResult) error {
	if err := result.Artifacts.Ensure(); err != nil {
		return err
	}
	if err := model.WriteJSON(result.Artifacts.ReportJSON, result); err != nil {
		return err
	}
	if err := model.WriteText(result.Artifacts.SummaryMD, RenderSummaryMarkdown(result)); err != nil {
		return err
	}
	if len(result.Diff.Changes) > 0 {
		if err := model.WriteText(result.Artifacts.DiffMD, RenderDiff(result.Diff)); err != nil {
			return err
		}
	}
	latest := filepath.Join(filepath.Dir(result.Artifacts.RunDir), "latest")
	_ = os.Remove(latest)
	_ = os.Symlink(result.Artifacts.RunDir, latest)
	return nil
}

func RenderTerminal(result model.RunResult) string {
	lines := []string{
		fmt.Sprintf("RESULT: %s", result.Status),
		fmt.Sprintf("Run ID: %s", result.RunID),
		fmt.Sprintf("Config: %s", result.ConfigPath),
		fmt.Sprintf("Mode: %s", result.Mode),
		fmt.Sprintf("Collector image: %s", result.CollectorImage),
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
	lines = append(lines, "- "+result.Artifacts.ReportJSON)
	lines = append(lines, "- "+result.Artifacts.SummaryMD)
	lines = append(lines, "- "+result.Artifacts.GraphMMD)
	if result.Artifacts.GraphSVG != "" {
		lines = append(lines, "- "+result.Artifacts.GraphSVG)
	}
	lines = append(lines, "- "+result.Artifacts.CollectorLog)
	lines = append(lines, "- "+result.Artifacts.PatchedConfig)
	if len(result.Diff.Changes) > 0 {
		lines = append(lines, "- "+result.Artifacts.DiffMD)
	}
	return strings.Join(lines, "\n")
}

func RenderSummaryMarkdown(result model.RunResult) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Meridian: %s\n\n", emojiStatus(result.Status)))
	b.WriteString(fmt.Sprintf("**Config:** `%s`  \n", result.ConfigPath))
	b.WriteString(fmt.Sprintf("**Mode:** `%s`  \n", result.Mode))
	b.WriteString(fmt.Sprintf("**Collector image:** `%s`  \n", result.CollectorImage))
	b.WriteString(fmt.Sprintf("**Run:** `%s`\n\n", result.StartedAt.Format(time.RFC3339)))
	b.WriteString("### Summary\n")
	b.WriteString(fmt.Sprintf("- Validate: %s\n", findingsStatus(result.Findings)))
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
		for _, change := range topDiffChanges(result.Diff.Changes, 5) {
			b.WriteString(fmt.Sprintf("- **%s:** %s\n", strings.ToUpper(string(change.Severity)), change.Message))
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

func RenderDiff(diff model.DiffResult) string {
	if len(diff.Changes) == 0 {
		return "No diff findings."
	}
	lines := []string{"Diff findings:"}
	for _, change := range diff.Changes {
		lines = append(lines, fmt.Sprintf("- [%s] %s", strings.ToUpper(string(change.Severity)), change.Message))
		if change.ReviewHint != "" {
			lines = append(lines, "  hint: "+change.ReviewHint)
		}
	}
	return strings.Join(lines, "\n")
}

func emojiStatus(status string) string {
	if status == "PASS" {
		return "PASS ✅"
	}
	return "FAIL ❌"
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

func WriteAnnotations(result model.RunResult) {
	count := 0
	for _, finding := range result.Findings {
		if finding.Severity != model.SeverityFail || count >= 3 {
			continue
		}
		fmt.Fprintf(os.Stdout, "::error title=%s::%s\n", finding.Code, finding.Message)
		count++
	}
	for _, assertion := range result.Assertions {
		if assertion.Status != "FAIL" || count >= 3 {
			continue
		}
		fmt.Fprintf(os.Stdout, "::error title=%s::%s (observed %s expected %s)\n", assertion.ID, assertion.Message, assertion.Observed, assertion.Expected)
		count++
	}
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
		b.WriteString(fmt.Sprintf("**Assertion:** `%s`  \n", assertion.ID))
		b.WriteString(fmt.Sprintf("**Observed:** %s  \n", assertion.Observed))
		b.WriteString(fmt.Sprintf("**Expected:** %s  \n", assertion.Expected))
		if len(assertion.LikelyCauses) > 0 {
			b.WriteString("**Likely causes:**\n")
			for _, cause := range assertion.LikelyCauses {
				b.WriteString(fmt.Sprintf("- %s\n", cause))
			}
		}
		if len(assertion.NextSteps) > 0 {
			b.WriteString("**Next steps:**\n")
			for _, step := range assertion.NextSteps {
				b.WriteString(fmt.Sprintf("- %s\n", step))
			}
		}
		return b.String()
	}
	for _, finding := range result.Findings {
		if finding.Severity != model.SeverityFail {
			continue
		}
		return fmt.Sprintf("**Validation:** %s\n", finding.Message)
	}
	return ""
}

func ciArtifactNames(result model.RunResult) []string {
	items := []string{"report.json", "summary.md", "graph.mmd", "collector.log", "config.patched.yaml"}
	if result.Artifacts.GraphSVG != "" {
		items = append(items, "graph.svg")
	}
	if len(result.Diff.Changes) > 0 {
		items = append(items, "diff.md")
	}
	return items
}
