package deterministicio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go.temporal.io/server/tools/gomad3/target"
)

var libcRegenerationSources = []string{
	"libc_darwin.go", "libc_darwin_arm64.go", "libc_unix.go",
	"syscall_musl.go", "libc_musl.go", "libc_musl_linux_amd64.go",
}

// Modernc libc is rewritten through syntax-aware transformations rather than
// the ordinary literal-anchor table. It still uses the same approval digest
// and staged publication transaction as every other adapter.
func regenerateLibcAdapter(ctx context.Context, request AdapterRegenerationRequest) (AdapterRegeneration, error) {
	if request.Version == "" || request.Sum == "" || request.PreviousModule == "" || request.CandidateModule == "" || request.GoCommand == "" || request.Scratch == "" {
		return AdapterRegeneration{}, errors.New("libc regeneration request is incomplete")
	}
	previousRewrites, _, err := rewriteLibcModule(request.PreviousModule)
	if err != nil {
		return AdapterRegeneration{}, fmt.Errorf("verify pinned libc rewrite: %w", err)
	}
	candidateRewrites, _, err := rewriteLibcModuleUnpinned(request.CandidateModule)
	if err != nil {
		return AdapterRegeneration{}, &AdapterRegenerationBlockedError{Err: fmt.Errorf("derive candidate libc rewrite: %w", err)}
	}
	previous := AdapterAnchors{
		Version: libcPinnedVersion(), Sum: libcPinnedSum(), GoModSum: request.PreviousGoModSum,
		PreparedPackage: libcModulePath, PreparedSourceSetSHA256: map[string]string{},
	}
	proposed := AdapterAnchors{
		Version: request.Version, Sum: request.Sum, GoModSum: request.GoModSum,
		PreparedPackage: libcModulePath, PreparedSourceSetSHA256: map[string]string{},
	}
	for platform, digest := range libcPreparedSourceSetSHA256ByHost {
		previous.PreparedSourceSetSHA256[platform] = digest
	}
	previous.OriginalSourceInventorySHA256, err = digestAdapterSourceInventory(request.PreviousModule)
	if err != nil {
		return AdapterRegeneration{}, err
	}
	proposed.OriginalSourceInventorySHA256, err = digestAdapterSourceInventory(request.CandidateModule)
	if err != nil {
		return AdapterRegeneration{}, err
	}
	previousCopy := filepath.Join(request.Scratch, "previous-libc")
	if _, err := copyLibcModule(request.PreviousModule, previousCopy, previousRewrites); err != nil {
		return AdapterRegeneration{}, err
	}
	previous.ReplacementSourceInventorySHA256, err = digestAdapterSourceInventory(previousCopy)
	if err != nil {
		return AdapterRegeneration{}, err
	}
	candidateCopy := filepath.Join(request.Scratch, "modernc-libc")
	if _, err := copyLibcModule(request.CandidateModule, candidateCopy, candidateRewrites); err != nil {
		return AdapterRegeneration{}, err
	}
	proposed.ReplacementSourceInventorySHA256, err = digestAdapterSourceInventory(candidateCopy)
	if err != nil {
		return AdapterRegeneration{}, err
	}
	for _, platform := range sortedKeys(previous.PreparedSourceSetSHA256) {
		goos, goarch, _ := strings.Cut(platform, "/")
		proposed.PreparedSourceSetSHA256[platform], err = target.AdapterPreparedSourceSetSHA256(ctx, request.GoCommand, candidateCopy, libcModulePath, goos, goarch)
		if err != nil {
			return AdapterRegeneration{}, err
		}
	}
	regeneration := AdapterRegeneration{Schema: AdapterRegenerationSchema, Module: libcModulePath, Previous: previous, Proposed: proposed}
	for _, path := range libcRegenerationSources {
		oldSource, err := readAdapterSource(libcModulePath, request.PreviousModule, path)
		if err != nil {
			return AdapterRegeneration{}, err
		}
		newSource, err := readCandidateSource(libcModulePath, request.Version, request.CandidateModule, path)
		if err != nil {
			return AdapterRegeneration{}, err
		}
		previous.Rewrites = append(previous.Rewrites, RewriteAnchors{Path: path, SourceSHA256: digestBytes(oldSource), ReplacementSHA256: digestBytes(previousRewrites[path])})
		proposed.Rewrites = append(proposed.Rewrites, RewriteAnchors{Path: path, SourceSHA256: digestBytes(newSource), ReplacementSHA256: digestBytes(candidateRewrites[path])})
		regeneration.Sources = append(regeneration.Sources, ChangedSource{
			Path: path, PreviousSHA256: digestBytes(oldSource), CandidateSHA256: digestBytes(newSource),
			Previous: oldSource, Candidate: newSource,
		})
	}
	regeneration.Previous, regeneration.Proposed = previous, proposed
	regeneration.ApprovalSHA256, err = regeneration.approval()
	return regeneration, err
}

