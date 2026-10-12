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

func (app application) launch(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomad", flag.ContinueOnError)
	manifest := flags.String("installation", "", "selected backend installation manifest")
	flags.SetOutput(io.Discard)
	// Only leading --installation options are global. Any other leading
	// argument is the command, so the native grammar keeps reporting an unknown
	// command rather than a flag-package error.
	commandStart := installationOptionsEnd(arguments)
	if err := flags.Parse(arguments[:commandStart]); err != nil {
		status := writeCommandError(stderr, 2, "%v\n", err)
		flags.SetOutput(stderr)
		flags.Usage()
		return status
	}
	commandArguments := arguments[commandStart:]
	// Private process protocols are not operator commands and do not load an
	// installation or participate in the public command registry.
	if len(commandArguments) != 0 && isPrivateMode(commandArguments[0]) {
		return app.runPrivateMode(commandArguments[0], stderr)
	}
	path, discovered, fromWASMEnvironment := *manifest, false, false
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
					path, discovered = candidate, true
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
	// A native manifest found beside the executable is the native resolver's
	// lowest-precedence source: --toolchain-root and GOMAD3_TOOLCHAIN_DIR win,
	// and the resolver reports a bad manifest only when neither applies.
	if discovered && filepath.Base(path) == toolchain.InstallationManifestName && (headerError != nil || header.Backend == "native") {
		path, header, headerError = "", installationHeader{Backend: "native"}, nil
	}
	registry := backends.Registry{
		"native": native.New(native.Config{}).Capabilities(),
		"wasm":   wasmbackend.CapabilitiesForProfile(header.Options.Config.Profile),
	}
	// An unambiguous backend-owned verb, such as install or report, is usable
	// before an installation exists. Shared verbs use the selected manifest.
	selected := registry[header.Backend]
	if (path == "" || headerError != nil) && len(commandArguments) != 0 && !supportsCommand(&selected, commandArguments[0]) {
		for name, capabilities := range registry {
			if supportsCommand(&capabilities, commandArguments[0]) {
				header.Backend = name
				break
			}
		}
	}
	capabilities, err := registry.Resolve(header.Backend)
	if err != nil {
		return writeCommandError(stderr, 2, "%v\n", err)
	}
	// Other backends keep the flag-package top-level grammar of the former
	// gomad-wasm launcher.
	if header.Backend != "native" && len(commandArguments) != 0 && strings.HasPrefix(commandArguments[0], "-") {
		err := flags.Parse(commandArguments)
		if errors.Is(err, flag.ErrHelp) {
			return writeCommandError(stderr, 2, "%s", commandUsage(&capabilities))
		}
		if err != nil {
			status := writeCommandError(stderr, 2, "%v\n", err)
			flags.SetOutput(stderr)
			flags.Usage()
			return status
		}
		commandArguments = flags.Args()
	}
	if headerError != nil && len(commandArguments) != 0 && !sourceFreeCommand(commandArguments[0]) && !commandHelp(commandArguments) {
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
		app.nativeManifest = path
	}
	return app.run(commandArguments, stdout, stderr)
}

// installationOptionsEnd is the index of the first argument after the leading
// --installation options.
func installationOptionsEnd(arguments []string) int {
	index := 0
	for index < len(arguments) && strings.HasPrefix(arguments[index], "-") {
		name, _, assigned := strings.Cut(strings.TrimPrefix(arguments[index][1:], "-"), "=")
		if name != "installation" {
			break
		}
		index++
		if !assigned && index < len(arguments) {
			index++
		}
	}
	return index
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
