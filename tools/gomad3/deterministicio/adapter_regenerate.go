package deterministicio

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

// AdapterRegenerationSchema names the canonical review an adapter
// regeneration's approval digest covers.
const AdapterRegenerationSchema = "gomad3.adapter-regeneration-review/v1"

// AdapterRegenerationRequest names an adapter and the exact module version its
// anchors are re-derived for. Both module directories hold extracted module
// sources, for example from a private module cache.
type AdapterRegenerationRequest struct {
	Module string
	// Version, Sum, and GoModSum identify the candidate module exactly.
	Version, Sum, GoModSum string
	// PreviousGoModSum is the pinned version's go.mod hash, so pinned test
	// data that records it can be updated.
	PreviousGoModSum string
	// PreviousModule holds the pinned version; CandidateModule the new one.
	PreviousModule, CandidateModule string
	// GoCommand is the pinned Go release, which selects each platform's
	// prepared package files.
	GoCommand string
	// Scratch is an empty directory for the candidate replacement copy.
	Scratch string
}

// AdapterAnchors are the exact identities an adapter pins for one module
// version.
type AdapterAnchors struct {
	Version                          string            `json:"version"`
	Sum                              string            `json:"sum"`
	GoModSum                         string            `json:"go_mod_sum,omitempty"`
	OriginalSourceInventorySHA256    string            `json:"original_source_inventory_sha256"`
	ReplacementSourceInventorySHA256 string            `json:"replacement_source_inventory_sha256"`
	PreparedPackage                  string            `json:"prepared_package"`
	PreparedSourceSetSHA256          map[string]string `json:"prepared_source_set_sha256"`
	Rewrites                         []RewriteAnchors  `json:"rewrites"`
}

// RewriteAnchors are one rewritten file's identities.
type RewriteAnchors struct {
	Path              string `json:"path"`
	SourceSHA256      string `json:"source_sha256"`
	Base              string `json:"base,omitempty"`
	BaseSHA256        string `json:"base_sha256,omitempty"`
	ReplacementSHA256 string `json:"replacement_sha256"`
}

// ChangedSource is one upstream file an adapter rewrite reads, in the pinned
// and the candidate version. Unchanged files are listed too, so the review
// names every input of the rewrite.
type ChangedSource struct {
	Path            string `json:"path"`
	PreviousSHA256  string `json:"previous_sha256"`
	CandidateSHA256 string `json:"candidate_sha256"`
	Previous        []byte `json:"-"`
	Candidate       []byte `json:"-"`
}

// AdapterRegeneration is the reviewed outcome of re-deriving an adapter's
// anchors. ApprovalSHA256 covers everything but the source bytes, which the
// source digests bind.
type AdapterRegeneration struct {
	Schema         string          `json:"schema"`
	Module         string          `json:"module"`
	Previous       AdapterAnchors  `json:"previous"`
	Proposed       AdapterAnchors  `json:"proposed"`
	Sources        []ChangedSource `json:"sources"`
	ApprovalSHA256 string          `json:"-"`
}

// AdapterSourceMissingError reports a rewritten file that the candidate
// version no longer provides as a regular file.
type AdapterSourceMissingError struct {
	Module, Version, Path string
}

func (err *AdapterSourceMissingError) Error() string {
	return fmt.Sprintf("%s@%s no longer provides rewritten file %s as a regular file", err.Module, err.Version, err.Path)
}

// AdapterNotRegenerableError reports an adapter whose preparation is not
// expressed as anchored rewrites, so its anchors are re-derived by hand.
type AdapterNotRegenerableError struct {
	Module string
}

func (err *AdapterNotRegenerableError) Error() string {
	return fmt.Sprintf("adapter %s is not prepared from exact-occurrence anchors; regenerate it by hand", err.Module)
}

type AdapterRegenerationBlockedError struct{ Err error }

