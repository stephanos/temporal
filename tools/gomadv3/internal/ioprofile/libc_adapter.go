package ioprofile

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	gomadversion "go.temporal.io/server/tools/gomadv3/internal/version"
)

const (
	libcModulePath         = "modernc.org/libc"
	libcDarwinSHA256       = "sha256:46fc04624c96033980a81d8eeb9b4d73daff0c6cae511931456f2c72a75fcb7e"
	libcDarwinArm64SHA256  = "sha256:6c725881029bda79d32b8e29be850b45ec8e359a0d5d2f52bc634f93dcae4e99"
	libcUnixSHA256         = "sha256:b4350edb7222f6f4e2a8f8eb079ab0fbbc18e2be74762b68b17205ac3ead4f4a"
	gomadLibcAdapterSHA256 = "sha256:de831957b7a6e5cf7c79785ea5026bbbda4486b179d7e843b22f00a985296a2c"
	maximumModuleFiles     = 5000
	maximumModuleBytes     = 512 << 20
)

func prepareModerncLibc(moduleCache, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
	moduleSource, err := filepath.EvalSymlinks(filepath.Join(moduleCache, "modernc.org", "libc@"+identity.Version))
	if err != nil {
		return adapterPreparation{}, fmt.Errorf("resolve pinned modernc libc module: %w", err)
	}
	rewrites, source, err := rewriteLibcModule(moduleSource)
	if err != nil {
		return adapterPreparation{}, err
	}
	moduleReplacement := filepath.Join(root, "modernc-libc")
	replacement, err := copyLibcModule(moduleSource, moduleReplacement, rewrites)
	if err != nil {
		return adapterPreparation{}, err
	}
	return adapterPreparation{
		replacement: moduleReplacement,
		evidence: BuildAdapter{
			Module: identity.Module, Version: identity.Version, Sum: identity.Sum,
			Source: source, Replacement: replacement, SourceSHA256: libcDarwinSHA256,
			ReplacementSHA256: digestBytes(rewrites["libc_darwin.go"]),
		},
	}, nil
}

func rewriteLibcModule(moduleSource string) (map[string][]byte, string, error) {
	if digestBytes([]byte(gomadLibcAdapterSource)) != gomadLibcAdapterSHA256 {
		return nil, "", errors.New("modernc libc adapter template identity mismatch")
	}
	identities := map[string]string{
		"libc_darwin.go":       libcDarwinSHA256,
		"libc_darwin_arm64.go": libcDarwinArm64SHA256,
		"libc_unix.go":         libcUnixSHA256,
	}
	rewrites := make(map[string][]byte, len(identities)+1)
	for relative, identity := range identities {
		path := filepath.Join(moduleSource, relative)
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			return nil, "", fmt.Errorf("pinned modernc libc source %q is not a regular file", relative)
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return nil, "", fmt.Errorf("read pinned modernc libc source %q: %w", relative, err)
		}
		if digestBytes(contents) != identity {
			return nil, "", fmt.Errorf("pinned modernc libc source %q identity mismatch", relative)
		}
		rewrites[relative] = contents
	}
	var err error
	rewrites["libc_darwin.go"], err = rewriteLibcDarwin(rewrites["libc_darwin.go"])
	if err != nil {
		return nil, "", err
	}
	rewrites["libc_darwin_arm64.go"], err = rewriteLibcDarwinArm64(rewrites["libc_darwin_arm64.go"])
	if err != nil {
		return nil, "", err
	}
	rewrites["libc_unix.go"], err = rewriteLibcUnix(rewrites["libc_unix.go"])
	if err != nil {
		return nil, "", err
	}
	rewrites["gomad_darwin.go"] = []byte(gomadLibcAdapterSource)
	return rewrites, filepath.Join(moduleSource, "libc_darwin.go"), nil
}

