package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/salman-frs/meridian/internal/assert"
	"github.com/salman-frs/meridian/internal/capture"
	"github.com/salman-frs/meridian/internal/configio"
	"github.com/salman-frs/meridian/internal/diffing"
	"github.com/salman-frs/meridian/internal/graph"
	"github.com/salman-frs/meridian/internal/model"
	"github.com/salman-frs/meridian/internal/patch"
	"github.com/salman-frs/meridian/internal/report"
	"github.com/salman-frs/meridian/internal/runtime"
	"github.com/salman-frs/meridian/internal/validate"
	"github.com/spf13/cobra"
)

const (
	defaultCollectorImage = "otel/opentelemetry-collector-contrib@sha256:e7c92c715f28ff142f3bcaccd4fc5603cf4c71276ef09954a38eb4038500a5a5"
	defaultOutputDir      = "./meridian-artifacts"
)

type GlobalOptions struct {
	ConfigPath string
	ConfigDir  string
	EnvFile    string
	EnvInline  []string
	Format     string
	Output     string
	Quiet      bool
	Verbose    bool
	NoColor    bool
}

type RuntimeOptions struct {
	Engine         string
	Mode           string
	CollectorImage string
	Timeout        time.Duration
	StartupTimeout time.Duration
	InjectTimeout  time.Duration
	CaptureTimeout time.Duration
	Pipelines      []string
	AssertionsFile string
	KeepContainers bool
	Seed           int64
	ChangedOnly    bool
	RenderGraph    string
	CaptureSamples int
	Diff           DiffOptions
}

type DiffOptions struct {
	OldPath   string
	NewPath   string
	BaseRef   string
	HeadRef   string
	Threshold string
}

func NewRootCommand() *cobra.Command {
	opts := &GlobalOptions{}
	runtimeOpts := &RuntimeOptions{
		Engine:         string(model.RuntimeEngineAuto),
		Mode:           string(model.RuntimeModeSafe),
		CollectorImage: defaultCollectorImage,
		Timeout:        30 * time.Second,
		StartupTimeout: 10 * time.Second,
		InjectTimeout:  5 * time.Second,
		CaptureTimeout: 10 * time.Second,
		Seed:           42,
		RenderGraph:    "mermaid",
		CaptureSamples: 5,
		Diff: DiffOptions{
			Threshold: "low",
		},
	}

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

func addRuntimeFlags(cmd *cobra.Command, opts *RuntimeOptions) {
	cmd.Flags().StringVar(&opts.Engine, "engine", string(model.RuntimeEngineAuto), "container engine: auto|docker|containerd")
	cmd.Flags().StringVar(&opts.Mode, "mode", string(model.RuntimeModeSafe), "runtime mode: safe|tee|live")
	cmd.Flags().StringVar(&opts.CollectorImage, "collector-image", defaultCollectorImage, "collector image to run")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 30*time.Second, "overall runtime timeout")
	cmd.Flags().DurationVar(&opts.StartupTimeout, "startup-timeout", 10*time.Second, "collector startup timeout")
	cmd.Flags().DurationVar(&opts.InjectTimeout, "inject-timeout", 5*time.Second, "telemetry injection timeout")
	cmd.Flags().DurationVar(&opts.CaptureTimeout, "capture-timeout", 10*time.Second, "capture wait timeout")
	cmd.Flags().StringSliceVar(&opts.Pipelines, "pipelines", nil, "limit runtime checks to specific signals or pipelines")
	cmd.Flags().StringVar(&opts.AssertionsFile, "assertions", "", "custom assertions YAML file")
	cmd.Flags().BoolVar(&opts.KeepContainers, "keep-containers", false, "keep the collector container running after the test")
	cmd.Flags().Int64Var(&opts.Seed, "seed", 42, "deterministic generation seed")
	cmd.Flags().BoolVar(&opts.ChangedOnly, "changed-only", false, "require explicit diff inputs and include only diff-aware review hints")
	cmd.Flags().StringVar(&opts.RenderGraph, "render-graph", "mermaid", "additional graph artifact for runtime commands: mermaid|svg|none")
	cmd.Flags().IntVar(&opts.CaptureSamples, "capture-samples", 5, "maximum captured telemetry samples to persist per signal")
}

