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

type OutputFormat string

const (
	outputFormatHuman OutputFormat = "human"
	outputFormatJSON  OutputFormat = "json"
)

type GraphRenderMode string

const (
	graphRenderNone    GraphRenderMode = "none"
	graphRenderMermaid GraphRenderMode = "mermaid"
	graphRenderSVG     GraphRenderMode = "svg"
)

type graphOutputRender string

const (
	graphOutputRenderMermaid graphOutputRender = "mermaid"
	graphOutputRenderDOT     graphOutputRender = "dot"
	graphOutputRenderSVG     graphOutputRender = "svg"
	graphOutputRenderNone    graphOutputRender = "none"
)

type GlobalOptions struct {
	ConfigPaths     []string
	ConfigPath      string
	ConfigDir       string
	EnvFile         string
	EnvInline       []string
	Format          string
	Output          string
	CollectorBinary string
	Quiet           bool
	Verbose         bool
	NoColor         bool
}

type SemanticOptions struct {
	Engine         string
	CollectorImage string
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

type ResolvedRuntimeOptions struct {
	Engine         model.RuntimeEngine
	Mode           model.RuntimeMode
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
	RenderGraph    GraphRenderMode
	CaptureSamples int
	Diff           DiffOptions
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

func newSemanticOptions() *SemanticOptions {
	return &SemanticOptions{
		Engine:         string(model.RuntimeEngineAuto),
		CollectorImage: defaultCollectorImage,
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
	cmd.Flags().StringVar(&opts.AssertionsFile, "assertions", "", "assertions or contracts YAML file")
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

func addSemanticFlags(cmd *cobra.Command, opts *SemanticOptions) {
	cmd.Flags().StringVar(&opts.Engine, "engine", string(model.RuntimeEngineAuto), "container engine used for collector image semantic validation: auto|docker|containerd")
	cmd.Flags().StringVar(&opts.CollectorImage, "collector-image", defaultCollectorImage, "collector image used when --collector-binary is not provided")
}

func validateGlobalOptions(global *GlobalOptions) error {
	if len(configSources(global)) == 0 && global.ConfigDir == "" {
		return errors.New("either --config or --config-dir is required")
	}
	_, err := parseOutputFormat(global.Format)
	return err
}

func validateRuntimeOptions(global *GlobalOptions, runtimeOpts *RuntimeOptions) error {
	_, err := resolveRuntimeOptions(global, runtimeOpts)
	return err
}

func resolveRuntimeOptions(global *GlobalOptions, runtimeOpts *RuntimeOptions) (ResolvedRuntimeOptions, error) {
	if err := validateGlobalOptions(global); err != nil {
		return ResolvedRuntimeOptions{}, err
	}
	if runtimeOpts.Timeout <= 0 || runtimeOpts.StartupTimeout <= 0 || runtimeOpts.InjectTimeout <= 0 || runtimeOpts.CaptureTimeout <= 0 {
		return ResolvedRuntimeOptions{}, errors.New("runtime timeouts must all be greater than zero")
	}
	if runtimeOpts.CaptureSamples <= 0 {
		return ResolvedRuntimeOptions{}, errors.New("--capture-samples must be greater than zero")
	}
	if runtimeOpts.ChangedOnly && !hasDiffInputs(runtimeOpts) {
		return ResolvedRuntimeOptions{}, errors.New("--changed-only requires explicit diff inputs")
	}

	engine, err := parseRuntimeEngine(runtimeOpts.Engine)
	if err != nil {
		return ResolvedRuntimeOptions{}, err
	}
	mode, err := parseRuntimeMode(runtimeOpts.Mode)
	if err != nil {
		return ResolvedRuntimeOptions{}, err
	}
	renderGraph, err := parseGraphRenderMode(runtimeOpts.RenderGraph)
	if err != nil {
		return ResolvedRuntimeOptions{}, err
	}

	return ResolvedRuntimeOptions{
		Engine:         engine,
		Mode:           mode,
		CollectorImage: runtimeOpts.CollectorImage,
		Timeout:        runtimeOpts.Timeout,
		StartupTimeout: runtimeOpts.StartupTimeout,
		InjectTimeout:  runtimeOpts.InjectTimeout,
		CaptureTimeout: runtimeOpts.CaptureTimeout,
		Pipelines:      runtimeOpts.Pipelines,
		AssertionsFile: runtimeOpts.AssertionsFile,
		KeepContainers: runtimeOpts.KeepContainers,
		Seed:           runtimeOpts.Seed,
		ChangedOnly:    runtimeOpts.ChangedOnly,
		RenderGraph:    renderGraph,
		CaptureSamples: runtimeOpts.CaptureSamples,
		Diff:           runtimeOpts.Diff,
	}, nil
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
	sources := configSources(global)
	if len(sources) == 0 {
		return ""
	}
	return sources[0]
}

func isJSONOutput(global *GlobalOptions) bool {
	format, err := parseOutputFormat(global.Format)
	return err == nil && format == outputFormatJSON
}

func parseOutputFormat(value string) (OutputFormat, error) {
	switch OutputFormat(value) {
	case outputFormatHuman, outputFormatJSON:
		return OutputFormat(value), nil
	default:
		return "", fmt.Errorf("unsupported --format %q", value)
	}
}

func parseRuntimeMode(value string) (model.RuntimeMode, error) {
	switch model.RuntimeMode(value) {
	case model.RuntimeModeSafe, model.RuntimeModeTee, model.RuntimeModeLive:
		return model.RuntimeMode(value), nil
	default:
		return "", fmt.Errorf("unsupported --mode %q", value)
	}
}

func parseRuntimeEngine(value string) (model.RuntimeEngine, error) {
	switch model.RuntimeEngine(value) {
	case model.RuntimeEngineAuto, model.RuntimeEngineDocker, model.RuntimeEngineContainerd:
		return model.RuntimeEngine(value), nil
	default:
		return "", fmt.Errorf("unsupported --engine %q", value)
	}
}

func parseGraphRenderMode(value string) (GraphRenderMode, error) {
	switch GraphRenderMode(value) {
	case graphRenderNone, graphRenderMermaid, graphRenderSVG:
		return GraphRenderMode(value), nil
	default:
		return "", fmt.Errorf("unsupported --render-graph %q", value)
	}
}

func parseGraphOutputRender(value string) (graphOutputRender, error) {
	switch graphOutputRender(value) {
	case graphOutputRenderMermaid, graphOutputRenderDOT, graphOutputRenderSVG, graphOutputRenderNone:
		return graphOutputRender(value), nil
	default:
		return "", fmt.Errorf("unsupported --render %q", value)
	}
}

func configSources(global *GlobalOptions) []string {
	if len(global.ConfigPaths) > 0 {
		return append([]string{}, global.ConfigPaths...)
	}
	if global.ConfigPath != "" {
		return []string{global.ConfigPath}
	}
	return nil
}
