package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/salman-frs/meridian/internal/model"
	"github.com/spf13/cobra"
)

type commandOutput struct {
	cmd    *cobra.Command
	global *GlobalOptions
}

func newCommandOutput(cmd *cobra.Command, global *GlobalOptions) commandOutput {
	return commandOutput{cmd: cmd, global: global}
}

func (o commandOutput) PrintHuman(text string) error {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	_, err := fmt.Fprintln(o.cmd.OutOrStdout(), text)
	return err
}

func (o commandOutput) PrintJSON(v any) error {
	return writeJSON(o.cmd.OutOrStdout(), v)
}

func (o commandOutput) PrintResult(human string, payload any) error {
	if isJSONOutput(o.global) {
		return o.PrintJSON(payload)
	}
	return o.PrintHuman(human)
}

func (o commandOutput) PrintVerbose(text string) error {
	if !o.global.Verbose || o.global.Quiet || isJSONOutput(o.global) || strings.TrimSpace(text) == "" {
		return nil
	}
	_, err := fmt.Fprintf(o.cmd.ErrOrStderr(), "VERBOSE: %s\n", text)
	return err
}

func (o commandOutput) PrintVerbosef(format string, args ...any) error {
	return o.PrintVerbose(fmt.Sprintf(format, args...))
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
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

func renderValidationReport(findings []model.Finding, localStage model.SemanticStage, semantic model.SemanticReport) string {
	lines := []string{"Validation stages:"}
	lines = append(lines, "- local-load: "+localStage.Status+stageMessage(localStage.Message))
	if semantic.Enabled || semantic.SkippedReason != "" || len(semantic.Stages) > 0 {
		if semantic.Enabled {
			lines = append(lines, "- semantic: "+semantic.Status+stageMessage(semantic.Target))
		} else {
			lines = append(lines, "- semantic: SKIP"+stageMessage(semantic.SkippedReason))
		}
		for _, stage := range semantic.Stages {
			lines = append(lines, "- semantic/"+stage.Name+": "+stage.Status+stageMessage(stage.Message))
		}
	}
	lines = append(lines, "", renderFindings(findings))
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

func resolveRunPath(outputDir string, runDir string, child string) string {
	if runDir == "" {
		return filepath.Join(outputDir, "runs", "latest", child)
	}
	return filepath.Join(runDir, child)
}

func printFile(w io.Writer, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, string(data))
	return nil
}

func printCaptureDir(w io.Writer, path string) error {
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
		if _, err := fmt.Fprintf(w, "== %s ==\n%s\n", entry.Name(), string(data)); err != nil {
			return err
		}
	}
	return nil
}

func renderTimingDetails(timings map[string]string) string {
	if len(timings) == 0 {
		return ""
	}
	order := []string{"config_load", "semantic", "validate", "graph", "patch", "diff", "runtime", "total"}
	lines := []string{"timings:"}
	for _, key := range order {
		if value := strings.TrimSpace(timings[key]); value != "" {
			lines = append(lines, fmt.Sprintf("- %s: %s", key, value))
		}
	}
	return strings.Join(lines, "\n")
}

func stageMessage(message string) string {
	if strings.TrimSpace(message) == "" {
		return ""
	}
	return " (" + strings.TrimSpace(message) + ")"
}