func addDiffFlags(cmd *cobra.Command, opts *DiffOptions) {
	cmd.Flags().StringVar(&opts.OldPath, "old", "", "old collector config file used for diff-aware review")
	cmd.Flags().StringVar(&opts.NewPath, "new", "", "new collector config file used for diff-aware review")
	cmd.Flags().StringVar(&opts.BaseRef, "base", "", "git base ref used to materialize the old config")
	cmd.Flags().StringVar(&opts.HeadRef, "head", "", "git head ref used to materialize the new config")
	cmd.Flags().StringVar(&opts.Threshold, "severity-threshold", "low", "minimum diff severity threshold: low|medium|high")
}

func loadConfig(global *GlobalOptions) (model.ConfigModel, error) {
	if err := validateGlobalOptions(global); err != nil {
		return model.ConfigModel{}, &model.ExitError{Code: 2, Err: err}
	}
	cfg, err := configio.LoadConfig(configio.LoadOptions{
		ConfigPath: global.ConfigPath,
		ConfigDir:  global.ConfigDir,
		EnvFile:    global.EnvFile,
		EnvInline:  global.EnvInline,
	})
	if err != nil {
		return model.ConfigModel{}, &model.ExitError{Code: 2, Err: err}
	}
	return cfg, nil
}

func loadRuntimeEnv(global *GlobalOptions, cfg model.ConfigModel) (map[string]string, error) {
	env, err := configio.LoadEnv(global.EnvFile, global.EnvInline, true)
	if err != nil {
		return nil, &model.ExitError{Code: 2, Err: err}
	}
	filtered := map[string]string{}
	for _, ref := range cfg.EnvReferences {
		if value, ok := env[ref.Name]; ok {
			filtered[ref.Name] = value
		}
	}
	return filtered, nil
}

