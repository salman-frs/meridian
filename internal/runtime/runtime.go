package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/salman-frs/meridian/internal/assert"
	"github.com/salman-frs/meridian/internal/capture"
	"github.com/salman-frs/meridian/internal/generator"
	"github.com/salman-frs/meridian/internal/model"
)

type Options struct {
	Engine         model.RuntimeEngine
	ResolvedEngine ResolvedEngine
	CollectorImage string
	Timeout        time.Duration
	StartupTimeout time.Duration
	InjectTimeout  time.Duration
	CaptureTimeout time.Duration
	KeepContainers bool
}

type Runner struct {
	options Options
	adapter engineAdapter
	now     func() time.Time
	sleep   func(time.Duration)
	runCmd  func(args ...string) ([]byte, error)
}

type RunRequest struct {
	Config      model.ConfigModel
	Original    model.ConfigModel
	Plan        model.TestPlan
	Artifacts   model.ArtifactManifest
	Seed        int64
	Assertions  string
	CaptureSink *capture.InMemorySink
	Env         map[string]string
}

type RunResult struct {
	Plan             model.TestPlan
	Captures         []model.SignalCapture
	CustomAssertions []model.AssertionSpec
	Contracts        []model.ContractSpec
	ContainerID      string
	ReproCommand     string
}

func NewRunner(options Options) *Runner {
	adapter := options.ResolvedEngine.adapterOrNil()
	if adapter == nil {
		adapter, _ = resolveEngine(options.Engine)
	}
	return &Runner{
		options: options,
		adapter: adapter,
		now:     time.Now,
		sleep:   time.Sleep,
	}
}

func (r *Runner) Run(req RunRequest) (RunResult, error) {
	if err := r.ensureAdapter(); err != nil {
		return RunResult{}, err
	}
	if err := r.adapter.Preflight(); err != nil {
		return RunResult{}, &model.ExitError{Code: 3, Err: err}
	}

	address, err := req.CaptureSink.Start(req.Plan.CapturePort)
	if err != nil {
		return RunResult{}, &model.ExitError{Code: 3, Err: fmt.Errorf("failed to start capture sink: %w", err)}
	}
	defer req.CaptureSink.Stop()

	suite, err := r.loadAssertionSuite(req.Assertions, req.Plan.RunID)
	if err != nil {
		return RunResult{}, err
	}
	req.Plan = r.configurePlan(req.Plan, address)
	if len(suite.Fixtures) > 0 {
		req.Plan.Fixtures = append([]string{}, suite.Fixtures...)
	}

	containerID, logs, err := r.startCollector(req)
	if err != nil {
		_ = r.persistCollectorLogs(req.Artifacts.CollectorLog, logs)
		return RunResult{}, err
	}
	defer r.cleanupContainer(containerID)

	req.Plan.InjectedAt = r.now().UTC()
	if err := r.injectTelemetry(req.Plan, req.Seed); err != nil {
		return RunResult{}, err
	}

	captures := r.waitForCapture(req.CaptureSink.Snapshot, req.Plan.Signals)
	if err := req.CaptureSink.Persist(); err != nil {
		return RunResult{}, err
	}
	if err := req.CaptureSink.PersistNormalized(req.Artifacts.CaptureNormalizedJSON); err != nil {
		return RunResult{}, err
	}

	logs = r.collectorLogs(containerID)
	if err := r.persistCollectorLogs(req.Artifacts.CollectorLog, logs); err != nil {
		return RunResult{}, err
	}

	return RunResult{
		Plan:             req.Plan,
		Captures:         captures,
		CustomAssertions: suite.Assertions,
		Contracts:        suite.Contracts,
		ContainerID:      containerID,
		ReproCommand:     reproCommand(req),
	}, nil
}

func (r *Runner) ensureAdapter() error {
	if r.adapter != nil {
		return nil
	}
	adapter, err := resolveEngine(r.options.Engine)
	if err != nil {
		return &model.ExitError{Code: 3, Err: err}
	}
	r.adapter = adapter
	return nil
}

func (r *Runner) configurePlan(plan model.TestPlan, address string) model.TestPlan {
	plan.Engine = r.adapter.Engine()
	plan.RuntimeBackend = r.adapter.RuntimeBackend()
	plan.CaptureEndpoint = r.adapter.CaptureEndpoint(address, plan.CapturePort)
	plan.InjectionEndpoint = fmt.Sprintf("127.0.0.1:%d", plan.InjectionPort)
	return plan
}

func (r *Runner) persistCollectorLogs(path string, logs []byte) error {
	if len(logs) == 0 {
		return nil
	}
	return os.WriteFile(path, logs, 0o644)
}

func (r *Runner) injectTelemetry(plan model.TestPlan, seed int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), r.options.InjectTimeout)
	defer cancel()

	gen := generator.New(plan.InjectionEndpoint, seed)
	if err := gen.Send(ctx, plan); err != nil {
		return &model.ExitError{Code: 3, Err: fmt.Errorf("failed to inject telemetry: %w", err)}
	}
	return nil
}

func (r *Runner) loadAssertionSuite(path string, runID string) (model.AssertionFile, error) {
	suite, err := assert.LoadSuite(path, runID)
	if err != nil {
		return model.AssertionFile{}, &model.ExitError{Code: 2, Err: fmt.Errorf("failed to load assertions or contracts: %w", err)}
	}
	return suite, nil
}

func firstPath(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func collectorReady(logs string) bool {
	return strings.Contains(logs, "Everything is ready") ||
		strings.Contains(logs, "Starting") ||
		strings.Contains(logs, "Serving")
}

func trimOutput(output []byte) string {
	text := strings.TrimSpace(string(output))
	if text == "" {
		return errors.New("no command output").Error()
	}
	return text
}
