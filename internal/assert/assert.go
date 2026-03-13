package assert

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/salman-frs/meridian/internal/capture"
	"github.com/salman-frs/meridian/internal/model"
	"gopkg.in/yaml.v3"
)

func Evaluate(sink *capture.InMemorySink, captures []model.SignalCapture, custom []model.AssertionSpec, plan model.TestPlan) []model.AssertionResult {
	_ = sink
	results := []model.AssertionResult{}
	for _, signal := range plan.Signals {
		capture := findCapture(captures, signal)
		results = append(results, model.AssertionResult{
			ID:       string(signal) + "-received",
			Severity: model.SeverityFail,
			Signal:   signal,
			Status:   status(capture.Count >= 1),
			Message:  "received at least one item",
			Observed: countString(capture.Count),
			Expected: ">= 1",
			Duration: 0,
			LikelyCauses: []string{
				"collector pipeline dropped all telemetry",
				"collector receiver failed to bind or start",
				"patched exporter could not reach Meridian capture sink",
			},
			NextSteps: []string{
				"inspect collector.log for receiver and exporter errors",
				"open config.patched.yaml to confirm injected receiver and exporter wiring",
				"rerun with --keep-containers for deeper investigation",
			},
		})
		results = append(results, model.AssertionResult{
			ID:       string(signal) + "-received-within",
			Severity: model.SeverityFail,
			Signal:   signal,
			Status:   status(receivedWithinWindow(capture, plan)),
			Message:  "received telemetry within the configured capture timeout",
			Observed: observedWindow(capture, plan),
			Expected: "<= " + plan.CaptureTimeout,
			Duration: 0,
		})
		runIDPresent := false
		for _, sample := range capture.Samples {
			if sample["run_id"] == plan.RunID {
				runIDPresent = true
				break
			}
		}
		results = append(results, model.AssertionResult{
			ID:       string(signal) + "-run-id",
			Severity: model.SeverityFail,
			Signal:   signal,
			Status:   status(runIDPresent),
			Message:  "received telemetry is correlated with the current run id",
			Observed: observedRunID(runIDPresent, plan.RunID),
			Expected: plan.RunID,
		})
		results = append(results, model.AssertionResult{
			ID:       string(signal) + "-decode-errors",
			Severity: model.SeverityFail,
			Signal:   signal,
			Status:   status(len(capture.Errors) == 0),
			Message:  "capture sink decoded the payload without errors",
			Observed: countString(len(capture.Errors)),
			Expected: "0",
		})
	}

	for _, spec := range custom {
		results = append(results, evaluateCustom(spec, captures, plan.RunID))
	}
	return results
}

func LoadCustomAssertions(path string, runID string) ([]model.AssertionSpec, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file model.AssertionFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	assertions := make([]model.AssertionSpec, 0, len(file.Assertions))
	for _, spec := range file.Assertions {
		assertions = append(assertions, applyDefaults(spec, file.Defaults, runID))
	}
	return assertions, nil
}

func evaluateCustom(spec model.AssertionSpec, captures []model.SignalCapture, runID string) model.AssertionResult {
	start := time.Now()
	capture := findCapture(captures, spec.Signal)
	matches := 0
	for _, sample := range capture.Samples {
		if matchesSample(sample, spec) {
			matches++
		}
	}
	pass := true
	expected := "custom assertion"
	if spec.Expect.MinCount > 0 {
		pass = matches >= spec.Expect.MinCount
		expected = ">=" + countString(spec.Expect.MinCount)
	}
	if spec.Expect.Exists != nil {
		pass = (matches > 0) == *spec.Expect.Exists
		if *spec.Expect.Exists {
			expected = "at least one match"
		} else {
			expected = "no matches"
		}
	}
	return model.AssertionResult{
		ID:       spec.ID,
		Severity: spec.Severity,
		Signal:   spec.Signal,
		Status:   status(pass),
		Message:  "custom assertion evaluated",
		Observed: countString(matches),
		Expected: expected,
		Duration: time.Since(start),
		LikelyCauses: []string{
			"processor mutation or routing behavior changed",
			"the filter condition in assertions.yaml no longer matches the output",
		},
		NextSteps: []string{
			"inspect capture samples under captures/",
			"compare the assertion filter against config.patched.yaml and graph.mmd",
			"update assertions only if the behavior change is intentional",
		},
	}
}

