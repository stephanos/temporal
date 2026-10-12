package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/backends/native"
	"go.temporal.io/server/tools/gomad3/backends/native/toolchain"
	wasmbackend "go.temporal.io/server/tools/gomad3/backends/wasm/backend"
	wasmcommand "go.temporal.io/server/tools/gomad3/backends/wasm/command"
	wasminstallation "go.temporal.io/server/tools/gomad3/backends/wasm/installation"
	"go.temporal.io/server/tools/gomad3/hostfs"
)

type installationHeader struct {
	Backend string `json:"backend"`
	Schema  string `json:"schema"`
	Options struct{ Config struct{ Profile string } }
}

func runLauncher(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomad", flag.ContinueOnError)
	manifest := flags.String("installation", "", "selected backend installation manifest")
	flags.SetOutput(io.Discard)
	err := flags.Parse(arguments)
	help := errors.Is(err, flag.ErrHelp)
	if err != nil && !help {
		fmt.Fprintln(stderr, err)
		flags.SetOutput(stderr)
		flags.Usage()
		return 2
	}
	app := hostApplication()
	// Private process protocols are not operator commands and do not load an
	// installation or participate in the public command registry.
	if len(flags.Args()) != 0 && isPrivateMode(flags.Arg(0)) {
		return app.runPrivateMode(flags.Arg(0), stderr)
	}
	path, fromWASMEnvironment := *manifest, false
	if path == "" {
		path = app.environment(wasminstallation.Environment)
		fromWASMEnvironment = path != ""
	}
	if path == "" {
		if executable, err := app.executablePath(); err == nil {
			if resolved, err := filepath.EvalSymlinks(executable); err == nil {
				executable = resolved
			}
			for _, candidate := range []string{
				filepath.Join(filepath.Dir(executable), toolchain.InstallationManifestName),
				filepath.Join(filepath.Dir(executable), wasminstallation.Sibling),
				filepath.Join(filepath.Dir(filepath.Dir(executable)), toolchain.InstallationManifestName),
			} {
				if _, err := os.Lstat(candidate); err == nil {
					path = candidate
					break
				}
			}
		}
	}
	header := installationHeader{Backend: "native"}
	var headerError error
	if fromWASMEnvironment {
		header.Backend = "wasm"
	}
	if path != "" {
		data, err := hostfs.ReadBounded(path, 16<<20)
		if err == nil {
			var decoded installationHeader
			if err := json.Unmarshal(data, &decoded); err != nil {
				headerError = fmt.Errorf("decode installation header: %w", err)
			} else if decoded.Backend == "" {
				switch decoded.Schema {
				case "gomad3.installation/v1":
					decoded.Backend = "native"
				case wasminstallation.Schema:
					decoded.Backend = "wasm"
				default:
					headerError = fmt.Errorf("unknown installation schema %q", decoded.Schema)
				}
			}
			if headerError == nil {
				header = decoded
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			headerError = fmt.Errorf("read installation header: %w", err)
		}
	}
	registry := backends.Registry{
		"native": native.New(native.Config{}).Capabilities(),
		"wasm":   wasmbackend.CapabilitiesForProfile(header.Options.Config.Profile),
	}
	// An unambiguous backend-owned verb, such as install or report, is usable
	// before an installation exists. Shared verbs use the selected manifest.
	selected := registry[header.Backend]
	if (path == "" || headerError != nil) && len(flags.Args()) != 0 && !supportsCommand(&selected, flags.Arg(0)) {
		for name, capabilities := range registry {
			if supportsCommand(&capabilities, flags.Arg(0)) {
				header.Backend = name
				break
			}
		}
	}
	capabilities, err := registry.Resolve(header.Backend)
	if err != nil {
		return writeCommandError(stderr, 2, "%v\n", err)
	}
	if help {
		return writeCommandError(stderr, 2, "%s", commandUsage(&capabilities))
	}
	if headerError != nil && len(flags.Args()) != 0 && !sourceFreeCommand(flags.Arg(0)) && !commandHelp(flags.Args()[1:]) {
		return writeCommandError(stderr, 2, "%v\n", headerError)
	}
	if path != "" {
		path, err = filepath.Abs(path)
		if err != nil {
			return writeCommandError(stderr, 2, "resolve installation manifest: %v\n", err)
		}
		app.selectedManifest = path
	}
	app.capabilities = &capabilities
	app.replayHook = wasmcommand.ReplayRetainedCluster
	if header.Backend == "wasm" {
		resolve := func() (wasminstallation.Description, error) {
			executable, err := app.executablePath()
			if err != nil {
				return wasminstallation.Description{}, err
			}
			return wasminstallation.Resolve(wasminstallation.Spec{Executable: executable, ExplicitManifest: path})
		}
		app.backend = func() (BackendInstallation, error) {
			description, err := resolve()
			if err != nil {
				return BackendInstallation{}, err
			}
			provider, err := description.Provider()
			if err != nil {
				return BackendInstallation{}, err
			}
			return BackendInstallation{Name: capabilities.Name, Provider: provider, ReplayCommand: []string{"gomad", "--installation", description.Path, "replay"}}, nil
		}
		app.command = func(args []string, out, diagnostic io.Writer) (int, bool) {
			return wasmcommand.Run(args, out, diagnostic, resolve)
		}
	} else if path != "" {
		app.manifestRoot = func() (string, error) {
			data, err := hostfs.ReadBounded(path, 64<<10)
			if err != nil {
				return "", err
			}
			var manifest struct {
				Schema, Backend string
				ToolchainRoot   string `json:"toolchain_root"`
			}
			decoder := json.NewDecoder(strings.NewReader(string(data)))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&manifest); err != nil {
				return "", err
			}
			if err := decoder.Decode(new(any)); err != io.EOF {
				return "", errors.New("installation manifest has trailing data")
			}
			if manifest.Schema != "gomad3.installation/v1" || manifest.ToolchainRoot == "" {
				return "", errors.New("native installation schema and toolchain root are required")
			}
			root := manifest.ToolchainRoot
			if !filepath.IsAbs(root) {
				root = filepath.Join(filepath.Dir(path), root)
			}
			return filepath.Abs(root)
		}
	}
	return app.run(flags.Args(), stdout, stderr)
}

func sourceFreeCommand(command string) bool {
	return command == "inspect" || command == "recover" || command == "report"
}

func commandHelp(arguments []string) bool {
	for _, argument := range arguments {
		if argument == "--" {
			return false
		}
		if argument == "-h" || argument == "--h" || argument == "-help" || argument == "--help" {
			return true
		}
	}
	return false
}

func commandUsage(capabilities *backends.Capabilities) string {
	if capabilities == nil || capabilities.IOTranscript {
		return usage
	}
	var output strings.Builder
	output.WriteString("usage: gomad [--installation FILE] COMMAND [flags]\ncommands:\n")
	for _, command := range capabilities.Commands {
		fmt.Fprintf(&output, "  %s\n", command)
	}
	return output.String()
}
