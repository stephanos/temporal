package deterministicio

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

const (
	libcModulePath         = "modernc.org/libc"
	libcDarwinSHA256       = "sha256:46fc04624c96033980a81d8eeb9b4d73daff0c6cae511931456f2c72a75fcb7e"
	libcDarwinArm64SHA256  = "sha256:6c725881029bda79d32b8e29be850b45ec8e359a0d5d2f52bc634f93dcae4e99"
	libcUnixSHA256         = "sha256:b4350edb7222f6f4e2a8f8eb079ab0fbbc18e2be74762b68b17205ac3ead4f4a"
	gomadLibcAdapterSHA256 = "sha256:751f42d790ea150f57977ae75189909eeb8ad0b55f3aee7bd5ede3e0f92f10cd"
	// The linux/amd64 build is a musl translation that reaches the kernel only
	// through the trampolines in syscall_musl.go, so the Linux model hooks the
	// syscall number there instead of individual libc functions.
	libcSyscallMuslSHA256       = "sha256:16a646a7d874493b0145fb13a45458c5b59eb9a131b975c93392a3f44b605578"
	libcMuslSHA256              = "sha256:2ae49f1d62addfa66305cd75b1ed67cee20cd0adb7bbff52ddf166b2e0e82d92"
	libcMuslLinuxAmd64SHA256    = "sha256:97cd9c7f1c6f1063e29685cf115c8f3f0530b3b9ce992935035dd52800fca8b1"
	gomadLibcLinuxAdapterSHA256 = "sha256:dc8e6f1bf6311e909d079a303857a5001a2d983afa34ea2465715e159b5c824f"
)

// Both platforms' sources are rewritten in every copy; the prepared source set
// differs because each platform compiles its own file set.
var libcPreparedSourceSetSHA256 = hostPin(map[string]string{
	"darwin/arm64": "sha256:8e1663c90aa178a706929ae94f248051781e4278ca83991d9a5fc6fe05321833",
	"linux/amd64":  "sha256:7dc5085b840868004fdccdf1526242f3da62cae0df0f06a0c772432e487ba403",
})

