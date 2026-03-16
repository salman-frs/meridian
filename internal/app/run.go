package app

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/salman-frs/meridian/internal/assert"
	"github.com/salman-frs/meridian/internal/capture"
	"github.com/salman-frs/meridian/internal/collector"
	"github.com/salman-frs/meridian/internal/configio"
	"github.com/salman-frs/meridian/internal/diffing"
	"github.com/salman-frs/meridian/internal/graph"
	"github.com/salman-frs/meridian/internal/model"
	"github.com/salman-frs/meridian/internal/patch"
	"github.com/salman-frs/meridian/internal/report"
	"github.com/salman-frs/meridian/internal/runtime"
	"github.com/salman-frs/meridian/internal/validate"
)

type RunService struct {
	now func() time.Time
}

type runInputs struct {
	configSources []string
	sourceConfig  model.ConfigModel
	runtimeConfig model.ConfigModel
	envValues     map[string]string
	semanticEnv   map[string]string
	loadDuration  time.Duration
	validatedOpts ResolvedRuntimeOptions
}

type staticRunData struct {
	findings         []model.Finding
	graph            model.GraphModel
	validateDuration time.Duration
	graphDuration    time.Duration
}

type runtimePreparation struct {
	artifacts     model.ArtifactManifest
	ports         model.RuntimePorts
	engine        runtime.ResolvedEngine
	patchedConfig model.ConfigModel
	plan          model.TestPlan
	patchDuration time.Duration
}

func NewRunService() RunService {
	return RunService{now: time.Now}
}

