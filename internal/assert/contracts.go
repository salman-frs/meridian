package assert

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/salman-frs/meridian/internal/model"
)

func EvaluateContracts(items []model.NormalizedTelemetry, specs []model.ContractSpec, plan model.TestPlan) []model.ContractResult {
	results := make([]model.ContractResult, 0, len(specs))
	for _, spec := range specs {
		results = append(results, evaluateContract(items, spec, plan))
	}
	return results
}

func evaluateContract(items []model.NormalizedTelemetry, spec model.ContractSpec, plan model.TestPlan) model.ContractResult {
	matches := filterContractItems(items, spec)
	failures := []string{}
	count := len(matches)

	if spec.Expect.Exists != nil {
		if *spec.Expect.Exists && count == 0 {
			failures = append(failures, "expected at least one matching item, but found none")
		}
		if !*spec.Expect.Exists && count > 0 {
			failures = append(failures, fmt.Sprintf("expected no matching items, but found %d", count))
		}
	}
	if spec.Expect.ExactCount != nil && count != *spec.Expect.ExactCount {
		failures = append(failures, fmt.Sprintf("expected exactly %d matching items, but found %d", *spec.Expect.ExactCount, count))
	}
	if spec.Expect.MinCount > 0 && count < spec.Expect.MinCount {
		failures = append(failures, fmt.Sprintf("expected at least %d matching items, but found %d", spec.Expect.MinCount, count))
	}
	if spec.Expect.MaxCount != nil && count > *spec.Expect.MaxCount {
		failures = append(failures, fmt.Sprintf("expected at most %d matching items, but found %d", *spec.Expect.MaxCount, count))
	}
	if count > 0 {
		failures = append(failures, evaluatePresenceContracts(matches, spec.Expect)...)
		failures = append(failures, evaluateStringExpectations(matches, "equal", spec.Expect.Equals, func(actual string, expected string) bool {
			return actual == expected
		})...)
		failures = append(failures, evaluateStringExpectations(matches, "contain", spec.Expect.Contains, func(actual string, expected string) bool {
			return strings.Contains(actual, expected)
		})...)
		failures = append(failures, evaluateRegexExpectations(matches, spec.Expect.Regex)...)
		failures = append(failures, evaluateMetricExpectation(matches, spec.Expect.MetricValue)...)
	}

	result := model.ContractResult{
		ID:       spec.ID,
		Severity: spec.Severity,
		Signal:   spec.Signal,
		Fixture:  spec.Fixture,
		Status:   status(len(failures) == 0),
		Message:  "contract satisfied",
		Observed: fmt.Sprintf("%d matching item(s)", count),
		Expected: describeContractExpectations(spec.Expect),
		LikelyCauses: []string{
			"collector mutation or routing behavior changed after the config edit",
			"the selected fixture did not produce the output shape this contract expects",
		},
		NextSteps: []string{
			"inspect capture.normalized.json for the concrete post-processor output",
			"compare contracts.json or contracts.md with config.patched.yaml and diff.md",
			"adjust the contract only if the behavior change is intentional",
		},
	}
	if len(failures) > 0 {
		result.Message = "contract failed"
		result.Diff = failures
	}
	if result.Fixture == "" && len(plan.Fixtures) == 1 {
		result.Fixture = plan.Fixtures[0]
	}
	return result
}

