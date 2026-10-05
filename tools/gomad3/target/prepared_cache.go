package target

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	"go.temporal.io/server/tools/gomad3/record"
	targetbuild "go.temporal.io/server/tools/gomad3/target/internal/build"
	"go.temporal.io/server/tools/gomad3/toolchain/installation"
)

const preparedTargetSchema = "gomad3.prepared-target/v2"

// maximumEmbeddedFileBytes bounds one embedded file the identity reads.
const maximumEmbeddedFileBytes = 64 << 20

// maximumPreparedTargetBytes bounds the binaries one toolchain build retains;
// the least recently restored entries leave first. A ./tests binary is about
// 150 MiB, so the bound holds a handful of large closures while the small
// fixture binaries of a test run cannot push one out.
var maximumPreparedTargetBytes = uint64(2 << 30)

// preparedTargetIdentity is every input of a go target build: a binary built
// from one identity is byte-identical to any other, so repetitions and
// workloads that share it can share the binary. The reviewed closure carries
// each package's source digests, module versions and sums, and adapter
// replacements; the identity adds what the closure does not review: the
// files packages embed, the module files the build reads, the overlay, the
// build flags the target kind and capability mode select, the toolchain, and
// the Go settings of the build environment.
type preparedTargetIdentity struct {
	Schema         string           `json:"schema"`
	GoVersion      string           `json:"go_version"`
	BuildKey       string           `json:"build_key"`
	TargetGOOS     string           `json:"target_goos"`
	TargetGOARCH   string           `json:"target_goarch"`
	BinaryVersion  string           `json:"binary_go_version"`
	Environment    []string         `json:"environment"`
	Kind           Kind             `json:"kind"`
	Package        string           `json:"package"`
	BuildTags      []string         `json:"build_tags"`
	CapabilityMode CapabilityMode   `json:"capability_mode"`
	Overlay        record.SHA256    `json:"overlay_sha256,omitempty"`
	ModFile        record.SHA256    `json:"modfile_sha256,omitempty"`
	ModuleFiles    record.SHA256    `json:"module_files_sha256"`
	Closure        record.SHA256    `json:"closure_sha256"`
	Embedded       []embeddedSource `json:"embedded"`
	Modules        []moduleSource   `json:"modules"`
}

// moduleSource binds a dependency module's language version and, for a
// module without a version (a local replacement), its go.mod contents; the
// main module's files are ModuleFiles.
type moduleSource struct {
	Path      string        `json:"path"`
	Version   string        `json:"version,omitempty"`
	GoVersion string        `json:"go_version,omitempty"`
	GoMod     record.SHA256 `json:"go_mod_sha256,omitempty"`
}

type embeddedSource struct {
	ImportPath string        `json:"import_path"`
	ForTest    string        `json:"for_test,omitempty"`
	Name       string        `json:"name"`
	SHA256     record.SHA256 `json:"sha256"`
}

// preparedTargetRecord is the cache entry beside a retained binary; restore
// trusts the binary only when it still hashes to the recorded digest.
type preparedTargetRecord struct {
	Schema   string                 `json:"schema"`
	Identity preparedTargetIdentity `json:"identity"`
	SHA256   string                 `json:"sha256"`
	Size     record.Uint64String    `json:"size"`
}

// preparedTargetCacheRoot places the cache beside the toolchain build's
// target build cache; tests redirect it.
var preparedTargetCacheRoot = func(toolchain installation.Description) string {
	return toolchain.PinnedBuild().PreparedTargets()
}

type preparedTargetCache struct {
	root     string
	identity preparedTargetIdentity
	digest   string
}

func openPreparedTargetCache(spec Spec, tags []string, toolchain pinnedToolchain, commandDirectory, packageArgument string, review CapabilityReview, packages []listedPackage) (*preparedTargetCache, error) {
	identity, err := newPreparedTargetIdentity(spec, tags, toolchain.ToolchainIdentity, commandDirectory, packageArgument, review, packages)
	if err != nil {
		return nil, err
	}
	encoded, err := canonicaljson.CanonicalJSON(identity)
	if err != nil {
		return nil, fmt.Errorf("encode prepared target identity: %w", err)
	}
	digest := sha256.Sum256(encoded)
	root := preparedTargetCacheRoot(toolchain.installation)
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create prepared target cache: %w", err)
	}
	return &preparedTargetCache{root: root, identity: identity, digest: hex.EncodeToString(digest[:])}, nil
}

