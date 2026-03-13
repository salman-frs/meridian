package app

import (
	"errors"
	"fmt"
	"os"

	"github.com/salman-frs/meridian/internal/diffing"
	"github.com/salman-frs/meridian/internal/graph"
	"github.com/salman-frs/meridian/internal/model"
	"github.com/salman-frs/meridian/internal/report"
	"github.com/salman-frs/meridian/internal/validate"
	"github.com/spf13/cobra"
)

// NewRootCommand builds the Meridian CLI.
func NewRootCommand() *cobra.Command {
	opts := &GlobalOptions{}
	runtimeOpts := newRuntimeOptions()

	root := &cobra.Command{
		Use:           "meridian",
		Short:         "Review and runtime-test OpenTelemetry Collector configs",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.PersistentFlags().StringVarP(&opts.ConfigPath, "config", "c", "", "path to collector config YAML")
	root.PersistentFlags().StringVar(&opts.ConfigDir, "config-dir", "", "path to a rendered collector config directory")
	root.PersistentFlags().StringVar(&opts.EnvFile, "env-file", "", "dotenv file used for config interpolation")
	root.PersistentFlags().StringArrayVar(&opts.EnvInline, "env", nil, "inline KEY=VALUE env vars")
	root.PersistentFlags().StringVar(&opts.Format, "format", "human", "output format: human|json")
	root.PersistentFlags().StringVar(&opts.Output, "output", defaultOutputDir, "artifact output directory")
	root.PersistentFlags().BoolVar(&opts.Quiet, "quiet", false, "suppress human progress output")
	root.PersistentFlags().BoolVar(&opts.Verbose, "verbose", false, "enable verbose output")
	root.PersistentFlags().BoolVar(&opts.NoColor, "no-color", false, "disable colorized output")

	root.AddCommand(newValidateCommand(opts))
	root.AddCommand(newGraphCommand(opts))
	root.AddCommand(newDiffCommand(opts))
	root.AddCommand(newTestCommand(opts, runtimeOpts))
	root.AddCommand(newCheckCommand(opts, runtimeOpts))
	root.AddCommand(newCICommand(opts, runtimeOpts))
	root.AddCommand(newDebugCommand(opts))
	root.AddCommand(newVersionCommand())
	root.AddCommand(newCompletionCommand(root))
	return root
}

// ExitCode maps Meridian errors to process exit codes.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *model.ExitError
	if errors.As(err, &exitErr) && exitErr != nil {
		return exitErr.Code
	}
	return 1
}

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
			if global.Format == "json" {
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

func newGraphCommand(global *GlobalOptions) *cobra.Command {
	var renderMode string
	var outPath string
	var view string
	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Build a pipeline graph for the collector config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(global)
			if err != nil {
				return err
			}
			g := graph.Build(cfg)
			rendered := graph.RenderMermaid(g)
			if view != "" {
				fmt.Fprintln(cmd.OutOrStdout(), graph.RenderTable(cfg))
			}
			if global.Format == "json" {
				return printJSON(g)
			}
			if renderMode == "none" {
				return nil
			}
			if renderMode == "dot" {
				rendered = graph.RenderDOT(g)
			}
			if renderMode == "svg" {
				dot := graph.RenderDOT(g)
				svg, err := graph.RenderSVG(dot)
				if err != nil {
					return &model.ExitError{Code: 3, Err: fmt.Errorf("graphviz dot is required for --render=svg: %w", err)}
				}
				if outPath == "" {
					outPath = "graph.svg"
				}
				return os.WriteFile(outPath, svg, 0o644)
			}
			if outPath != "" {
				return os.WriteFile(outPath, []byte(rendered), 0o644)
			}
			fmt.Fprintln(cmd.OutOrStdout(), rendered)
			return nil
		},
	}
	cmd.Flags().StringVar(&renderMode, "render", "mermaid", "render mode: mermaid|dot|svg|none")
	cmd.Flags().StringVar(&outPath, "out", "", "write graph output to a file")
	cmd.Flags().StringVar(&view, "view", "table", "terminal view: ascii|table")
	return cmd
}

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
			if global.Format == "json" {
				return printJSON(result)
			}
			fmt.Fprintln(cmd.OutOrStdout(), report.RenderDiff(result))
			return nil
		},
	}
	addDiffFlags(cmd, &opts)
	return cmd
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
			result, err := executeRun(global, runtimeOpts, true)
			if err != nil {
				return err
			}
			summary := report.RenderSummaryMarkdown(result)
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
			report.WriteAnnotations(result)
			if prMode && !global.Quiet {
				fmt.Fprintln(cmd.OutOrStdout(), report.RenderPRComment(result))
			}
			fmt.Fprintln(cmd.OutOrStdout(), report.RenderTerminal(result))
			if result.Status != "PASS" {
				return &model.ExitError{Code: 2}
			}
			return nil
		},
	}
	addRuntimeFlags(cmd, runtimeOpts)
	addDiffFlags(cmd, &runtimeOpts.Diff)
	cmd.Flags().StringVar(&summaryFile, "summary-file", "", "write GitHub summary markdown to a file")
	cmd.Flags().StringVar(&jsonFile, "json-file", "", "write the JSON report to a file")
	cmd.Flags().StringVar(&prCommentFile, "pr-comment-file", "", "write the PR comment markdown to a file")
	cmd.Flags().BoolVar(&prMode, "pr", false, "print PR-comment markdown to stdout")
	return cmd
}

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

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the Meridian version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "meridian dev")
		},
	}
}

func newCompletionCommand(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return root.GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return root.GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				return &model.ExitError{Code: 2, Err: fmt.Errorf("unsupported shell %q", args[0])}
			}
		},
	}
}