func loadConfig(global *GlobalOptions) (model.ConfigModel, error) {
	if err := validateGlobalOptions(global); err != nil {
		return model.ConfigModel{}, &model.ExitError{Code: 2, Err: err}
	}
	cfg, err := configio.LoadConfig(configio.LoadOptions{
		ConfigPaths: configSources(global),
		ConfigDir:   global.ConfigDir,
		EnvFile:     global.EnvFile,
		EnvInline:   global.EnvInline,
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

func (s RunService) Execute(global *GlobalOptions, runtimeOpts *RuntimeOptions, includeDiff bool) (model.RunResult, error) {
	inputs, err := s.loadInputs(global, runtimeOpts)
	if err != nil {
		return model.RunResult{}, err
	}

	startedAt := s.now().UTC()
	runID := fmt.Sprintf("%s-%d", startedAt.Format("20060102-150405.000000000"), os.Getpid())

	prep, err := s.prepareExecution(inputs.validatedOpts, global.Output, runID)
	if err != nil {
		return model.RunResult{}, err
	}

	result := model.RunResult{
		RunID:          runID,
		ConfigPath:     primaryConfigSource(inputs.configSources),
		Status:         "PASS",
		Engine:         prep.engine.Engine(),
		RuntimeBackend: prep.engine.RuntimeBackend(),
		Mode:           inputs.validatedOpts.Mode,
		CollectorImage: inputs.validatedOpts.CollectorImage,
		StartedAt:      startedAt,
		Timings: map[string]string{
			"config_load": inputs.loadDuration.String(),
		},
		Ports:     prep.ports,
		Artifacts: prep.artifacts,
	}

	semanticStart := s.now()
	semanticResult, err := s.runSemanticValidation(global, inputs, prep.engine)
	if err != nil {
		return model.RunResult{}, err
	}
	result.Semantic = semanticResult
	result.Findings = append(result.Findings, semanticResult.Findings...)
	result.Timings["semantic"] = s.now().Sub(semanticStart).String()

	runtimeConfig, runtimeConfigSource, err := selectRuntimeConfig(inputs, semanticResult)
	if err != nil {
		return model.RunResult{}, &model.ExitError{Code: 3, Err: err}
	}
	inputs.runtimeConfig = runtimeConfig
	result.RuntimeConfigSource = runtimeConfigSource
	if len(inputs.envValues) == 0 {
		if envValues, envErr := loadRuntimeEnv(global, inputs.runtimeConfig); envErr == nil {
			inputs.envValues = envValues
		}
	}

	staticData, err := s.writeStaticArtifacts(inputs.runtimeConfig, prep.artifacts, inputs.validatedOpts.RenderGraph)
	if err != nil {
		return model.RunResult{}, err
	}
	result.Findings = append(result.Findings, staticData.findings...)
	result.Graph = staticData.graph
	result.Timings["validate"] = staticData.validateDuration.String()
	result.Timings["graph"] = staticData.graphDuration.String()

	prep, err = s.patchRuntime(inputs.runtimeConfig, inputs.validatedOpts, prep)
	if err != nil {
		return model.RunResult{}, err
	}
	result.Ports = prep.ports
	result.Plan = prep.plan
	result.Engine = prep.engine.Engine()
	result.RuntimeBackend = prep.engine.RuntimeBackend()
	result.Timings["patch"] = prep.patchDuration.String()

	if err := s.attachDiff(&result, global, runtimeOpts, includeDiff); err != nil {
		return model.RunResult{}, err
	}

	if shouldFail(result.Findings, "fail") {
		return s.finishValidationFailure(result)
	}

	return s.executeRuntimeHarness(result, prep.engine, inputs.runtimeConfig, prep.patchedConfig, inputs.envValues, inputs.validatedOpts)
}

func (s RunService) loadInputs(global *GlobalOptions, runtimeOpts *RuntimeOptions) (runInputs, error) {
	validatedOpts, err := resolveRuntimeOptions(global, runtimeOpts)
	if err != nil {
		return runInputs{}, &model.ExitError{Code: 2, Err: err}
	}
	configSources, err := configio.ExpandConfigSources(configio.LoadOptions{
		ConfigPaths: configSources(global),
		ConfigDir:   global.ConfigDir,
	})
	if err != nil {
		return runInputs{}, &model.ExitError{Code: 2, Err: err}
	}

	cfgLoadStart := s.now()
	semanticEnv, err := configio.LoadEnv(global.EnvFile, global.EnvInline, true)
	if err != nil {
		return runInputs{}, err
	}
	sourceCfg, err := loadConfig(global)
	if err != nil && !isNoLocalConfigSource(err) {
		return runInputs{}, err
	}
	if err != nil {
		sourceCfg = model.ConfigModel{SourcePaths: configSources}
	}
	envValues, err := loadRuntimeEnv(global, sourceCfg)
	if err != nil && !isNoLocalConfigSource(err) {
		return runInputs{}, err
	}
	if err != nil {
		envValues = map[string]string{}
	}

	return runInputs{
		configSources: configSources,
		sourceConfig:  sourceCfg,
		envValues:     envValues,
		semanticEnv:   semanticEnv,
		loadDuration:  s.now().Sub(cfgLoadStart),
		validatedOpts: validatedOpts,
	}, nil
}

func (s RunService) prepareExecution(runtimeOpts ResolvedRuntimeOptions, outputPath string, runID string) (runtimePreparation, error) {
	outputDir, err := filepath.Abs(outputPath)
	if err != nil {
		return runtimePreparation{}, &model.ExitError{Code: 3, Err: err}
	}

	artifacts := model.NewArtifactManifest(outputDir, runID)
	if err := artifacts.Ensure(); err != nil {
		return runtimePreparation{}, &model.ExitError{Code: 3, Err: err}
	}

	ports, err := reserveRuntimePorts()
	if err != nil {
		return runtimePreparation{}, &model.ExitError{Code: 3, Err: err}
	}

	engine, err := runtime.Resolve(runtimeOpts.Engine)
	if err != nil {
		return runtimePreparation{}, &model.ExitError{Code: 3, Err: err}
	}

	return runtimePreparation{
		artifacts: artifacts,
		ports:     ports,
		engine:    engine,
	}, nil
}

func (s RunService) patchRuntime(cfg model.ConfigModel, runtimeOpts ResolvedRuntimeOptions, prep runtimePreparation) (runtimePreparation, error) {
	patchStart := s.now()
	patchedConfig, plan, err := patch.Build(cfg, patch.Options{
		RunID:           filepath.Base(prep.artifacts.RunDir),
		Engine:          prep.engine.Engine(),
		RuntimeBackend:  prep.engine.RuntimeBackend(),
		Mode:            runtimeOpts.Mode,
		CollectorImage:  runtimeOpts.CollectorImage,
		Timeout:         runtimeOpts.Timeout,
		StartupTimeout:  runtimeOpts.StartupTimeout,
		InjectTimeout:   runtimeOpts.InjectTimeout,
		CaptureTimeout:  runtimeOpts.CaptureTimeout,
		PipelineArgs:    runtimeOpts.Pipelines,
		InjectionPort:   prep.ports.InjectionGRPC,
		CapturePort:     prep.ports.CaptureGRPC,
		CaptureEndpoint: prep.engine.CaptureEndpoint(fmt.Sprintf("127.0.0.1:%d", prep.ports.CaptureGRPC), prep.ports.CaptureGRPC),
		CaptureSamples:  runtimeOpts.CaptureSamples,
	})
	if err != nil {
		return runtimePreparation{}, &model.ExitError{Code: 2, Err: err}
	}
	if err := model.WriteText(prep.artifacts.PatchedConfig, patchedConfig.CanonicalYAML); err != nil {
		return runtimePreparation{}, err
	}

	return runtimePreparation{
		artifacts:     prep.artifacts,
		ports:         prep.ports,
		engine:        prep.engine,
		patchedConfig: patchedConfig,
		plan:          plan,
		patchDuration: s.now().Sub(patchStart),
	}, nil
}

func (s RunService) writeStaticArtifacts(cfg model.ConfigModel, artifacts model.ArtifactManifest, renderGraph GraphRenderMode) (staticRunData, error) {
	validateStart := s.now()
	findings := validate.Run(cfg)
	validateDuration := s.now().Sub(validateStart)

	graphStart := s.now()
	g := graph.Build(cfg)
	if err := model.WriteText(artifacts.GraphMMD, graph.RenderMermaid(g)); err != nil {
		return staticRunData{}, err
	}
	graphDuration := s.now().Sub(graphStart)

	if renderGraph == graphRenderSVG {
		svg, err := graph.RenderSVG(graph.RenderDOT(g))
		if err != nil {
			return staticRunData{}, &model.ExitError{Code: 3, Err: fmt.Errorf("graphviz dot is required for --render-graph=svg: %w", err)}
		}
		artifacts.GraphSVG = filepath.Join(artifacts.RunDir, "graph.svg")
		if err := os.WriteFile(artifacts.GraphSVG, svg, 0o644); err != nil {
			return staticRunData{}, err
		}
	}

	return staticRunData{
		findings:         findings,
		graph:            g,
		validateDuration: validateDuration,
		graphDuration:    graphDuration,
	}, nil
}

func (s RunService) attachDiff(result *model.RunResult, global *GlobalOptions, runtimeOpts *RuntimeOptions, includeDiff bool) error {
	if !includeDiff {
		return nil
	}
	if !hasDiffInputs(runtimeOpts) {
		result.Diff = diffing.Empty()
		return nil
	}

	diffStart := s.now()
	semanticEnv, err := configio.LoadEnv(global.EnvFile, global.EnvInline, true)
	if err != nil {
		return &model.ExitError{Code: 2, Err: err}
	}
	diffResult, err := diffing.Run(diffing.Options{
		OldPath:         runtimeOpts.Diff.OldPath,
		NewPath:         diffNewPath(global, runtimeOpts),
		BaseRef:         runtimeOpts.Diff.BaseRef,
		HeadRef:         runtimeOpts.Diff.HeadRef,
		EnvFile:         global.EnvFile,
		EnvInline:       global.EnvInline,
		Env:             semanticEnv,
		Threshold:       runtimeOpts.Diff.Threshold,
		CollectorBinary: global.CollectorBinary,
		CollectorImage:  runtimeOpts.CollectorImage,
		Engine:          model.RuntimeEngine(runtimeOpts.Engine),
	})
	if err != nil {
		return &model.ExitError{Code: 2, Err: err}
	}

	result.Diff = diffResult
	result.Timings["diff"] = s.now().Sub(diffStart).String()
	return nil
}

func (s RunService) finishValidationFailure(result model.RunResult) (model.RunResult, error) {
	result.Status = "FAIL"
	result.Message = "validation failed before runtime execution"
	result.FinishedAt = s.now().UTC()
	result.Timings["total"] = result.FinishedAt.Sub(result.StartedAt).String()
	if err := report.WriteBundle(result); err != nil {
		return model.RunResult{}, err
	}
	return result, nil
}

func (s RunService) executeRuntimeHarness(result model.RunResult, engine runtime.ResolvedEngine, originalCfg model.ConfigModel, patchedConfig model.ConfigModel, envValues map[string]string, runtimeOpts ResolvedRuntimeOptions) (model.RunResult, error) {
	runtimeStart := s.now()
	sink := capture.NewInMemorySink(result.RunID, result.Artifacts.CapturesDir, runtimeOpts.CaptureSamples)
	runner := runtime.NewRunner(runtime.Options{
		Engine:         runtimeOpts.Engine,
		ResolvedEngine: engine,
		CollectorImage: runtimeOpts.CollectorImage,
		Timeout:        runtimeOpts.Timeout,
		StartupTimeout: runtimeOpts.StartupTimeout,
		InjectTimeout:  runtimeOpts.InjectTimeout,
		CaptureTimeout: runtimeOpts.CaptureTimeout,
		KeepContainers: runtimeOpts.KeepContainers,
	})

	runtimeResult, err := runner.Run(runtime.RunRequest{
		Config:      patchedConfig,
		Original:    originalCfg,
		Plan:        result.Plan,
		Artifacts:   result.Artifacts,
		Seed:        runtimeOpts.Seed,
		Assertions:  runtimeOpts.AssertionsFile,
		CaptureSink: sink,
		Env:         envValues,
	})
	if err != nil {
		result.Status = "FAIL"
		result.Message = err.Error()
		result.FinishedAt = s.now().UTC()
		result.Timings["runtime"] = s.now().Sub(runtimeStart).String()
		result.Timings["total"] = result.FinishedAt.Sub(result.StartedAt).String()
		_ = report.WriteBundle(result)
		return model.RunResult{}, err
	}

	result.Timings["runtime"] = s.now().Sub(runtimeStart).String()
	result.Captures = runtimeResult.Captures
	result.Plan = runtimeResult.Plan
	result.RuntimeBackend = runtimeResult.Plan.RuntimeBackend
	result.Assertions = assert.Evaluate(sink, runtimeResult.Captures, runtimeResult.CustomAssertions, runtimeResult.Plan)
	result.Contracts = assert.EvaluateContracts(sink.Normalized(), runtimeResult.Contracts, runtimeResult.Plan)
	result.ContainerID = runtimeResult.ContainerID
	result.ReproCommand = runtimeResult.ReproCommand
	for _, assertion := range result.Assertions {
		if assertion.Status == "FAIL" {
			result.Status = "FAIL"
			break
		}
	}
	if result.Status == "PASS" {
		for _, contract := range result.Contracts {
			if contract.Status == "FAIL" {
				result.Status = "FAIL"
				break
			}
		}
	}
	result.FinishedAt = s.now().UTC()
	if result.Status == "FAIL" && result.Message == "" {
		if len(result.Contracts) > 0 {
			result.Message = "runtime contracts failed"
		} else {
			result.Message = "runtime assertions failed"
		}
	}
	result.Timings["total"] = result.FinishedAt.Sub(result.StartedAt).String()
	if err := report.WriteBundle(result); err != nil {
		return model.RunResult{}, err
	}
	return result, nil
}

func shouldRetryRuntimeRun(err error) bool {
	var exitErr *model.ExitError
	if !errors.As(err, &exitErr) || exitErr == nil || exitErr.Code != 3 {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "port is already allocated") || strings.Contains(message, "address already in use")
}

func reserveRuntimePorts() (model.RuntimePorts, error) {
	injectionPort, err := reservePort()
	if err != nil {
		return model.RuntimePorts{}, err
	}

	capturePort, err := reservePort()
	if err != nil {
		return model.RuntimePorts{}, err
	}
	for capturePort == injectionPort {
		capturePort, err = reservePort()
		if err != nil {
			return model.RuntimePorts{}, err
		}
	}

	return model.RuntimePorts{
		InjectionGRPC: injectionPort,
		CaptureGRPC:   capturePort,
	}, nil
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

func (s RunService) runSemanticValidation(global *GlobalOptions, inputs runInputs, engine runtime.ResolvedEngine) (model.SemanticReport, error) {
	report, err := collector.Analyze(collector.Options{
		ConfigSources:   inputs.configSources,
		ConfigModel:     inputs.sourceConfig,
		Env:             inputs.semanticEnv,
		CollectorBinary: global.CollectorBinary,
		CollectorImage:  inputs.validatedOpts.CollectorImage,
		Engine:          model.RuntimeEngine(inputs.validatedOpts.Engine),
		ResolvedEngine:  engine,
		RequireSemantic: true,
	})
	if err != nil {
		return model.SemanticReport{}, &model.ExitError{Code: 3, Err: err}
	}
	return report, nil
}

func primaryConfigSource(sources []string) string {
	if len(sources) == 0 {
		return ""
	}
	return sources[0]
}

func isNoLocalConfigSource(err error) bool {
	var exitErr *model.ExitError
	if errors.As(err, &exitErr) && exitErr != nil {
		return errors.Is(exitErr.Err, configio.ErrNoLocalConfigSource)
	}
	return errors.Is(err, configio.ErrNoLocalConfigSource)
}

func selectRuntimeConfig(inputs runInputs, semantic model.SemanticReport) (model.ConfigModel, string, error) {
	if runtimeConfigFromSources(inputs.configSources, inputs.sourceConfig) {
		return inputs.sourceConfig, describeSourceConfig(inputs.configSources), nil
	}
	if strings.TrimSpace(semantic.FinalConfig) != "" {
		cfg, err := collector.LoadEffectiveConfig(semantic.FinalConfig, primaryConfigSource(inputs.configSources)+"#print-config")
		if err != nil {
			return model.ConfigModel{}, "", fmt.Errorf("failed to parse collector-rendered config: %w", err)
		}
		cfg.SourcePaths = append([]string{}, inputs.configSources...)
		return cfg, "collector-rendered effective config", nil
	}
	return model.ConfigModel{}, "", errors.New("runtime requires collector print-config when non-local config sources are used")
}

func hasNonLocalSources(sources []string) bool {
	localCount := len(configio.LocalConfigSources(sources))
	return localCount != len(sources)
}

func runtimeConfigFromSources(sources []string, cfg model.ConfigModel) bool {
	if len(cfg.Raw) == 0 {
		return false
	}
	for _, source := range sources {
		if !configio.IsMaterializableConfigSource(source) {
			return false
		}
	}
	return true
}

func describeSourceConfig(sources []string) string {
	if hasNonLocalSources(sources) {
		return "source-merged config"
	}
	return "repo-local config"
}
