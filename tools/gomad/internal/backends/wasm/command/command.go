package command

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
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/backend"
	simulation "go.temporal.io/server/tools/gomad/internal/backends/wasm/cluster"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/installation"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/qualification"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	gomadqualification "go.temporal.io/server/tools/gomad/internal/evidence/qualification"
	hostexec "go.temporal.io/server/tools/gomad/internal/host/exec"
	"go.temporal.io/server/tools/gomad/internal/runner"
)

func Run(arguments []string, stdout, stderr io.Writer, resolve func() (installation.Description, error)) (int, bool) {
	if len(arguments) == 0 {
		return 0, false
	}
	if arguments[0] == "install" {
		return install(arguments[1:], stdout, stderr), true
	}
	if arguments[0] == "report" {
		return report(arguments[1:], stdout, stderr), true
	}
	if arguments[0] == "qualify" {
		return qualify(arguments[1:], stdout, stderr, resolve), true
	}
	if arguments[0] == "discover" {
		if len(arguments) != 1 {
			return failure(stderr, fmt.Errorf("discover accepts no command arguments"), 2), true
		}
		description, err := resolve()
		if err != nil {
			return failure(stderr, err, 3), true
		}
		response := struct {
			installation.Description
			Capabilities backends.Capabilities `json:"capabilities"`
		}{description, backend.CapabilitiesForProfile(description.Manifest.Options.Config.Profile)}
		if err := json.NewEncoder(stdout).Encode(response); err != nil {
			return failure(stderr, err, 3), true
		}
		return 0, true
	}
	return 0, false
}

// RetainedClusterReplay returns the retained cluster replay hook. Replay
// executes the resolved installation's helper, never one carried by the artifact.
func RetainedClusterReplay(resolve func() (installation.Description, error)) func(context.Context, runner.ReplaySpec) (runner.ReplayResult, bool, error) {
	return func(ctx context.Context, spec runner.ReplaySpec) (runner.ReplayResult, bool, error) {
		return replayRetainedCluster(ctx, spec, resolve)
	}
}

func replayRetainedCluster(ctx context.Context, spec runner.ReplaySpec, resolve func() (installation.Description, error)) (replay runner.ReplayResult, handled bool, retErr error) {
	opened, err := artifact.OpenArtifact(spec.ArtifactPath)
	if err != nil {
		return runner.ReplayResult{}, false, nil
	}
	manifest := opened.Manifest()
	if err := opened.Close(); err != nil {
		return runner.ReplayResult{}, false, err
	}
	if manifest.Target.Backend == nil || manifest.Target.Backend.Evidence == nil || !simulation.IsRetainedSchema(manifest.Target.Backend.Evidence.Schema) {
		return runner.ReplayResult{}, false, nil
	}
	if spec.Inspection != nil {
		return runner.ReplayResult{}, false, nil
	}
	if spec.VerifyOnly || spec.ObservedDir != "" || spec.ToolchainRoot != "" {
		return runner.ReplayResult{}, true, &runner.ReplayPreflightError{Err: fmt.Errorf("retained WASM cluster replay does not support --verify-only, --observed, or --toolchain-root")}
	}
	// Older retained schemas are refused before an installation is resolved.
	if err := simulation.CheckRetainedSchema(manifest.Target.Backend.Evidence.Schema); err != nil {
		return runner.ReplayResult{}, true, err
	}
	description, err := resolve()
	if err != nil {
		return runner.ReplayResult{}, true, fmt.Errorf("retained WASM cluster replay requires a WASM installation: %w", err)
	}
	options := description.Manifest.Options
	installed := simulation.InstalledHelper{Path: options.HelperPath, SHA256: strings.TrimPrefix(options.HelperSHA256, "sha256:"), Engine: options.Engine}
	directory, err := os.MkdirTemp("", "gomad-wasm-cluster-replay-")
	if err != nil {
		return runner.ReplayResult{}, true, err
	}
	defer func() { retErr = errors.Join(retErr, os.RemoveAll(directory)) }()
	result, _, err := qualification.ReplayClusterArtifact(ctx, spec.ArtifactPath, directory, installed)
	if err != nil {
		return runner.ReplayResult{}, true, err
	}
	return result, true, nil
}

