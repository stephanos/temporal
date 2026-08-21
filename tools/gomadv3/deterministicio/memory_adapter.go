package deterministicio

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.temporal.io/server/tools/gomadv3/target"
	gomadversion "go.temporal.io/server/tools/gomadv3/toolchain/version"
)

const (
	memoryModulePath                       = "modernc.org/memory"
	memoryVersion                          = "v1.11.0"
	memorySum                              = "h1:o4QC8aMQzmcwCK3t3Ux/ZHmwFPzE6hf2Y5LbkRs+hbI="
	memoryOriginalSourceInventorySHA256    = "sha256:4d829c24cc1718026fee9455b47449cfa15d8e241bbbfd9da6136435fd81881f"
	memoryMmapSourceSHA256                 = "sha256:d487e0d7f447b25397874a79e53c0e42b8568ed0503b562c959c83e8ef47f0a7"
	memoryMmapReplacementSHA256            = "sha256:c8a86dca80f526b39f0a855f59552d1085a352fb7fef47b86da827b854ab88ad"
	memoryReplacementSourceInventorySHA256 = "sha256:720c0239c80b4f8bcbebe1cd887451b8e58554f857d09f3f9b9dff939ce3f24e"
	memoryPreparedSourceSetSHA256          = "sha256:f58c119822204a56f5dee48029c1c6ac2888a22ca062f7ea078000248194ce36"
	memoryMmapPath                         = "mmap_unix.go"
)

func prepareModerncMemory(moduleCache, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
	if identity.Module != memoryModulePath || identity.Version != memoryVersion || identity.Sum != memorySum {
		return adapterPreparation{}, errors.New("modernc memory adapter identity mismatch")
	}
	moduleSource, err := filepath.EvalSymlinks(filepath.Join(moduleCache, "modernc.org", "memory@"+identity.Version))
	if err != nil {
		return adapterPreparation{}, fmt.Errorf("resolve pinned modernc memory module: %w", err)
	}
	originalInventory, err := target.DigestAdapterSourceInventory(moduleSource)
	if err != nil {
		return adapterPreparation{}, fmt.Errorf("hash pinned modernc memory source inventory: %w", err)
	}
	if originalInventory != memoryOriginalSourceInventorySHA256 {
		return adapterPreparation{}, fmt.Errorf("pinned modernc memory source inventory identity mismatch: got %s, want %s", originalInventory, memoryOriginalSourceInventorySHA256)
	}
	source, err := os.ReadFile(filepath.Join(moduleSource, memoryMmapPath))
	if err != nil {
		return adapterPreparation{}, fmt.Errorf("read pinned modernc memory source: %w", err)
	}
	rewritten, err := rewriteModerncMemory(source)
	if err != nil {
		return adapterPreparation{}, err
	}
	moduleReplacement := filepath.Join(root, "modernc-memory")
	if err := copyAdapterModule(moduleSource, moduleReplacement, map[string][]byte{memoryMmapPath: rewritten}, defaultAdapterCopyLimits); err != nil {
		return adapterPreparation{}, fmt.Errorf("copy modernc memory adapter module: %w", err)
	}
	replacementInventory, err := target.DigestAdapterSourceInventory(moduleReplacement)
	if err != nil {
		return adapterPreparation{}, fmt.Errorf("hash modernc memory replacement inventory: %w", err)
	}
	if replacementInventory != memoryReplacementSourceInventorySHA256 {
		return adapterPreparation{}, fmt.Errorf("modernc memory replacement inventory identity mismatch: got %s, want %s", replacementInventory, memoryReplacementSourceInventorySHA256)
	}
	return adapterPreparation{
		replacement: moduleReplacement,
		evidence: BuildAdapter{
			Module: identity.Module, Version: identity.Version, Sum: identity.Sum,
			Source: filepath.Join(moduleSource, memoryMmapPath), ReplacementRoot: moduleReplacement, Replacement: filepath.Join(moduleReplacement, memoryMmapPath),
			PreparedPackage:                  memoryModulePath,
			SourceSHA256:                     memoryMmapSourceSHA256,
			ReplacementSHA256:                memoryMmapReplacementSHA256,
			OriginalSourceInventorySHA256:    originalInventory,
			ReplacementSourceInventorySHA256: replacementInventory,
			PreparedSourceSetSHA256:          memoryPreparedSourceSetSHA256,
		},
	}, nil
}

func rewriteModerncMemory(source []byte) ([]byte, error) {
	if digestBytes(source) != memoryMmapSourceSHA256 {
		return nil, errors.New("pinned modernc memory source identity mismatch")
	}
	rewrites := []struct {
		anchor      []byte
		replacement []byte
	}{
		{
			anchor:      []byte("var (\n\tosPageMask = osPageSize - 1\n\tosPageSize = os.Getpagesize()\n)\n"),
			replacement: []byte("var (\n\tosPageMask = osPageSize - 1\n\tosPageSize = os.Getpagesize()\n)\n\n//go:linkname gomadMemoryEnabled internal/gomadio.Enabled\nfunc gomadMemoryEnabled() bool\n\n//go:linkname gomadMemoryMap internal/gomadio.AnonymousMap\nfunc gomadMemoryMap(size, alignment uintptr) uintptr\n\n//go:linkname gomadMemoryUnmap internal/gomadio.AnonymousUnmap\nfunc gomadMemoryUnmap(address, size uintptr) bool\n"),
		},
		{
			anchor:      []byte("func unmap(addr uintptr, size int) error {\n\treturn unix.MunmapPtr(unsafe.Pointer(addr), uintptr(size))\n}\n"),
			replacement: []byte("func unmap(addr uintptr, size int) error {\n\tif gomadMemoryEnabled() {\n\t\tif !gomadMemoryUnmap(addr, uintptr(size)) {\n\t\t\treturn unix.EINVAL\n\t\t}\n\t\treturn nil\n\t}\n\treturn unix.MunmapPtr(unsafe.Pointer(addr), uintptr(size))\n}\n"),
		},
		{
			anchor:      []byte("\tsize = roundup(size, osPageSize)\n"),
			replacement: []byte("\tsize = roundup(size, osPageSize)\n\tif gomadMemoryEnabled() {\n\t\tp := gomadMemoryMap(uintptr(size), pageSize)\n\t\tif p == 0 {\n\t\t\treturn 0, 0, unix.ENOMEM\n\t\t}\n\t\treturn p, size, nil\n\t}\n"),
		},
	}
	result := append([]byte(nil), source...)
	for _, rewrite := range rewrites {
		if bytes.Count(result, rewrite.anchor) != 1 {
			return nil, errors.New("pinned modernc memory rewrite anchor mismatch")
		}
		result = bytes.Replace(result, rewrite.anchor, rewrite.replacement, 1)
	}
	if got := digestBytes(result); got != memoryMmapReplacementSHA256 {
		return nil, fmt.Errorf("modernc memory replacement identity mismatch: got %s, want %s", got, memoryMmapReplacementSHA256)
	}
	return result, nil
}
