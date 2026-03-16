package report

import (
	"fmt"
	"strings"

	"github.com/salman-frs/meridian/internal/model"
)

func RenderContractsMarkdown(contracts []model.ContractResult) string {
	if len(contracts) == 0 {
		return "No contract results.\n"
	}
	lines := []string{"# Contract Results", ""}
	for _, contract := range contracts {
		lines = append(lines, fmt.Sprintf("## %s", contract.ID))
		lines = append(lines, fmt.Sprintf("- Status: %s", contract.Status))
		lines = append(lines, fmt.Sprintf("- Severity: %s", contract.Severity))
		lines = append(lines, fmt.Sprintf("- Signal: %s", contract.Signal))
		if contract.Fixture != "" {
			lines = append(lines, fmt.Sprintf("- Fixture: %s", contract.Fixture))
		}
		if contract.Expected != "" {
			lines = append(lines, fmt.Sprintf("- Expected: %s", contract.Expected))
		}
		if contract.Observed != "" {
			lines = append(lines, fmt.Sprintf("- Observed: %s", contract.Observed))
		}
		lines = append(lines, fmt.Sprintf("- Message: %s", contract.Message))
		if len(contract.Diff) > 0 {
			lines = append(lines, "- Diff:")
			for _, item := range contract.Diff {
				lines = append(lines, "  - "+item)
			}
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func contractStatus(contracts []model.ContractResult) string {
	if len(contracts) == 0 {
		return "not configured"
	}
	failures := 0
	for _, contract := range contracts {
		if contract.Status == "FAIL" {
			failures++
		}
	}
	if failures == 0 {
		return fmt.Sprintf("PASS (%d contract(s))", len(contracts))
	}
	return fmt.Sprintf("FAIL (%d/%d contract(s) failed)", failures, len(contracts))
}