func (err *AdapterRegenerationBlockedError) Error() string { return err.Err.Error() }
func (err *AdapterRegenerationBlockedError) Unwrap() error { return err.Err }

// IsAdapterRegenerationBlocked reports whether err means the candidate cannot
// be regenerated without a person editing the adapter: an anchor that no
// longer matches exactly once, or a rewritten file that is gone.
func IsAdapterRegenerationBlocked(err error) bool {
	var anchor *AnchorMismatchError
	var missing *AdapterSourceMissingError
	var custom *AdapterNotRegenerableError
	var blocked *AdapterRegenerationBlockedError
	return errors.As(err, &anchor) || errors.As(err, &missing) || errors.As(err, &custom) || errors.As(err, &blocked)
}

// RegenerableAdapters lists the registered adapters whose anchors the
// regeneration command can re-derive.
func RegenerableAdapters() []string {
	var modules []string
	for _, definition := range deterministicAdapters.definitions {
		if definition.implementation.rewritten != nil || definition.identity.Module == libcModulePath {
			modules = append(modules, definition.identity.Module)
		}
	}
	return modules
}

func registeredRewrittenModule(module string) (rewrittenModule, error) {
	for _, definition := range deterministicAdapters.definitions {
		if definition.identity.Module != module {
			continue
		}
		if definition.implementation.rewritten == nil {
			return rewrittenModule{}, &AdapterNotRegenerableError{Module: module}
		}
		return *definition.implementation.rewritten, nil
	}
	return rewrittenModule{}, fmt.Errorf("no adapter is registered for %s", module)
}

// RegenerateAdapter re-derives the registered adapter's anchors for the
// candidate module. Each rewrite is applied by its existing exact-occurrence
// anchors; nothing is written outside request.Scratch.
func RegenerateAdapter(ctx context.Context, request AdapterRegenerationRequest) (AdapterRegeneration, error) {
	if request.Module == libcModulePath {
		return regenerateLibcAdapter(ctx, request)
	}
	spec, err := registeredRewrittenModule(request.Module)
	if err != nil {
		return AdapterRegeneration{}, err
	}
	return regenerateRewrittenModule(ctx, spec, request)
}

