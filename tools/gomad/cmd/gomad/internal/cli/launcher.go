package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/backends"
	wasmbackend "go.temporal.io/server/tools/gomad/internal/backends/wasm/backend"
	wasmcommand "go.temporal.io/server/tools/gomad/internal/backends/wasm/command"
	wasminstallation "go.temporal.io/server/tools/gomad/internal/backends/wasm/installation"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
	"go.temporal.io/server/tools/gomad/internal/runner"
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
	// argument is the command, so the grammar keeps reporting an unknown
	// command rather than a flag-package error.
	commandStart := installationOptionsEnd(arguments)
	if err := flags.Parse(arguments[:commandStart]); err != nil {
		status := writeCommandError(stderr, 2, "%v\n", err)
		flags.SetOutput(stderr)
		flags.Usage()
		return status
	}
	commandArguments := arguments[commandStart:]
	path := *manifest
	if path == "" {
		path = app.environment(wasminstallation.Environment)
	}
	if path == "" {
		if executable, err := app.executablePath(); err == nil {
			if resolved, err := filepath.EvalSymlinks(executable); err == nil {
				executable = resolved
			}
			candidate := filepath.Join(filepath.Dir(executable), wasminstallation.Sibling)
			if _, err := os.Lstat(candidate); err == nil {
				path = candidate
			}
		}
	}
	header := installationHeader{Backend: "wasm"}
	var headerError error
	if path != "" {
		data, err := hostfs.ReadBounded(path, 16<<20)
		if err == nil {
			var decoded installationHeader
			if err := json.Unmarshal(data, &decoded); err != nil {
				headerError = fmt.Errorf("decode installation header: %w", err)
			} else if decoded.Backend == "" {
				switch decoded.Schema {
				case "gomad.installation/v1":
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
		"wasm": wasmbackend.CapabilitiesForProfile(header.Options.Config.Profile),
	}
	capabilities, err := registry.Resolve(header.Backend)
	if err != nil {
		return writeCommandError(stderr, 2, "%v\n", err)
	}
	if path == "" {
		capabilities.Commands = append(append([]string(nil), capabilities.Commands...), "doctor")
	}
	if len(commandArguments) != 0 && strings.HasPrefix(commandArguments[0], "-") {
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
	}
	app.capabilities = &capabilities
	resolve := func() (wasminstallation.Description, error) {
		executable, err := app.executablePath()
		if err != nil {
			return wasminstallation.Description{}, err
		}
		return wasminstallation.Resolve(wasminstallation.Spec{Executable: executable, ExplicitManifest: path})
	}
	app.replayHook = wasmcommand.RetainedClusterReplay(resolve)
	app.clusterInspection = wasmcommand.RetainedClusterInspection
	if path == "" {
		retained := app.replayHook
		app.replayHook = func(ctx context.Context, spec runner.ReplaySpec) (runner.ReplayResult, bool, error) {
			result, handled, err := retained(ctx, spec)
			if handled || err != nil {
				return result, handled, err
			}
			opened, err := artifact.OpenArtifact(spec.ArtifactPath)
			if err != nil {
				return runner.ReplayResult{}, true, &runner.ReplayPreflightError{Err: err}
			}
			return result, false, opened.Close()
		}
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
