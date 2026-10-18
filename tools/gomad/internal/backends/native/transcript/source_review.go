package transcript

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	nativebuild "go.temporal.io/server/tools/gomad/internal/backends/native/build"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
)

// SourceReviewProfile derives a declared platform's source identity without
// creating an executable profile or changing Default's host admission.
func SourceReviewProfile(goos, goarch string) (record.IOContract, error) {
	platform := goos + "/" + goarch
	if !slices.Contains(BoundaryPlatforms(), platform) {
		return record.IOContract{}, fmt.Errorf("unsupported source review platform %s", platform)
	}
	definition := *deterministicProfile.definition
	definition.target.GOOS, definition.target.GOARCH = goos, goarch
	return mustSpec(definition).Identity(), nil
}

// PrepareSourceReviewAdapters keeps exact rewrite and inventory verification,
// then verifies each prepared package against the declared platform's pin.
func PrepareSourceReviewAdapters(ctx context.Context, spec target.Spec, goCommand, goos, goarch string) (target.Spec, []BuildAdapter, error) {
	profile, err := SourceReviewProfile(goos, goarch)
	if err != nil {
		return target.Spec{}, nil, err
	}
	if len(spec.AdapterReplacements) != 0 {
		return target.Spec{}, nil, errors.New("source review already contains adapter replacement evidence")
	}
	workingDirectory, err := nativebuild.ModuleDirectory(spec)
	if err != nil {
		return target.Spec{}, nil, err
	}
	selected, _, err := deterministicAdapters.selected(workingDirectory)
	if err != nil {
		return target.Spec{}, nil, err
	}
	if len(selected) == 0 {
		return spec, []BuildAdapter{}, nil
	}
	if _, err := requireAdapterSums(workingDirectory, selected); err != nil {
		return target.Spec{}, nil, err
	}
	// These stock-Go operations resolve sources only; no patched installation
	// or running-host native profile is required.
	sourceRoot, err := nativebuild.SourceReviewGoRoot(ctx, goCommand)
	if err != nil {
		return target.Spec{}, nil, err
	}
	for _, adapter := range selected {
		identity := target.ModuleIdentity{Path: adapter.identity.Module, Version: adapter.identity.Version, Sum: adapter.identity.Sum}
		if err := nativebuild.DownloadModule(ctx, sourceRoot, identity); err != nil {
			return target.Spec{}, nil, err
		}
	}
	moduleCache, err := nativebuild.ReadModuleCache(ctx, sourceRoot)
	if err != nil {
		return target.Spec{}, nil, err
	}
	prepared, adapters, err := deterministicAdapters.prepare(spec, moduleCache)
	if err != nil {
		return target.Spec{}, nil, err
	}
	prepared.AdapterReplacements = make([]target.AdapterReplacement, len(adapters))
	for index := range adapters {
		adapter := &adapters[index]
		var pins map[string]string
		implementation := selected[index].implementation
		if implementation.rewritten != nil {
			pins = implementation.rewritten.preparedSourceSetSHA256ByHost
		} else if adapter.Module == libcModulePath {
			pins = libcPreparedSourceSetSHA256ByHost
		}
		want := pins[goos+"/"+goarch]
		if want == "" {
			return target.Spec{}, nil, fmt.Errorf("adapter %s has no source-set pin for %s/%s", adapter.Module, goos, goarch)
		}
		packageDirectory := adapter.ReplacementRoot
		if adapter.PreparedPackage != adapter.Module {
			packageDirectory = filepath.Join(packageDirectory, filepath.FromSlash(strings.TrimPrefix(adapter.PreparedPackage, adapter.Module+"/")))
		}
		digest, err := nativebuild.AdapterPreparedSourceSetSHA256(ctx, goCommand, packageDirectory, adapter.PreparedPackage, goos, goarch)
		if err != nil {
			return target.Spec{}, nil, err
		}
		if digest != want {
			return target.Spec{}, nil, fmt.Errorf("adapter %s source review identity mismatch: got %s, want %s", adapter.Module, digest, want)
		}
		adapter.PreparedSourceSetSHA256 = want
		prepared.AdapterReplacements[index] = projectAdapterReplacement(profile, *adapter)
	}
	return prepared, adapters, nil
}
