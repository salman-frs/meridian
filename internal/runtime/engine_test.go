package runtime

import (
	"errors"
	"reflect"
	"testing"

	"github.com/salman-frs/meridian/internal/model"
)

func TestResolveEngineAutoPrefersDocker(t *testing.T) {
	prevLookup := lookupPath
	prevGOOS := currentGOOS
	t.Cleanup(func() {
		lookupPath = prevLookup
		currentGOOS = prevGOOS
	})

	currentGOOS = "linux"
	lookupPath = func(file string) (string, error) {
		switch file {
		case "docker":
			return "/usr/bin/docker", nil
		case "nerdctl":
			return "/usr/bin/nerdctl", nil
		default:
			return "", errors.New("missing")
		}
	}

	adapter, err := ResolveEngine(model.RuntimeEngineAuto)
	if err != nil {
		t.Fatalf("ResolveEngine(auto) returned error: %v", err)
	}
	if got := adapter.Engine(); got != model.RuntimeEngineDocker {
		t.Fatalf("ResolveEngine(auto) picked %q, want %q", got, model.RuntimeEngineDocker)
	}
}

func TestResolveEngineAutoFallsBackToContainerdOnLinux(t *testing.T) {
	prevLookup := lookupPath
	prevGOOS := currentGOOS
	t.Cleanup(func() {
		lookupPath = prevLookup
		currentGOOS = prevGOOS
	})

	currentGOOS = "linux"
	lookupPath = func(file string) (string, error) {
		switch file {
		case "docker":
			return "", errors.New("missing")
		case "nerdctl":
			return "/usr/bin/nerdctl", nil
		default:
			return "", errors.New("missing")
		}
	}

	adapter, err := ResolveEngine(model.RuntimeEngineAuto)
	if err != nil {
		t.Fatalf("ResolveEngine(auto) returned error: %v", err)
	}
	if got := adapter.Engine(); got != model.RuntimeEngineContainerd {
		t.Fatalf("ResolveEngine(auto) picked %q, want %q", got, model.RuntimeEngineContainerd)
	}
	if got := adapter.RuntimeBackend(); got != "nerdctl" {
		t.Fatalf("ResolveEngine(auto) backend = %q, want nerdctl", got)
	}
}

func TestResolveEngineAutoFallsBackToContainerdViaLimaOnDarwin(t *testing.T) {
	prevLookup := lookupPath
	prevGOOS := currentGOOS
	t.Cleanup(func() {
		lookupPath = prevLookup
		currentGOOS = prevGOOS
	})

	currentGOOS = "darwin"
	lookupPath = func(file string) (string, error) {
		switch file {
		case "docker":
			return "", errors.New("missing")
		case "lima":
			return "/opt/homebrew/bin/lima", nil
		default:
			return "", errors.New("missing")
		}
	}

	adapter, err := ResolveEngine(model.RuntimeEngineAuto)
	if err != nil {
		t.Fatalf("ResolveEngine(auto) returned error: %v", err)
	}
	if got := adapter.Engine(); got != model.RuntimeEngineContainerd {
		t.Fatalf("ResolveEngine(auto) picked %q, want %q", got, model.RuntimeEngineContainerd)
	}
	if got := adapter.RuntimeBackend(); got != "lima nerdctl" {
		t.Fatalf("ResolveEngine(auto) backend = %q, want lima nerdctl", got)
	}
}

func TestResolveEngineContainerdUsesLimaOnDarwin(t *testing.T) {
	prevLookup := lookupPath
	prevGOOS := currentGOOS
	t.Cleanup(func() {
		lookupPath = prevLookup
		currentGOOS = prevGOOS
	})

	currentGOOS = "darwin"
	lookupPath = func(file string) (string, error) {
		if file == "lima" {
			return "/opt/homebrew/bin/lima", nil
		}
		return "", errors.New("missing")
	}

	adapter, err := ResolveEngine(model.RuntimeEngineContainerd)
	if err != nil {
		t.Fatalf("ResolveEngine(containerd) returned error: %v", err)
	}
	if got := adapter.RuntimeBackend(); got != "lima nerdctl" {
		t.Fatalf("ResolveEngine(containerd) backend = %q, want lima nerdctl", got)
	}
}

func TestResolveEngineContainerdOnDarwinRequiresLima(t *testing.T) {
	prevLookup := lookupPath
	prevGOOS := currentGOOS
	t.Cleanup(func() {
		lookupPath = prevLookup
		currentGOOS = prevGOOS
	})

	currentGOOS = "darwin"
	lookupPath = func(file string) (string, error) {
		return "", errors.New("missing")
	}

	if _, err := ResolveEngine(model.RuntimeEngineContainerd); err == nil {
		t.Fatal("ResolveEngine(containerd) succeeded on darwin without Lima, want error")
	}
}

func TestDockerRunArgs(t *testing.T) {
	adapter := newDockerAdapter()
	req := RunRequest{
		Plan: model.TestPlan{
			RunID:          "run-123",
			CollectorImage: "otel/test:latest",
			InjectionPort:  4317,
		},
		Artifacts: model.ArtifactManifest{
			PatchedConfig: "/tmp/config.yaml",
		},
		Env: map[string]string{
			"B": "2",
			"A": "1",
		},
	}

	got := adapter.RunArgs(req)
	want := []string{
		"run", "-d",
		"--name", "meridian-run-123",
		"-p", "4317:4317",
		"--add-host", "host.docker.internal:host-gateway",
		"-v", "/tmp/config.yaml:/etc/meridian/config.yaml:ro",
		"-e", "A=1",
		"-e", "B=2",
		"otel/test:latest",
		"--config=/etc/meridian/config.yaml",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RunArgs mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestLinuxContainerdRunArgsUseHostNetwork(t *testing.T) {
	adapter := newLinuxContainerdAdapter()
	req := RunRequest{
		Plan: model.TestPlan{
			RunID:          "run-123",
			CollectorImage: "otel/test:latest",
			InjectionPort:  4317,
		},
		Artifacts: model.ArtifactManifest{
			PatchedConfig: "/tmp/config.yaml",
		},
	}

	got := adapter.RunArgs(req)
	wantPrefix := []string{"run", "-d", "--name", "meridian-run-123", "--network", "host"}
	if !reflect.DeepEqual(got[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("RunArgs prefix mismatch\n got: %#v\nwant: %#v", got[:len(wantPrefix)], wantPrefix)
	}
}

func TestDarwinContainerdRunArgsUsePortPublishing(t *testing.T) {
	adapter := newDarwinContainerdAdapter()
	req := RunRequest{
		Plan: model.TestPlan{
			RunID:          "run-123",
			CollectorImage: "otel/test:latest",
			InjectionPort:  4317,
		},
		Artifacts: model.ArtifactManifest{
			PatchedConfig: "/tmp/config.yaml",
		},
	}

	got := adapter.RunArgs(req)
	wantPrefix := []string{"run", "-d", "--name", "meridian-run-123", "-p", "4317:4317"}
	if !reflect.DeepEqual(got[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("RunArgs prefix mismatch\n got: %#v\nwant: %#v", got[:len(wantPrefix)], wantPrefix)
	}
	if endpoint := adapter.CaptureEndpoint("127.0.0.1:4318", 4318); endpoint != "host.lima.internal:4318" {
		t.Fatalf("CaptureEndpoint = %q, want host.lima.internal:4318", endpoint)
	}
}
