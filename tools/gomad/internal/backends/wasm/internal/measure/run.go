package measure

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/measurement"
)

var ErrRefused = errors.New("measurement refused")

type Options struct {
	Repo    string
	Inputs  map[string]string
	Environ []string
	Log     io.Writer
	Keep    bool
}

type runKey struct{}

type runScope struct {
	*runResources
	options                       Options
	name                          string
	outputVariable, requestedRoot string
	mu                            sync.Mutex
	failures                      error
	cleanups                      []func(context.Context) error
}

type runResources struct {
	light   chan struct{}
	heavy   chan struct{}
	warm    sync.Mutex
	reports sync.Map
	overlay struct {
		once           sync.Once
		path, identity string
		err            error
	}
}

var producers = map[string]func(context.Context) error{
	"compiled-identity-owners":      measureCompiledIdentityOwners,
	"diagnostic-cluster-baseline":   measureDiagnosticClusterBaseline,
	"diagnostic-helper-build":       measureDiagnosticHelperBuild,
	"failing-suite-profile":         measureFailingSuiteProfile,
	"guest-profile":                 measureGuestProfile,
	"guest-variants":                measureGuestVariants,
	"host-call-profile":             measureHostCallProfile,
	"legacy-workflow-profile":       measureLegacyWorkflowProfile,
	"matched-release-candidate":     measureMatchedReleaseCandidate,
	"matched-suite":                 measureMatchedSuite,
	"migration":                     measureMigration,
	"namespace-activity":            measureNamespaceActivity,
	"namespace-sample":              measureNamespaceSample,
	"operation-retention":           measureOperationRetention,
	"runtime-cost-profile":          measureRuntimeCostProfile,
	"scratch-smoke-profile":         measureScratchSmokeProfile,
	"terminal-exit":                 measureTerminalToProcessExit,
	"terminal-exit-helper-build":    measureTerminalExitHelperBuild,
	"terminal-exit-helper-stage":    measureTerminalExitHelperStage,
	"warm-preparation":              measureWarmPreparation,
	"warm-preparation-helper-build": measureWarmPreparationHelperBuild,
	"warm-preparation-prime":        measureWarmPreparationPrime,
}

func Run(ctx context.Context, name string, options Options) error {
	producer, ok := producers[name]
	if !ok {
		return fmt.Errorf("%w: unknown measurement %q", ErrRefused, name)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !filepath.IsAbs(options.Repo) || filepath.Clean(options.Repo) != options.Repo {
		return fmt.Errorf("%w: repository must be an absolute clean path", ErrRefused)
	}
	for _, entry := range options.Environ {
		if key, _, _ := strings.Cut(entry, "="); strings.HasPrefix(key, "GOMAD_") {
			return fmt.Errorf("%w: unset %s; provide measurement inputs explicitly", ErrRefused, key)
		}
	}
	for _, required := range measurement.Inputs(name) {
		if required.Required && options.Inputs[required.Variable] == "" {
			return fmt.Errorf("%w: missing --%s", ErrRefused, measurement.Flag(required.Variable))
		}
	}
	options.Inputs = maps.Clone(options.Inputs)
	if options.Inputs == nil {
		options.Inputs = map[string]string{}
	}
	if options.Inputs["GOMAD_WASM_TOOLCHAIN_CACHE"] == "" {
		options.Inputs["GOMAD_WASM_TOOLCHAIN_CACHE"] = filepath.Join(options.Repo, "tools/gomad/.toolchain")
	}
	options.Environ = append([]string(nil), options.Environ...)
	if options.Log == nil {
		options.Log = io.Discard
	}
	scope := &runScope{runResources: &runResources{}, options: options, name: name}
	limit := func(name string) int {
		if value, err := strconv.Atoi(options.Inputs[name]); err == nil && value > 0 {
			return value
		}
		return 4
	}
	scope.light = make(chan struct{}, limit("GOMAD_WASM_PROTOTYPE_PARALLEL"))
	scope.heavy = make(chan struct{}, limit("GOMAD_WASM_PROTOTYPE_HEAVY"))
	return scope.execute(context.WithValue(ctx, runKey{}, scope), producer)
}

func (scope *runScope) execute(ctx context.Context, producer func(context.Context) error) error {
	retained, err := scope.beginRetention(ctx)
	if err != nil {
		return err
	}
	recordFailure(ctx, producer(ctx))
	for {
		scope.mu.Lock()
		index := len(scope.cleanups) - 1
		if index < 0 {
			scope.mu.Unlock()
			break
		}
		cleanup := scope.cleanups[index]
		scope.cleanups = scope.cleanups[:index]
		scope.mu.Unlock()
		recordFailure(ctx, cleanup(ctx))
	}
	scope.mu.Lock()
	err = errors.Join(scope.failures, ctx.Err())
	scope.mu.Unlock()
	if retained != nil {
		err = errors.Join(err, retained.finish(ctx, err == nil))
	}
	return err
}

func scopeFor(ctx context.Context) *runScope { return ctx.Value(runKey{}).(*runScope) }

func recordFailure(ctx context.Context, err error) {
	if err == nil {
		return
	}
	scope := scopeFor(ctx)
	scope.mu.Lock()
	defer scope.mu.Unlock()
	scope.failures = errors.Join(scope.failures, err)
}

func failed(ctx context.Context) bool {
	scope := scopeFor(ctx)
	scope.mu.Lock()
	defer scope.mu.Unlock()
	return scope.failures != nil
}

func input(ctx context.Context, name string) string {
	value, _ := lookupInput(ctx, name)
	return value
}

func lookupInput(ctx context.Context, name string) (string, bool) {
	scope := scopeFor(ctx)
	if value, ok := scope.options.Inputs[name]; ok {
		return value, true
	}
	if !strings.HasPrefix(name, "GOMAD_") {
		for _, entry := range scope.options.Environ {
			if value, ok := strings.CutPrefix(entry, name+"="); ok {
				return value, true
			}
		}
	}
	return "", false
}

func setInput(ctx context.Context, name, value string) { scopeFor(ctx).options.Inputs[name] = value }

func environ(ctx context.Context) []string {
	scope := scopeFor(ctx)
	values := map[string]string{}
	for _, entry := range scope.options.Environ {
		if key, value, ok := strings.Cut(entry, "="); ok {
			values[key] = value
		}
	}
	maps.Copy(values, scope.options.Inputs)
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+values[key])
	}
	return result
}

