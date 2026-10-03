// Package adapterregen re-derives a deterministic I/O adapter's anchors for a
// new exact module version. A dry run prints the changed upstream source of
// every rewritten file, the proposed anchors, and an approval digest over
// both; an apply with that digest publishes the adapter constants, the version
// descriptor entry, the pinned test data, and every generated output derived
// from them as one transaction.
package adapterregen

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	compatibility "go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
	"go.temporal.io/server/tools/gomad3/internal/hostexec"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

const (
	commandTimeout     = 20 * time.Minute
	commandOutputLimit = 64 << 20
)

// InputError reports a request the command cannot act on as given.
type InputError struct{ Err error }

func (err *InputError) Error() string { return err.Err.Error() }
func (err *InputError) Unwrap() error { return err.Err }

// BlockedError reports a regeneration that needs a person: an anchor that no
// longer matches exactly once, a rewritten file gone upstream, an adapter not
// built from anchors, a staged output that fails generation or verification,
// or a checkout that changed while the transaction ran.
type BlockedError struct{ Err error }

func (err *BlockedError) Error() string { return err.Err.Error() }
func (err *BlockedError) Unwrap() error { return err.Err }

// Spec describes one regeneration. Root is the Gomad source module root.
type Spec struct {
	Root        string
	Module      string
	Version     string
	GoCommand   string
	Environment []string
	// Approval, when set, applies the regeneration whose review digest it
	// names; when empty the run is a dry run that writes nothing.
	Approval string

	// Generators and Verifiers run in the staged copy; nil selects the
	// defaults. Each command runs with the staged root as its directory and
	// may name "{root}", "{go}", "{module}", and "{candidate}" placeholders.
	Generators [][]string
	Verifiers  [][]string
	// Tidy runs in each staged test fixture module whose requirement moved;
	// nil selects "go mod tidy".
	Tidy []string
	// StageOnly, with Approval, stages and verifies the regeneration and
	// reports every file the apply would publish, then publishes nothing.
	StageOnly bool

	// afterStage, beforeApplyFile, and residualScan are test seams.
	afterStage      func() error
	beforeApplyFile func(index int) error
	residualScan    func(string, deterministicio.AdapterRegeneration) ([]string, error)
}

// Result is a dry run's review or an apply's publication.
type Result struct {
	Regeneration deterministicio.AdapterRegeneration `json:"regeneration"`
	Diffs        []SourceDiff                        `json:"diffs"`
	StalePacks   []StalePack                         `json:"stale_packs"`
	Applied      bool                                `json:"applied"`
	Published    []string                            `json:"published,omitempty"`
	// Staged lists every file the apply publishes, with the diff of each
	// go.mod and go.sum, for a stage-only run and an apply.
	Staged []StagedFile `json:"staged,omitempty"`
	// Residual lists checked-in references to the previous module version
	// that the transaction left for a person, as path:line.
	Residual []string `json:"residual,omitempty"`
	// Warnings are problems after a complete publication.
	Warnings []string `json:"warnings,omitempty"`
}

// StagedFile is one file a staged regeneration adds, changes, or removes.
type StagedFile struct {
	Path   string `json:"path"`
	Change string `json:"change"`
	Diff   string `json:"diff,omitempty"`
}

// SourceDiff is one rewritten upstream file's change between the versions.
type SourceDiff struct {
	Path    string `json:"path"`
	Changed bool   `json:"changed"`
	Diff    string `json:"diff,omitempty"`
}

// StalePack is a compatibility-pack binding of the regenerated adapter that
// still names its previous identity; the pack refresh repairs it.
type StalePack struct {
	File      string   `json:"file"`
	ID        string   `json:"id"`
	Location  string   `json:"location"`
	Platforms []string `json:"platforms"`
}

