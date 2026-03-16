package app

import (
	"fmt"

	"github.com/salman-frs/meridian/internal/model"
	"github.com/salman-frs/meridian/internal/validate"
	"github.com/spf13/cobra"
)

func newValidateCommand(global *GlobalOptions) *cobra.Command {
	var failOn string
	var rules string
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Run static validation against a collector config",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = rules
			cfg, err := loadConfig(global)
			if err != nil {
				return err
			}
			findings := validate.Run(cfg)
			if isJSONOutput(global) {
				return printJSON(map[string]any{"findings": findings, "summary": summarizeFindings(findings)})
			}
			fmt.Fprintln(cmd.OutOrStdout(), renderFindings(findings))
			if shouldFail(findings, failOn) {
				return &model.ExitError{Code: 2}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&failOn, "fail-on", "fail", "treat warn or fail findings as command failures")
	cmd.Flags().StringVar(&rules, "rules", "default", "validation profile: default|minimal|all")
	return cmd
}
