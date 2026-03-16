package app

import (
	"fmt"

	"github.com/salman-frs/meridian/internal/diffing"
	"github.com/salman-frs/meridian/internal/model"
	"github.com/salman-frs/meridian/internal/report"
	"github.com/spf13/cobra"
)

func newDiffCommand(global *GlobalOptions) *cobra.Command {
	opts := DiffOptions{Threshold: "low"}
	semanticOpts := newSemanticOptions()
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Compare two collector configs and classify risky changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := diffing.Run(diffing.Options{
				OldPath:         opts.OldPath,
				NewPath:         opts.NewPath,
				BaseRef:         opts.BaseRef,
				HeadRef:         opts.HeadRef,
				EnvFile:         global.EnvFile,
				EnvInline:       global.EnvInline,
				Threshold:       opts.Threshold,
				CollectorBinary: global.CollectorBinary,
				CollectorImage:  semanticOpts.CollectorImage,
				Engine:          model.RuntimeEngine(semanticOpts.Engine),
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
	addSemanticFlags(cmd, semanticOpts)
	return cmd
}