func executeRun(global *GlobalOptions, runtimeOpts *RuntimeOptions, includeDiff bool) (model.RunResult, error) {
	if err := validateRuntimeOptions(global, runtimeOpts); err != nil {
		return model.RunResult{}, &model.ExitError{Code: 2, Err: err}
	}
	startedAt := time.Now().UTC()
	cfgLoadStart := time.Now()
	cfg, err := loadConfig(global)
	if err != nil {
		return model.RunResult{}, err
	}
	configLoadDuration := time.Since(cfgLoadStart)
	envValues, err := loadRuntimeEnv(global, cfg)
	if err != nil {
		return model.RunResult{}, err
	}
	runID := fmt.Sprintf("%s-%d", startedAt.Format("20060102-150405.000000000"), os.Getpid())
	outputDir, err := filepath.Abs(global.Output)
	if err != nil {
		return model.RunResult{}, &model.ExitError{Code: 3, Err: err}
	}
	artifacts := model.NewArtifactManifest(outputDir, runID)
	if err := artifacts.Ensure(); err != nil {
		return model.RunResult{}, &model.ExitError{Code: 3, Err: err}
	}
	injectionPort, err := reservePort()
	if err != nil {
		return model.RunResult{}, &model.ExitError{Code: 3, Err: err}
	}
	capturePort, err := reservePort()
	if err != nil {
		return model.RunResult{}, &model.ExitError{Code: 3, Err: err}
	}
	for capturePort == injectionPort {
		capturePort, err = reservePort()
		if err != nil {
			return model.RunResult{}, &model.ExitError{Code: 3, Err: err}
		}
	}

	validateStart := time.Now()
	findings := validate.Run(cfg)
	validateDuration := time.Since(validateStart)
	graphStart := time.Now()
	g := graph.Build(cfg)
	if err := model.WriteText(artifacts.GraphMMD, graph.RenderMermaid(g)); err != nil {
		return model.RunResult{}, err
	}
	graphDuration := time.Since(graphStart)
	if runtimeOpts.RenderGraph == "svg" {
		svg, err := graph.RenderSVG(graph.RenderDOT(g))
		if err != nil {
			return model.RunResult{}, &model.ExitError{Code: 3, Err: fmt.Errorf("graphviz dot is required for --render-graph=svg: %w", err)}
		}
		artifacts.GraphSVG = filepath.Join(artifacts.RunDir, "graph.svg")
		if err := os.WriteFile(artifacts.GraphSVG, svg, 0o644); err != nil {
			return model.RunResult{}, err
		}
	}

	patchStart := time.Now()
	engineAdapter, err := runtime.ResolveEngine(model.RuntimeEngine(runtimeOpts.Engine))
	if err != nil {
		return model.RunResult{}, &model.ExitError{Code: 3, Err: err}
	}
	captureEndpoint := engineAdapter.CaptureEndpoint(fmt.Sprintf("127.0.0.1:%d", capturePort), capturePort)
	patchedConfig, plan, patchErr := patch.Build(cfg, patch.Options{
		RunID:           runID,
		Engine:          engineAdapter.Engine(),
		RuntimeBackend:  engineAdapter.RuntimeBackend(),
		Mode:            model.RuntimeMode(runtimeOpts.Mode),
		CollectorImage:  runtimeOpts.CollectorImage,
		Timeout:         runtimeOpts.Timeout,
		StartupTimeout:  runtimeOpts.StartupTimeout,
		InjectTimeout:   runtimeOpts.InjectTimeout,
		CaptureTimeout:  runtimeOpts.CaptureTimeout,
		PipelineArgs:    runtimeOpts.Pipelines,
		InjectionPort:   injectionPort,
		CapturePort:     capturePort,
		CaptureEndpoint: captureEndpoint,
		CaptureSamples:  runtimeOpts.CaptureSamples,
	})
	if patchErr != nil {
		return model.RunResult{}, &model.ExitError{Code: 2, Err: patchErr}
	}
	patchDuration := time.Since(patchStart)
	if err := model.WriteText(artifacts.PatchedConfig, patchedConfig.CanonicalYAML); err != nil {
		return model.RunResult{}, err
	}

	result := model.RunResult{
		RunID:          runID,
		ConfigPath:     cfg.PrimarySourcePath(),
		Status:         "PASS",
		Engine:         engineAdapter.Engine(),
		RuntimeBackend: engineAdapter.RuntimeBackend(),
		Mode:           model.RuntimeMode(runtimeOpts.Mode),
		CollectorImage: runtimeOpts.CollectorImage,
		StartedAt:      startedAt,
		Timings: map[string]string{
			"config_load": configLoadDuration.String(),
			"validate":    validateDuration.String(),
			"graph":       graphDuration.String(),
			"patch":       patchDuration.String(),
		},
		Ports: model.RuntimePorts{
			InjectionGRPC: injectionPort,
			CaptureGRPC:   capturePort,
		},
		Findings:  findings,
		Graph:     g,
		Plan:      plan,
		Artifacts: artifacts,
	}
	if includeDiff && hasDiffInputs(global, runtimeOpts) {
		diffStart := time.Now()
		diffResult, err := diffing.Run(diffing.Options{
			OldPath:   runtimeOpts.Diff.OldPath,
			NewPath:   diffNewPath(global, runtimeOpts),
			BaseRef:   runtimeOpts.Diff.BaseRef,
			HeadRef:   runtimeOpts.Diff.HeadRef,
			EnvFile:   global.EnvFile,
			EnvInline: global.EnvInline,
			Threshold: runtimeOpts.Diff.Threshold,
		})
		if err != nil {
			return model.RunResult{}, &model.ExitError{Code: 2, Err: err}
		}
		result.Diff = diffResult
		result.Timings["diff"] = time.Since(diffStart).String()
	} else if includeDiff {
		result.Diff = diffing.Empty()
	}

	if shouldFail(findings, "fail") {
		result.Status = "FAIL"
		result.Message = "validation failed before runtime execution"
		result.FinishedAt = time.Now().UTC()
		result.Timings["total"] = time.Since(startedAt).String()
		if err := report.WriteBundle(result); err != nil {
			return model.RunResult{}, err
		}
		return result, nil
	}

	runtimeStart := time.Now()
	sink := capture.NewInMemorySink(runID, artifacts.CapturesDir, runtimeOpts.CaptureSamples)
	runner := runtime.NewRunner(runtime.Options{
		Engine:         engineAdapter.Engine(),
		CollectorImage: runtimeOpts.CollectorImage,
		Timeout:        runtimeOpts.Timeout,
		StartupTimeout: runtimeOpts.StartupTimeout,
		InjectTimeout:  runtimeOpts.InjectTimeout,
		CaptureTimeout: runtimeOpts.CaptureTimeout,
		KeepContainers: runtimeOpts.KeepContainers,
	})
	runtimeResult, err := runner.Run(runtime.RunRequest{
		Config:      patchedConfig,
		Original:    cfg,
		Plan:        plan,
		Artifacts:   artifacts,
		Seed:        runtimeOpts.Seed,
		Assertions:  runtimeOpts.AssertionsFile,
		CaptureSink: sink,
		Env:         envValues,
	})
	if err != nil {
		result.Status = "FAIL"
		result.Message = err.Error()
		result.FinishedAt = time.Now().UTC()
		result.Timings["runtime"] = time.Since(runtimeStart).String()
		result.Timings["total"] = time.Since(startedAt).String()
		_ = report.WriteBundle(result)
		return model.RunResult{}, err
	}
	result.Timings["runtime"] = time.Since(runtimeStart).String()
	result.Captures = runtimeResult.Captures
	result.Plan = runtimeResult.Plan
	result.RuntimeBackend = runtimeResult.Plan.RuntimeBackend
	result.Assertions = assert.Evaluate(sink, runtimeResult.Captures, runtimeResult.CustomAssertions, runtimeResult.Plan)
	result.ContainerID = runtimeResult.ContainerID
	result.ReproCommand = runtimeResult.ReproCommand
	for _, assertion := range result.Assertions {
		if assertion.Status == "FAIL" {
			result.Status = "FAIL"
			break
		}
	}
	result.FinishedAt = time.Now().UTC()
	if result.Status == "FAIL" && result.Message == "" {
		result.Message = "runtime assertions failed"
	}
	result.Timings["total"] = time.Since(startedAt).String()
	if err := report.WriteBundle(result); err != nil {
		return model.RunResult{}, err
	}
	return result, nil
}

