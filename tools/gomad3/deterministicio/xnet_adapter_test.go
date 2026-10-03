package deterministicio

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/target"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

func TestRewriteXNetSocketDeniesRawSocketOptions(t *testing.T) {
	sysSource, emptySource := readPinnedXNetSocketSources(t)
	rewrittenSys, err := rewriteAdapterSource(xnetModulePath, xnetRewrites[0], sysSource)
	if err != nil {
		t.Fatal(err)
	}
	rewrittenEmpty, err := rewriteAdapterSource(xnetModulePath, xnetRewrites[1], emptySource)
	if err != nil {
		t.Fatal(err)
	}
	for _, retained := range []string{
		"func recvmsg(", "func sendmsg(", "func addrToSockaddr(", "func sockaddrToAddr(",
	} {
		if !strings.Contains(string(rewrittenSys), retained) {
			t.Fatalf("rewritten sys_unix.go omitted %q", retained)
		}
	}
	for _, removed := range []string{
		"go:linkname", "syscall_getsockopt", "syscall_setsockopt", "\"unsafe\"",
	} {
		if strings.Contains(string(rewrittenSys), removed) {
			t.Fatalf("rewritten sys_unix.go retained %q", removed)
		}
	}
	if !strings.Contains(string(rewrittenSys), "return 0, unix.ENOTSUP") || !strings.Contains(string(rewrittenSys), "return unix.ENOTSUP") {
		t.Fatalf("rewritten sys_unix.go does not deny raw socket options: %s", rewrittenSys)
	}
	if !strings.Contains(string(rewrittenEmpty), "//go:build ignore") || strings.Contains(string(rewrittenEmpty), "//go:build darwin") || !strings.Contains(string(rewrittenEmpty), "This exists solely so we can linkname in symbols from syscall.") {
		t.Fatalf("rewritten empty.s = %s", rewrittenEmpty)
	}
	if got := digestBytes(rewrittenSys); got != xnetSocketReplacementSHA256 {
		t.Fatalf("rewritten sys_unix.go digest = %q, want %q", got, xnetSocketReplacementSHA256)
	}
	if got := digestBytes(rewrittenEmpty); got != xnetEmptyReplacementSHA256 {
		t.Fatalf("rewritten empty.s digest = %q, want %q", got, xnetEmptyReplacementSHA256)
	}
}

func TestRewriteXNetSocketRejectsSourceIdentityDrift(t *testing.T) {
	sysSource, emptySource := readPinnedXNetSocketSources(t)
	if _, err := rewriteAdapterSource(xnetModulePath, xnetRewrites[0], append(sysSource, '\n')); err == nil {
		t.Fatal("x/net rewrite accepted changed sys_unix.go")
	}
	if _, err := rewriteAdapterSource(xnetModulePath, xnetRewrites[1], append(emptySource, '\n')); err == nil {
		t.Fatal("x/net rewrite accepted changed empty.s")
	}
}

func TestPrepareXNetRecordsExactPrivateReplacement(t *testing.T) {
	identity := gomadversion.AdapterIdentity{Module: xnetModulePath, Version: xnetVersion, Sum: xnetSum}
	prepared, err := prepareXNet(pinnedModuleCache(t), t.TempDir(), identity)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.replacement != prepared.evidence.ReplacementRoot || prepared.evidence.Module != identity.Module || prepared.evidence.Version != identity.Version || prepared.evidence.Sum != identity.Sum {
		t.Fatalf("prepared adapter = %#v", prepared)
	}
	if prepared.evidence.OriginalSourceInventorySHA256 != xnetOriginalSourceInventorySHA256 || prepared.evidence.ReplacementSourceInventorySHA256 != xnetReplacementSourceInventorySHA256 {
		t.Fatalf("adapter inventory evidence = %#v", prepared.evidence)
	}
	if prepared.evidence.PreparedPackage != "golang.org/x/net/internal/socket" || prepared.evidence.PreparedSourceSetSHA256 != xnetPreparedSocketSourceSetSHA256 {
		t.Fatalf("prepared package evidence = %#v", prepared.evidence)
	}
	contents, err := os.ReadFile(prepared.evidence.Replacement)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(contents), "go:linkname") || !strings.Contains(string(contents), "return unix.ENOTSUP") {
		t.Fatalf("replacement source = %s", contents)
	}
}

func TestPrepareXNetRejectsChangedIdentity(t *testing.T) {
	identity := gomadversion.AdapterIdentity{Module: xnetModulePath, Version: "v0.57.1", Sum: "h1:changed"}
	if _, err := prepareXNet(pinnedModuleCache(t), t.TempDir(), identity); err == nil {
		t.Fatal("prepareXNet() accepted a changed identity")
	}
}

func TestXNetPreparedPackageSourceSetIdentity(t *testing.T) {
	workingDirectory := t.TempDir()
	moduleFile := "module golang.org/x/net/adaptertest\n\ngo 1.26.4\n\nrequire (\n\tgolang.org/x/net v0.58.0\n\tgolang.org/x/sys v0.47.0 // indirect\n)\n"
	sumFile := "golang.org/x/net v0.58.0 h1:ynWG7rqYi4ccpTEuPZ2QGWHktVEM9DMCj9yzDE0Q7To=\ngolang.org/x/net v0.58.0/go.mod h1:KpXc8iv+r3XplLAG/f7Jsf9RPszJzdR0f58q9vGOuEU=\ngolang.org/x/sys v0.47.0 h1:o7XGOvZQCADBQQ4Y7VNq2dRWQR7JmOUW8Kxx4ZsNgWs=\ngolang.org/x/sys v0.47.0/go.mod h1:4GL1E5IUh+htKOUEOaiffhrAeqysfVGipDYzABqnCmw=\n"
	for name, contents := range map[string]string{
		"go.mod":  moduleFile,
		"go.sum":  sumFile,
		"main.go": "package main\n\nimport _ \"golang.org/x/net/ipv4\"\n\nfunc main() {}\n",
	} {
		if err := os.WriteFile(filepath.Join(workingDirectory, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	toolchainRoot, err := filepath.Abs(filepath.Join("..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	spec, adapters, err := Default().PrepareBuildAdapters(target.Spec{
		Kind: target.KindGoRun, Source: ".", WorkingDir: workingDirectory,
		PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot,
	}, pinnedModuleCache(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(adapters) != 1 || adapters[0].Module != xnetModulePath {
		t.Fatalf("selected adapters = %#v", adapters)
	}
	if _, err := target.ReviewCapabilities(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
}

func readPinnedXNetSocketSources(t *testing.T) ([]byte, []byte) {
	t.Helper()
	root := filepath.Join(pinnedModuleCache(t), "golang.org", "x", "net@v0.58.0", "internal", "socket")
	sysSource, err := os.ReadFile(filepath.Join(root, "sys_unix.go"))
	if err != nil {
		t.Fatal(err)
	}
	emptySource, err := os.ReadFile(filepath.Join(root, "empty.s"))
	if err != nil {
		t.Fatal(err)
	}
	return sysSource, emptySource
}
