package app

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newDebugCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Inspect artifacts from a previous Meridian run",
	}
	var runDir string
	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "Print collector logs from a run directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			return printFile(resolveRunPath(runDir, "collector.log"))
		},
	}
	captureCmd := &cobra.Command{
		Use:   "capture",
		Short: "Print capture samples from a run directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			return printCaptureDir(resolveRunPath(runDir, "captures"))
		},
	}
	bundleCmd := &cobra.Command{
		Use:   "bundle",
		Short: "Print the run bundle manifest",
		RunE: func(cmd *cobra.Command, args []string) error {
			reportPath := resolveRunPath(runDir, "report.json")
			data, err := os.ReadFile(reportPath)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		},
	}
	for _, sub := range []*cobra.Command{logsCmd, captureCmd, bundleCmd} {
		sub.Flags().StringVar(&runDir, "run", "", "run directory")
		cmd.AddCommand(sub)
	}
	return cmd
}
