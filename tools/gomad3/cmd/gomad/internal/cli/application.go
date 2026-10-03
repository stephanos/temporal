package cli

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
	"go.temporal.io/server/tools/gomad3/toolchain"
)

type application struct {
	executablePath       string
	executableSet        bool
	resolvedInstallation toolchain.Installation
	installationRootArg  string
	installationPath     string
	installationSet      bool
	runnerBuild          string
	runnerBuildSet       bool
	privateCommands      privateCommands
	privateCommandsSet   bool
}

type privateCommands struct {
	supervisor  []string
	coordinator []string
}

func newApplication() *application {
	return &application{}
}

func applicationWithExecutable(executable string) *application {
	return &application{executablePath: executable, executableSet: true}
}

func (app *application) executable() (string, error) {
	if app.executableSet {
		return app.executablePath, nil
	}
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	app.executablePath = executable
	app.executableSet = true
	return executable, nil
}

func (app *application) installation(explicitToolchainRoot string) (toolchain.Installation, string, error) {
	if app.installationSet && app.installationRootArg == explicitToolchainRoot {
		return app.resolvedInstallation, app.installationPath, nil
	}
	executable, err := app.executable()
	if err != nil {
		return toolchain.Installation{}, "", fmt.Errorf("resolve gomad executable: %w", err)
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return toolchain.Installation{}, "", fmt.Errorf("resolve gomad executable path: %w", err)
	}
	resolved, err := toolchain.ResolveInstallation(toolchain.InstallationSpec{
		Executable: executable, ExplicitToolchainRoot: explicitToolchainRoot, EnvironmentToolchainRoot: os.Getenv("GOMAD3_TOOLCHAIN_DIR"),
	})
	if err != nil {
		return toolchain.Installation{}, "", fmt.Errorf("resolve Gomad installation: %w", err)
	}
	app.resolvedInstallation = resolved
	app.installationRootArg = explicitToolchainRoot
	app.installationPath = executable
	app.installationSet = true
	return resolved, executable, nil
}

func (app *application) identity(explicitToolchainRoot string) (toolchainRoot, executable, runnerBuild string, err error) {
	resolved, executable, err := app.installation(explicitToolchainRoot)
	if err != nil {
		return "", "", "", err
	}
	if !app.runnerBuildSet {
		bytes, readErr := os.ReadFile(executable)
		if readErr != nil {
			return "", "", "", fmt.Errorf("hash gomad executable: %w", readErr)
		}
		digest := sha256.Sum256(bytes)
		app.runnerBuild = fmt.Sprintf("sha256:%x", digest)
		app.runnerBuildSet = true
	}
	app.commands(executable)
	return resolved.ToolchainRoot, executable, app.runnerBuild, nil
}

func (app *application) commands(executable string) privateCommands {
	if !app.privateCommandsSet {
		app.privateCommands = privateCommandsFor(executable)
		app.privateCommandsSet = true
	}
	return app.privateCommands
}

func privateCommandsFor(executable string) privateCommands {
	return privateCommands{
		supervisor: []string{executable, "__supervisor"}, coordinator: []string{executable, "__coordinator"},
	}
}

func (app *application) dispatchPrivateMode(mode string) error {
	return runner.DispatchPrivateMode(mode, os.Stdin, os.Stdout)
}

func (app *application) check(config Config) Report {
	digest, err := hashExecutable(config.RunnerPath)
	config.runnerDigest = digest
	config.runnerDigestError = err
	config.runnerDigestSet = true
	if err == nil {
		app.runnerBuild = string(digest)
		app.runnerBuildSet = true
	}
	return Check(config)
}

func hashExecutable(path string) (digest record.SHA256, retErr error) {
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
		//nolint:staticcheck // Preserve the doctor report's existing repair message.
		return "", fmt.Errorf("Runner is not a regular executable: %s", path)
	}
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("hash Runner: %w", err)
	}
	return record.SHA256(fmt.Sprintf("sha256:%x", hasher.Sum(nil))), nil
}
