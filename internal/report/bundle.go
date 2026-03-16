package report

import (
	"fmt"
	"os"
	"path/filepath"

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