func filterContractItems(items []model.NormalizedTelemetry, spec model.ContractSpec) []model.NormalizedTelemetry {
	out := make([]model.NormalizedTelemetry, 0, len(items))
	for _, item := range items {
		if item.Signal != spec.Signal {
			continue
		}
		if spec.Fixture != "" && item.Fixture != spec.Fixture {
			continue
		}
		if !matchesContractWhere(item, spec.Where) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func matchesContractWhere(item model.NormalizedTelemetry, where model.AssertionWhere) bool {
	for key, value := range where.Attributes {
		if !matchesNormalizedAttribute(item, key, value) {
			return false
		}
	}
	if where.SpanName != "" && item.SpanName != where.SpanName {
		return false
	}
	if where.MetricName != "" && item.MetricName != where.MetricName {
		return false
	}
	if where.Body != "" && !strings.Contains(item.Body, where.Body) {
		return false
	}
	return true
}

func evaluatePresenceContracts(items []model.NormalizedTelemetry, expect model.ContractExpect) []string {
	failures := []string{}
	for _, key := range expect.AttributesPresent {
		missing := 0
		for _, item := range items {
			if hasNormalizedAttribute(item, key) {
				continue
			}
			missing++
		}
		if missing > 0 {
			failures = append(failures, fmt.Sprintf("%q should be present in every matched item, but was missing in %d item(s)", key, missing))
		}
	}
	for _, key := range expect.AttributesAbsent {
		present := 0
		for _, item := range items {
			if !hasNormalizedAttribute(item, key) {
				continue
			}
			present++
		}
		if present > 0 {
			failures = append(failures, fmt.Sprintf("%q should be absent from every matched item, but was present in %d item(s)", key, present))
		}
	}
	return failures
}

func evaluateStringExpectations(items []model.NormalizedTelemetry, verb string, expectations map[string]string, compare func(actual string, expected string) bool) []string {
	failures := []string{}
	for field, expected := range expectations {
		failed := 0
		for _, item := range items {
			actual, ok := normalizedField(item, field)
			if !ok || !compare(actual, expected) {
				failed++
			}
		}
		if failed > 0 {
			failures = append(failures, fmt.Sprintf("%q should %s %q in every matched item, but %d item(s) did not", field, verb, expected, failed))
		}
	}
	return failures
}

func evaluateRegexExpectations(items []model.NormalizedTelemetry, expectations map[string]string) []string {
	failures := []string{}
	for field, pattern := range expectations {
		re, err := regexp.Compile(pattern)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%q has invalid regex %q: %v", field, pattern, err))
			continue
		}
		failed := 0
		for _, item := range items {
			actual, ok := normalizedField(item, field)
			if !ok || !re.MatchString(actual) {
				failed++
			}
		}
		if failed > 0 {
			failures = append(failures, fmt.Sprintf("%q should match regex %q in every matched item, but %d item(s) did not", field, pattern, failed))
		}
	}
	return failures
}

func evaluateMetricExpectation(items []model.NormalizedTelemetry, expect model.NumericExpectation) []string {
	if expect == (model.NumericExpectation{}) {
		return nil
	}
	failures := []string{}
	failed := 0
	for _, item := range items {
		if item.MetricValue == nil || !matchesNumericExpectation(*item.MetricValue, expect) {
			failed++
		}
	}
	if failed > 0 {
		failures = append(failures, fmt.Sprintf("metric_value should satisfy %s in every matched item, but %d item(s) did not", describeNumericExpectation(expect), failed))
	}
	return failures
}

func matchesNumericExpectation(value float64, expect model.NumericExpectation) bool {
	if expect.Eq != nil && value != *expect.Eq {
		return false
	}
	if expect.Gt != nil && value <= *expect.Gt {
		return false
	}
	if expect.Gte != nil && value < *expect.Gte {
		return false
	}
	if expect.Lt != nil && value >= *expect.Lt {
		return false
	}
	if expect.Lte != nil && value > *expect.Lte {
		return false
	}
	return true
}

func normalizedField(item model.NormalizedTelemetry, field string) (string, bool) {
	switch {
	case field == "fixture":
		return item.Fixture, item.Fixture != ""
	case field == "run_id":
		return item.RunID, item.RunID != ""
	case field == "span_name":
		return item.SpanName, item.SpanName != ""
	case field == "metric_name":
		return item.MetricName, item.MetricName != ""
	case field == "body":
		return item.Body, item.Body != ""
	case field == "metric_value":
		if item.MetricValue == nil {
			return "", false
		}
		return strconv.FormatFloat(*item.MetricValue, 'f', -1, 64), true
	case strings.HasPrefix(field, "attributes."):
		value, ok := item.Attributes[strings.TrimPrefix(field, "attributes.")]
		return fmt.Sprint(value), ok
	case strings.HasPrefix(field, "resource."):
		value, ok := item.Resource[strings.TrimPrefix(field, "resource.")]
		return fmt.Sprint(value), ok
	default:
		return "", false
	}
}

func matchesNormalizedAttribute(item model.NormalizedTelemetry, key string, expected string) bool {
	for _, field := range []string{key, "attributes." + key, "resource." + key} {
		actual, ok := normalizedField(item, field)
		if ok && actual == expected {
			return true
		}
	}
	return false
}

func hasNormalizedAttribute(item model.NormalizedTelemetry, key string) bool {
	for _, field := range []string{key, "attributes." + key, "resource." + key} {
		if _, ok := normalizedField(item, field); ok {
			return true
		}
	}
	return false
}

func describeContractExpectations(expect model.ContractExpect) string {
	parts := []string{}
	if expect.Exists != nil {
		if *expect.Exists {
			parts = append(parts, "exists")
		} else {
			parts = append(parts, "does not exist")
		}
	}
	if expect.ExactCount != nil {
		parts = append(parts, fmt.Sprintf("exact_count=%d", *expect.ExactCount))
	}
	if expect.MinCount > 0 {
		parts = append(parts, fmt.Sprintf("min_count=%d", expect.MinCount))
	}
	if expect.MaxCount != nil {
		parts = append(parts, fmt.Sprintf("max_count=%d", *expect.MaxCount))
	}
	if len(expect.AttributesPresent) > 0 {
		parts = append(parts, "attributes present")
	}
	if len(expect.AttributesAbsent) > 0 {
		parts = append(parts, "attributes absent")
	}
	if len(expect.Equals) > 0 {
		parts = append(parts, "equals checks")
	}
	if len(expect.Contains) > 0 {
		parts = append(parts, "contains checks")
	}
	if len(expect.Regex) > 0 {
		parts = append(parts, "regex checks")
	}
	if expect.MetricValue != (model.NumericExpectation{}) {
		parts = append(parts, "metric_value "+describeNumericExpectation(expect.MetricValue))
	}
	if len(parts) == 0 {
		return "no explicit expectations"
	}
	return strings.Join(parts, ", ")
}

func describeNumericExpectation(expect model.NumericExpectation) string {
	parts := []string{}
	if expect.Eq != nil {
		parts = append(parts, fmt.Sprintf("= %s", strconv.FormatFloat(*expect.Eq, 'f', -1, 64)))
	}
	if expect.Gt != nil {
		parts = append(parts, fmt.Sprintf("> %s", strconv.FormatFloat(*expect.Gt, 'f', -1, 64)))
	}
	if expect.Gte != nil {
		parts = append(parts, fmt.Sprintf(">= %s", strconv.FormatFloat(*expect.Gte, 'f', -1, 64)))
	}
	if expect.Lt != nil {
		parts = append(parts, fmt.Sprintf("< %s", strconv.FormatFloat(*expect.Lt, 'f', -1, 64)))
	}
	if expect.Lte != nil {
		parts = append(parts, fmt.Sprintf("<= %s", strconv.FormatFloat(*expect.Lte, 'f', -1, 64)))
	}
	return strings.Join(parts, ", ")
}

func applyContractDefaults(spec model.ContractSpec, defaults model.AssertionDefaults, fixtures []string, runID string) model.ContractSpec {
	if spec.Severity == "" {
		spec.Severity = model.SeverityFail
	}
	if spec.Fixture == "" && len(fixtures) == 1 {
		spec.Fixture = fixtures[0]
	}
	if spec.Expect.Exists == nil && spec.Expect.MinCount == 0 && spec.Expect.ExactCount == nil && spec.Expect.MaxCount == nil && defaults.MinCount > 0 {
		spec.Expect.MinCount = defaults.MinCount
	}
	replacer := strings.NewReplacer("{{RUN_ID}}", runID)
	spec.ID = replacer.Replace(spec.ID)
	spec.Fixture = replacer.Replace(spec.Fixture)
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
	for key, value := range spec.Expect.Equals {
		spec.Expect.Equals[key] = replacer.Replace(value)
	}
	for key, value := range spec.Expect.Contains {
		spec.Expect.Contains[key] = replacer.Replace(value)
	}
	for key, value := range spec.Expect.Regex {
		spec.Expect.Regex[key] = replacer.Replace(value)
	}
	return spec
}
