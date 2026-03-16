package runtime

import (
	"errors"
	"slices"
	"strings"
)

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

func commandError(output []byte) string {
	text := strings.TrimSpace(string(output))
	if text == "" {
		return errors.New("no command output").Error()
	}
	return text
}
