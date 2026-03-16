package assert

import (
	"testing"
	"time"

	"github.com/salman-frs/meridian/internal/model"
)

func TestEvaluateIncludesCustomAssertions(t *testing.T) {
	t.Parallel()

	plan := model.TestPlan{
		RunID:          "run-123",
		Signals:        []model.SignalType{model.SignalTraces},
		CaptureTimeout: "10s",
		InjectedAt:     time.Unix(0, 0).UTC(),
	}
	captures := []model.SignalCapture{
		{
			Signal:          model.SignalTraces,
			Count:           1,
			FirstReceivedAt: time.Unix(1, 0).UTC(),
			Samples: []map[string]any{
				{
					"run_id":    "run-123",
					"span_name": "checkout",
					"attributes": map[string]any{
						"service.name": "storefront",
					},
				},
			},
		},
	}

	results := Evaluate(nil, captures, []model.AssertionSpec{
		{
			ID:       "service-present",
			Severity: model.SeverityFail,
			Signal:   model.SignalTraces,
			Where: model.AssertionWhere{
				SpanName: "checkout",
			},
			Expect: model.AssertionExpect{
				MinCount:          1,
				AttributesPresent: []string{"service.name"},
			},
		},
	}, plan)

	if len(results) != 5 {
		t.Fatalf("Evaluate() len = %d, want 5", len(results))
	}
	if results[len(results)-1].Status != "PASS" {
		t.Fatalf("custom assertion status = %s, want PASS", results[len(results)-1].Status)
	}
}