func newPreparedTargetIdentity(spec Spec, tags []string, toolchain ToolchainIdentity, commandDirectory, packageArgument string, review CapabilityReview, packages []listedPackage) (preparedTargetIdentity, error) {
	closure, err := canonicaljson.CanonicalJSON(review.Closure)
	if err != nil {
		return preparedTargetIdentity{}, fmt.Errorf("encode reviewed closure: %w", err)
	}
	identity := preparedTargetIdentity{
		Schema: preparedTargetSchema, GoVersion: toolchain.GoVersion, BuildKey: toolchain.BuildKey,
		TargetGOOS: toolchain.TargetGOOS, TargetGOARCH: toolchain.TargetGOARCH, BinaryVersion: targetbuild.BinaryGoVersion(toolchain.GoVersion),
		Environment: goEnvironment(targetbuild.Environment()), Kind: spec.Kind, Package: packageArgument,
		BuildTags: append([]string{}, tags...), CapabilityMode: spec.CapabilityMode, Closure: record.HashBytes(closure), Embedded: []embeddedSource{},
	}
	if spec.BuildOverlay != "" {
		identity.Overlay, err = overlayDigest(spec.BuildOverlay, commandDirectory)
		if err != nil {
			return preparedTargetIdentity{}, err
		}
	}
	if spec.BuildModFile != "" {
		modFile := spec.BuildModFile
		if !filepath.IsAbs(modFile) {
			modFile = filepath.Join(commandDirectory, modFile)
		}
		identity.ModFile, err = filesDigest(modFile, strings.TrimSuffix(modFile, ".mod")+".sum")
		if err != nil {
			return preparedTargetIdentity{}, err
		}
	}
	identity.ModuleFiles, err = filesDigest(filepath.Join(commandDirectory, "go.mod"), filepath.Join(commandDirectory, "go.sum"))
	if err != nil {
		return preparedTargetIdentity{}, err
	}
	for _, pkg := range packages {
		if pkg.Standard {
			continue
		}
		for _, name := range sortedSetCopy(append(append(append([]string{}, pkg.EmbedFiles...), pkg.TestEmbedFiles...), pkg.XTestEmbedFiles...)) {
			if pkg.Dir == "" || filepath.IsAbs(name) || strings.HasPrefix(filepath.ToSlash(name), "../") {
				return preparedTargetIdentity{}, fmt.Errorf("inspect target embedded file %s: invalid path %q", pkg.ImportPath, name)
			}
			contents, err := hostfs.ReadBounded(filepath.Join(pkg.Dir, filepath.FromSlash(name)), maximumEmbeddedFileBytes)
			if err != nil {
				return preparedTargetIdentity{}, fmt.Errorf("inspect target embedded file %s: unreadable %s: %w", pkg.ImportPath, name, err)
			}
			identity.Embedded = append(identity.Embedded, embeddedSource{ImportPath: pkg.ImportPath, ForTest: pkg.ForTest, Name: filepath.ToSlash(name), SHA256: record.HashBytes(contents)})
		}
	}
	identity.Modules, err = moduleSources(packages)
	if err != nil {
		return preparedTargetIdentity{}, err
	}
	sort.Slice(identity.Embedded, func(i, j int) bool {
		left, right := identity.Embedded[i], identity.Embedded[j]
		if left.ImportPath != right.ImportPath {
			return left.ImportPath < right.ImportPath
		}
		if left.ForTest != right.ForTest {
			return left.ForTest < right.ForTest
		}
		return left.Name < right.Name
	})
	return identity, nil
}