func prepareModerncLibc(moduleCache, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
	moduleSource, err := filepath.EvalSymlinks(filepath.Join(moduleCache, "modernc.org", "libc@"+identity.Version))
	if err != nil {
		return adapterPreparation{}, fmt.Errorf("resolve pinned modernc libc module: %w", err)
	}
	originalInventory, err := digestAdapterSourceInventory(moduleSource)
	if err != nil {
		return adapterPreparation{}, fmt.Errorf("hash pinned modernc libc source inventory: %w", err)
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
	replacementInventory, err := digestAdapterSourceInventory(moduleReplacement)
	if err != nil {
		return adapterPreparation{}, fmt.Errorf("hash modernc libc replacement inventory: %w", err)
	}
	return adapterPreparation{
		replacement: moduleReplacement,
		evidence: BuildAdapter{
			Module: identity.Module, Version: identity.Version, Sum: identity.Sum,
			Source: source, ReplacementRoot: moduleReplacement, Replacement: replacement, SourceSHA256: libcDarwinSHA256,
			PreparedPackage:                  libcModulePath,
			ReplacementSHA256:                digestBytes(rewrites["libc_darwin.go"]),
			OriginalSourceInventorySHA256:    originalInventory,
			ReplacementSourceInventorySHA256: replacementInventory,
			PreparedSourceSetSHA256:          libcPreparedSourceSetSHA256,
		},
	}, nil
}

func rewriteLibcModule(moduleSource string) (map[string][]byte, string, error) {
	if digestBytes([]byte(gomadLibcAdapterSource)) != gomadLibcAdapterSHA256 || digestBytes([]byte(gomadLibcLinuxAdapterSource)) != gomadLibcLinuxAdapterSHA256 {
		return nil, "", errors.New("modernc libc adapter template identity mismatch")
	}
	identities := map[string]string{
		"libc_darwin.go":           libcDarwinSHA256,
		"libc_darwin_arm64.go":     libcDarwinArm64SHA256,
		"libc_unix.go":             libcUnixSHA256,
		"syscall_musl.go":          libcSyscallMuslSHA256,
		"libc_musl.go":             libcMuslSHA256,
		"libc_musl_linux_amd64.go": libcMuslLinuxAmd64SHA256,
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
	rewrites["syscall_musl.go"], err = rewriteLibcSyscallMusl(rewrites["syscall_musl.go"])
	if err != nil {
		return nil, "", err
	}
	for _, relative := range []string{"libc_musl.go", "libc_musl_linux_amd64.go"} {
		rewrites[relative], err = denyHostCapabilityCalls(rewrites[relative])
		if err != nil {
			return nil, "", err
		}
	}
	rewrites["gomad_darwin.go"] = []byte(gomadLibcAdapterSource)
	rewrites["gomad_linux.go"] = []byte(gomadLibcLinuxAdapterSource)
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
		{header: "func Xmmap(t *TLS, addr uintptr, length types.Size_t, prot, flags, fd int32, offset types.Off_t) uintptr {", body: "\tif result, handled := gomadMmap(t, addr, uint64(length), prot, flags, fd, int64(offset)); handled { return result }\n"},
		{header: "func Xmunmap(t *TLS, addr uintptr, length types.Size_t) int32 {", body: "\tif result, handled := gomadMunmap(t, addr, uint64(length)); handled { return result }\n"},
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

func rewriteLibcSyscallMusl(contents []byte) ([]byte, error) {
	return rewriteFunctions(contents, []functionRewrite{
		{header: "func ___syscall_cp(tls *TLS, n, a, b, c, d, e, f long) long {", body: "\tif result, handled := gomadSyscall(tls, n, a, b, c, d, e, f); handled { return result }\n"},
		{header: "func X__syscall0(tls *TLS, n long) long {", body: "\tif result, handled := gomadSyscall(tls, n, 0, 0, 0, 0, 0, 0); handled { return result }\n"},
		{header: "func X__syscall1(tls *TLS, n, a1 long) long {", body: "\tif result, handled := gomadSyscall(tls, n, a1, 0, 0, 0, 0, 0); handled { return result }\n"},
		{header: "func X__syscall2(tls *TLS, n, a1, a2 long) long {", body: "\tif result, handled := gomadSyscall(tls, n, a1, a2, 0, 0, 0, 0); handled { return result }\n"},
		{header: "func X__syscall3(tls *TLS, n, a1, a2, a3 long) long {", body: "\tif result, handled := gomadSyscall(tls, n, a1, a2, a3, 0, 0, 0); handled { return result }\n"},
		{header: "func X__syscall4(tls *TLS, n, a1, a2, a3, a4 long) long {", body: "\tif result, handled := gomadSyscall(tls, n, a1, a2, a3, a4, 0, 0); handled { return result }\n"},
		{header: "func X__syscall5(tls *TLS, n, a1, a2, a3, a4, a5 long) long {", body: "\tif result, handled := gomadSyscall(tls, n, a1, a2, a3, a4, a5, 0); handled { return result }\n"},
		{header: "func X__syscall6(tls *TLS, n, a1, a2, a3, a4, a5, a6 long) long {", body: "\tif result, handled := gomadSyscall(tls, n, a1, a2, a3, a4, a5, a6); handled { return result }\n"},
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
	if err := copyAdapterModule(source, destination, replacements, defaultAdapterCopyLimits); err != nil {
		return "", fmt.Errorf("copy modernc libc adapter module: %w", err)
	}
	return filepath.Join(destination, "libc_darwin.go"), nil
}

//go:embed adapterdata/modernc_libc_darwin.go.tmpl
var gomadLibcAdapterSource string

//go:embed adapterdata/modernc_libc_linux.go.tmpl
var gomadLibcLinuxAdapterSource string