func qualify(arguments []string, stdout, stderr io.Writer, resolve func() (installation.Description, error)) int {
	operatorStart := hostexec.CurrentProcessResourceUsage()
	flags := flag.NewFlagSet("gomad qualify", flag.ContinueOnError)
	flags.SetOutput(stderr)
	path := flags.String("spec", "", "qualification owner JSON spec; installation supplies backend Options")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 0 || *path == "" {
		return failure(stderr, fmt.Errorf("qualify requires --spec FILE and no positional arguments"), 2)
	}
	spec, err := qualification.ReadSpec(*path)
	if err != nil {
		return failure(stderr, err, 2)
	}
	spec.OperatorStart = operatorStart
	description, err := resolve()
	if err != nil {
		return failure(stderr, err, 3)
	}
	spec.Options = description.Manifest.Options
	spec.Command = append([]string{"gomad", "--installation", description.Path, "qualify"}, arguments...)
	spec, err = qualification.Freeze(context.Background(), spec)
	if err != nil {
		return failure(stderr, err, 3)
	}
	result, err := qualification.Run(context.Background(), spec)
	if err != nil {
		return failure(stderr, err, 3)
	}
	resultPath, receipt, err := qualification.WriteResultObserved(spec.ArtifactRoot, result)
	if err != nil && resultPath == "" {
		return failure(stderr, err, 3)
	}
	return emitQualificationResult(stdout, stderr, resultPath, result, receipt, err)
}

func emitQualificationResult(stdout, stderr io.Writer, resultPath string, result qualification.Result, receipt qualification.PublicationReceipt, publicationErr error) int {
	if !filepath.IsAbs(resultPath) || len(resultPath) > 1024 {
		return failure(stderr, fmt.Errorf("qualification result_path must be absolute and at most 1024 bytes"), 3)
	}
	encoded, err := json.Marshal(struct {
		Path    string                           `json:"result_path"`
		Receipt qualification.PublicationReceipt `json:"operator_receipt"`
	}{resultPath, receipt})
	if err != nil {
		return qualificationResponseFailure(stderr, "qualification response encoding failed", err)
	}
	if len(encoded)+1 > 8<<10 {
		return failure(stderr, fmt.Errorf("qualification response exceeds 8KiB limit"), 3)
	}
	encoded = append(encoded, '\n')
	if n, err := stdout.Write(encoded); err != nil {
		return qualificationResponseFailure(stderr, "qualification response output failed", err)
	} else if n != len(encoded) {
		return qualificationResponseFailure(stderr, "qualification response output failed", io.ErrShortWrite)
	}
	if publicationErr != nil {
		return qualificationResponseFailure(stderr, "qualification publication observation failed", publicationErr)
	}
	if !result.Manifest.Provisional && (result.CostFailure != "" || result.BudgetFailure != "" || result.Budget == nil || !result.Budget.WithinBudget) {
		return failure(stderr, fmt.Errorf("qualification cost/budget gate failed; inspect retained result %s", resultPath), 3)
	}
	return gomadqualification.ExitStatus(gomadqualification.ClassifyQualification(result.Qualification))
}

func qualificationResponseFailure(stderr io.Writer, reason string, err error) int {
	detail := err.Error()
	if len(detail) > 1024 {
		detail = detail[:1024] + " [truncated]"
	}
	return failure(stderr, errors.New(reason+": "+detail), 3)
}

func report(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomad report", flag.ContinueOnError)
	flags.SetOutput(stderr)
	path := flags.String("result", "", "retained content-addressed WASM qualification result")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 0 || *path == "" {
		return failure(stderr, fmt.Errorf("report requires --result FILE and no positional arguments"), 2)
	}
	result, err := qualification.OpenResult(*path)
	if err != nil {
		return failure(stderr, err, 3)
	}
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		return failure(stderr, err, 3)
	}
	return 0
}

