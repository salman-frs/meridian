package report

import (
	"fmt"
	"strings"

	"github.com/salman-frs/meridian/internal/model"
)

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
