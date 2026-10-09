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

// The private child modes the gomad executable serves for its own Runner. The
// Runner starts the supervisor and coordinator through the commands an
// installation spells, and derives the target bootstrap from the supervisor's
// executable.
const (
	coordinatorMode     = "__coordinator"
	supervisorMode      = "__supervisor"
	targetBootstrapMode = "__target_bootstrap"
)

// toolchainEnvironment names the environment variable that selects the
// toolchain root when --toolchain-root is absent.
const toolchainEnvironment = "GOMAD3_TOOLCHAIN_DIR"

// application is the construction one gomad invocation shares. Run builds it
// once from the host. It alone finds the running executable, resolves the
// pinned toolchain installation, digests the Runner build and spells the
// private child-mode commands; commands receive what it resolves and never
// consult the host for those themselves. A command resolves its installation
// only after it has validated its own input, so invalid input keeps precedence
// over an unusable installation.
type application struct {
	executable  func() (string, error)
	environment func(string) string
	cache       *installationCache
	// privateInput and privateOutput carry the coordinator's request and
	// response protocol; the supervisor and target bootstrap modes ignore
	// them and read their fixed descriptors. They are the process's standard
	// streams, not a command's writers.
	privateInput  io.Reader
	privateOutput io.Writer
	dispatch      func(mode string, input io.Reader, output io.Writer) error
}

// installation is what one operation runs with: the resolved toolchain root,
// and the executable and Runner build that serve its private child modes.
type installation struct {
	toolchainRoot string
	executable    string
	runnerBuild   string
}

type installationCache struct {
	explicitRoot string
	installed    installation
	set          bool
	executable   string
	runnerBuild  string
	identitySet  bool
}

func hostApplication() application {
	return application{
		executable: os.Executable, environment: os.Getenv,
		privateInput: os.Stdin, privateOutput: os.Stdout, dispatch: runner.DispatchPrivateMode,
		cache: new(installationCache),
	}
}

func (installation installation) supervisorCommand() []string {
	return []string{installation.executable, supervisorMode}
}

func (installation installation) coordinatorCommand() []string {
	return []string{installation.executable, coordinatorMode}
}

func isPrivateMode(mode string) bool {
	return mode == coordinatorMode || mode == targetBootstrapMode || mode == supervisorMode
}

func (app application) runPrivateMode(mode string, stderr io.Writer) int {
	if err := app.dispatch(mode, app.privateInput, app.privateOutput); err != nil {
		if _, printErr := fmt.Fprintln(stderr, err); printErr != nil {
			return 3
		}
		return 3
	}
	return 0
}

// executablePath reports the running gomad executable as the host names it.
func (app application) executablePath() (string, error) {
	executable, err := app.executable()
	if err != nil {
		return "", fmt.Errorf("resolve gomad executable: %w", err)
	}
	return executable, nil
}

func absoluteExecutable(executable string) (string, error) {
	absolute, err := filepath.Abs(executable)
	if err != nil {
		return "", fmt.Errorf("resolve gomad executable path: %w", err)
	}
	return absolute, nil
}

// resolveInstallation locates the toolchain installation serving executable:
// an explicit --toolchain-root first, then the environment, then the
// installation beside the executable.
func (app application) resolveInstallation(executable, explicitToolchainRoot string) (toolchain.Installation, error) {
	return toolchain.ResolveInstallation(toolchain.InstallationSpec{
		Executable: executable, ExplicitToolchainRoot: explicitToolchainRoot, EnvironmentToolchainRoot: app.environment(toolchainEnvironment),
	})
}

// install resolves everything an operation needs to start the Runner's child
// processes from this executable.
func (app application) install(explicitToolchainRoot string) (installation, error) {
	if app.cache != nil && app.cache.set && app.cache.explicitRoot == explicitToolchainRoot {
		return app.cache.installed, nil
	}
	executable := ""
	if app.cache != nil && app.cache.identitySet {
		executable = app.cache.executable
	} else {
		var err error
		executable, err = app.executablePath()
		if err != nil {
			return installation{}, err
		}
		executable, err = absoluteExecutable(executable)
		if err != nil {
			return installation{}, err
		}
	}
	resolved, err := app.resolveInstallation(executable, explicitToolchainRoot)
	if err != nil {
		return installation{}, fmt.Errorf("resolve Gomad installation: %w", err)
	}
	build := ""
	if app.cache != nil && app.cache.identitySet {
		build = app.cache.runnerBuild
	} else {
		digest, err := digestRunner(executable)
		if err != nil {
			return installation{}, fmt.Errorf("hash gomad executable: %w", err)
		}
		build = string(digest)
	}
	installed := installation{toolchainRoot: resolved.ToolchainRoot, executable: executable, runnerBuild: build}
	if app.cache != nil {
		app.cache.explicitRoot, app.cache.installed, app.cache.set = explicitToolchainRoot, installed, true
		app.cache.executable, app.cache.runnerBuild, app.cache.identitySet = executable, build, true
	}
	return installed, nil
}

// installedToolchain resolves only the toolchain root, for operations that
// start no Runner child process.
func (app application) installedToolchain(explicitToolchainRoot string) (string, error) {
	resolved, err := app.install(explicitToolchainRoot)
	return resolved.toolchainRoot, err
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