func install(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomad install", flag.ContinueOnError)
	flags.SetOutput(stderr)
	compiler := flags.String("compiler", "", "existing stock Go compiler executable")
	helper := flags.String("helper", "", "existing pinned WASM helper executable")
	runtimeRoot := flags.String("runtime-root", "", "canonical runtimesemantics source directory")
	cache := flags.String("cache", "", "private preparation cache directory")
	compileCache := flags.String("compile-cache", "", "private helper compiled-code cache directory")
	output := flags.String("output", "", "new installation manifest path")
	profile := flags.String("profile", wasi.StockProfile, "explicit WASM capability profile")
	version := flags.String("go-version", "go1.27.1", "pinned stock Go version")
	configPath := flags.String("config", "", "immutable guest configuration JSON")
	temporalCheckout := flags.String("temporal-checkout", "", "Temporal checkout whose SQLite schema is captured")
	temporalPackage := flags.String("temporal-package", "", "checkout-relative Temporal test package")
	memory := flags.Uint64("memory-bytes", 256<<20, "guest memory capacity")
	fuel := flags.Uint64("fuel", 4000000000, "deterministic guest instruction capacity")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 0 || *output == "" {
		return failure(stderr, fmt.Errorf("install requires --output FILE and named tool/cache paths"), 2)
	}
	config := installation.DefaultConfig(*profile)
	var buildTags []string
	if *temporalCheckout != "" || *temporalPackage != "" {
		for _, entry := range os.Environ() {
			if key, _, _ := strings.Cut(entry, "="); strings.HasPrefix(key, "GOMAD_") {
				return failure(stderr, fmt.Errorf("unset %s; Temporal setup accepts explicit inputs only", key), 2)
			}
		}
		if *temporalCheckout == "" || *temporalPackage == "" {
			return failure(stderr, fmt.Errorf("temporal setup requires --temporal-checkout and --temporal-package"), 2)
		}
		for _, input := range []struct{ name, value string }{{"compiler", *compiler}, {"helper", *helper}, {"cache", *cache}, {"runtime-root", *runtimeRoot}} {
			if input.value == "" {
				return failure(stderr, fmt.Errorf("temporal setup requires --%s", input.name), 2)
			}
		}
		conflict := ""
		flags.Visit(func(f *flag.Flag) {
			if f.Name == "config" || f.Name == "profile" {
				conflict = f.Name
			}
		})
		if conflict != "" {
			return failure(stderr, fmt.Errorf("temporal setup derives --%s", conflict), 2)
		}
		var err error
		config, err = installation.TemporalConfig(*temporalCheckout, *temporalPackage)
		if err != nil {
			return failure(stderr, err, 2)
		}
		for _, input := range []struct{ name, path string }{{"compiler", *compiler}, {"helper", *helper}} {
			if _, err := os.Lstat(input.path); err != nil {
				return failure(stderr, fmt.Errorf("temporal %s: %w", input.name, err), 2)
			}
		}
		buildTags = backend.TemporalGuestBuildTags()
		visited := map[string]bool{}
		flags.Visit(func(f *flag.Flag) { visited[f.Name] = true })
		if !visited["memory-bytes"] {
			*memory = 4 << 30
		}
		if !visited["fuel"] {
			*fuel = 1 << 50
		}
	}
	if *configPath != "" {
		var err error
		config, err = installation.ReadConfig(*configPath)
		if err != nil {
			return failure(stderr, err, 2)
		}
		if config.Profile != *profile {
			return failure(stderr, fmt.Errorf("--profile and guest configuration disagree"), 2)
		}
	}
	manifest, err := installation.Build(backend.Options{CompilerPath: *compiler, HelperPath: *helper, RuntimeRoot: *runtimeRoot, CacheRoot: *cache, CompileCacheRoot: *compileCache, BuildTags: buildTags, GoVersion: *version, MemoryBytes: *memory, Fuel: *fuel, Config: config})
	if err != nil {
		return failure(stderr, err, 2)
	}
	if err := installation.Write(*output, manifest); err != nil {
		return failure(stderr, err, 3)
	}
	if _, err := fmt.Fprintf(stdout, "gomad: installed WASM manifest %s profile=%s\n", *output, config.Profile); err != nil {
		return failure(stderr, err, 3)
	}
	return 0
}

func failure(stderr io.Writer, err error, status int) int {
	if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
		return 3
	}
	return status
}