func moduleSources(packages []listedPackage) ([]moduleSource, error) {
	byPath := map[string]moduleSource{}
	for _, pkg := range packages {
		if pkg.Standard || pkg.Module == nil || pkg.Module.Main {
			continue
		}
		effective := pkg.Module
		if effective.Replace != nil {
			effective = effective.Replace
		}
		source := moduleSource{Path: pkg.Module.Path, Version: effective.Version, GoVersion: cmp.Or(effective.GoVersion, pkg.Module.GoVersion)}
		if effective.Version == "" {
			if effective.GoMod == "" {
				return nil, fmt.Errorf("inspect target module %s: local module has no go.mod", pkg.Module.Path)
			}
			digest, err := filesDigest(effective.GoMod)
			if err != nil {
				return nil, fmt.Errorf("inspect target module %s: %w", pkg.Module.Path, err)
			}
			source.GoMod = digest
		}
		byPath[source.Path] = source
	}
	sources := make([]moduleSource, 0, len(byPath))
	for _, path := range slices.Sorted(maps.Keys(byPath)) {
		sources = append(sources, byPath[path])
	}
	return sources, nil
}

// goEnvironment keeps the Go settings of the build environment, which select
// code generation (GOAMD64, GOARM64, GOFIPS140, ...), and drops the rest.
func goEnvironment(environment []string) []string {
	settings := make([]string, 0, len(environment))
	for _, entry := range environment {
		if strings.HasPrefix(entry, "GO") {
			settings = append(settings, entry)
		}
	}
	sort.Strings(settings)
	return settings
}

func overlayDigest(path, commandDirectory string) (record.SHA256, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(commandDirectory, path)
	}
	overlay, err := loadBuildOverlay(path, commandDirectory)
	if err != nil {
		return "", err
	}
	replacements := make([]string, 0, len(overlay))
	for original := range overlay {
		replacements = append(replacements, original)
	}
	sort.Strings(replacements)
	hasher := sha256.New()
	for _, original := range replacements {
		contents, err := hostfs.ReadBounded(overlay[original], maximumCapabilitySourceBytes)
		if err != nil {
			return "", fmt.Errorf("read target build overlay replacement: %w", err)
		}
		digest := sha256.Sum256(contents)
		_, _ = hasher.Write(fmt.Appendf(nil, "%s\x00%x\n", original, digest))
	}
	return record.SHA256("sha256:" + hex.EncodeToString(hasher.Sum(nil))), nil
}

// filesDigest binds the contents of the named files; an absent file is a
// distinct state, not an error, since a module may have no go.sum.
func filesDigest(paths ...string) (record.SHA256, error) {
	hasher := sha256.New()
	for _, path := range paths {
		contents, err := hostfs.ReadBounded(path, maximumCapabilitySourceBytes)
		if errors.Is(err, os.ErrNotExist) {
			_, _ = hasher.Write(fmt.Appendf(nil, "%s\x00absent\n", filepath.Base(path)))
			continue
		}
		if err != nil {
			return "", fmt.Errorf("read target module file: %w", err)
		}
		digest := sha256.Sum256(contents)
		_, _ = hasher.Write(fmt.Appendf(nil, "%s\x00%x\n", filepath.Base(path), digest))
	}
	return record.SHA256("sha256:" + hex.EncodeToString(hasher.Sum(nil))), nil
}

func (cache *preparedTargetCache) entry() string {
	return filepath.Join(cache.root, cache.digest)
}

// restore copies the retained binary of this identity to targetPath and
// reports whether it did. A retained binary that no longer hashes to its
// record is discarded rather than trusted.
func (cache *preparedTargetCache) restore(targetPath string) (bool, error) {
	entry := cache.entry()
	recordPath := filepath.Join(entry, "record.json")
	contents, err := hostfs.ReadBounded(recordPath, 1<<20)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, cache.discard(fmt.Errorf("read prepared target record: %w", err))
	}
	var retained preparedTargetRecord
	if err := canonicaljson.StrictDecode(bytes.TrimSuffix(contents, []byte{'\n'}), &retained); err != nil {
		return false, cache.discard(fmt.Errorf("decode prepared target record: %w", err))
	}
	recordedIdentity, err := canonicaljson.CanonicalJSON(retained.Identity)
	if err != nil {
		return false, cache.discard(err)
	}
	expectedIdentity, err := canonicaljson.CanonicalJSON(cache.identity)
	if err != nil {
		return false, err
	}
	if retained.Schema != preparedTargetSchema || !bytes.Equal(recordedIdentity, expectedIdentity) {
		return false, cache.discard(errors.New("prepared target record does not describe this identity"))
	}
	if err := copyRegularFile(filepath.Join(entry, "target"), targetPath); err != nil {
		return false, cache.discard(errors.Join(fmt.Errorf("restore prepared target: %w", err), os.Remove(targetPath)))
	}
	hash, size, err := hashRegularFile(targetPath)
	if err != nil || hash != retained.SHA256 || size != uint64(retained.Size) {
		return false, cache.discard(errors.Join(errors.New("prepared target does not hash to its record"), err, os.Remove(targetPath)))
	}
	now := time.Now()
	if err := os.Chtimes(recordPath, now, now); err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("touch prepared target record: %w", err)
	}
	return true, nil
}

