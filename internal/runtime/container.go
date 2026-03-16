package runtime

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/salman-frs/meridian/internal/model"
)

func (r *Runner) cleanupContainer(containerID string) {
	if r.options.KeepContainers || containerID == "" {
		return
	}
	_, _ = r.commandOutput("rm", "-f", containerID)
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
