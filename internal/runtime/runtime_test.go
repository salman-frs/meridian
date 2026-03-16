package runtime

import (
	"os/exec"
	"reflect"
	"testing"
	"time"

	"github.com/salman-frs/meridian/internal/model"
)

func TestParseRunningState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{
			name:   "plain true",
			output: "true\n",
			want:   true,
		},
		{
			name:   "warning then true",
			output: "time=\"2026-03-13T18:30:25+01:00\" level=warning msg=\"failed to inspect NetNS\"\ntrue\n",
			want:   true,
		},
		{
			name:   "plain false",
			output: "false\n",
			want:   false,
		},
		{
			name:   "unexpected output",
			output: "warning only\n",
			want:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := parseRunningState([]byte(tt.output)); got != tt.want {
				t.Fatalf("parseRunningState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStartCollectorRetriesAfterEarlyExit(t *testing.T) {
	t.Parallel()

	current := time.Unix(0, 0)
	calls := []string{}
	runner := &Runner{
		options: Options{StartupTimeout: time.Second},
		adapter: fakeEngineAdapter{},
		now: func() time.Time {
			return current
		},
		sleep: func(d time.Duration) {
			current = current.Add(d)
		},
		runCmd: func(args ...string) ([]byte, error) {
			calls = append(calls, args[0])
			switch {
			case reflect.DeepEqual(args, []string{"run"}):
				runCount := 0
				for _, call := range calls {
					if call == "run" {
						runCount++
					}
				}
				if runCount == 1 {
					return []byte("cid-1\n"), nil
				}
				return []byte("cid-2\n"), nil
			case reflect.DeepEqual(args, []string{"logs", "cid-1"}):
				return []byte("boot failed"), nil
			case reflect.DeepEqual(args, []string{"inspect", "-f", "{{.State.Running}}", "cid-1"}):
				return []byte("false\n"), nil
			case reflect.DeepEqual(args, []string{"rm", "-f", "cid-1"}):
				return []byte(""), nil
			case reflect.DeepEqual(args, []string{"logs", "cid-2"}):
				return []byte("Serving"), nil
			default:
				t.Fatalf("unexpected command: %#v", args)
				return nil, nil
			}
		},
	}

	containerID, logs, err := runner.startCollector(RunRequest{})
	if err != nil {
		t.Fatalf("startCollector() error = %v", err)
	}
	if containerID != "cid-2" {
		t.Fatalf("startCollector() containerID = %q, want cid-2", containerID)
	}
	if string(logs) != "Serving" {
		t.Fatalf("startCollector() logs = %q, want Serving", string(logs))
	}
}

func TestWaitForCaptureReturnsWhenSignalsArrive(t *testing.T) {
	t.Parallel()

	current := time.Unix(0, 0)
	snapshots := 0
	runner := &Runner{
		options: Options{CaptureTimeout: time.Second},
		now: func() time.Time {
			return current
		},
		sleep: func(d time.Duration) {
			current = current.Add(d)
		},
	}

	got := runner.waitForCapture(func() []model.SignalCapture {
		snapshots++
		if snapshots < 2 {
			return []model.SignalCapture{{Signal: model.SignalTraces, Count: 0}}
		}
		return []model.SignalCapture{{Signal: model.SignalTraces, Count: 1}}
	}, []model.SignalType{model.SignalTraces})

	if len(got) != 1 || got[0].Count != 1 {
		t.Fatalf("waitForCapture() = %#v, want traces count 1", got)
	}
}

func TestCollectorReady(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		logs string
		want bool
	}{
		{name: "everything is ready", logs: "Everything is ready", want: true},
		{name: "starting", logs: "Starting collector", want: true},
		{name: "serving", logs: "Serving gRPC", want: true},
		{name: "not ready", logs: "retrying", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := collectorReady(tt.logs); got != tt.want {
				t.Fatalf("collectorReady() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewRunnerUsesResolvedEngine(t *testing.T) {
	t.Parallel()

	runner := NewRunner(Options{
		Engine:         model.RuntimeEngineAuto,
		ResolvedEngine: ResolvedEngine{adapter: fakeEngineAdapter{}},
	})

	if runner.adapter == nil {
		t.Fatal("NewRunner() adapter = nil, want resolved adapter")
	}
	if runner.adapter.Engine() != model.RuntimeEngineDocker {
		t.Fatalf("NewRunner() adapter.Engine() = %q", runner.adapter.Engine())
	}
}

type fakeEngineAdapter struct{}

func (fakeEngineAdapter) Engine() model.RuntimeEngine                            { return model.RuntimeEngineDocker }
func (fakeEngineAdapter) RuntimeBackend() string                                 { return "docker" }
func (fakeEngineAdapter) Command(args ...string) *exec.Cmd                       { return nil }
func (fakeEngineAdapter) CommandLabel() string                                   { return "fake" }
func (fakeEngineAdapter) Preflight() error                                       { return nil }
func (fakeEngineAdapter) CaptureEndpoint(address string, capturePort int) string { return address }
func (fakeEngineAdapter) RunArgs(req RunRequest) []string                        { return []string{"run"} }
