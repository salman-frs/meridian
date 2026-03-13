package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/salman-frs/meridian/internal/model"
)

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func renderFindings(findings []model.Finding) string {
	if len(findings) == 0 {
		return "Validation passed with no findings."
	}
	lines := make([]string, 0, len(findings)+1)
	lines = append(lines, "Validation findings:")
	for _, finding := range findings {
		lines = append(lines, "- "+model.FormatFinding(finding))
	}
	return strings.Join(lines, "\n")
}

func summarizeFindings(findings []model.Finding) map[string]int {
	summary := map[string]int{"info": 0, "warn": 0, "fail": 0}
	for _, finding := range findings {
		summary[string(finding.Severity)]++
	}
	return summary
}

func shouldFail(findings []model.Finding, failOn string) bool {
	for _, finding := range findings {
		if finding.Severity == model.SeverityFail {
			return true
		}
		if failOn == "warn" && finding.Severity == model.SeverityWarn {
			return true
		}
	}
	return false
}

func resolveRunPath(runDir string, child string) string {
	if runDir == "" {
		return filepath.Join(defaultOutputDir, "latest", child)
	}
	return filepath.Join(runDir, child)
}

func printFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	fmt.Print(string(data))
	return nil
}

func printCaptureDir(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(path, entry.Name()))
		if err != nil {
			return err
		}
		fmt.Printf("== %s ==\n%s\n", entry.Name(), string(data))
	}
	return nil
}
