package assert

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/salman-frs/meridian/internal/model"
)

func TestLoadSuiteSupportsFixturesAndContracts(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "contracts.yaml")
	if err := os.WriteFile(path, []byte(`
version: 2
defaults:
  min_count: 1
fixtures:
  - redaction
contracts:
  - id: redact-{{RUN_ID}}
    severity: fail
    signal: traces
    expect:
      attributes_absent:
        - http.request.header.authorization
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	suite, err := LoadSuite(path, "run-123")
	if err != nil {
		t.Fatalf("LoadSuite() error = %v", err)
	}
	if len(suite.Fixtures) != 1 || suite.Fixtures[0] != "redaction" {
		t.Fatalf("LoadSuite() fixtures = %#v", suite.Fixtures)
	}
	if len(suite.Contracts) != 1 {
		t.Fatalf("LoadSuite() contracts = %d, want 1", len(suite.Contracts))
	}
	if suite.Contracts[0].ID != "redact-run-123" {
		t.Fatalf("contract ID = %q, want placeholder replaced", suite.Contracts[0].ID)
	}
	if suite.Contracts[0].Fixture != "redaction" {
		t.Fatalf("contract fixture = %q, want redaction", suite.Contracts[0].Fixture)
	}
	if suite.Contracts[0].Expect.MinCount != 1 {
		t.Fatalf("contract min_count = %d, want 1", suite.Contracts[0].Expect.MinCount)
	}
}

func TestEvaluateContractsReportsFailuresAndPasses(t *testing.T) {
	t.Parallel()

	items := []model.NormalizedTelemetry{
		{
			Signal:   model.SignalTraces,
			Fixture:  "redaction",
			RunID:    "run-123",
			SpanName: "meridian.synthetic.redaction",
			Attributes: map[string]any{
				"http.request.header.authorization": "Bearer secret",
			},
		},
		{
			Signal:     model.SignalMetrics,
			Fixture:    "metric-transform",
			RunID:      "run-123",
			MetricName: "meridian.synthetic.metric.transformed",
			MetricValue: func() *float64 {
				value := 2.0
				return &value
			}(),
		},
	}

	results := EvaluateContracts(items, []model.ContractSpec{
		{
			ID:       "redaction-contract",
			Severity: model.SeverityFail,
			Signal:   model.SignalTraces,
			Fixture:  "redaction",
			Expect: model.ContractExpect{
				MinCount:         1,
				AttributesAbsent: []string{"http.request.header.authorization"},
			},
		},
		{
			ID:       "metric-transform-contract",
			Severity: model.SeverityFail,
			Signal:   model.SignalMetrics,
			Fixture:  "metric-transform",
			Expect: model.ContractExpect{
				MinCount: 1,
				MetricValue: model.NumericExpectation{
					Gte: func() *float64 {
						value := 1.0
						return &value
					}(),
				},
				Equals: map[string]string{
					"metric_name": "meridian.synthetic.metric.transformed",
				},
			},
		},
	}, model.TestPlan{})

	if len(results) != 2 {
		t.Fatalf("EvaluateContracts() len = %d, want 2", len(results))
	}
	if results[0].Status != "FAIL" {
		t.Fatalf("redaction contract status = %s, want FAIL", results[0].Status)
	}
	if results[1].Status != "PASS" {
		t.Fatalf("metric contract status = %s, want PASS", results[1].Status)
	}
}