func rewriteLibcDarwin(contents []byte) ([]byte, error) {
	result, err := denyHostCapabilityCalls(contents)
	if err != nil {
		return nil, err
	}
	rewrites := []functionRewrite{
		{header: "func Xclose(t *TLS, fd int32) int32 {", body: "\tif result, handled := gomadClose(t, fd); handled { return result }\n"},
		{header: "func Xfsync(t *TLS, fd int32) int32 {", body: "\tif result, handled := gomadSync(t, fd); handled { return result }\n"},
		{header: "func Xftruncate(t *TLS, fd int32, length types.Off_t) int32 {", body: "\tif result, handled := gomadTruncate(t, fd, int64(length)); handled { return result }\n"},
		{header: "func Xread(t *TLS, fd int32, buf uintptr, count types.Size_t) types.Ssize_t {", body: "\tif result, handled := gomadRead(t, fd, buf, uint64(count), 0, false); handled { return types.Ssize_t(result) }\n"},
		{header: "func Xwrite(t *TLS, fd int32, buf uintptr, count types.Size_t) types.Ssize_t {", body: "\tif result, handled := gomadWrite(t, fd, buf, uint64(count), 0, false); handled { return types.Ssize_t(result) }\n"},
		{header: "func Xpwrite(t *TLS, fd int32, buf uintptr, count types.Size_t, offset types.Off_t) types.Ssize_t {", body: "\tif result, handled := gomadWrite(t, fd, buf, uint64(count), int64(offset), true); handled { return types.Ssize_t(result) }\n"},
		{header: "func Xgetcwd(t *TLS, buf uintptr, size types.Size_t) uintptr {", body: "\tif result, handled := gomadGetcwd(t, buf, uint64(size)); handled { return result }\n"},
		{header: "func Xfchmod(t *TLS, fd int32, mode types.Mode_t) int32 {", body: "\tif result, handled := gomadDescriptorNoop(t, fd); handled { return result }\n"},
		{header: "func Xfchown(t *TLS, fd int32, owner types.Uid_t, group types.Gid_t) int32 {", body: "\tif result, handled := gomadDescriptorNoop(t, fd); handled { return result }\n"},
		{header: "func Xmmap(t *TLS, addr uintptr, length types.Size_t, prot, flags, fd int32, offset types.Off_t) uintptr {", body: "\tif result, handled := gomadMmap(t, fd); handled { return result }\n"},
		{header: "func Xgettimeofday(t *TLS, tv, tz uintptr) int32 {", body: "\tif result, handled := gomadGettimeofday(t, tv, tz); handled { return result }\n"},
		{header: "func Xgeteuid(t *TLS) types.Uid_t {", body: "\tif gomadLibcEnabled() { return 0 }\n"},
		{header: "func Xrmdir(t *TLS, pathname uintptr) int32 {", body: "\tif gomadLibcEnabled() { return gomadRemove(t, GoString(pathname)) }\n"},
	}
	result, err = rewriteFunctions(result, rewrites)
	if err != nil {
		return nil, err
	}
	result, err = injectAfter(result, "\tif args != 0 {\n\t\tmode = (types.Mode_t)(VaUint32(&args))\n\t}\n", "\tif gomadLibcEnabled() { return gomadOpen(t, GoString(pathname), flags, uint32(mode)) }\n", 2)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func rewriteLibcDarwinArm64(contents []byte) ([]byte, error) {
	result, err := denyHostCapabilityCalls(contents)
	if err != nil {
		return nil, err
	}
	return rewriteFunctions(result, []functionRewrite{
		{header: "func Xfcntl64(t *TLS, fd, cmd int32, args uintptr) (r int32) {", body: "\tif result, handled := gomadFcntl(t, fd, cmd, args); handled { return result }\n"},
		{header: "func Xlstat64(t *TLS, pathname, statbuf uintptr) int32 {", body: "\tif result, handled := gomadStatPath(t, GoString(pathname), statbuf); handled { return result }\n"},
		{header: "func Xstat64(t *TLS, pathname, statbuf uintptr) int32 {", body: "\tif result, handled := gomadStatPath(t, GoString(pathname), statbuf); handled { return result }\n"},
		{header: "func Xfstatfs(t *TLS, fd int32, buf uintptr) int32 {", body: "\tif result, handled := gomadStatfs(t, fd, buf); handled { return result }\n"},
		{header: "func Xstatfs(t *TLS, path uintptr, buf uintptr) int32 {", body: "\tif gomadLibcEnabled() { return gomadStatfsPath(t, GoString(path), buf) }\n"},
		{header: "func Xfstat64(t *TLS, fd int32, statbuf uintptr) int32 {", body: "\tif result, handled := gomadStatDescriptor(t, fd, statbuf); handled { return result }\n"},
		{header: "func Xlseek64(t *TLS, fd int32, offset types.Off_t, whence int32) types.Off_t {", body: "\tif result, handled := gomadSeek(t, fd, int64(offset), whence); handled { return types.Off_t(result) }\n"},
		{header: "func Xmkdir(t *TLS, path uintptr, mode types.Mode_t) int32 {", body: "\tif gomadLibcEnabled() { return gomadMkdir(t, GoString(path), uint32(mode)) }\n"},
		{header: "func Xunlink(t *TLS, pathname uintptr) int32 {", body: "\tif gomadLibcEnabled() { return gomadRemove(t, GoString(pathname)) }\n"},
		{header: "func Xaccess(t *TLS, pathname uintptr, mode int32) int32 {", body: "\tif gomadLibcEnabled() { return gomadAccess(t, GoString(pathname)) }\n"},
		{header: "func Xrename(t *TLS, oldpath, newpath uintptr) int32 {", body: "\tif gomadLibcEnabled() { return gomadRename(t, GoString(oldpath), GoString(newpath)) }\n"},
	})
}

func rewriteLibcUnix(contents []byte) ([]byte, error) {
	result, err := denyHostCapabilityCalls(contents)
	if err != nil {
		return nil, err
	}
	return rewriteFunctions(result, []functionRewrite{
		{header: "func Xpread(t *TLS, fd int32, buf uintptr, count types.Size_t, offset types.Off_t) types.Ssize_t {", body: "\tif result, handled := gomadRead(t, fd, buf, uint64(count), int64(offset), true); handled { return types.Ssize_t(result) }\n"},
	})
}

func denyHostCapabilityCalls(contents []byte) ([]byte, error) {
	files := token.NewFileSet()
	parsed, err := parser.ParseFile(files, "libc.go", contents, 0)
	if err != nil {
		return nil, fmt.Errorf("parse pinned modernc libc source: %w", err)
	}
	type insertion struct {
		offset int
		text   []byte
	}
	var insertions []insertion
	lateModels := map[string]struct{}{"Xopen": {}, "Xopen64": {}}
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		if _, modeled := lateModels[function.Name.Name]; modeled {
			continue
		}
		risky := false
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			qualifier, ok := selector.X.(*ast.Ident)
			if ok && (qualifier.Name == "unix" || qualifier.Name == "syscall" || qualifier.Name == "exec") {
				risky = true
				return false
			}
			return true
		})
		if risky {
			insertions = append(insertions, insertion{
				offset: files.Position(function.Body.Lbrace).Offset + 1,
				text:   []byte("\n\tif gomadLibcEnabled() { panic(\"gomad: unsupported modernc libc host capability: " + function.Name.Name + "\") }"),
			})
		}
	}
	result := append([]byte(nil), contents...)
	for index := len(insertions) - 1; index >= 0; index-- {
		insertion := insertions[index]
		result = append(result[:insertion.offset], append(insertion.text, result[insertion.offset:]...)...)
	}
	return result, nil
}

