package model

import "time"

type NormalizedTelemetry struct {
	Signal      SignalType     `json:"signal" yaml:"signal"`
	Fixture     string         `json:"fixture,omitempty" yaml:"fixture,omitempty"`
	RunID       string         `json:"run_id,omitempty" yaml:"run_id,omitempty"`
	SpanName    string         `json:"span_name,omitempty" yaml:"span_name,omitempty"`
	MetricName  string         `json:"metric_name,omitempty" yaml:"metric_name,omitempty"`
	Body        string         `json:"body,omitempty" yaml:"body,omitempty"`
	Resource    map[string]any `json:"resource,omitempty" yaml:"resource,omitempty"`
	Attributes  map[string]any `json:"attributes,omitempty" yaml:"attributes,omitempty"`
	MetricValue *float64       `json:"metric_value,omitempty" yaml:"metric_value,omitempty"`
	ReceivedAt  time.Time      `json:"received_at,omitempty" yaml:"received_at,omitempty"`
}

type ContractSpec struct {
	ID       string         `yaml:"id"`
	Severity Severity       `yaml:"severity"`
	Signal   SignalType     `yaml:"signal"`
	Fixture  string         `yaml:"fixture"`
	Where    AssertionWhere `yaml:"where"`
	Expect   ContractExpect `yaml:"expect"`
}

type ContractExpect struct {
	MinCount          int                `yaml:"min_count"`
	ExactCount        *int               `yaml:"exact_count"`
	MaxCount          *int               `yaml:"max_count"`
	Exists            *bool              `yaml:"exists"`
	AttributesPresent []string           `yaml:"attributes_present"`
	AttributesAbsent  []string           `yaml:"attributes_absent"`
	Equals            map[string]string  `yaml:"equals"`
	Contains          map[string]string  `yaml:"contains"`
	Regex             map[string]string  `yaml:"regex"`
	MetricValue       NumericExpectation `yaml:"metric_value"`
}

type NumericExpectation struct {
	Eq  *float64 `yaml:"eq"`
	Gt  *float64 `yaml:"gt"`
	Gte *float64 `yaml:"gte"`
	Lt  *float64 `yaml:"lt"`
	Lte *float64 `yaml:"lte"`
}

type ContractResult struct {
	ID           string     `json:"id" yaml:"id"`
	Severity     Severity   `json:"severity" yaml:"severity"`
	Signal       SignalType `json:"signal" yaml:"signal"`
	Fixture      string     `json:"fixture,omitempty" yaml:"fixture,omitempty"`
	Status       string     `json:"status" yaml:"status"`
	Message      string     `json:"message" yaml:"message"`
	Observed     string     `json:"observed,omitempty" yaml:"observed,omitempty"`
	Expected     string     `json:"expected,omitempty" yaml:"expected,omitempty"`
	Diff         []string   `json:"diff,omitempty" yaml:"diff,omitempty"`
	LikelyCauses []string   `json:"likely_causes,omitempty" yaml:"likely_causes,omitempty"`
	NextSteps    []string   `json:"next_steps,omitempty" yaml:"next_steps,omitempty"`
}
