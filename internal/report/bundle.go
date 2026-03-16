package report

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/salman-frs/meridian/internal/model"
)

func WriteBundle(result model.RunResult) error {
	if err := result.Artifacts.Ensure(); err != nil {
		return err
	}
	if result.Semantic.Enabled {
		if len(result.Semantic.Findings) > 0 {
			if err := model.WriteJSON(result.Artifacts.SemanticJSON, result.Semantic.Findings); err != nil {
				return err
			}
		}
		if len(result.Semantic.Components) > 0 || result.Semantic.RawComponents != "" {
			payload := map[string]any{
				"components": result.Semantic.Components,
				"raw":        result.Semantic.RawComponents,
			}
			if err := model.WriteJSON(result.Artifacts.ComponentsJSON, payload); err != nil {
				return err
			}
		}
		if strings.TrimSpace(result.Semantic.FinalConfig) != "" {
			if err := model.WriteText(result.Artifacts.FinalConfig, result.Semantic.FinalConfig); err != nil {
				return err
			}
		}
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
	if err := model.WriteJSON(result.Artifacts.ContractsJSON, result.Contracts); err != nil {
		return err
	}
	if err := model.WriteText(result.Artifacts.ContractsMD, RenderContractsMarkdown(result.Contracts)); err != nil {
		return err
	}
	latest := filepath.Join(filepath.Dir(result.Artifacts.RunDir), "latest")
	_ = os.Remove(latest)
	_ = os.Symlink(result.Artifacts.RunDir, latest)
	return nil
}

func WriteAnnotations(w io.Writer, result model.RunResult) {
	count := 0
	for _, contract := range result.Contracts {
		if contract.Status != "FAIL" || count >= 3 {
			continue
		}
		fmt.Fprintf(w, "::error title=%s::%s (%s)\n", contract.ID, contract.Message, strings.Join(contract.Diff, "; "))
		count++
	}
	for _, finding := range result.Findings {
		if finding.Severity != model.SeverityFail || count >= 3 {
			continue
		}
		fmt.Fprintf(w, "::error title=%s::%s\n", finding.Code, finding.Message)
		count++
	}
	for _, assertion := range result.Assertions {
		if assertion.Status != "FAIL" || count >= 3 {
			continue
		}
		fmt.Fprintf(w, "::error title=%s::%s (observed %s expected %s)\n", assertion.ID, assertion.Message, assertion.Observed, assertion.Expected)
		count++
	}
}
