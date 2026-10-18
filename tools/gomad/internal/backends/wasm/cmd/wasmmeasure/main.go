// Command wasmmeasure runs the WASM backend's measurement producers (fn-187.2)
// outside the test runner. Each subcommand is one producer; its inputs are
// flags, never the environment. Build the command, then run it; `go run`
// reports every failure as exit status 1:
//
//	cd tools/gomad && go build -o /abs/bin/wasmmeasure ./internal/backends/wasm/cmd/wasmmeasure
//	/abs/bin/wasmmeasure runtime-cost-profile --repo /abs/checkout --profile-root /abs/evidence \
//	    --test-helper /abs/gomad-wasmhost --stock-go /abs/go
//
// wasmmeasure builds the --repo checkout's measurement command with --go
// (default: go on PATH) and runs its production executor, so the producer,
// runtime and oracles that run are the ones under --repo. Producers keep their
// admission, identity and host-evidence checks and their retained artifact schemas. No test
// target invokes this command. Exit status: 0 success, 1 failure, 2 refusal.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/measure"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/measurement"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	args, start := os.Args[1:], producer(produce)
	if len(args) > 0 && args[0] == "--internal-run" {
		args, start = args[1:], execute
	}
	code := run(ctx, args, os.Environ(), os.Stdout, os.Stderr, start)
	stop()
	os.Exit(code)
}

// invocation is one validated producer run.
type invocation struct {
	name, repo, goCommand string
	inputs                map[string]string
	environ               []string
}

// producer runs an invocation and returns its exit status.
type producer func(ctx context.Context, call invocation, stdout, stderr io.Writer) int

