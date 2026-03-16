package model

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Severity string

const (
	SeverityInfo Severity = "info"
	SeverityWarn Severity = "warn"
	SeverityFail Severity = "fail"
)

type SignalType string

const (
	SignalTraces  SignalType = "traces"
	SignalMetrics SignalType = "metrics"
	SignalLogs    SignalType = "logs"
)

type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type ConfigModel struct {
	SourcePaths     []string                 `json:"source_paths" yaml:"source_paths"`
	Raw             map[string]any           `json:"raw" yaml:"raw"`
	CanonicalYAML   string                   `json:"canonical_yaml" yaml:"canonical_yaml"`
	Receivers       map[string]Component     `json:"receivers" yaml:"receivers"`
	Processors      map[string]Component     `json:"processors" yaml:"processors"`
	Exporters       map[string]Component     `json:"exporters" yaml:"exporters"`
	Connectors      map[string]Component     `json:"connectors" yaml:"connectors"`
	Extensions      map[string]Component     `json:"extensions" yaml:"extensions"`
	Pipelines       map[string]PipelineModel `json:"pipelines" yaml:"pipelines"`
	EnvReferences   []EnvReference           `json:"env_references" yaml:"env_references"`
	MissingEnvNames []string                 `json:"missing_env_names" yaml:"missing_env_names"`
}

type Component struct {
	Name   string         `json:"name" yaml:"name"`
	Kind   string         `json:"kind" yaml:"kind"`
	Config map[string]any `json:"config" yaml:"config"`
}

type PipelineModel struct {
	Name       string     `json:"name" yaml:"name"`
	Signal     SignalType `json:"signal" yaml:"signal"`
	Receivers  []string   `json:"receivers" yaml:"receivers"`
	Processors []string   `json:"processors" yaml:"processors"`
	Exporters  []string   `json:"exporters" yaml:"exporters"`
}

type EnvReference struct {
	Name       string `json:"name" yaml:"name"`
	Original   string `json:"original" yaml:"original"`
	HasValue   bool   `json:"has_value" yaml:"has_value"`
	SourcePath string `json:"source_path,omitempty" yaml:"source_path,omitempty"`
	SourceKey  string `json:"source_key,omitempty" yaml:"source_key,omitempty"`
}

type SourceLocation struct {
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
	Key  string `json:"key,omitempty" yaml:"key,omitempty"`
}

type Finding struct {
	Severity    Severity       `json:"severity" yaml:"severity"`
	Code        string         `json:"code" yaml:"code"`
	Message     string         `json:"message" yaml:"message"`
	Location    SourceLocation `json:"location,omitempty" yaml:"location,omitempty"`
	Remediation string         `json:"remediation,omitempty" yaml:"remediation,omitempty"`
	NextStep    string         `json:"next_step,omitempty" yaml:"next_step,omitempty"`
}

