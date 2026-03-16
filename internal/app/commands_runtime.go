package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/salman-frs/meridian/internal/model"
	"github.com/salman-frs/meridian/internal/report"
	"github.com/spf13/cobra"
)

var runServiceFactory = NewRunService
var executeRunService = func(service RunService, global *GlobalOptions, runtimeOpts *RuntimeOptions, includeDiff bool) (model.RunResult, error) {
	return service.Execute(global, runtimeOpts, includeDiff)
}

func newTestCommand(global *GlobalOptions, runtimeOpts *RuntimeOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Run the runtime harness against the collector config",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHarness(global, runtimeOpts, false, cmd)
		},
	}
	addRuntimeFlags(cmd, runtimeOpts)
	addArtifactOutputFlag(cmd, global)
	return cmd
}

func newCheckCommand(global *GlobalOptions, runtimeOpts *RuntimeOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run Meridian's opinionated end-to-end confidence workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHarness(global, runtimeOpts, true, cmd)
		},
	}
	addRuntimeFlags(cmd, runtimeOpts)
	addDiffFlags(cmd, &runtimeOpts.Diff)
	addArtifactOutputFlag(cmd, global)
	return cmd
}

func newCICommand(global *GlobalOptions, runtimeOpts *RuntimeOptions) *cobra.Command {
	var summaryFile string
	var jsonFile string
	var prCommentFile string
	var prMode bool
	cmd := &cobra.Command{
		Use:   "ci",
		Short: "CI-friendly compatibility wrapper around check",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := newCommandOutput(cmd, global)
			_ = out.PrintVerbose("starting CI compatibility workflow")
			result, err := executeRunService(runServiceFactory(), global, runtimeOpts, true)
			if err != nil {
				return err
			}
			summary := report.RenderSummaryMarkdown(result)
			if err := writeCIOutputs(result, summary, summaryFile, jsonFile, prCommentFile); err != nil {
				return err
			}
			annotationWriter := cmd.OutOrStdout()
			if isJSONOutput(global) {
				annotationWriter = cmd.ErrOrStderr()
			}
			report.WriteAnnotations(annotationWriter, result)
			if prMode && !global.Quiet && !isJSONOutput(global) {
				fmt.Fprintln(cmd.OutOrStdout(), report.RenderPRComment(result))
			}
			if err := out.PrintResult(report.RenderTerminal(result), result); err != nil {
				return err
			}
			_ = out.PrintVerbose(renderVerboseResultDetails(result))
			if result.Status != "PASS" {
				return &model.ExitError{Code: 2}
			}
			return nil
		},
	}
	addRuntimeFlags(cmd, runtimeOpts)
	addDiffFlags(cmd, &runtimeOpts.Diff)
	addArtifactOutputFlag(cmd, global)
	cmd.Flags().StringVar(&summaryFile, "summary-file", "", "write GitHub summary markdown to a file")
	cmd.Flags().StringVar(&jsonFile, "json-file", "", "write the JSON report to a file")
	cmd.Flags().StringVar(&prCommentFile, "pr-comment-file", "", "write the PR comment markdown to a file")
	cmd.Flags().BoolVar(&prMode, "pr", false, "print PR-comment markdown to stdout")
	return cmd
}

func runHarness(global *GlobalOptions, runtimeOpts *RuntimeOptions, includeDiff bool, cmd *cobra.Command) error {
	out := newCommandOutput(cmd, global)
	service := runServiceFactory()
	_ = out.PrintVerbose("starting runtime harness")
	result, err := executeRunService(service, global, runtimeOpts, includeDiff)
	if err != nil && shouldRetryRuntimeRun(err) {
		_ = out.PrintVerbosef("retrying runtime harness after transient runtime failure: %v", err)
		result, err = executeRunService(service, global, runtimeOpts, includeDiff)
	}
	if err != nil {
		return err
	}
	if err := out.PrintResult(report.RenderTerminal(result), result); err != nil {
		return err
	}
	_ = out.PrintVerbose(renderVerboseResultDetails(result))
	if result.Status != "PASS" {
		return &model.ExitError{Code: 2}
	}
	return nil
}

func writeCIOutputs(result model.RunResult, summary string, summaryFile string, jsonFile string, prCommentFile string) error {
	if summaryFile != "" {
		if err := os.WriteFile(summaryFile, []byte(summary), 0o644); err != nil {
			return err
		}
	}
	if jsonFile != "" {
		if err := model.WriteJSON(jsonFile, result); err != nil {
			return err
		}
	}
	if prCommentFile != "" {
		if err := os.WriteFile(prCommentFile, []byte(report.RenderPRComment(result)), 0o644); err != nil {
			return err
		}
	}
	if path := os.Getenv("GITHUB_STEP_SUMMARY"); path != "" {
		_ = os.WriteFile(path, []byte(summary), 0o644)
	}
	return nil
}

func renderVerboseResultDetails(result model.RunResult) string {
	lines := []string{}
	if result.Semantic.Target != "" {
		lines = append(lines, "semantic target: "+result.Semantic.Target)
	}
	if timingDetails := renderTimingDetails(result.Timings); timingDetails != "" {
		lines = append(lines, timingDetails)
	}
	return strings.Join(lines, "\n")
}