func matchesSample(sample map[string]any, spec model.AssertionSpec) bool {
	for key, value := range spec.Where.Attributes {
		if !matchesAttribute(sample, key, value) {
			return false
		}
	}
	if spec.Where.SpanName != "" && sample["span_name"] != spec.Where.SpanName {
		return false
	}
	if spec.Where.MetricName != "" && sample["metric_name"] != spec.Where.MetricName {
		return false
	}
	if spec.Where.Body != "" && !strings.Contains(toString(sample["body"]), spec.Where.Body) {
		return false
	}
	if len(spec.Expect.AttributesPresent) > 0 {
		for _, key := range spec.Expect.AttributesPresent {
			if !hasAttribute(sample, key) {
				return false
			}
		}
	}
	if len(spec.Expect.AttributesAbsent) > 0 {
		for _, key := range spec.Expect.AttributesAbsent {
			if hasAttribute(sample, key) {
				return false
			}
		}
	}
	return true
}

func matchesAttribute(sample map[string]any, key string, expected string) bool {
	for _, field := range []string{"attributes", "resource"} {
		if attrs, ok := sample[field].(map[string]any); ok && toString(attrs[key]) == expected {
			return true
		}
	}
	return false
}

func hasAttribute(sample map[string]any, key string) bool {
	for _, field := range []string{"attributes", "resource"} {
		if attrs, ok := sample[field].(map[string]any); ok {
			if _, ok := attrs[key]; ok {
				return true
			}
		}
	}
	return false
}

func findCapture(captures []model.SignalCapture, signal model.SignalType) model.SignalCapture {
	for _, capture := range captures {
		if capture.Signal == signal {
			return capture
		}
	}
	return model.SignalCapture{Signal: signal}
}

func status(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}

func receivedWithinWindow(capture model.SignalCapture, plan model.TestPlan) bool {
	if capture.Count == 0 {
		return false
	}
	if capture.FirstReceivedAt.IsZero() || plan.InjectedAt.IsZero() {
		return true
	}
	window, err := time.ParseDuration(plan.CaptureTimeout)
	if err != nil {
		return true
	}
	return capture.FirstReceivedAt.Sub(plan.InjectedAt) <= window
}

func observedWindow(capture model.SignalCapture, plan model.TestPlan) string {
	if capture.FirstReceivedAt.IsZero() || plan.InjectedAt.IsZero() {
		return "not-recorded"
	}
	return capture.FirstReceivedAt.Sub(plan.InjectedAt).String()
}

func countString(count int) string {
	return strconv.Itoa(count)
}

func observedRunID(ok bool, runID string) string {
	if ok {
		return runID
	}
	return "missing"
}

func toString(v any) string {
	switch typed := v.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(v)
	}
}

func applyDefaults(spec model.AssertionSpec, defaults model.AssertionDefaults, runID string) model.AssertionSpec {
	if spec.Severity == "" {
		spec.Severity = model.SeverityFail
	}
	if spec.Expect.Exists == nil && spec.Expect.MinCount == 0 && defaults.MinCount > 0 {
		spec.Expect.MinCount = defaults.MinCount
	}
	replacer := strings.NewReplacer("{{RUN_ID}}", runID)
	spec.ID = replacer.Replace(spec.ID)
	spec.Where.SpanName = replacer.Replace(spec.Where.SpanName)
	spec.Where.MetricName = replacer.Replace(spec.Where.MetricName)
	spec.Where.Body = replacer.Replace(spec.Where.Body)
	for key, value := range spec.Where.Attributes {
		spec.Where.Attributes[key] = replacer.Replace(value)
	}
	for i, item := range spec.Expect.AttributesPresent {
		spec.Expect.AttributesPresent[i] = replacer.Replace(item)
	}
	for i, item := range spec.Expect.AttributesAbsent {
		spec.Expect.AttributesAbsent[i] = replacer.Replace(item)
	}
	return spec
}