func libcPinnedVersion() string {
	for _, adapter := range deterministicAdapters.definitions {
		if adapter.identity.Module == libcModulePath {
			return adapter.identity.Version
		}
	}
	return ""
}

func libcPinnedSum() string {
	for _, adapter := range deterministicAdapters.definitions {
		if adapter.identity.Module == libcModulePath {
			return adapter.identity.Sum
		}
	}
	return ""
}

func (regeneration AdapterRegeneration) libcSourceEdits(root string) (map[string][]byte, error) {
	path := filepath.Join(root, "deterministicio", "libc_adapter.go")
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(regeneration.Previous.Rewrites) != len(libcRegenerationSources) || len(regeneration.Proposed.Rewrites) != len(libcRegenerationSources) {
		return nil, errors.New("libc regeneration has incomplete source evidence")
	}
	for index, name := range libcRegenerationSources {
		old, next := regeneration.Previous.Rewrites[index], regeneration.Proposed.Rewrites[index]
		if old.Path != name || next.Path != name || old.SourceSHA256 == "" || next.SourceSHA256 == "" {
			return nil, fmt.Errorf("libc regeneration source %s is inconsistent", name)
		}
		contents, err = replaceLibcLiteral(contents, old.SourceSHA256, next.SourceSHA256)
		if err != nil {
			return nil, fmt.Errorf("replace libc source pin %s: %w", name, err)
		}
	}
	for _, platform := range sortedKeys(regeneration.Previous.PreparedSourceSetSHA256) {
		contents, err = replaceLibcLiteral(contents, regeneration.Previous.PreparedSourceSetSHA256[platform], regeneration.Proposed.PreparedSourceSetSHA256[platform])
		if err != nil {
			return nil, fmt.Errorf("replace libc prepared source set %s: %w", platform, err)
		}
	}
	formatted, err := format.Source(contents)
	if err != nil {
		return nil, err
	}
	return map[string][]byte{"deterministicio/libc_adapter.go": formatted}, nil
}

func replaceLibcLiteral(contents []byte, old, next string) ([]byte, error) {
	if old == next {
		return contents, nil
	}
	if old == "" || next == "" {
		return nil, errors.New("libc pin is empty")
	}
	before, after := []byte(strconv.Quote(old)), []byte(strconv.Quote(next))
	if count := bytes.Count(contents, before); count != 1 {
		return nil, fmt.Errorf("libc pin %s occurs %d times, want exactly once", old, count)
	}
	return bytes.Replace(contents, before, after, 1), nil
}

func verifyLibcAdapter(ctx context.Context, moduleDirectory, goCommand string) (retErr error) {
	rewrites, _, err := rewriteLibcModule(moduleDirectory)
	if err != nil {
		return err
	}
	scratch, err := os.MkdirTemp("", "gomad3-libc-verify-")
	if err != nil {
		return err
	}
	defer func() {
		if cleanupErr := os.RemoveAll(scratch); cleanupErr != nil {
			if retErr == nil {
				retErr = cleanupErr
			} else {
				retErr = errors.Join(retErr, cleanupErr)
			}
		}
	}()
	replacement := filepath.Join(scratch, "modernc-libc")
	if _, err := copyLibcModule(moduleDirectory, replacement, rewrites); err != nil {
		return err
	}
	for platform, pinned := range libcPreparedSourceSetSHA256ByHost {
		goos, goarch, _ := strings.Cut(platform, "/")
		actual, err := target.AdapterPreparedSourceSetSHA256(ctx, goCommand, replacement, libcModulePath, goos, goarch)
		if err != nil {
			return err
		}
		if actual != pinned {
			return fmt.Errorf("libc prepared source set %s has %s, want %s", platform, actual, pinned)
		}
	}
	return nil
}
