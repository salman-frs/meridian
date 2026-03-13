package app

import (
	"errors"
	"fmt"
	"time"

	"github.com/salman-frs/meridian/internal/model"
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

func newRuntimeOptions() *RuntimeOptions {
	return &RuntimeOptions{
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
	if runtimeOpts.ChangedOnly && !hasDiffInputs(runtimeOpts) {
		return errors.New("--changed-only requires explicit diff inputs")
	}
	return nil
}

func hasDiffInputs(runtimeOpts *RuntimeOptions) bool {
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