func regenerateRewrittenModule(ctx context.Context, spec rewrittenModule, request AdapterRegenerationRequest) (AdapterRegeneration, error) {
	if request.Module != spec.module || request.Version == "" || request.Sum == "" || request.GoCommand == "" || request.Scratch == "" {
		return AdapterRegeneration{}, errors.New("adapter regeneration request is incomplete")
	}
	if request.Version == spec.version {
		return AdapterRegeneration{}, fmt.Errorf("%s is already pinned at %s", spec.module, spec.version)
	}
	if err := verifyAdapterModuleInventory(spec.module, request.PreviousModule, spec.originalInventorySHA256); err != nil {
		return AdapterRegeneration{}, err
	}
	var err error
	previous := pinnedAnchors(spec)
	previous.GoModSum = request.PreviousGoModSum
	proposed := AdapterAnchors{
		Version: request.Version, Sum: request.Sum, GoModSum: request.GoModSum,
		PreparedPackage: spec.preparedPackage, PreparedSourceSetSHA256: map[string]string{},
	}
	proposed.OriginalSourceInventorySHA256, err = digestAdapterSourceInventory(request.CandidateModule)
	if err != nil {
		return AdapterRegeneration{}, fmt.Errorf("hash candidate %s source inventory: %w", spec.module, err)
	}
	sources := map[string]ChangedSource{}
	readPair := func(relative string) ([]byte, error) {
		candidate, err := readCandidateSource(spec.module, request.Version, request.CandidateModule, relative)
		if err != nil {
			return nil, err
		}
		if _, found := sources[relative]; !found {
			before, err := readAdapterSource(spec.module, request.PreviousModule, relative)
			if err != nil {
				return nil, err
			}
			sources[relative] = ChangedSource{
				Path: relative, PreviousSHA256: digestBytes(before), CandidateSHA256: digestBytes(candidate),
				Previous: before, Candidate: candidate,
			}
		}
		return candidate, nil
	}
	replacements := make(map[string][]byte, len(spec.rewrites))
	derived := make([]sourceRewrite, len(spec.rewrites))
	for index, rewrite := range spec.rewrites {
		contents, err := readPair(rewrite.path)
		if err != nil {
			return AdapterRegeneration{}, err
		}
		next := rewrite
		next.sourceSHA256 = digestBytes(contents)
		input, inputPath := contents, rewrite.path
		if rewrite.base != "" {
			input, err = readPair(rewrite.base)
			if err != nil {
				return AdapterRegeneration{}, err
			}
			inputPath = rewrite.base
			next.baseSHA256 = digestBytes(input)
		}
		result, err := applyAdapterAnchors(spec.module, inputPath, rewrite.rewrites, input)
		if err != nil {
			return AdapterRegeneration{}, err
		}
		next.replacementSHA256 = digestBytes(result)
		replacements[rewrite.path] = result
		derived[index] = next
		proposed.Rewrites = append(proposed.Rewrites, rewriteAnchors(next))
	}
	replacementRoot := filepath.Join(request.Scratch, spec.replacementDirectory)
	if err := copyAdapterModule(request.CandidateModule, replacementRoot, replacements, defaultAdapterCopyLimits); err != nil {
		return AdapterRegeneration{}, fmt.Errorf("copy candidate %s adapter module: %w", spec.module, err)
	}
	proposed.ReplacementSourceInventorySHA256, err = digestAdapterSourceInventory(replacementRoot)
	if err != nil {
		return AdapterRegeneration{}, fmt.Errorf("hash candidate %s replacement inventory: %w", spec.module, err)
	}
	packageDirectory := replacementRoot
	if spec.preparedPackage != spec.module {
		relative, found := strings.CutPrefix(spec.preparedPackage, spec.module+"/")
		if !found {
			return AdapterRegeneration{}, fmt.Errorf("%s prepared package %s is outside the module", spec.module, spec.preparedPackage)
		}
		packageDirectory = filepath.Join(replacementRoot, filepath.FromSlash(relative))
	}
	for _, platform := range sortedKeys(spec.preparedSourceSetSHA256ByHost) {
		goos, goarch, _ := strings.Cut(platform, "/")
		proposed.PreparedSourceSetSHA256[platform], err = target.AdapterPreparedSourceSetSHA256(ctx, request.GoCommand, packageDirectory, spec.preparedPackage, goos, goarch)
		if err != nil {
			return AdapterRegeneration{}, err
		}
	}
	// The proposed anchors must satisfy the same fail-closed preparation the
	// build runs, so an approval never records anchors the build rejects.
	candidateSpec := spec
	candidateSpec.version, candidateSpec.sum = request.Version, request.Sum
	candidateSpec.cacheElements = []string{request.CandidateModule}
	candidateSpec.originalInventorySHA256 = proposed.OriginalSourceInventorySHA256
	candidateSpec.replacementInventorySHA256 = proposed.ReplacementSourceInventorySHA256
	candidateSpec.preparedSourceSetSHA256ByHost = proposed.PreparedSourceSetSHA256
	candidateSpec.rewrites = derived
	check, err := os.MkdirTemp(request.Scratch, "check-")
	if err != nil {
		return AdapterRegeneration{}, err
	}
	if _, err := prepareRewrittenModule("", check, adapterIdentity(candidateSpec), candidateSpec); err != nil {
		return AdapterRegeneration{}, fmt.Errorf("proposed %s anchors fail preparation: %w", spec.module, err)
	}
	regeneration := AdapterRegeneration{Schema: AdapterRegenerationSchema, Module: spec.module, Previous: previous, Proposed: proposed}
	for _, path := range sortedKeys(sources) {
		regeneration.Sources = append(regeneration.Sources, sources[path])
	}
	regeneration.ApprovalSHA256, err = regeneration.approval()
	if err != nil {
		return AdapterRegeneration{}, err
	}
	return regeneration, nil
}

