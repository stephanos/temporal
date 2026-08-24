package deterministicio

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

const (
	xnetModulePath                       = "golang.org/x/net"
	xnetVersion                          = "v0.57.0"
	xnetSum                              = "h1:K5+3DljvIuDG9/Jv9rvyMywYNFCQ9RSUY6OOTTkT+tE="
	xnetOriginalSourceInventorySHA256    = "sha256:f58ae09eeed3e297521e4057277d53ad2ff03a4c6ce80029a6db584cc213baba"
	xnetSocketSourceSHA256               = "sha256:facf54b3bc8b1e36552241cdf5bf3f5cd1010cf864f995cb0cf2ed3830036d6c"
	xnetEmptySourceSHA256                = "sha256:0d09f2c52fc60c2d411818b538de77927fbf43ad530214066e26315922f5bdd6"
	xnetSocketReplacementSHA256          = "sha256:f7469c5b887c0c443d55bf7e03add926e55bacd5cfed76870c180635ca9d2bb8"
	xnetEmptyReplacementSHA256           = "sha256:061172228faecc3a10af82b3bb3fdb88cbc5ae4bb712c36f915c49787ceaa455"
	xnetReplacementSourceInventorySHA256 = "sha256:bca9285d07d302281304f53e96a9352f7c656a4b91c8d408cb9d4089f7991883"
	xnetPreparedSocketSourceSetSHA256    = "sha256:10df56ce136dff1eca6c15283ddee843f7ef016fb994fa915d321dca77566c97"
	xnetSocketPath                       = "internal/socket/sys_unix.go"
	xnetEmptyPath                        = "internal/socket/empty.s"
)

func prepareXNet(moduleCache, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
	if identity.Module != xnetModulePath || identity.Version != xnetVersion || identity.Sum != xnetSum {
		return adapterPreparation{}, errors.New("x/net adapter identity mismatch")
	}
	moduleSource, err := filepath.EvalSymlinks(filepath.Join(moduleCache, "golang.org", "x", "net@"+identity.Version))
	if err != nil {
		return adapterPreparation{}, fmt.Errorf("resolve pinned x/net module: %w", err)
	}
	if err := verifyXNetModule(moduleSource); err != nil {
		return adapterPreparation{}, err
	}
	sysSource, err := readXNetAdapterSource(moduleSource, xnetSocketPath)
	if err != nil {
		return adapterPreparation{}, err
	}
	emptySource, err := readXNetAdapterSource(moduleSource, xnetEmptyPath)
	if err != nil {
		return adapterPreparation{}, err
	}
	rewrittenSys, rewrittenEmpty, err := rewriteXNetSocket(sysSource, emptySource)
	if err != nil {
		return adapterPreparation{}, err
	}
	moduleReplacement := filepath.Join(root, "golang-x-net")
	replacements := map[string][]byte{xnetSocketPath: rewrittenSys, xnetEmptyPath: rewrittenEmpty}
	if err := copyAdapterModule(moduleSource, moduleReplacement, replacements, defaultAdapterCopyLimits); err != nil {
		return adapterPreparation{}, fmt.Errorf("copy x/net adapter module: %w", err)
	}
	replacementInventory, err := digestAdapterSourceInventory(moduleReplacement)
	if err != nil {
		return adapterPreparation{}, fmt.Errorf("hash x/net replacement inventory: %w", err)
	}
	if replacementInventory != xnetReplacementSourceInventorySHA256 {
		return adapterPreparation{}, fmt.Errorf("x/net replacement inventory identity mismatch: got %s, want %s", replacementInventory, xnetReplacementSourceInventorySHA256)
	}
	return adapterPreparation{
		replacement: moduleReplacement,
		evidence: BuildAdapter{
			Module: identity.Module, Version: identity.Version, Sum: identity.Sum,
			Source: filepath.Join(moduleSource, filepath.FromSlash(xnetSocketPath)), ReplacementRoot: moduleReplacement, Replacement: filepath.Join(moduleReplacement, filepath.FromSlash(xnetSocketPath)),
			PreparedPackage:                  xnetModulePath + "/internal/socket",
			SourceSHA256:                     xnetSocketSourceSHA256,
			ReplacementSHA256:                xnetSocketReplacementSHA256,
			OriginalSourceInventorySHA256:    xnetOriginalSourceInventorySHA256,
			ReplacementSourceInventorySHA256: replacementInventory,
			PreparedSourceSetSHA256:          xnetPreparedSocketSourceSetSHA256,
		},
	}, nil
}

