package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/gomad3/hostfs"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad_wasm/toolchain"
)

func (p *Provider) runtimeOverlay(ctx context.Context, dir string, environment []string, includeDirectory string) (string, string, map[string]string, error) {
	if !filepath.IsAbs(p.options.RuntimeRoot) || !filepath.IsAbs(p.options.CacheRoot) {
		return "", "", nil, errors.New("absolute WASM runtime source and cache roots are required")
	}
	stockRoot := filepath.Dir(filepath.Dir(includeDirectory))
	moduleCache, err := command(ctx, p.options.CompilerPath, []string{"env", "GOMODCACHE"}, dir, environment, 64<<10)
	if err != nil {
		return "", "", nil, err
	}
	resolvedRoot, err := filepath.EvalSymlinks(stockRoot)
	if err != nil {
		return "", "", nil, err
	}
	cacheRoot := strings.TrimSpace(string(moduleCache))
	if resolvedCache, resolveErr := filepath.EvalSymlinks(cacheRoot); resolveErr == nil {
		cacheRoot = resolvedCache
	} else if !errors.Is(resolveErr, os.ErrNotExist) {
		return "", "", nil, resolveErr
	}
	relative, err := filepath.Rel(cacheRoot, resolvedRoot)
	if err != nil {
		return "", "", nil, err
	}
	if relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", "", nil, errors.New("cooperative compiler distribution must be outside the module cache")
	}
	path, identity, err := toolchain.BuildOverlay(stockRoot, p.options.RuntimeRoot, filepath.Join(p.options.CacheRoot, "runtime-overlay"))
	if err != nil {
		return "", "", nil, err
	}
	data, err := hostfs.ReadBounded(path, maximumProvenanceBytes)
	if err != nil {
		return "", "", nil, err
	}
	var overlay struct{ Replace map[string]string }
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&overlay); err != nil {
		return "", "", nil, err
	}
	return path, identity, overlay.Replace, nil
}

func (p *Provider) runtimeInputs(overlayPath string) ([]sourceIdentity, error) {
	inputs := []sourceIdentity{}
	for _, path := range []string{overlayPath, filepath.Join(p.options.RuntimeRoot, "gomad.go"), filepath.Join(p.options.RuntimeRoot, "gomad_choicewire_generated.go")} {
		data, err := hostfs.ReadBounded(path, 512<<20)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, sourceIdentity{Path: path, SHA256: record.HashBytes(data)})
	}
	return inputs, nil
}