func (regeneration AdapterRegeneration) approval() (string, error) {
	encoded, err := canonicaljson.CanonicalJSON(regeneration)
	if err != nil {
		return "", fmt.Errorf("encode adapter regeneration review: %w", err)
	}
	return string(record.DomainHash(AdapterRegenerationSchema, encoded)), nil
}

func pinnedAnchors(spec rewrittenModule) AdapterAnchors {
	anchors := AdapterAnchors{
		Version: spec.version, Sum: spec.sum,
		OriginalSourceInventorySHA256: spec.originalInventorySHA256, ReplacementSourceInventorySHA256: spec.replacementInventorySHA256,
		PreparedPackage: spec.preparedPackage, PreparedSourceSetSHA256: map[string]string{},
	}
	for platform, pin := range spec.preparedSourceSetSHA256ByHost {
		anchors.PreparedSourceSetSHA256[platform] = pin
	}
	for _, rewrite := range spec.rewrites {
		anchors.Rewrites = append(anchors.Rewrites, rewriteAnchors(rewrite))
	}
	return anchors
}

func rewriteAnchors(rewrite sourceRewrite) RewriteAnchors {
	return RewriteAnchors{Path: rewrite.path, SourceSHA256: rewrite.sourceSHA256, Base: rewrite.base, BaseSHA256: rewrite.baseSHA256, ReplacementSHA256: rewrite.replacementSHA256}
}

func readCandidateSource(module, version, moduleRoot, relative string) ([]byte, error) {
	info, err := os.Lstat(filepath.Join(moduleRoot, filepath.FromSlash(relative)))
	if errors.Is(err, fs.ErrNotExist) || err == nil && !info.Mode().IsRegular() {
		return nil, &AdapterSourceMissingError{Module: module, Version: version, Path: relative}
	}
	if err != nil {
		return nil, err
	}
	return readAdapterSource(module, moduleRoot, relative)
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return slices.Clip(keys)
}

// VerifyRegisteredAdapter checks a checkout's adapter for module against the
// module source in moduleDirectory: the registered identity from the version
// descriptor, every pinned digest the preparation checks, and each platform's
// prepared source-set pin, recomputed from the prepared replacement.
func VerifyRegisteredAdapter(ctx context.Context, module, moduleDirectory, goCommand string) error {
	if module == libcModulePath {
		return verifyLibcAdapter(ctx, moduleDirectory, goCommand)
	}
	spec, err := registeredRewrittenModule(module)
	if err != nil {
		return err
	}
	var identity gomadversion.AdapterIdentity
	for _, definition := range deterministicAdapters.definitions {
		if definition.identity.Module == module {
			identity = definition.identity
		}
	}
	scratch, err := os.MkdirTemp("", "gomad3-adapter-verify-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(scratch)
	local := spec
	local.cacheElements = []string{moduleDirectory}
	prepared, err := prepareRewrittenModule("", scratch, identity, local)
	if err != nil {
		return err
	}
	directory := prepared.replacement
	if spec.preparedPackage != spec.module {
		directory = filepath.Join(directory, filepath.FromSlash(strings.TrimPrefix(spec.preparedPackage, spec.module+"/")))
	}
	for _, platform := range sortedKeys(spec.preparedSourceSetSHA256ByHost) {
		goos, goarch, _ := strings.Cut(platform, "/")
		got, err := target.AdapterPreparedSourceSetSHA256(ctx, goCommand, directory, spec.preparedPackage, goos, goarch)
		if err != nil {
			return err
		}
		if want := spec.preparedSourceSetSHA256ByHost[platform]; got != want {
			return fmt.Errorf("%s %s prepared source set is %s, pinned %s", module, platform, got, want)
		}
	}
	return nil
}