// Run performs a dry run, or an apply when spec.Approval is set.
func Run(ctx context.Context, spec Spec) (Result, error) {
	if spec.Root == "" || spec.Module == "" || spec.Version == "" || spec.GoCommand == "" {
		return Result{}, &InputError{Err: errors.New("adapter regeneration needs a root, module, version, and go command")}
	}
	if !filepath.IsAbs(spec.Root) || !filepath.IsAbs(spec.GoCommand) {
		return Result{}, &InputError{Err: errors.New("adapter regeneration root and go command must be absolute paths")}
	}
	if spec.StageOnly && spec.Approval == "" {
		return Result{}, &InputError{Err: errors.New("a stage-only run needs the approval digest of the dry run it stages")}
	}
	if pending, err := publicationPending(spec.Root); err != nil {
		return Result{}, err
	} else if pending && (spec.Approval == "" || spec.StageOnly) {
		return Result{}, &BlockedError{Err: errors.New("an interrupted adapter publication is pending; run with --recover to complete it")}
	}
	var pinned *gomadversion.AdapterIdentity
	for _, identity := range gomadversion.Adapters {
		if identity.Module == spec.Module {
			pinned = &identity
		}
	}
	if pinned == nil {
		return Result{}, &InputError{Err: fmt.Errorf("no adapter is pinned for %s", spec.Module)}
	}
	if pinned.Version == spec.Version {
		return Result{}, &InputError{Err: fmt.Errorf("%s is already pinned at %s", spec.Module, spec.Version)}
	}
	if err := requirePinnedGo(ctx, spec); err != nil {
		return Result{}, err
	}
	work, err := os.MkdirTemp("", "gomad3-adapter-regenerate-")
	if err != nil {
		return Result{}, err
	}
	defer removeWritable(work)
	downloads, err := download(ctx, spec, filepath.Join(work, "modcache"), []string{pinned.Version, spec.Version})
	if err != nil {
		return Result{}, err
	}
	previous, candidate := downloads[pinned.Version], downloads[spec.Version]
	if previous.Sum != pinned.Sum {
		return Result{}, fmt.Errorf("downloaded %s@%s has sum %s, pinned %s", spec.Module, pinned.Version, previous.Sum, pinned.Sum)
	}
	scratch := filepath.Join(work, "derive")
	if err := os.Mkdir(scratch, 0o700); err != nil {
		return Result{}, err
	}
	regeneration, err := deterministicio.RegenerateAdapter(ctx, deterministicio.AdapterRegenerationRequest{
		Module: spec.Module, Version: spec.Version, Sum: candidate.Sum, GoModSum: candidate.GoModSum, PreviousGoModSum: previous.GoModSum,
		PreviousModule: previous.Dir, CandidateModule: candidate.Dir, GoCommand: spec.GoCommand, Scratch: scratch,
	})
	if err != nil {
		if deterministicio.IsAdapterRegenerationBlocked(err) {
			return Result{}, &BlockedError{Err: err}
		}
		return Result{}, err
	}
	result := Result{Regeneration: regeneration, Diffs: diffs(regeneration)}
	result.StalePacks, err = stalePacks(spec.Root, spec.Module, spec.Version)
	if err != nil {
		return Result{}, err
	}
	if spec.Approval == "" {
		return result, nil
	}
	if spec.Approval != regeneration.ApprovalSHA256 {
		return Result{}, &InputError{Err: fmt.Errorf("approval %s does not match the review digest %s of %s@%s", spec.Approval, regeneration.ApprovalSHA256, spec.Module, spec.Version)}
	}
	published, err := apply(ctx, spec, regeneration, candidate.Dir)
	if err != nil {
		return Result{}, err
	}
	result.Staged = published.staged
	if spec.StageOnly {
		return result, nil
	}
	result.Applied, result.Published, result.Residual, result.Warnings = true, published.published, published.residual, published.warnings
	return result, nil
}

func diffs(regeneration deterministicio.AdapterRegeneration) []SourceDiff {
	result := make([]SourceDiff, len(regeneration.Sources))
	for index, source := range regeneration.Sources {
		result[index] = SourceDiff{Path: source.Path, Changed: source.PreviousSHA256 != source.CandidateSHA256, Diff: unifiedDiff(source.Path, source.Previous, source.Candidate)}
	}
	return result
}

// stalePacks lists the pack bindings of module that will still name its
// previous identity once the adapter pins version.
func stalePacks(root, module, version string) ([]StalePack, error) {
	names, err := filepath.Glob(filepath.Join(root, "internal", "compatibilitypack", "packs", "*.json"))
	if err != nil {
		return nil, err
	}
	stale := []StalePack{}
	for _, name := range names {
		contents, err := os.ReadFile(name)
		if err != nil {
			return nil, err
		}
		pack, err := compatibility.DecodePack(contents)
		if err != nil {
			return nil, fmt.Errorf("decode compatibility pack %s: %w", filepath.Base(name), err)
		}
		check := func(location string, bound compatibility.PackModule) {
			adapter := bound.Replacement.Adapter
			if adapter != nil && adapter.Module == module && adapter.Version != version {
				stale = append(stale, StalePack{File: filepath.Base(name), ID: pack.ID, Location: location, Platforms: slices.Clone(pack.Governance.Platforms)})
			}
		}
		for _, bound := range pack.Activation {
			check("activation "+bound.Path, bound)
		}
		for _, rule := range pack.Rules {
			check("rule "+rule.ImportPath, rule.Module)
		}
	}
	return stale, nil
}

type downloaded struct {
	Dir, Sum, GoModSum string
}

