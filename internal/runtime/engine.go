package runtime

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"slices"
	"strings"

	"github.com/salman-frs/meridian/internal/model"
)

type engineAdapter interface {
	Engine() model.RuntimeEngine
	RuntimeBackend() string
	Command(args ...string) *exec.Cmd
	CommandLabel() string
	Preflight() error
	CaptureEndpoint(address string, capturePort int) string
	RunArgs(req RunRequest) []string
}

type cliAdapter struct {
	engine          model.RuntimeEngine
	runtimeBackend  string
	binary          string
	prefix          []string
	hostNetwork     bool
	hostAlias       string
	extraRunArgs    []string
	versionArgs     []string
	preflightTarget string
}

var (
	lookupPath  = exec.LookPath
	currentGOOS = runtime.GOOS
)

func (a cliAdapter) Engine() model.RuntimeEngine {
	return a.engine
}

func (a cliAdapter) RuntimeBackend() string {
	return a.runtimeBackend
}

func (a cliAdapter) Command(args ...string) *exec.Cmd {
	fullArgs := append(slices.Clone(a.prefix), args...)
	return exec.Command(a.binary, fullArgs...)
}

func (a cliAdapter) CommandLabel() string {
	parts := append([]string{a.binary}, a.prefix...)
	return strings.Join(parts, " ")
}

func (a cliAdapter) Preflight() error {
	if _, err := lookupPath(a.binary); err != nil {
		return fmt.Errorf("%s is required for runtime tests", a.binary)
	}
	cmd := a.Command(a.versionArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		text := strings.TrimSpace(string(output))
		if text == "" {
			text = err.Error()
		}
		return fmt.Errorf("failed to contact the %s via %s: %s", a.preflightTarget, a.CommandLabel(), text)
	}
	return nil
}

func (a cliAdapter) CaptureEndpoint(address string, capturePort int) string {
	if a.hostNetwork {
		return fmt.Sprintf("127.0.0.1:%d", capturePort)
	}
	return strings.ReplaceAll(address, "127.0.0.1", a.hostAlias)
}

func (a cliAdapter) RunArgs(req RunRequest) []string {
	args := []string{
		"run", "-d",
		"--name", "meridian-" + sanitizeName(req.Plan.RunID),
	}
	if a.hostNetwork {
		args = append(args, "--network", "host")
	} else {
		args = append(args, "-p", fmt.Sprintf("%d:%d", req.Plan.InjectionPort, req.Plan.InjectionPort))
	}
	args = append(args, a.extraRunArgs...)
	args = append(args, "-v", req.Artifacts.PatchedConfig+":/etc/meridian/config.yaml:ro")
	keys := make([]string, 0, len(req.Env))
	for key := range req.Env {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		value := req.Env[key]
		args = append(args, "-e", key+"="+value)
	}
	args = append(args,
		req.Plan.CollectorImage,
		"--config=/etc/meridian/config.yaml",
	)
	return args
}

func resolveEngine(requested model.RuntimeEngine) (engineAdapter, error) {
	switch requested {
	case "", model.RuntimeEngineAuto:
		if _, err := lookupPath("docker"); err == nil {
			return newDockerAdapter(), nil
		}
		if currentGOOS == "linux" {
			if _, err := lookupPath("nerdctl"); err == nil {
				return newLinuxContainerdAdapter(), nil
			}
		}
		if currentGOOS == "darwin" {
			if _, err := lookupPath("lima"); err == nil {
				return newDarwinContainerdAdapter(), nil
			}
		}
		return nil, errors.New("no supported container engine found; install Docker, install nerdctl on Linux, or install Lima for containerd on macOS")
	case model.RuntimeEngineDocker:
		return newDockerAdapter(), nil
	case model.RuntimeEngineContainerd:
		switch currentGOOS {
		case "linux":
			return newLinuxContainerdAdapter(), nil
		case "darwin":
			if _, err := lookupPath("lima"); err != nil {
				return nil, errors.New("containerd on macOS requires Lima with its bundled nerdctl; install Lima and start a VM, or use --engine docker")
			}
			return newDarwinContainerdAdapter(), nil
		default:
			return nil, fmt.Errorf("containerd engine is not supported on %s", currentGOOS)
		}
	default:
		return nil, fmt.Errorf("unsupported --engine %q", requested)
	}
}

func newDockerAdapter() engineAdapter {
	return cliAdapter{
		engine:          model.RuntimeEngineDocker,
		runtimeBackend:  "docker",
		binary:          "docker",
		hostAlias:       "host.docker.internal",
		extraRunArgs:    []string{"--add-host", "host.docker.internal:host-gateway"},
		versionArgs:     []string{"version", "--format", "{{.Server.Version}}"},
		preflightTarget: "Docker daemon",
	}
}

func newLinuxContainerdAdapter() engineAdapter {
	return cliAdapter{
		engine:          model.RuntimeEngineContainerd,
		runtimeBackend:  "nerdctl",
		binary:          "nerdctl",
		hostNetwork:     true,
		versionArgs:     []string{"version"},
		preflightTarget: "containerd/nerdctl runtime",
	}
}

func newDarwinContainerdAdapter() engineAdapter {
	return cliAdapter{
		engine:          model.RuntimeEngineContainerd,
		runtimeBackend:  "lima nerdctl",
		binary:          "lima",
		prefix:          []string{"nerdctl"},
		hostAlias:       "host.lima.internal",
		versionArgs:     []string{"version"},
		preflightTarget: "Lima containerd/nerdctl runtime",
	}
}