type functionRewrite struct {
	header string
	body   string
}

func rewriteFunctions(contents []byte, rewrites []functionRewrite) ([]byte, error) {
	result := append([]byte(nil), contents...)
	for _, rewrite := range rewrites {
		var err error
		result, err = injectAfter(result, rewrite.header+"\n", rewrite.body, 1)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func injectAfter(contents []byte, anchor, addition string, expected int) ([]byte, error) {
	if bytes.Count(contents, []byte(anchor)) != expected {
		return nil, fmt.Errorf("pinned modernc libc rewrite anchor mismatch for %q", strings.TrimSpace(anchor))
	}
	return bytes.ReplaceAll(contents, []byte(anchor), []byte(anchor+addition)), nil
}

func copyLibcModule(source, destination string, replacements map[string][]byte) (string, error) {
	if err := os.Mkdir(destination, 0o700); err != nil {
		return "", fmt.Errorf("create modernc libc adapter module: %w", err)
	}
	files := 0
	bytesCopied := int64(0)
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return errors.New("invalid modernc libc module path")
		}
		targetPath := filepath.Join(destination, relative)
		if entry.IsDir() {
			if relative == "." {
				return nil
			}
			return os.Mkdir(targetPath, 0o700)
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("modernc libc module contains a non-regular file")
		}
		files++
		bytesCopied += info.Size()
		if files > maximumModuleFiles || bytesCopied > maximumModuleBytes {
			return errors.New("modernc libc module exceeds adapter bounds")
		}
		contents := replacements[relative]
		if contents == nil {
			contents, err = os.ReadFile(path)
			if err != nil {
				return err
			}
		}
		return writeExclusive(targetPath, contents)
	})
	if err != nil {
		return "", fmt.Errorf("copy modernc libc adapter module: %w", err)
	}
	adapterPath := filepath.Join(destination, "gomad_darwin.go")
	if err := writeExclusive(adapterPath, replacements["gomad_darwin.go"]); err != nil {
		return "", err
	}
	return filepath.Join(destination, "libc_darwin.go"), nil
}

func writeExclusive(path string, contents []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o400)
	if err != nil {
		return fmt.Errorf("create deterministic I/O adapter file: %w", err)
	}
	if _, err := file.Write(contents); err != nil {
		return errors.Join(fmt.Errorf("write deterministic I/O adapter file: %w", err), file.Close())
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close deterministic I/O adapter file: %w", err)
	}
	return nil
}

func digestBytes(contents []byte) string {
	digest := sha256.Sum256(contents)
	return "sha256:" + hex.EncodeToString(digest[:])
}

//go:embed adapterdata/modernc_libc_darwin.go.tmpl
var gomadLibcAdapterSource string
