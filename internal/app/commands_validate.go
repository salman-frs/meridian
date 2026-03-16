package app

import (
	"errors"

	"github.com/salman-frs/meridian/internal/collector"
	"github.com/salman-frs/meridian/internal/configio"
	"github.com/salman-frs/meridian/internal/model"
	"github.com/salman-frs/meridian/internal/validate"
	"github.com/spf13/cobra"
)

func newValidateCommand(global *GlobalOptions) *cobra.Command {
	var failOn string
	var rules string
	semanticOpts := newSemanticOptions()
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Run static validation against a collector config",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := newCommandOutput(cmd, global)

			sources, err := configio.ExpandConfigSources(configio.LoadOptions{
				ConfigPaths: configSources(global),
				ConfigDir:   global.ConfigDir,
			})
			if err != nil {
				return &model.ExitError{Code: 2, Err: err}
			}

			localStage := model.SemanticStage{Name: "local-load", Status: "PASS"}
			cfg, err := configio.LoadConfig(configio.LoadOptions{
				ConfigPaths: configSources(global),
				ConfigDir:   global.ConfigDir,
				EnvFile:     global.EnvFile,
				EnvInline:   global.EnvInline,
			})

			findings := []model.Finding{}
			switch {
			case err == nil:
				findings = validate.Run(cfg)
			case errors.Is(err, configio.ErrNoLocalConfigSource):
				localStage.Status = "SKIP"
				localStage.Message = "no local YAML sources were available for repo-side parsing"
				cfg = model.ConfigModel{SourcePaths: sources}
				findings = append(findings, model.Finding{
					Severity:    model.SeverityInfo,
					Code:        "local-load-skipped",
					Message:     "local structural validation was skipped because all config sources are collector-native URIs",
					Remediation: "provide at least one local YAML file if you want Meridian's repo-side graph and structure checks",
					NextStep:    "use --collector-binary or --collector-image to rely on collector-native semantic validation for URI-only configs",
				})
			default:
				return &model.ExitError{Code: 2, Err: err}
			}
			semanticEnv, err := configio.LoadEnv(global.EnvFile, global.EnvInline, true)
			if err != nil {
				return &model.ExitError{Code: 2, Err: err}
			}

			semantic, err := collector.Analyze(collector.Options{
				ConfigSources:   sources,
				ConfigModel:     cfg,
				Env:             semanticEnv,
				CollectorBinary: global.CollectorBinary,
				CollectorImage:  semanticOpts.CollectorImage,
				Engine:          model.RuntimeEngine(semanticOpts.Engine),
				RequireSemantic: false,
			})
			if err != nil {
				return &model.ExitError{Code: 3, Err: err}
			}
			findings = append(findings, semantic.Findings...)
			_ = out.PrintVerbosef("semantic target: %s (%s)", semantic.Target, semantic.Source)

			if isJSONOutput(global) {
				return out.PrintJSON(map[string]any{
					"config_sources": sources,
					"local_load":     localStage,
					"semantic":       semantic,
					"findings":       findings,
					"summary":        summarizeFindings(findings),
				})
			}

			if err := out.PrintHuman(renderValidationReport(findings, localStage, semantic)); err != nil {
				return err
			}
			if localStage.Status == "SKIP" && !semantic.Enabled {
				return &model.ExitError{Code: 2, Err: errors.New("validate had no local config to parse and semantic validation was skipped")}
			}
			if shouldFail(findings, failOn) {
				return &model.ExitError{Code: 2}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&failOn, "fail-on", "fail", "treat warn or fail findings as command failures")
	cmd.Flags().StringVar(&rules, "rules", "default", "validation profile: default|minimal|all")
	_ = cmd.Flags().MarkHidden("rules")
	_ = cmd.Flags().MarkDeprecated("rules", "validation profiles are not implemented; the default rule set is always used")
	addSemanticFlags(cmd, semanticOpts)
	return cmd
}
