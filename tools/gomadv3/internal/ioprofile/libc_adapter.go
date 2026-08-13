package ioprofile

import (
	"bytes"
	"crypto/sha256"
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

	"go.temporal.io/server/tools/gomadv3/internal/target"
	gomadversion "go.temporal.io/server/tools/gomadv3/internal/version"
)

const (
	libcModulePath        = "modernc.org/libc"
	libcModuleVersion     = gomadversion.ModerncLibcVersion
	libcDarwinSHA256      = "sha256:46fc04624c96033980a81d8eeb9b4d73daff0c6cae511931456f2c72a75fcb7e"
	libcDarwinArm64SHA256 = "sha256:6c725881029bda79d32b8e29be850b45ec8e359a0d5d2f52bc634f93dcae4e99"
	libcUnixSHA256        = "sha256:b4350edb7222f6f4e2a8f8eb079ab0fbbc18e2be74762b68b17205ac3ead4f4a"
	maximumModuleFiles    = 5000
	maximumModuleBytes    = 512 << 20
)

type BuildOverlay struct {
	Path              string
	Source            string
	Replacement       string
	SourceSHA256      string
	ReplacementSHA256 string
}

func (profile ProfileSpec) PrepareBuildOverlay(spec target.Spec, moduleCache string) (target.Spec, BuildOverlay, error) {
	definition, err := profile.validated()
	if err != nil {
		return target.Spec{}, BuildOverlay{}, err
	}
	return definition.prepareBuildOverlay(spec, moduleCache)
}

func prepareDeterministicBuildOverlay(spec target.Spec, moduleCache string) (target.Spec, BuildOverlay, error) {
	workingDirectory, err := filepath.Abs(spec.WorkingDir)
	if err != nil {
		return target.Spec{}, BuildOverlay{}, fmt.Errorf("resolve target working directory: %w", err)
	}
	moduleFile, err := os.ReadFile(filepath.Join(workingDirectory, "go.mod"))
	if err != nil {
		return target.Spec{}, BuildOverlay{}, fmt.Errorf("read target module file: %w", err)
	}
	detected, err := detectLibcVersion(moduleFile)
	if err != nil {
		return target.Spec{}, BuildOverlay{}, err
	}
	if detected == "" {
		return spec, BuildOverlay{}, nil
	}
	if detected != libcModuleVersion {
		return target.Spec{}, BuildOverlay{}, fmt.Errorf("unsupported %s version %q", libcModulePath, detected)
	}
	if moduleCache == "" || spec.PreparationRoot == "" {
		return target.Spec{}, BuildOverlay{}, errors.New("deterministic I/O build adapter requires module cache and preparation root")
	}
	moduleSource, err := filepath.EvalSymlinks(filepath.Join(moduleCache, "modernc.org", "libc@"+libcModuleVersion))
	if err != nil {
		return target.Spec{}, BuildOverlay{}, fmt.Errorf("resolve pinned modernc libc module: %w", err)
	}
	rewrites, source, err := rewriteLibcModule(moduleSource)
	if err != nil {
		return target.Spec{}, BuildOverlay{}, err
	}
	preparationRoot, err := filepath.Abs(spec.PreparationRoot)
	if err != nil {
		return target.Spec{}, BuildOverlay{}, fmt.Errorf("resolve deterministic I/O preparation root: %w", err)
	}
	root := filepath.Join(preparationRoot, ".io-adapter")
	if err := os.Mkdir(root, 0o700); err != nil {
		return target.Spec{}, BuildOverlay{}, fmt.Errorf("create deterministic I/O adapter directory: %w", err)
	}
	moduleReplacement := filepath.Join(root, "libc")
	replacement, err := copyLibcModule(moduleSource, moduleReplacement, rewrites)
	if err != nil {
		return target.Spec{}, BuildOverlay{}, err
	}
	if bytes.Contains(moduleFile, []byte("replace "+libcModulePath)) {
		return target.Spec{}, BuildOverlay{}, fmt.Errorf("target module already replaces %s", libcModulePath)
	}
	moduleFile = append(moduleFile, []byte("\nreplace "+libcModulePath+" => "+moduleReplacement+"\n")...)
	modFilePath := filepath.Join(root, "gomad.mod")
	if err := writeExclusive(modFilePath, moduleFile); err != nil {
		return target.Spec{}, BuildOverlay{}, err
	}
	sumFile, err := os.ReadFile(filepath.Join(workingDirectory, "go.sum"))
	if err != nil {
		return target.Spec{}, BuildOverlay{}, fmt.Errorf("read target module sums: %w", err)
	}
	if err := writeExclusive(filepath.Join(root, "gomad.sum"), sumFile); err != nil {
		return target.Spec{}, BuildOverlay{}, err
	}
	spec.BuildModFile = modFilePath
	return spec, BuildOverlay{
		Path: modFilePath, Source: source, Replacement: replacement, SourceSHA256: libcDarwinSHA256,
		ReplacementSHA256: digestBytes(rewrites["libc_darwin.go"]),
	}, nil
}