func verifyXNetModule(moduleRoot string) error {
	inventory, err := digestAdapterSourceInventory(moduleRoot)
	if err != nil {
		return fmt.Errorf("hash pinned x/net source inventory: %w", err)
	}
	if inventory != xnetOriginalSourceInventorySHA256 {
		return fmt.Errorf("pinned x/net source inventory identity mismatch: got %s, want %s", inventory, xnetOriginalSourceInventorySHA256)
	}
	return nil
}

func readXNetAdapterSource(moduleRoot, relative string) ([]byte, error) {
	path := filepath.Join(moduleRoot, filepath.FromSlash(relative))
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("pinned x/net source is not a regular file: %s", relative)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pinned x/net source %s: %w", relative, err)
	}
	return contents, nil
}

func rewriteXNetSocket(sysSource, emptySource []byte) ([]byte, []byte, error) {
	if digestBytes(sysSource) != xnetSocketSourceSHA256 || digestBytes(emptySource) != xnetEmptySourceSHA256 {
		return nil, nil, errors.New("pinned x/net socket source identity mismatch")
	}
	rewrittenSys, err := rewriteXNetSocketSource(sysSource)
	if err != nil {
		return nil, nil, err
	}
	rewrittenEmpty, err := replaceXNetAnchor(emptySource, []byte("//go:build darwin"), []byte("//go:build !darwin"))
	if err != nil {
		return nil, nil, err
	}
	if got := digestBytes(rewrittenSys); got != xnetSocketReplacementSHA256 {
		return nil, nil, fmt.Errorf("x/net socket replacement identity mismatch: got %s, want %s", got, xnetSocketReplacementSHA256)
	}
	if got := digestBytes(rewrittenEmpty); got != xnetEmptyReplacementSHA256 {
		return nil, nil, fmt.Errorf("x/net empty assembly replacement identity mismatch: got %s, want %s", got, xnetEmptyReplacementSHA256)
	}
	return rewrittenSys, rewrittenEmpty, nil
}

func rewriteXNetSocketSource(contents []byte) ([]byte, error) {
	rewrites := []struct {
		anchor      []byte
		replacement []byte
	}{
		{anchor: []byte("\t\"unsafe\"\n")},
		{anchor: []byte("//go:linkname syscall_getsockopt syscall.getsockopt\nfunc syscall_getsockopt(s, level, name int, val unsafe.Pointer, vallen *uint32) error\n\n//go:linkname syscall_setsockopt syscall.setsockopt\nfunc syscall_setsockopt(s, level, name int, val unsafe.Pointer, vallen uintptr) error\n\n")},
		{
			anchor:      []byte("func getsockopt(s uintptr, level, name int, b []byte) (int, error) {\n\tl := uint32(len(b))\n\terr := syscall_getsockopt(int(s), level, name, unsafe.Pointer(&b[0]), &l)\n\treturn int(l), err\n}\n"),
			replacement: []byte("func getsockopt(s uintptr, level, name int, b []byte) (int, error) {\n\treturn 0, unix.ENOTSUP\n}\n"),
		},
		{
			anchor:      []byte("func setsockopt(s uintptr, level, name int, b []byte) error {\n\treturn syscall_setsockopt(int(s), level, name, unsafe.Pointer(&b[0]), uintptr(len(b)))\n}\n"),
			replacement: []byte("func setsockopt(s uintptr, level, name int, b []byte) error {\n\treturn unix.ENOTSUP\n}\n"),
		},
	}
	result := append([]byte(nil), contents...)
	for _, rewrite := range rewrites {
		var err error
		result, err = replaceXNetAnchor(result, rewrite.anchor, rewrite.replacement)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func replaceXNetAnchor(contents, anchor, replacement []byte) ([]byte, error) {
	if bytes.Count(contents, anchor) != 1 {
		return nil, fmt.Errorf("pinned x/net rewrite anchor mismatch for %q", anchor)
	}
	return bytes.Replace(contents, anchor, replacement, 1), nil
}
