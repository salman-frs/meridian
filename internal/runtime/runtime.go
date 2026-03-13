package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/salman-frs/meridian/internal/assert"
	"github.com/salman-frs/meridian/internal/capture"
	"github.com/salman-frs/meridian/internal/generator"
	"github.com/salman-frs/meridian/internal/model"
)

type Options struct {
	Engine         model.RuntimeEngine
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
	ContainerID      string
	ReproCommand     string
}

func NewRunner(options Options) *Runner {
	adapter, err := ResolveEngine(options.Engine)
	if err != nil {
		return &Runner{
			options: options,
			now:     time.Now,
			sleep:   time.Sleep,
		}
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

	req.Plan = r.configurePlan(req.Plan, address)

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
	customAssertions, err := r.loadCustomAssertions(req.Assertions, req.Plan.RunID)
	if err != nil {
		return RunResult{}, err
	}

	logs = r.collectorLogs(containerID)
	if err := r.persistCollectorLogs(req.Artifacts.CollectorLog, logs); err != nil {
		return RunResult{}, err
	}

	return RunResult{
		Plan:             req.Plan,
		Captures:         captures,
		CustomAssertions: customAssertions,
		ContainerID:      containerID,
		ReproCommand:     reproCommand(req),
	}, nil
}

func (r *Runner) ensureAdapter() error {
	if r.adapter != nil {
		return nil
	}
	adapter, err := ResolveEngine(r.options.Engine)
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

func (r *Runner) cleanupContainer(containerID string) {
	if r.options.KeepContainers || containerID == "" {
		return
	}
	_, _ = r.commandOutput("rm", "-f", containerID)
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

func (r *Runner) loadCustomAssertions(path string, runID string) ([]model.AssertionSpec, error) {
	customAssertions, err := assert.LoadCustomAssertions(path, runID)
	if err != nil {
		return nil, &model.ExitError{Code: 2, Err: fmt.Errorf("failed to load custom assertions: %w", err)}
	}
	return customAssertions, nil
}

func (r *Runner) startCollector(req RunRequest) (string, []byte, error) {
	var lastErr error
	var lastLogs []byte
	for attempt := 0; attempt < 2; attempt++ {
		containerID, logs, ready, exitedEarly, err := r.startCollectorAttempt(req)
		if err == nil && ready {
			return containerID, logs, nil
		}
		r.cleanupContainer(containerID)
		if err != nil {
			lastErr = err
			lastLogs = logs
		}
		if !exitedEarly {
			break
		}
	}
	if lastErr != nil {
		return "", lastLogs, lastErr
	}
	return "", lastLogs, &model.ExitError{Code: 3, Err: errors.New("collector did not become ready before startup timeout")}
}

func (r *Runner) startCollectorAttempt(req RunRequest) (string, []byte, bool, bool, error) {
	runArgs := r.adapter.RunArgs(req)
	output, err := r.commandOutput(runArgs...)
	if err != nil {
		return "", output, false, false, &model.ExitError{Code: 3, Err: fmt.Errorf("failed to start collector container with %s via %s: %s", r.adapter.Engine(), r.adapter.CommandLabel(), trimOutput(output))}
	}
	containerID := strings.TrimSpace(string(output))
	deadline := r.now().Add(r.options.StartupTimeout)
	for r.now().Before(deadline) {
		logs := r.collectorLogs(containerID)
		text := string(logs)
		if collectorReady(text) {
			return containerID, logs, true, false, nil
		}
		if !r.containerRunning(containerID) {
			return containerID, logs, false, true, &model.ExitError{Code: 3, Err: fmt.Errorf("collector exited before it became ready: %s", strings.TrimSpace(text))}
		}
		r.sleep(500 * time.Millisecond)
	}
	if !r.containerRunning(containerID) {
		logs := r.collectorLogs(containerID)
		return containerID, logs, false, true, &model.ExitError{Code: 3, Err: fmt.Errorf("collector exited before startup timeout: %s", strings.TrimSpace(string(logs)))}
	}
	logs := r.collectorLogs(containerID)
	return containerID, logs, true, false, nil
}

func firstPath(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func (r *Runner) waitForCapture(snapshot func() []model.SignalCapture, signals []model.SignalType) []model.SignalCapture {
	deadline := r.now().Add(r.options.CaptureTimeout)
	for {
		captures := snapshot()
		if allSignalsCaptured(captures, signals) || r.now().After(deadline) {
			return captures
		}
		r.sleep(200 * time.Millisecond)
	}
}

func allSignalsCaptured(captures []model.SignalCapture, signals []model.SignalType) bool {
	for _, signal := range signals {
		capture := model.SignalCapture{Signal: signal}
		for _, item := range captures {
			if item.Signal == signal {
				capture = item
				break
			}
		}
		if capture.Count < 1 {
			return false
		}
	}
	return true
}

func (r *Runner) containerRunning(containerID string) bool {
	output, err := r.commandOutput("inspect", "-f", "{{.State.Running}}", containerID)
	if err != nil {
		return false
	}
	return parseRunningState(output)
}

func (r *Runner) collectorLogs(containerID string) []byte {
	output, _ := r.commandOutput("logs", containerID)
	return output
}

func (r *Runner) commandOutput(args ...string) ([]byte, error) {
	if r.runCmd != nil {
		return r.runCmd(args...)
	}
	return r.adapter.Command(args...).CombinedOutput()
}

func collectorReady(logs string) bool {
	return strings.Contains(logs, "Everything is ready") ||
		strings.Contains(logs, "Starting") ||
		strings.Contains(logs, "Serving")
}

func reproCommand(req RunRequest) string {
	parts := []string{
		"meridian", "test",
		"-c", firstPath(req.Original.SourcePaths),
		"--engine=" + string(req.Plan.Engine),
		"--mode=" + string(req.Plan.Mode),
		"--collector-image", req.Plan.CollectorImage,
		"--timeout", req.Plan.Timeout,
		"--startup-timeout", req.Plan.StartupTimeout,
		"--inject-timeout", req.Plan.InjectTimeout,
		"--capture-timeout", req.Plan.CaptureTimeout,
		"--keep-containers",
	}
	if req.Assertions != "" {
		parts = append(parts, "--assertions", req.Assertions)
	}
	keys := make([]string, 0, len(req.Env))
	for key := range req.Env {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		parts = append(parts, "--env", key+"=<redacted>")
	}
	return strings.Join(parts, " ")
}

func sanitizeName(value string) string {
	replacer := strings.NewReplacer(":", "-", "/", "-", " ", "-", "@", "-", "=", "-", "+", "-", "%", "-")
	return replacer.Replace(value)
}

func trimOutput(output []byte) string {
	text := strings.TrimSpace(string(output))
	if text == "" {
		return errors.New("no command output").Error()
	}
	return text
}

func parseRunningState(output []byte) bool {
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		switch strings.TrimSpace(lines[i]) {
		case "true":
			return true
		case "false":
			return false
		}
	}
	return false
}
