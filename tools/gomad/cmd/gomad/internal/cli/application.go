package cli

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/runner"
	"go.temporal.io/server/tools/gomad/internal/runner/inspect"
)

// application holds the backend and host dependencies for one invocation.
type application struct {
	capabilities *backends.Capabilities
	command      func([]string, io.Writer, io.Writer) (int, bool)
	backend      func() (BackendInstallation, error)
	replayHook   func(context.Context, runner.ReplaySpec) (runner.ReplayResult, bool, error)
	// clusterInspection projects a backend's retained cluster evidence.
	clusterInspection func(string) (*inspect.ClusterInspection, error)
	executable        func() (string, error)
	environment       func(string) string
}

// installation holds the provider and Runner identity used by one operation.
type installation struct {
	replayCommand string
	backend       backends.Provider
	backendName   string
	toolchainRoot string
	executable    string
	runnerBuild   string
}

func hostApplication() application {
	return application{
		executable: os.Executable, environment: os.Getenv,
	}
}

// executablePath reports the running gomad executable as the host names it.
func (app application) executablePath() (string, error) {
	executable, err := app.executable()
	if err != nil {
		return "", fmt.Errorf("resolve gomad executable: %w", err)
	}
	return executable, nil
}

// install resolves the backend after command input validation.
func (app application) install(explicitToolchainRoot string) (installation, error) {
	if app.backend != nil {
		if explicitToolchainRoot != "" {
			return installation{}, fmt.Errorf("--toolchain-root is unavailable with an external backend installation")
		}
		resolved, err := app.backend()
		if err != nil {
			return installation{}, err
		}
		if resolved.Name == "" || resolved.Provider == nil {
			return installation{}, fmt.Errorf("external backend name and provider are required")
		}
		executable, err := app.executablePath()
		if err != nil {
			return installation{}, err
		}
		digest, err := digestRunner(executable)
		if err != nil {
			return installation{}, err
		}
		var quoted []string
		for _, argument := range resolved.ReplayCommand {
			quoted = append(quoted, quoteArgument(argument))
		}
		return installation{backend: resolved.Provider, backendName: resolved.Name, runnerBuild: string(digest), replayCommand: strings.Join(quoted, " ")}, nil
	}
	return installation{}, fmt.Errorf("backend installation is required")
}

// digestRunner is the Runner build identity of the gomad executable at path.
func digestRunner(path string) (_ record.SHA256, retErr error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open Runner: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil && retErr == nil {
			retErr = fmt.Errorf("close Runner: %w", err)
		}
	}()
	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("stat Runner: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return "", fmt.Errorf("Runner is not a regular executable: %s", path)
	}
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("hash Runner: %w", err)
	}
	return record.SHA256(fmt.Sprintf("sha256:%x", hasher.Sum(nil))), nil
}
