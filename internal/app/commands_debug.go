package app

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/salman-frs/meridian/internal/model"
	"github.com/salman-frs/meridian/internal/report"
	"github.com/spf13/cobra"
)

func newDebugCommand(global *GlobalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Inspect artifacts from a previous Meridian run",
	}
	var runDir string
	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "Print collector logs from a run directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			return printFile(cmd.OutOrStdout(), resolveRunPath(global.Output, runDir, "collector.log"))
		},
	}
	captureCmd := &cobra.Command{
		Use:   "capture",
		Short: "Print capture samples from a run directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			return printCaptureDir(cmd.OutOrStdout(), resolveRunPath(global.Output, runDir, "captures"))
		},
	}
	summaryCmd := &cobra.Command{
		Use:   "summary",
		Short: "Print the stored markdown summary",
		RunE: func(cmd *cobra.Command, args []string) error {
			return printFile(cmd.OutOrStdout(), resolveRunPath(global.Output, runDir, "summary.md"))
		},
	}
	bundleCmd := &cobra.Command{
		Use:   "bundle",
		Short: "Print the run bundle manifest",
		RunE: func(cmd *cobra.Command, args []string) error {
			reportPath := resolveRunPath(global.Output, runDir, "report.json")
			data, err := os.ReadFile(reportPath)
			if err != nil {
				return err
			}
			if isJSONOutput(global) {
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}
			var result model.RunResult
			if err := json.Unmarshal(data, &result); err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), report.RenderBundleSummary(result))
			return err
		},
	}
	addArtifactOutputFlag(cmd, global)
	for _, sub := range []*cobra.Command{logsCmd, captureCmd, summaryCmd, bundleCmd} {
		sub.Flags().StringVar(&runDir, "run", "", "run directory")
		cmd.AddCommand(sub)
	}
	return cmd
}