// produce builds the measured checkout's production measurement executor.
func produce(ctx context.Context, call invocation, stdout, stderr io.Writer) (code int) {
	scratch, err := os.MkdirTemp("", "wasmmeasure-")
	if err != nil {
		return writeStatus(stderr, measurement.ExitFailed, "wasmmeasure: %v\n", err)
	}
	defer func() {
		if err := os.RemoveAll(scratch); err != nil {
			code = writeStatus(stderr, measurement.ExitFailed, "wasmmeasure: cleanup: %v\n", err)
		}
	}()
	binary := filepath.Join(scratch, "wasmmeasure")
	build := exec.CommandContext(ctx, call.goCommand, "build", "-trimpath=false", "-o", binary, "./internal/backends/wasm/cmd/wasmmeasure")
	build.Dir = filepath.Join(call.repo, "tools", "gomad")
	build.Env = append(append([]string(nil), call.environ...), "GOROOT=", "GOENV=off", "GOWORK=off", "GOFLAGS=", "GOTOOLCHAIN=local", "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH, "GOEXPERIMENT=", "GOCACHEPROG=", "CGO_ENABLED=0")
	build.Stdout, build.Stderr = stderr, stderr
	if err := build.Run(); err != nil {
		return writeStatus(stderr, measurement.ExitFailed, "wasmmeasure %s: building %s: %v\n", call.name, build.Dir, err)
	}
	args := []string{"--internal-run", call.name, "--repo", call.repo, "--go", call.goCommand}
	variables := make([]string, 0, len(call.inputs))
	for variable := range call.inputs {
		variables = append(variables, variable)
	}
	sort.Strings(variables)
	for _, variable := range variables {
		args = append(args, "--"+measurement.Flag(variable), call.inputs[variable])
	}
	command := exec.CommandContext(ctx, binary, args...)
	command.Dir, command.Env = build.Dir, call.environ
	command.Stdout, command.Stderr = stdout, stderr
	err = command.Run()
	exit, exited := errors.AsType[*exec.ExitError](err)
	switch {
	case err == nil:
		return 0
	case exited && (exit.ExitCode() == measurement.ExitFailed || exit.ExitCode() == measurement.ExitRefused):
		return exit.ExitCode()
	default:
		return writeStatus(stderr, measurement.ExitFailed, "wasmmeasure %s: %v\n", call.name, err)
	}
}

func execute(ctx context.Context, call invocation, stdout, stderr io.Writer) int {
	_, source, _, ok := runtime.Caller(0)
	compiled, compiledErr := os.Stat(source)
	selected, selectedErr := os.Stat(filepath.Join(call.repo, "tools", "gomad", "internal", "backends", "wasm", "cmd", "wasmmeasure", "main.go"))
	if !ok || compiledErr != nil || selectedErr != nil || !os.SameFile(compiled, selected) {
		return writeStatus(stderr, measurement.ExitRefused, "wasmmeasure: executor was not built from --repo\n")
	}
	err := measure.Run(ctx, call.name, measure.Options{Repo: call.repo, Inputs: call.inputs, Environ: call.environ, Log: stdout})
	if err == nil {
		return 0
	}
	if errors.Is(err, measure.ErrRefused) {
		return writeStatus(stderr, measurement.ExitRefused, "wasmmeasure %s: %v\n", call.name, err)
	}
	return writeStatus(stderr, measurement.ExitFailed, "wasmmeasure %s: %v\n", call.name, err)
}

func writeStatus(output io.Writer, status int, format string, args ...any) int {
	if _, err := fmt.Fprintf(output, format, args...); err != nil {
		return measurement.ExitFailed
	}
	return status
}

func usage(output io.Writer) error {
	if _, err := fmt.Fprintln(output, "usage: wasmmeasure <measurement> --repo <absolute checkout> [--go <go>] [inputs]\nmeasurements:"); err != nil {
		return err
	}
	for _, name := range measurement.Names() {
		var required []string
		for _, input := range measurement.Inputs(name) {
			if input.Required {
				required = append(required, "--"+measurement.Flag(input.Variable))
			}
		}
		if _, err := fmt.Fprintf(output, "  %-30s %s\n", name, strings.Join(required, " ")); err != nil {
			return err
		}
	}
	return nil
}

func run(ctx context.Context, args, environ []string, stdout, stderr io.Writer, start producer) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		if err := usage(stdout); err != nil {
			return measurement.ExitFailed
		}
		if len(args) == 0 {
			return measurement.ExitRefused
		}
		return 0
	}
	name := args[0]
	inputs := measurement.Inputs(name)
	if inputs == nil {
		if code := writeStatus(stderr, measurement.ExitRefused, "wasmmeasure: unknown measurement %q\n", name); code == measurement.ExitFailed {
			return code
		}
		if err := usage(stderr); err != nil {
			return measurement.ExitFailed
		}
		return measurement.ExitRefused
	}
	// Inputs are flags only: an inherited GOMAD_* variable would be an input
	// the command line does not show.
	for _, entry := range environ {
		if key, _, _ := strings.Cut(entry, "="); strings.HasPrefix(key, "GOMAD_") {
			return writeStatus(stderr, measurement.ExitRefused, "wasmmeasure: unset %s; pass inputs as flags\n", key)
		}
	}
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	var parseOutput strings.Builder
	flags.SetOutput(&parseOutput)
	repo := flags.String("repo", "", "absolute checkout whose measurement command is built and run (required)")
	goCommand := flags.String("go", "go", "go command that builds the checkout's measurement command")
	values := map[string]*string{}
	for _, input := range inputs {
		help := "sets " + input.Variable
		if input.Required {
			help += " (required)"
		}
		values[input.Variable] = flags.String(measurement.Flag(input.Variable), "", help)
	}
	if err := flags.Parse(args[1:]); err != nil {
		return writeStatus(stderr, measurement.ExitRefused, "%s", parseOutput.String())
	}
	if flags.NArg() != 0 {
		return writeStatus(stderr, measurement.ExitRefused, "wasmmeasure: unexpected arguments %q\n", flags.Args())
	}
	if !filepath.IsAbs(*repo) || filepath.Clean(*repo) != *repo {
		return writeStatus(stderr, measurement.ExitRefused, "wasmmeasure: --repo must be an absolute clean checkout path\n")
	}
	if info, err := os.Stat(filepath.Join(*repo, "tools", "gomad", "internal", "backends", "wasm", "internal", "measure")); err != nil || !info.IsDir() {
		return writeStatus(stderr, measurement.ExitRefused, "wasmmeasure: %s is not a gomad checkout with wasmmeasure producers\n", *repo)
	}
	call := invocation{name: name, repo: *repo, goCommand: *goCommand, inputs: map[string]string{}, environ: environ}
	var missing []string
	for _, input := range inputs {
		value := *values[input.Variable]
		if value != "" {
			call.inputs[input.Variable] = value
		} else if input.Required {
			missing = append(missing, "--"+measurement.Flag(input.Variable))
		}
	}
	if len(missing) != 0 {
		return writeStatus(stderr, measurement.ExitRefused, "wasmmeasure %s: missing required %s\n", name, strings.Join(missing, ", "))
	}
	return start(ctx, call, stdout, stderr)
}
