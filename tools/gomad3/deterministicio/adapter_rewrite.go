package deterministicio

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

// sourceRewrite pins one module source file together with the anchored edits
// that turn it into its deterministic replacement. Every anchor must occur
// exactly once, and both the input and the output are bound to exact digests
// so an upstream edit fails the build instead of shifting the rewrite.
type sourceRewrite struct {
	path                            string
	sourceSHA256, replacementSHA256 string
	rewrites                        []anchorRewrite
}

type anchorRewrite struct {
	anchor      []byte
	replacement []byte
}

// rewrittenModule describes an adapter that copies a pinned module and
// replaces a fixed set of its files. The first rewrite is the one the build
// evidence names as the adapter's source.
type rewrittenModule struct {
	module, version, sum       string
	cacheElements              []string
	replacementDirectory       string
	originalInventorySHA256    string
	replacementInventorySHA256 string
	preparedPackage            string
	preparedSourceSetSHA256    string
	rewrites                   []sourceRewrite
}

func prepareRewrittenModule(moduleCache, root string, identity gomadversion.AdapterIdentity, spec rewrittenModule) (adapterPreparation, error) {
	if identity.Module != spec.module || identity.Version != spec.version || identity.Sum != spec.sum {
		return adapterPreparation{}, fmt.Errorf("%s adapter identity mismatch", spec.module)
	}
	if len(spec.rewrites) == 0 {
		return adapterPreparation{}, fmt.Errorf("%s adapter has no rewrites", spec.module)
	}
	moduleSource, err := filepath.EvalSymlinks(filepath.Join(append([]string{moduleCache}, spec.cacheElements...)...))
	if err != nil {
		return adapterPreparation{}, fmt.Errorf("resolve pinned %s module: %w", spec.module, err)
	}
	if err := verifyAdapterModuleInventory(spec.module, moduleSource, spec.originalInventorySHA256); err != nil {
		return adapterPreparation{}, err
	}
	replacements := make(map[string][]byte, len(spec.rewrites))
	for _, rewrite := range spec.rewrites {
		contents, err := readAdapterSource(spec.module, moduleSource, rewrite.path)
		if err != nil {
			return adapterPreparation{}, err
		}
		replacements[rewrite.path], err = rewriteAdapterSource(spec.module, rewrite, contents)
		if err != nil {
			return adapterPreparation{}, err
		}
	}
	moduleReplacement := filepath.Join(root, spec.replacementDirectory)
	if err := copyAdapterModule(moduleSource, moduleReplacement, replacements, defaultAdapterCopyLimits); err != nil {
		return adapterPreparation{}, fmt.Errorf("copy %s adapter module: %w", spec.module, err)
	}
	replacementInventory, err := digestAdapterSourceInventory(moduleReplacement)
	if err != nil {
		return adapterPreparation{}, fmt.Errorf("hash %s replacement inventory: %w", spec.module, err)
	}
	if replacementInventory != spec.replacementInventorySHA256 {
		return adapterPreparation{}, fmt.Errorf("%s replacement inventory identity mismatch: got %s, want %s", spec.module, replacementInventory, spec.replacementInventorySHA256)
	}
	primary := spec.rewrites[0]
	return adapterPreparation{
		replacement: moduleReplacement,
		evidence: BuildAdapter{
			Module: identity.Module, Version: identity.Version, Sum: identity.Sum,
			Source: filepath.Join(moduleSource, filepath.FromSlash(primary.path)), ReplacementRoot: moduleReplacement, Replacement: filepath.Join(moduleReplacement, filepath.FromSlash(primary.path)),
			PreparedPackage:                  spec.preparedPackage,
			SourceSHA256:                     primary.sourceSHA256,
			ReplacementSHA256:                primary.replacementSHA256,
			OriginalSourceInventorySHA256:    spec.originalInventorySHA256,
			ReplacementSourceInventorySHA256: replacementInventory,
			PreparedSourceSetSHA256:          spec.preparedSourceSetSHA256,
		},
	}, nil
}

func verifyAdapterModuleInventory(module, moduleRoot, want string) error {
	inventory, err := digestAdapterSourceInventory(moduleRoot)
	if err != nil {
		return fmt.Errorf("hash pinned %s source inventory: %w", module, err)
	}
	if inventory != want {
		return fmt.Errorf("pinned %s source inventory identity mismatch: got %s, want %s", module, inventory, want)
	}
	return nil
}

func readAdapterSource(module, moduleRoot, relative string) ([]byte, error) {
	path := filepath.Join(moduleRoot, filepath.FromSlash(relative))
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("pinned %s source is not a regular file: %s", module, relative)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pinned %s source %s: %w", module, relative, err)
	}
	return contents, nil
}

func rewriteAdapterSource(module string, rewrite sourceRewrite, contents []byte) ([]byte, error) {
	if digestBytes(contents) != rewrite.sourceSHA256 {
		return nil, fmt.Errorf("pinned %s source identity mismatch for %s", module, rewrite.path)
	}
	if len(rewrite.rewrites) == 0 {
		return nil, errors.New("adapter source rewrite has no anchors")
	}
	result := append([]byte(nil), contents...)
	for _, step := range rewrite.rewrites {
		if bytes.Count(result, step.anchor) != 1 {
			return nil, fmt.Errorf("pinned %s rewrite anchor mismatch for %s: %q", module, rewrite.path, step.anchor)
		}
		result = bytes.Replace(result, step.anchor, step.replacement, 1)
	}
	if got := digestBytes(result); got != rewrite.replacementSHA256 {
		return nil, fmt.Errorf("%s replacement identity mismatch for %s: got %s, want %s", module, rewrite.path, got, rewrite.replacementSHA256)
	}
	return result, nil
}
