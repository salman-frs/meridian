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
	"github.com/spf13/cobra"
)

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
	cfg, envValues, configLoadDuration, err := loadRunInputs(global)
	if err != nil {
		return model.RunResult{}, err
	}

	runID := fmt.Sprintf("%s-%d", startedAt.Format("20060102-150405.000000000"), os.Getpid())
	artifacts, ports, err := prepareRunArtifacts(global.Output, runID)
	if err != nil {
		return model.RunResult{}, err
	}

	artifacts, findings, g, validateDuration, graphDuration, err := writeStaticArtifacts(cfg, artifacts, runtimeOpts.RenderGraph)
	if err != nil {
		return model.RunResult{}, err
	}

	engineAdapter, patchedConfig, plan, patchDuration, err := buildPatchedRuntimeConfig(cfg, runtimeOpts, runID, ports)
	if err != nil {
		return model.RunResult{}, err
	}
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
		Ports:     ports,
		Findings:  findings,
		Graph:     g,
		Plan:      plan,
		Artifacts: artifacts,
	}

	if err := maybeAttachDiff(&result, global, runtimeOpts, includeDiff); err != nil {
		return model.RunResult{}, err
	}

	if shouldFail(findings, "fail") {
		return finishValidationFailure(result)
	}

	return executeRuntimeHarness(result, cfg, patchedConfig, envValues, runtimeOpts)
}

func loadRunInputs(global *GlobalOptions) (model.ConfigModel, map[string]string, time.Duration, error) {
	cfgLoadStart := time.Now()
	cfg, err := loadConfig(global)
	if err != nil {
		return model.ConfigModel{}, nil, 0, err
	}
	envValues, err := loadRuntimeEnv(global, cfg)
	if err != nil {
		return model.ConfigModel{}, nil, 0, err
	}
	return cfg, envValues, time.Since(cfgLoadStart), nil
}

func prepareRunArtifacts(outputPath string, runID string) (model.ArtifactManifest, model.RuntimePorts, error) {
	outputDir, err := filepath.Abs(outputPath)
	if err != nil {
		return model.ArtifactManifest{}, model.RuntimePorts{}, &model.ExitError{Code: 3, Err: err}
	}
	artifacts := model.NewArtifactManifest(outputDir, runID)
	if err := artifacts.Ensure(); err != nil {
		return model.ArtifactManifest{}, model.RuntimePorts{}, &model.ExitError{Code: 3, Err: err}
	}
	ports, err := reserveRuntimePorts()
	if err != nil {
		return model.ArtifactManifest{}, model.RuntimePorts{}, &model.ExitError{Code: 3, Err: err}
	}
	return artifacts, ports, nil
}

func writeStaticArtifacts(cfg model.ConfigModel, artifacts model.ArtifactManifest, renderGraph string) (model.ArtifactManifest, []model.Finding, model.GraphModel, time.Duration, time.Duration, error) {
	validateStart := time.Now()
	findings := validate.Run(cfg)
	validateDuration := time.Since(validateStart)

	graphStart := time.Now()
	g := graph.Build(cfg)
	if err := model.WriteText(artifacts.GraphMMD, graph.RenderMermaid(g)); err != nil {
		return model.ArtifactManifest{}, nil, model.GraphModel{}, 0, 0, err
	}
	graphDuration := time.Since(graphStart)

	if renderGraph == "svg" {
		svg, err := graph.RenderSVG(graph.RenderDOT(g))
		if err != nil {
			return model.ArtifactManifest{}, nil, model.GraphModel{}, 0, 0, &model.ExitError{Code: 3, Err: fmt.Errorf("graphviz dot is required for --render-graph=svg: %w", err)}
		}
		artifacts.GraphSVG = filepath.Join(artifacts.RunDir, "graph.svg")
		if err := os.WriteFile(artifacts.GraphSVG, svg, 0o644); err != nil {
			return model.ArtifactManifest{}, nil, model.GraphModel{}, 0, 0, err
		}
	}

	return artifacts, findings, g, validateDuration, graphDuration, nil
}

func buildPatchedRuntimeConfig(cfg model.ConfigModel, runtimeOpts *RuntimeOptions, runID string, ports model.RuntimePorts) (runtimeAdapter, model.ConfigModel, model.TestPlan, time.Duration, error) {
	patchStart := time.Now()
	engineAdapter, err := runtime.ResolveEngine(model.RuntimeEngine(runtimeOpts.Engine))
	if err != nil {
		return nil, model.ConfigModel{}, model.TestPlan{}, 0, &model.ExitError{Code: 3, Err: err}
	}

	captureEndpoint := engineAdapter.CaptureEndpoint(fmt.Sprintf("127.0.0.1:%d", ports.CaptureGRPC), ports.CaptureGRPC)
	patchedConfig, plan, err := patch.Build(cfg, patch.Options{
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
		InjectionPort:   ports.InjectionGRPC,
		CapturePort:     ports.CaptureGRPC,
		CaptureEndpoint: captureEndpoint,
		CaptureSamples:  runtimeOpts.CaptureSamples,
	})
	if err != nil {
		return nil, model.ConfigModel{}, model.TestPlan{}, 0, &model.ExitError{Code: 2, Err: err}
	}

	return engineAdapter, patchedConfig, plan, time.Since(patchStart), nil
}

type runtimeAdapter interface {
	Engine() model.RuntimeEngine
	RuntimeBackend() string
	CaptureEndpoint(address string, capturePort int) string
}

func maybeAttachDiff(result *model.RunResult, global *GlobalOptions, runtimeOpts *RuntimeOptions, includeDiff bool) error {
	if !includeDiff {
		return nil
	}
	if !hasDiffInputs(runtimeOpts) {
		result.Diff = diffing.Empty()
		return nil
	}

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
		return &model.ExitError{Code: 2, Err: err}
	}

	result.Diff = diffResult
	result.Timings["diff"] = time.Since(diffStart).String()
	return nil
}

func finishValidationFailure(result model.RunResult) (model.RunResult, error) {
	result.Status = "FAIL"
	result.Message = "validation failed before runtime execution"
	result.FinishedAt = time.Now().UTC()
	result.Timings["total"] = time.Since(result.StartedAt).String()
	if err := report.WriteBundle(result); err != nil {
		return model.RunResult{}, err
	}
	return result, nil
}

func executeRuntimeHarness(result model.RunResult, originalCfg model.ConfigModel, patchedConfig model.ConfigModel, envValues map[string]string, runtimeOpts *RuntimeOptions) (model.RunResult, error) {
	runtimeStart := time.Now()
	sink := capture.NewInMemorySink(result.RunID, result.Artifacts.CapturesDir, runtimeOpts.CaptureSamples)
	runner := runtime.NewRunner(runtime.Options{
		Engine:         result.Engine,
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
		result.FinishedAt = time.Now().UTC()
		result.Timings["runtime"] = time.Since(runtimeStart).String()
		result.Timings["total"] = time.Since(result.StartedAt).String()
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
	result.Timings["total"] = time.Since(result.StartedAt).String()
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