// discard removes a cache entry that cannot be trusted, so the caller builds;
// only a failure to remove it is an error.
func (cache *preparedTargetCache) discard(reason error) error {
	if err := os.RemoveAll(cache.entry()); err != nil {
		return errors.Join(fmt.Errorf("discard prepared target: %w", err), reason)
	}
	return nil
}

// publish retains a copy of the freshly built binary under this identity. A
// concurrent publication of the same identity is byte-identical, so whichever
// entry lands first stays.
func (cache *preparedTargetCache) publish(targetPath string) (retErr error) {
	hash, size, err := hashRegularFile(targetPath)
	if err != nil {
		return fmt.Errorf("hash prepared target: %w", err)
	}
	staging, err := os.MkdirTemp(cache.root, ".publish-")
	if err != nil {
		return fmt.Errorf("stage prepared target: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(staging); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("remove prepared target staging: %w", err))
		}
	}()
	if err := copyRegularFile(targetPath, filepath.Join(staging, "target")); err != nil {
		return fmt.Errorf("stage prepared target: %w", err)
	}
	if err := os.Chmod(filepath.Join(staging, "target"), 0o500); err != nil {
		return fmt.Errorf("stage prepared target: %w", err)
	}
	encoded, err := canonicaljson.CanonicalJSON(preparedTargetRecord{Schema: preparedTargetSchema, Identity: cache.identity, SHA256: hash, Size: record.Uint64String(size)})
	if err != nil {
		return fmt.Errorf("encode prepared target record: %w", err)
	}
	if err := writePreparedFile(filepath.Join(staging, "record.json"), append(encoded, '\n'), 0o400); err != nil {
		return fmt.Errorf("stage prepared target record: %w", err)
	}
	if err := os.Rename(staging, cache.entry()); err != nil {
		if _, statErr := os.Stat(filepath.Join(cache.entry(), "record.json")); statErr == nil {
			return nil
		}
		return fmt.Errorf("publish prepared target: %w", err)
	}
	return cache.evict()
}

// evict keeps the most recently restored or published entries within
// maximumPreparedTargetBytes, never the entry this cache addresses.
func (cache *preparedTargetCache) evict() error {
	entries, err := os.ReadDir(cache.root)
	if err != nil {
		return fmt.Errorf("list prepared targets: %w", err)
	}
	type retained struct {
		name     string
		size     uint64
		modified time.Time
	}
	var total uint64
	candidates := make([]retained, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		info, err := os.Stat(filepath.Join(cache.root, entry.Name(), "record.json"))
		if err != nil {
			continue
		}
		target, err := os.Stat(filepath.Join(cache.root, entry.Name(), "target"))
		if err != nil || target.Size() < 0 {
			continue
		}
		total += uint64(target.Size())
		if entry.Name() != cache.digest {
			candidates = append(candidates, retained{name: entry.Name(), size: uint64(target.Size()), modified: info.ModTime()})
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].modified.Before(candidates[j].modified) })
	for _, candidate := range candidates {
		if total <= maximumPreparedTargetBytes {
			break
		}
		if err := os.RemoveAll(filepath.Join(cache.root, candidate.name)); err != nil {
			return fmt.Errorf("evict prepared target: %w", err)
		}
		total -= candidate.size
	}
	return nil
}