func detectLibcVersion(contents []byte) (string, error) {
	inRequireBlock := false
	for _, line := range strings.Split(string(contents), "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "//", 2)[0])
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "replace" && len(fields) > 1 && fields[1] == libcModulePath {
			return "", fmt.Errorf("target module already replaces %s", libcModulePath)
		}
		if fields[0] == "require" {
			if len(fields) == 2 && fields[1] == "(" {
				inRequireBlock = true
				continue
			}
			if len(fields) >= 3 && fields[1] == libcModulePath {
				return fields[2], nil
			}
		}
		if inRequireBlock {
			if fields[0] == ")" {
				inRequireBlock = false
				continue
			}
			if len(fields) >= 2 && fields[0] == libcModulePath {
				return fields[1], nil
			}
		}
	}
	return "", nil
}

func rewriteLibcModule(moduleSource string) (map[string][]byte, string, error) {
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

const gomadLibcAdapterSource = `// Copyright 2026 The Libc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin

package libc

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
	"modernc.org/libc/fcntl"
)

//go:linkname gomadLibcEnabled internal/gomadio.Enabled
func gomadLibcEnabled() bool

//go:linkname gomadLibcOpen internal/gomadio.LibcOpen
func gomadLibcOpen(string, int, uint32) (int32, syscall.Errno)

//go:linkname gomadLibcClose internal/gomadio.LibcClose
func gomadLibcClose(int32) syscall.Errno

//go:linkname gomadLibcRead internal/gomadio.LibcRead
func gomadLibcRead(int32, uintptr, uint64, int64, bool) (int64, syscall.Errno)

//go:linkname gomadLibcWrite internal/gomadio.LibcWrite
func gomadLibcWrite(int32, uintptr, uint64, int64, bool) (int64, syscall.Errno)

//go:linkname gomadLibcSeek internal/gomadio.LibcSeek
func gomadLibcSeek(int32, int64, int) (int64, syscall.Errno)

//go:linkname gomadLibcTruncate internal/gomadio.LibcTruncate
func gomadLibcTruncate(int32, int64) syscall.Errno

//go:linkname gomadLibcSync internal/gomadio.LibcSync
func gomadLibcSync(int32) syscall.Errno

//go:linkname gomadLibcRemove internal/gomadio.LibcRemove
func gomadLibcRemove(string) syscall.Errno

//go:linkname gomadLibcRename internal/gomadio.LibcRename
func gomadLibcRename(string, string) syscall.Errno

//go:linkname gomadLibcMkdir internal/gomadio.LibcMkdir
func gomadLibcMkdir(string, uint32) syscall.Errno

//go:linkname gomadLibcAccess internal/gomadio.LibcAccess
func gomadLibcAccess(string) syscall.Errno

//go:linkname gomadLibcStat internal/gomadio.LibcStat
func gomadLibcStat(string, int32) (uint32, int64, syscall.Errno)

//go:linkname gomadLibcIsDescriptor internal/gomadio.LibcIsDescriptor
func gomadLibcIsDescriptor(int32) bool

//go:linkname gomadLibcNow internal/gomadio.LibcNow
func gomadLibcNow() (int64, int64)

func gomadErrno(t *TLS, errno syscall.Errno) int32 {
	if errno == 0 { return 0 }
	t.setErrno(errno)
	return -1
}

func gomadOpen(t *TLS, name string, flags int32, mode uint32) int32 {
	fd, errno := gomadLibcOpen(name, int(flags), mode)
	if errno != 0 { return gomadErrno(t, errno) }
	return fd
}

func gomadClose(t *TLS, fd int32) (int32, bool) {
	if !gomadLibcEnabled() { return 0, false }
	if !gomadLibcIsDescriptor(fd) { return gomadErrno(t, syscall.EBADF), true }
	return gomadErrno(t, gomadLibcClose(fd)), true
}

func gomadRead(t *TLS, fd int32, address uintptr, size uint64, offset int64, positional bool) (int64, bool) {
	if !gomadLibcEnabled() { return 0, false }
	if !gomadLibcIsDescriptor(fd) { return int64(gomadErrno(t, syscall.EBADF)), true }
	result, errno := gomadLibcRead(fd, address, size, offset, positional)
	if errno != 0 { return int64(gomadErrno(t, errno)), true }
	return result, true
}

func gomadWrite(t *TLS, fd int32, address uintptr, size uint64, offset int64, positional bool) (int64, bool) {
	if !gomadLibcEnabled() { return 0, false }
	if !gomadLibcIsDescriptor(fd) { return int64(gomadErrno(t, syscall.EBADF)), true }
	result, errno := gomadLibcWrite(fd, address, size, offset, positional)
	if errno != 0 { return int64(gomadErrno(t, errno)), true }
	return result, true
}

func gomadSeek(t *TLS, fd int32, offset int64, whence int32) (int64, bool) {
	if !gomadLibcEnabled() { return 0, false }
	if !gomadLibcIsDescriptor(fd) { return int64(gomadErrno(t, syscall.EBADF)), true }
	result, errno := gomadLibcSeek(fd, offset, int(whence))
	if errno != 0 { return int64(gomadErrno(t, errno)), true }
	return result, true
}

func gomadTruncate(t *TLS, fd int32, size int64) (int32, bool) {
	if !gomadLibcEnabled() { return 0, false }
	if !gomadLibcIsDescriptor(fd) { return gomadErrno(t, syscall.EBADF), true }
	return gomadErrno(t, gomadLibcTruncate(fd, size)), true
}

func gomadSync(t *TLS, fd int32) (int32, bool) {
	if !gomadLibcEnabled() { return 0, false }
	if !gomadLibcIsDescriptor(fd) { return gomadErrno(t, syscall.EBADF), true }
	return gomadErrno(t, gomadLibcSync(fd)), true
}

func gomadGetcwd(t *TLS, address uintptr, size uint64) (uintptr, bool) {
	if !gomadLibcEnabled() { return 0, false }
	if size < 2 || address == 0 { gomadErrno(t, syscall.ERANGE); return 0, true }
	buffer := unsafe.Slice((*byte)(unsafe.Pointer(address)), int(size))
	buffer[0], buffer[1] = '/', 0
	return address, true
}

func gomadStatPath(t *TLS, name string, address uintptr) (int32, bool) {
	if !gomadLibcEnabled() { return 0, false }
	return gomadFillStat(t, name, -1, address), true
}

func gomadStatDescriptor(t *TLS, fd int32, address uintptr) (int32, bool) {
	if !gomadLibcEnabled() { return 0, false }
	if !gomadLibcIsDescriptor(fd) { return gomadErrno(t, syscall.EBADF), true }
	return gomadFillStat(t, "", fd, address), true
}

func gomadFillStat(t *TLS, name string, fd int32, address uintptr) int32 {
	mode, size, errno := gomadLibcStat(name, fd)
	if errno != 0 { return gomadErrno(t, errno) }
	if address == 0 { return gomadErrno(t, syscall.EFAULT) }
	*(*unix.Stat_t)(unsafe.Pointer(address)) = unix.Stat_t{Mode: uint16(mode), Nlink: 1, Size: size, Blocks: (size + 511) / 512, Blksize: 4096}
	return 0
}

func gomadStatfs(t *TLS, fd int32, address uintptr) (int32, bool) {
	if !gomadLibcEnabled() { return 0, false }
	if !gomadLibcIsDescriptor(fd) { return gomadErrno(t, syscall.EBADF), true }
	return gomadFillStatfs(t, address), true
}

func gomadStatfsPath(t *TLS, name string, address uintptr) int32 {
	if errno := gomadLibcAccess(name); errno != 0 { return gomadErrno(t, errno) }
	return gomadFillStatfs(t, address)
}

func gomadFillStatfs(t *TLS, address uintptr) int32 {
	if address == 0 { return gomadErrno(t, syscall.EFAULT) }
	*(*unix.Statfs_t)(unsafe.Pointer(address)) = unix.Statfs_t{Bsize: 4096, Iosize: 4096, Blocks: 1 << 30, Bfree: 1 << 29, Bavail: 1 << 29, Files: 1 << 30, Ffree: 1 << 29}
	return 0
}

func gomadMkdir(t *TLS, name string, mode uint32) int32 { return gomadErrno(t, gomadLibcMkdir(name, mode)) }
func gomadRemove(t *TLS, name string) (result int32) { return gomadErrno(t, gomadLibcRemove(name)) }
func gomadRename(t *TLS, oldName, newName string) int32 { return gomadErrno(t, gomadLibcRename(oldName, newName)) }
func gomadAccess(t *TLS, name string) int32 { return gomadErrno(t, gomadLibcAccess(name)) }

func gomadDescriptorNoop(t *TLS, fd int32) (int32, bool) {
	if !gomadLibcEnabled() { return 0, false }
	if !gomadLibcIsDescriptor(fd) { return gomadErrno(t, syscall.EBADF), true }
	return 0, true
}

func gomadFcntl(t *TLS, fd, cmd int32, args uintptr) (int32, bool) {
	if !gomadLibcEnabled() { return 0, false }
	if !gomadLibcIsDescriptor(fd) { return gomadErrno(t, syscall.EBADF), true }
	switch cmd {
	case fcntl.F_GETLK:
		lock := *(*uintptr)(unsafe.Pointer(args))
		(*unix.Flock_t)(unsafe.Pointer(lock)).Type = fcntl.F_UNLCK
		return 0, true
	case fcntl.F_SETLK, fcntl.F_SETLKW, fcntl.F_GETFL, fcntl.F_FULLFSYNC, fcntl.F_SETFD, fcntl.F_SETFL:
		return 0, true
	default:
		return gomadErrno(t, syscall.ENOTSUP), true
	}
}

func gomadMmap(t *TLS, fd int32) (uintptr, bool) {
	if !gomadLibcEnabled() { return 0, false }
	gomadErrno(t, syscall.ENODEV)
	return ^uintptr(0), true
}

func gomadGettimeofday(t *TLS, tv, tz uintptr) (int32, bool) {
	if !gomadLibcEnabled() { return 0, false }
	if tz != 0 || tv == 0 { return gomadErrno(t, syscall.EINVAL), true }
	seconds, microseconds := gomadLibcNow()
	*(*unix.Timeval)(unsafe.Pointer(tv)) = unix.Timeval{Sec: seconds, Usec: int32(microseconds)}
	return 0, true
}
`
