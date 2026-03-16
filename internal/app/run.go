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
	config        model.ConfigModel
	envValues     map[string]string
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

func (s RunService) Execute(global *GlobalOptions, runtimeOpts *RuntimeOptions, includeDiff bool) (model.RunResult, error) {
	inputs, err := s.loadInputs(global, runtimeOpts)
	if err != nil {
		return model.RunResult{}, err
	}

	startedAt := s.now().UTC()
	runID := fmt.Sprintf("%s-%d", startedAt.Format("20060102-150405.000000000"), os.Getpid())

	prep, err := s.prepareRuntime(inputs.config, inputs.validatedOpts, global.Output, runID)
	if err != nil {
		return model.RunResult{}, err
	}

	staticData, err := s.writeStaticArtifacts(inputs.config, prep.artifacts, inputs.validatedOpts.RenderGraph)
	if err != nil {
		return model.RunResult{}, err
	}

	result := model.RunResult{
		RunID:          runID,
		ConfigPath:     inputs.config.PrimarySourcePath(),
		Status:         "PASS",
		Engine:         prep.engine.Engine(),
		RuntimeBackend: prep.engine.RuntimeBackend(),
		Mode:           inputs.validatedOpts.Mode,
		CollectorImage: inputs.validatedOpts.CollectorImage,
		StartedAt:      startedAt,
		Timings: map[string]string{
			"config_load": inputs.loadDuration.String(),
			"validate":    staticData.validateDuration.String(),
			"graph":       staticData.graphDuration.String(),
			"patch":       prep.patchDuration.String(),
		},
		Ports:     prep.ports,
		Findings:  staticData.findings,
		Graph:     staticData.graph,
		Plan:      prep.plan,
		Artifacts: prep.artifacts,
	}

	if err := s.attachDiff(&result, global, runtimeOpts, includeDiff); err != nil {
		return model.RunResult{}, err
	}

	if shouldFail(staticData.findings, "fail") {
		return s.finishValidationFailure(result)
	}

	return s.executeRuntimeHarness(result, prep.engine, inputs.config, prep.patchedConfig, inputs.envValues, inputs.validatedOpts)
}

func (s RunService) loadInputs(global *GlobalOptions, runtimeOpts *RuntimeOptions) (runInputs, error) {
	validatedOpts, err := resolveRuntimeOptions(global, runtimeOpts)
	if err != nil {
		return runInputs{}, &model.ExitError{Code: 2, Err: err}
	}

	cfgLoadStart := s.now()
	cfg, err := loadConfig(global)
	if err != nil {
		return runInputs{}, err
	}
	envValues, err := loadRuntimeEnv(global, cfg)
	if err != nil {
		return runInputs{}, err
	}

	return runInputs{
		config:        cfg,
		envValues:     envValues,
		loadDuration:  s.now().Sub(cfgLoadStart),
		validatedOpts: validatedOpts,
	}, nil
}

func (s RunService) prepareRuntime(cfg model.ConfigModel, runtimeOpts ResolvedRuntimeOptions, outputPath string, runID string) (runtimePreparation, error) {
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

	patchStart := s.now()
	engine, err := runtime.Resolve(runtimeOpts.Engine)
	if err != nil {
		return runtimePreparation{}, &model.ExitError{Code: 3, Err: err}
	}

	patchedConfig, plan, err := patch.Build(cfg, patch.Options{
		RunID:           runID,
		Engine:          engine.Engine(),
		RuntimeBackend:  engine.RuntimeBackend(),
		Mode:            runtimeOpts.Mode,
		CollectorImage:  runtimeOpts.CollectorImage,
		Timeout:         runtimeOpts.Timeout,
		StartupTimeout:  runtimeOpts.StartupTimeout,
		InjectTimeout:   runtimeOpts.InjectTimeout,
		CaptureTimeout:  runtimeOpts.CaptureTimeout,
		PipelineArgs:    runtimeOpts.Pipelines,
		InjectionPort:   ports.InjectionGRPC,
		CapturePort:     ports.CaptureGRPC,
		CaptureEndpoint: engine.CaptureEndpoint(fmt.Sprintf("127.0.0.1:%d", ports.CaptureGRPC), ports.CaptureGRPC),
		CaptureSamples:  runtimeOpts.CaptureSamples,
	})
	if err != nil {
		return runtimePreparation{}, &model.ExitError{Code: 2, Err: err}
	}
	if err := model.WriteText(artifacts.PatchedConfig, patchedConfig.CanonicalYAML); err != nil {
		return runtimePreparation{}, err
	}

	return runtimePreparation{
		artifacts:     artifacts,
		ports:         ports,
		engine:        engine,
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
	result.ContainerID = runtimeResult.ContainerID
	result.ReproCommand = runtimeResult.ReproCommand
	for _, assertion := range result.Assertions {
		if assertion.Status == "FAIL" {
			result.Status = "FAIL"
			break
		}
	}
	result.FinishedAt = s.now().UTC()
	if result.Status == "FAIL" && result.Message == "" {
		result.Message = "runtime assertions failed"
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