// download fetches module versions into a private module cache from outside
// every module, so no checked-in go.sum changes. The exported proxy and
// checksum settings apply.
func download(ctx context.Context, spec Spec, cache string, versions []string) (map[string]downloaded, error) {
	directory := filepath.Join(filepath.Dir(cache), "download")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, err
	}
	arguments := []string{spec.GoCommand, "mod", "download", "-json"}
	for _, version := range versions {
		arguments = append(arguments, spec.Module+"@"+version)
	}
	result, err := hostexec.Run(ctx, hostexec.Request{
		Command: arguments, Dir: directory, Env: goEnvironment(spec.Environment, "GOMODCACHE="+cache, "GOFLAGS=-modcacherw", "GO111MODULE=on"),
		Timeout: commandTimeout, TerminateGrace: time.Second, OutputLimit: commandOutputLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("run go mod download: %w", err)
	}
	if result.Stdout.Truncated || result.WatchdogTimeout || result.Termination != hostexec.TerminationExit {
		return nil, errors.New("go mod download did not finish within its bounds")
	}
	modules := map[string]downloaded{}
	decoder := json.NewDecoder(bytes.NewReader(result.Stdout.RawBytes))
	for {
		var listed struct {
			Version, Dir, Sum, GoModSum, Error string
		}
		if err := decoder.Decode(&listed); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, fmt.Errorf("decode go mod download output: %w", err)
		}
		if listed.Error != "" {
			return nil, fmt.Errorf("download %s@%s: %s", spec.Module, listed.Version, listed.Error)
		}
		modules[listed.Version] = downloaded{Dir: listed.Dir, Sum: listed.Sum, GoModSum: listed.GoModSum}
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("go mod download failed: %s", strings.TrimSpace(string(result.Stderr.RawBytes)))
	}
	for _, version := range versions {
		if modules[version].Dir == "" || modules[version].Sum == "" {
			return nil, fmt.Errorf("go mod download did not report %s@%s", spec.Module, version)
		}
	}
	return modules, nil
}

// requirePinnedGo checks that the go command is the pinned Go release, whose
// release tags select each platform's prepared package files.
func requirePinnedGo(ctx context.Context, spec Spec) error {
	result, err := hostexec.Run(ctx, hostexec.Request{
		Command: []string{spec.GoCommand, "env", "GOVERSION"}, Dir: spec.Root, Env: goEnvironment(spec.Environment),
		Timeout: time.Minute, TerminateGrace: time.Second, OutputLimit: 4096,
	})
	if err != nil {
		return fmt.Errorf("run go env: %w", err)
	}
	if result.Termination != hostexec.TerminationExit || result.ExitCode != 0 {
		return fmt.Errorf("go env GOVERSION failed: %s", strings.TrimSpace(string(result.Stderr.RawBytes)))
	}
	version, _, _ := strings.Cut(strings.TrimSpace(string(result.Stdout.RawBytes)), " ")
	if version != gomadversion.GoVersion {
		return &InputError{Err: fmt.Errorf("adapter regeneration needs the pinned %s go command; %s reports %s", gomadversion.GoVersion, spec.GoCommand, version)}
	}
	return nil
}

// goEnvironment passes the caller's environment through, minus the settings
// this package owns and those extra sets. The caller's module cache passes
// through to staged commands; only download sets a private one.
func goEnvironment(environment []string, extra ...string) []string {
	owned := map[string]bool{"GOFLAGS": true, "GOTOOLCHAIN": true, "GOWORK": true, "GO111MODULE": true, "GOMADSEED": true, "GOMAD3_CHILD_SEED": true}
	for _, entry := range extra {
		name, _, _ := strings.Cut(entry, "=")
		owned[name] = true
	}
	filtered := make([]string, 0, len(environment)+len(extra)+2)
	for _, entry := range environment {
		name, _, _ := strings.Cut(entry, "=")
		if !owned[name] {
			filtered = append(filtered, entry)
		}
	}
	filtered = append(filtered, "GOTOOLCHAIN=local", "GOWORK=off")
	return append(filtered, extra...)
}

// removeWritable removes a tree the go command made read-only.
func removeWritable(root string) {
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err == nil && entry.IsDir() {
			_ = os.Chmod(path, 0o700)
		}
		return nil
	})
	_ = os.RemoveAll(root)
}

// Verify checks the checkout's compiled adapter for module against the module
// source in moduleDirectory under every pin; the staged transaction runs it
// from the staged copy.
func Verify(ctx context.Context, module, moduleDirectory, goCommand string) error {
	return deterministicio.VerifyRegisteredAdapter(ctx, module, moduleDirectory, goCommand)
}

// Regenerable lists the adapters whose anchors Run can re-derive.
func Regenerable() []string {
	return deterministicio.RegenerableAdapters()
}
