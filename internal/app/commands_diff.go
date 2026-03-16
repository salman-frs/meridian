package app

import (
	"fmt"

	"github.com/salman-frs/meridian/internal/diffing"
	"github.com/salman-frs/meridian/internal/report"
	"github.com/spf13/cobra"
)

func newDiffCommand(global *GlobalOptions) *cobra.Command {
	opts := DiffOptions{Threshold: "low"}
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Compare two collector configs and classify risky changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := diffing.Run(diffing.Options{
				OldPath:   opts.OldPath,
				NewPath:   opts.NewPath,
				BaseRef:   opts.BaseRef,
				HeadRef:   opts.HeadRef,
				EnvFile:   global.EnvFile,
				EnvInline: global.EnvInline,
				Threshold: opts.Threshold,
			})
			if err != nil {
				return err
			}
			if isJSONOutput(global) {
				return printJSON(result)
			}
			fmt.Fprintln(cmd.OutOrStdout(), report.RenderDiff(result))
			return nil
		},
	}
	addDiffFlags(cmd, &opts)
	return cmd
}