func runHarness(global *GlobalOptions, runtimeOpts *RuntimeOptions, includeDiff bool, cmd *cobra.Command) error {
	result, err := executeRun(global, runtimeOpts, includeDiff)
	if err != nil && shouldRetryRuntimeRun(err) {
		result, err = executeRun(global, runtimeOpts, includeDiff)
	}
	if err != nil {
		return err
	}
	if global.Format == "json" {
		if err := printJSON(result); err != nil {
			return err
		}
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), report.RenderTerminal(result))
	}
	if result.Status != "PASS" {
		return &model.ExitError{Code: 2}
	}
	return nil
}

func shouldRetryRuntimeRun(err error) bool {
	var exitErr *model.ExitError
	if !errors.As(err, &exitErr) || exitErr == nil || exitErr.Code != 3 {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "port is already allocated") || strings.Contains(message, "address already in use")
}

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

func validateGlobalOptions(global *GlobalOptions) error {
	if global.ConfigPath == "" && global.ConfigDir == "" {
		return errors.New("either --config or --config-dir is required")
	}
	if global.ConfigPath != "" && global.ConfigDir != "" {
		return errors.New("use either --config or --config-dir, not both")
	}
	switch global.Format {
	case "human", "json":
		return nil
	default:
		return fmt.Errorf("unsupported --format %q", global.Format)
	}
}

func validateRuntimeOptions(global *GlobalOptions, runtimeOpts *RuntimeOptions) error {
	if err := validateGlobalOptions(global); err != nil {
		return err
	}
	switch runtimeOpts.Mode {
	case string(model.RuntimeModeSafe), string(model.RuntimeModeTee), string(model.RuntimeModeLive):
	default:
		return fmt.Errorf("unsupported --mode %q", runtimeOpts.Mode)
	}
	switch runtimeOpts.Engine {
	case string(model.RuntimeEngineAuto), string(model.RuntimeEngineDocker), string(model.RuntimeEngineContainerd):
	default:
		return fmt.Errorf("unsupported --engine %q", runtimeOpts.Engine)
	}
	switch runtimeOpts.RenderGraph {
	case "none", "mermaid", "svg":
	default:
		return fmt.Errorf("unsupported --render-graph %q", runtimeOpts.RenderGraph)
	}
	if runtimeOpts.Timeout <= 0 || runtimeOpts.StartupTimeout <= 0 || runtimeOpts.InjectTimeout <= 0 || runtimeOpts.CaptureTimeout <= 0 {
		return errors.New("runtime timeouts must all be greater than zero")
	}
	if runtimeOpts.CaptureSamples <= 0 {
		return errors.New("--capture-samples must be greater than zero")
	}
	if runtimeOpts.ChangedOnly && !hasDiffInputs(global, runtimeOpts) {
		return errors.New("--changed-only requires explicit diff inputs")
	}
	return nil
}

func hasDiffInputs(_ *GlobalOptions, runtimeOpts *RuntimeOptions) bool {
	if runtimeOpts.Diff.OldPath != "" || runtimeOpts.Diff.NewPath != "" {
		return true
	}
	if runtimeOpts.Diff.BaseRef != "" || runtimeOpts.Diff.HeadRef != "" {
		return true
	}
	return false
}

func diffNewPath(global *GlobalOptions, runtimeOpts *RuntimeOptions) string {
	if runtimeOpts.Diff.NewPath != "" {
		return runtimeOpts.Diff.NewPath
	}
	return global.ConfigPath
}

func reservePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, errors.New("failed to allocate a TCP port")
	}
	return addr.Port, nil
}
