//go:build gomad_native

package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/native"
	"go.temporal.io/server/tools/gomad/internal/backends/native/toolchain"
	"go.temporal.io/server/tools/gomad/internal/runner"
)

const nativeBuiltIn = true
const nativeInstallationManifest = toolchain.InstallationManifestName

func configureNative(app *application) {
	app.nativeProvider = func(installed installation) backends.Provider { return native.New(nativeConfiguration(installed)) }
}

func registerNative(registry backends.Registry) {
	registry["native"] = native.New(native.Config{}).Capabilities()
}

func nativeConfiguration(installed installation) native.Config {
	return native.Config{ToolchainRoot: installed.toolchainRoot, SupervisorCommand: []string{installed.executable, supervisorMode}, CoordinatorCommand: []string{installed.executable, coordinatorMode}, BootstrapCommand: []string{installed.executable, targetBootstrapMode}, RunnerBuild: installed.runnerBuild}
}

func validateRequiredSemanticProbes(required []string) error {
	_, err := native.New(native.Config{}).Describe(context.Background(), backends.Configuration{RequiredSemanticProbes: required})
	return err
}

func (app application) resolveInstallation(executable, explicitToolchainRoot string) (nativeInstallation, error) {
	resolved, err := toolchain.ResolveInstallation(toolchain.InstallationSpec{
		Executable: executable, ExplicitToolchainRoot: explicitToolchainRoot, EnvironmentToolchainRoot: app.environment(toolchainEnvironment),
		Manifest: app.nativeManifest,
	})
	return nativeInstallation{resolved.ToolchainRoot, resolved.Source, resolved.RepairInstruction}, err
}

func dispatchPrivateMode(mode string, input io.Reader, output io.Writer) error {
	if mode == coordinatorMode {
		data, err := io.ReadAll(io.LimitReader(input, (16<<20)+1))
		if err != nil {
			return fmt.Errorf("decode coordinator request: %w", err)
		}
		if len(data) > 16<<20 {
			return fmt.Errorf("coordinator request exceeds size bound")
		}
		config, wire, err := native.CoordinatorInput(data)
		if err != nil {
			return err
		}
		return runner.CoordinatorMain(bytes.NewReader(wire), output, native.New(config))
	}
	handled, err := native.DispatchPrivateMode(mode, input, output)
	if !handled && err == nil {
		return fmt.Errorf("unknown private runner mode %q", mode)
	}
	return err
}
