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
		return &Runner{options: options}
	}
	return &Runner{options: options, adapter: adapter}
}

func (r *Runner) Run(req RunRequest) (RunResult, error) {
	if r.adapter == nil {
		adapter, err := ResolveEngine(r.options.Engine)
		if err != nil {
			return RunResult{}, &model.ExitError{Code: 3, Err: err}
		}
		r.adapter = adapter
	}
	if err := r.adapter.Preflight(); err != nil {
		return RunResult{}, &model.ExitError{Code: 3, Err: err}
	}
	address, err := req.CaptureSink.Start(req.Plan.CapturePort)
	if err != nil {
		return RunResult{}, &model.ExitError{Code: 3, Err: fmt.Errorf("failed to start capture sink: %w", err)}
	}
	defer req.CaptureSink.Stop()

	req.Plan.Engine = r.adapter.Engine()
	req.Plan.RuntimeBackend = r.adapter.RuntimeBackend()
	req.Plan.CaptureEndpoint = r.adapter.CaptureEndpoint(address, req.Plan.CapturePort)
	req.Plan.InjectionEndpoint = fmt.Sprintf("127.0.0.1:%d", req.Plan.InjectionPort)

	containerID, logs, err := r.startCollector(req)
	if err != nil {
		if len(logs) > 0 {
			_ = os.WriteFile(req.Artifacts.CollectorLog, logs, 0o644)
		}
		return RunResult{}, err
	}
	defer func() {
		if r.options.KeepContainers || containerID == "" {
			return
		}
		_, _ = r.adapter.Command("rm", "-f", containerID).CombinedOutput()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), r.options.InjectTimeout)
	defer cancel()
	gen := generator.New(req.Plan.InjectionEndpoint, req.Seed)
	req.Plan.InjectedAt = time.Now().UTC()
	if err := gen.Send(ctx, req.Plan); err != nil {
		return RunResult{}, &model.ExitError{Code: 3, Err: fmt.Errorf("failed to inject telemetry: %w", err)}
	}

	captures := r.waitForCapture(req)
	if err := req.CaptureSink.Persist(); err != nil {
		return RunResult{}, err
	}
	customAssertions, err := assert.LoadCustomAssertions(req.Assertions, req.Plan.RunID)
	if err != nil {
		return RunResult{}, &model.ExitError{Code: 2, Err: fmt.Errorf("failed to load custom assertions: %w", err)}
	}

	logs, _ = r.adapter.Command("logs", containerID).CombinedOutput()
	if err := os.WriteFile(req.Artifacts.CollectorLog, logs, 0o644); err != nil {
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

func (r *Runner) startCollector(req RunRequest) (string, []byte, error) {
	var lastErr error
	var lastLogs []byte
	for attempt := 0; attempt < 2; attempt++ {
		containerID, logs, ready, exitedEarly, err := r.startCollectorAttempt(req)
		if err == nil && ready {
			return containerID, logs, nil
		}
		if containerID != "" && !r.options.KeepContainers {
			_, _ = r.adapter.Command("rm", "-f", containerID).CombinedOutput()
		}
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
	output, err := r.adapter.Command(runArgs...).CombinedOutput()
	if err != nil {
		return "", output, false, false, &model.ExitError{Code: 3, Err: fmt.Errorf("failed to start collector container with %s via %s: %s", r.adapter.Engine(), r.adapter.CommandLabel(), trimOutput(output))}
	}
	containerID := strings.TrimSpace(string(output))
	deadline := time.Now().Add(r.options.StartupTimeout)
	for time.Now().Before(deadline) {
		logs, _ := r.adapter.Command("logs", containerID).CombinedOutput()
		text := string(logs)
		if strings.Contains(text, "Everything is ready") || strings.Contains(text, "Starting") || strings.Contains(text, "Serving") {
			return containerID, logs, true, false, nil
		}
		if !r.containerRunning(containerID) {
			return containerID, logs, false, true, &model.ExitError{Code: 3, Err: fmt.Errorf("collector exited before it became ready: %s", strings.TrimSpace(text))}
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !r.containerRunning(containerID) {
		logs, _ := r.adapter.Command("logs", containerID).CombinedOutput()
		return containerID, logs, false, true, &model.ExitError{Code: 3, Err: fmt.Errorf("collector exited before startup timeout: %s", strings.TrimSpace(string(logs)))}
	}
	logs, _ := r.adapter.Command("logs", containerID).CombinedOutput()
	return containerID, logs, true, false, nil
}

func firstPath(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func (r *Runner) waitForCapture(req RunRequest) []model.SignalCapture {
	deadline := time.Now().Add(r.options.CaptureTimeout)
	for {
		captures := req.CaptureSink.Snapshot()
		if allSignalsCaptured(captures, req.Plan.Signals) || time.Now().After(deadline) {
			return captures
		}
		time.Sleep(200 * time.Millisecond)
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
	output, err := r.adapter.Command("inspect", "-f", "{{.State.Running}}", containerID).CombinedOutput()
	if err != nil {
		return false
	}
	return parseRunningState(output)
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