func runName(ctx context.Context) string { return scopeFor(ctx).name }

func rerunRecord(ctx context.Context, goCommand, repo, name string, inputs map[string]string, shell bool) map[string]any {
	scope := scopeFor(ctx)
	if scope.outputVariable != "" {
		inputs = maps.Clone(inputs)
		inputs[scope.outputVariable] = scope.requestedRoot
	}
	return measurement.RerunRecord(goCommand, repo, name, inputs, shell)
}

func addCleanup(ctx context.Context, cleanup func(context.Context) error) {
	scope := scopeFor(ctx)
	scope.mu.Lock()
	defer scope.mu.Unlock()
	scope.cleanups = append(scope.cleanups, cleanup)
}

func temporaryDirectory(ctx context.Context) (string, error) {
	path, err := os.MkdirTemp(input(ctx, "TMPDIR"), "gomad-measure-")
	if err != nil {
		return "", err
	}
	addCleanup(ctx, func(context.Context) error { return os.RemoveAll(path) })
	return path, nil
}

func absolute(ctx context.Context, path string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	return filepath.Join(scopeFor(ctx).options.Repo, "tools/gomad/internal/backends/wasm/wasi", path), nil
}

func logf(ctx context.Context, format string, args ...any) {
	writeLog(ctx, fmt.Sprintf(format, args...))
}

func log(ctx context.Context, args ...any) { writeLog(ctx, fmt.Sprint(args...)) }

func writeLog(ctx context.Context, message string) {
	scope := scopeFor(ctx)
	scope.mu.Lock()
	defer scope.mu.Unlock()
	_, err := fmt.Fprintf(scope.options.Log, "%s: %s\n", scope.name, message)
	scope.failures = errors.Join(scope.failures, err)
}

func runStep(ctx context.Context, name string, step func(context.Context) error) (bool, error) {
	parent := scopeFor(ctx)
	child := &runScope{runResources: parent.runResources, options: parent.options, name: parent.name + "/" + name, outputVariable: parent.outputVariable, requestedRoot: parent.requestedRoot}
	child.options.Inputs = maps.Clone(parent.options.Inputs)
	err := child.execute(context.WithValue(ctx, runKey{}, child), step)
	recordFailure(ctx, err)
	if errors.Is(err, ErrRefused) || ctx.Err() != nil {
		return false, err
	}
	return err == nil, nil
}