type GraphNode struct {
	ID       string            `json:"id" yaml:"id"`
	Label    string            `json:"label" yaml:"label"`
	Kind     string            `json:"kind" yaml:"kind"`
	Signal   SignalType        `json:"signal,omitempty" yaml:"signal,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type GraphEdge struct {
	From   string `json:"from" yaml:"from"`
	To     string `json:"to" yaml:"to"`
	Label  string `json:"label,omitempty" yaml:"label,omitempty"`
	Signal string `json:"signal,omitempty" yaml:"signal,omitempty"`
}

type GraphModel struct {
	Nodes []GraphNode `json:"nodes" yaml:"nodes"`
	Edges []GraphEdge `json:"edges" yaml:"edges"`
}

type DiffChange struct {
	Severity   Severity `json:"severity" yaml:"severity"`
	Kind       string   `json:"kind" yaml:"kind"`
	Message    string   `json:"message" yaml:"message"`
	ReviewHint string   `json:"review_hint,omitempty" yaml:"review_hint,omitempty"`
}

type DiffResult struct {
	OldConfig          string       `json:"old_config,omitempty" yaml:"old_config,omitempty"`
	NewConfig          string       `json:"new_config,omitempty" yaml:"new_config,omitempty"`
	ComparedEffective  bool         `json:"compared_effective_config,omitempty" yaml:"compared_effective_config,omitempty"`
	OldEffectiveConfig string       `json:"old_effective_config,omitempty" yaml:"old_effective_config,omitempty"`
	NewEffectiveConfig string       `json:"new_effective_config,omitempty" yaml:"new_effective_config,omitempty"`
	Changes            []DiffChange `json:"changes" yaml:"changes"`
}

type RuntimeMode string

const (
	RuntimeModeSafe RuntimeMode = "safe"
	RuntimeModeTee  RuntimeMode = "tee"
	RuntimeModeLive RuntimeMode = "live"
)

type RuntimeEngine string

const (
	RuntimeEngineAuto       RuntimeEngine = "auto"
	RuntimeEngineDocker     RuntimeEngine = "docker"
	RuntimeEngineContainerd RuntimeEngine = "containerd"
)

type TestPlan struct {
	RunID             string        `json:"run_id" yaml:"run_id"`
	Engine            RuntimeEngine `json:"engine" yaml:"engine"`
	RuntimeBackend    string        `json:"runtime_backend,omitempty" yaml:"runtime_backend,omitempty"`
	Mode              RuntimeMode   `json:"mode" yaml:"mode"`
	CollectorImage    string        `json:"collector_image" yaml:"collector_image"`
	Pipelines         []string      `json:"pipelines" yaml:"pipelines"`
	Signals           []SignalType  `json:"signals" yaml:"signals"`
	Fixtures          []string      `json:"fixtures,omitempty" yaml:"fixtures,omitempty"`
	InjectedReceiver  string        `json:"injected_receiver" yaml:"injected_receiver"`
	CaptureEndpoint   string        `json:"capture_endpoint" yaml:"capture_endpoint"`
	InjectionEndpoint string        `json:"injection_endpoint" yaml:"injection_endpoint"`
	InjectionPort     int           `json:"injection_port" yaml:"injection_port"`
	CapturePort       int           `json:"capture_port" yaml:"capture_port"`
	Timeout           string        `json:"timeout" yaml:"timeout"`
	StartupTimeout    string        `json:"startup_timeout" yaml:"startup_timeout"`
	InjectTimeout     string        `json:"inject_timeout" yaml:"inject_timeout"`
	CaptureTimeout    string        `json:"capture_timeout" yaml:"capture_timeout"`
	CaptureSamples    int           `json:"capture_samples" yaml:"capture_samples"`
	InjectedAt        time.Time     `json:"injected_at,omitempty" yaml:"injected_at,omitempty"`
}

type RuntimePorts struct {
	InjectionGRPC int `json:"injection_grpc" yaml:"injection_grpc"`
	CaptureGRPC   int `json:"capture_grpc" yaml:"capture_grpc"`
}

type AssertionResult struct {
	ID           string        `json:"id" yaml:"id"`
	Severity     Severity      `json:"severity" yaml:"severity"`
	Signal       SignalType    `json:"signal" yaml:"signal"`
	Status       string        `json:"status" yaml:"status"`
	Message      string        `json:"message" yaml:"message"`
	Observed     string        `json:"observed,omitempty" yaml:"observed,omitempty"`
	Expected     string        `json:"expected,omitempty" yaml:"expected,omitempty"`
	Duration     time.Duration `json:"duration" yaml:"duration"`
	LikelyCauses []string      `json:"likely_causes,omitempty" yaml:"likely_causes,omitempty"`
	NextSteps    []string      `json:"next_steps,omitempty" yaml:"next_steps,omitempty"`
}

type SignalCapture struct {
	Signal          SignalType       `json:"signal" yaml:"signal"`
	Count           int              `json:"count" yaml:"count"`
	Samples         []map[string]any `json:"samples,omitempty" yaml:"samples,omitempty"`
	Errors          []string         `json:"errors,omitempty" yaml:"errors,omitempty"`
	FirstReceivedAt time.Time        `json:"first_received_at,omitempty" yaml:"first_received_at,omitempty"`
	LastReceivedAt  time.Time        `json:"last_received_at,omitempty" yaml:"last_received_at,omitempty"`
	Truncated       bool             `json:"truncated,omitempty" yaml:"truncated,omitempty"`
}

type RunResult struct {
	RunID               string            `json:"run_id" yaml:"run_id"`
	ConfigPath          string            `json:"config_path" yaml:"config_path"`
	RuntimeConfigSource string            `json:"runtime_config_source,omitempty" yaml:"runtime_config_source,omitempty"`
	Status              string            `json:"status" yaml:"status"`
	Message             string            `json:"message,omitempty" yaml:"message,omitempty"`
	Engine              RuntimeEngine     `json:"engine" yaml:"engine"`
	RuntimeBackend      string            `json:"runtime_backend,omitempty" yaml:"runtime_backend,omitempty"`
	Mode                RuntimeMode       `json:"mode" yaml:"mode"`
	CollectorImage      string            `json:"collector_image" yaml:"collector_image"`
	StartedAt           time.Time         `json:"started_at" yaml:"started_at"`
	FinishedAt          time.Time         `json:"finished_at" yaml:"finished_at"`
	Timings             map[string]string `json:"timings" yaml:"timings"`
	Ports               RuntimePorts      `json:"ports" yaml:"ports"`
	Findings            []Finding         `json:"findings,omitempty" yaml:"findings,omitempty"`
	Diff                DiffResult        `json:"diff,omitempty" yaml:"diff,omitempty"`
	Graph               GraphModel        `json:"graph,omitempty" yaml:"graph,omitempty"`
	Semantic            SemanticReport    `json:"semantic,omitempty" yaml:"semantic,omitempty"`
	Plan                TestPlan          `json:"plan,omitempty" yaml:"plan,omitempty"`
	Assertions          []AssertionResult `json:"assertions,omitempty" yaml:"assertions,omitempty"`
	Contracts           []ContractResult  `json:"contracts,omitempty" yaml:"contracts,omitempty"`
	Captures            []SignalCapture   `json:"captures,omitempty" yaml:"captures,omitempty"`
	Artifacts           ArtifactManifest  `json:"artifacts" yaml:"artifacts"`
	ContainerID         string            `json:"container_id,omitempty" yaml:"container_id,omitempty"`
	ReproCommand        string            `json:"repro_command,omitempty" yaml:"repro_command,omitempty"`
}

type ArtifactManifest struct {
	RunDir                string `json:"run_dir" yaml:"run_dir"`
	ReportJSON            string `json:"report_json" yaml:"report_json"`
	SummaryMD             string `json:"summary_md" yaml:"summary_md"`
	GraphMMD              string `json:"graph_mmd" yaml:"graph_mmd"`
	GraphSVG              string `json:"graph_svg,omitempty" yaml:"graph_svg,omitempty"`
	CollectorLog          string `json:"collector_log" yaml:"collector_log"`
	PatchedConfig         string `json:"patched_config" yaml:"patched_config"`
	FinalConfig           string `json:"final_config,omitempty" yaml:"final_config,omitempty"`
	CapturesDir           string `json:"captures_dir" yaml:"captures_dir"`
	CaptureNormalizedJSON string `json:"capture_normalized_json,omitempty" yaml:"capture_normalized_json,omitempty"`
	ComponentsJSON        string `json:"components_json,omitempty" yaml:"components_json,omitempty"`
	SemanticJSON          string `json:"semantic_json,omitempty" yaml:"semantic_json,omitempty"`
	DiffMD                string `json:"diff_md,omitempty" yaml:"diff_md,omitempty"`
	ContractsJSON         string `json:"contracts_json,omitempty" yaml:"contracts_json,omitempty"`
	ContractsMD           string `json:"contracts_md,omitempty" yaml:"contracts_md,omitempty"`
}

type SemanticStage struct {
	Name    string `json:"name" yaml:"name"`
	Status  string `json:"status" yaml:"status"`
	Message string `json:"message,omitempty" yaml:"message,omitempty"`
}

type CollectorComponent struct {
	Kind      string `json:"kind" yaml:"kind"`
	Name      string `json:"name" yaml:"name"`
	Stability string `json:"stability,omitempty" yaml:"stability,omitempty"`
	Raw       string `json:"raw,omitempty" yaml:"raw,omitempty"`
}

type SemanticReport struct {
	Enabled       bool                 `json:"enabled" yaml:"enabled"`
	Status        string               `json:"status,omitempty" yaml:"status,omitempty"`
	Source        string               `json:"source,omitempty" yaml:"source,omitempty"`
	Target        string               `json:"target,omitempty" yaml:"target,omitempty"`
	SkippedReason string               `json:"skipped_reason,omitempty" yaml:"skipped_reason,omitempty"`
	Stages        []SemanticStage      `json:"stages,omitempty" yaml:"stages,omitempty"`
	Findings      []Finding            `json:"findings,omitempty" yaml:"findings,omitempty"`
	Components    []CollectorComponent `json:"components,omitempty" yaml:"components,omitempty"`
	RawComponents string               `json:"raw_components,omitempty" yaml:"raw_components,omitempty"`
	FinalConfig   string               `json:"final_config,omitempty" yaml:"final_config,omitempty"`
	UsedForDiff   bool                 `json:"used_for_diff,omitempty" yaml:"used_for_diff,omitempty"`
}

type AssertionFile struct {
	Version    int               `yaml:"version"`
	Defaults   AssertionDefaults `yaml:"defaults"`
	Fixtures   []string          `yaml:"fixtures"`
	Assertions []AssertionSpec   `yaml:"assertions"`
	Contracts  []ContractSpec    `yaml:"contracts"`
}

type AssertionDefaults struct {
	Timeout  string `yaml:"timeout"`
	MinCount int    `yaml:"min_count"`
}

type AssertionSpec struct {
	ID       string          `yaml:"id"`
	Severity Severity        `yaml:"severity"`
	Signal   SignalType      `yaml:"signal"`
	Where    AssertionWhere  `yaml:"where"`
	Expect   AssertionExpect `yaml:"expect"`
}

type AssertionWhere struct {
	Attributes map[string]string `yaml:"attributes"`
	SpanName   string            `yaml:"span_name"`
	MetricName string            `yaml:"metric_name"`
	Body       string            `yaml:"body"`
}

type AssertionExpect struct {
	MinCount          int      `yaml:"min_count"`
	Exists            *bool    `yaml:"exists"`
	AttributesPresent []string `yaml:"attributes_present"`
	AttributesAbsent  []string `yaml:"attributes_absent"`
}

func NewArtifactManifest(baseOutput string, runID string) ArtifactManifest {
	runDir := filepath.Join(baseOutput, "runs", runID)
	return ArtifactManifest{
		RunDir:                runDir,
		ReportJSON:            filepath.Join(runDir, "report.json"),
		SummaryMD:             filepath.Join(runDir, "summary.md"),
		GraphMMD:              filepath.Join(runDir, "graph.mmd"),
		CollectorLog:          filepath.Join(runDir, "collector.log"),
		PatchedConfig:         filepath.Join(runDir, "config.patched.yaml"),
		FinalConfig:           filepath.Join(runDir, "config.final.yaml"),
		CapturesDir:           filepath.Join(runDir, "captures"),
		CaptureNormalizedJSON: filepath.Join(runDir, "capture.normalized.json"),
		ComponentsJSON:        filepath.Join(runDir, "collector-components.json"),
		SemanticJSON:          filepath.Join(runDir, "semantic-findings.json"),
		DiffMD:                filepath.Join(runDir, "diff.md"),
		ContractsJSON:         filepath.Join(runDir, "contracts.json"),
		ContractsMD:           filepath.Join(runDir, "contracts.md"),
	}
}

func (a ArtifactManifest) Ensure() error {
	if err := os.MkdirAll(a.RunDir, 0o755); err != nil {
		return err
	}
	return os.MkdirAll(a.CapturesDir, 0o755)
}

func (c ConfigModel) PipelineNames() []string {
	names := make([]string, 0, len(c.Pipelines))
	for name := range c.Pipelines {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (c ConfigModel) Signals() []SignalType {
	seen := map[SignalType]struct{}{}
	for _, p := range c.Pipelines {
		seen[p.Signal] = struct{}{}
	}
	out := make([]SignalType, 0, len(seen))
	for _, signal := range []SignalType{SignalTraces, SignalMetrics, SignalLogs} {
		if _, ok := seen[signal]; ok {
			out = append(out, signal)
		}
	}
	return out
}

func (c ConfigModel) PrimarySourcePath() string {
	if len(c.SourcePaths) == 0 {
		return ""
	}
	return c.SourcePaths[0]
}

func SeverityRank(severity Severity) int {
	switch severity {
	case SeverityFail:
		return 3
	case SeverityWarn:
		return 2
	case SeverityInfo:
		return 1
	default:
		return 0
	}
}

func DetectSignalType(name string) SignalType {
	switch {
	case strings.Contains(name, "trace"):
		return SignalTraces
	case strings.Contains(name, "metric"):
		return SignalMetrics
	case strings.Contains(name, "log"):
		return SignalLogs
	default:
		return SignalTraces
	}
}

func MarshalYAML(v any) (string, error) {
	out, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func WriteJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func WriteText(path string, text string) error {
	return os.WriteFile(path, []byte(text), 0o644)
}

func FormatFinding(f Finding) string {
	location := ""
	if f.Location.Path != "" || f.Location.Key != "" {
		location = fmt.Sprintf(" (%s%s)", f.Location.Path, keySuffix(f.Location.Key))
	}
	return fmt.Sprintf("[%s] %s%s", strings.ToUpper(string(f.Severity)), f.Message, location)
}

func keySuffix(key string) string {
	if key == "" {
		return ""
	}
	return "#" + key
}
