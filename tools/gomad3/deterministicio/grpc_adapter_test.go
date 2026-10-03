package deterministicio

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/target"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

// TestPinnedAdapterModuleInventories digests each adapted module in the
// pinned toolchain's module cache against its reviewed source inventory.
func TestPinnedAdapterModuleInventories(t *testing.T) {
	moduleCache := pinnedModuleCache(t)
	for _, test := range []struct{ name, moduleDirectory, want string }{
		{name: "gRPC", moduleDirectory: "google.golang.org/grpc@v1.83.2", want: "sha256:53960aeb3f1d34cfe2340c30365456689710cd7bf32b6faf6e39d6f5306fc9a9"},
		{name: "x/net", moduleDirectory: "golang.org/x/net@v0.58.0", want: xnetOriginalSourceInventorySHA256},
		{name: "modernc memory", moduleDirectory: "modernc.org/memory@v1.11.0", want: memoryOriginalSourceInventorySHA256},
	} {
		got, err := target.DigestAdapterSourceInventory(filepath.Join(moduleCache, filepath.FromSlash(test.moduleDirectory)))
		if err != nil {
			t.Fatal(err)
		}
		if got != test.want {
			t.Fatalf("%s module inventory = %q, want %q", test.name, got, test.want)
		}
	}
}

func TestRewriteGRPCKeepalivePreservesDialerWithoutHostControl(t *testing.T) {
	source := readPinnedGRPCKeepalive(t)
	rewritten, err := rewriteAdapterSource(grpcModulePath, grpcKeepaliveRewrite, source)
	if err != nil {
		t.Fatal(err)
	}
	for _, retained := range []string{
		"//go:build unix", "Copyright 2023 gRPC authors.",
		"// NetDialerWithTCPKeepalive returns a net.Dialer that enables TCP keepalives on",
		"func NetDialerWithTCPKeepalive() *net.Dialer {", "KeepAlive: time.Duration(-1)",
		"// This method is called after the underlying network socket is created,",
	} {
		if !strings.Contains(string(rewritten), retained) {
			t.Fatalf("rewritten source omitted %q", retained)
		}
	}
	for _, removed := range []string{"\"syscall\"", "\"golang.org/x/sys/unix\"", "Control:", "RawConn", "SetsockoptInt"} {
		if strings.Contains(string(rewritten), removed) {
			t.Fatalf("rewritten source retained %q", removed)
		}
	}
	const wantDigest = "sha256:8705566fa6ba58f69d8c8215227ddadad46794c333bca38fe6d5399d6be24e8c"
	if got := digestBytes(rewritten); got != wantDigest {
		t.Fatalf("rewritten source digest = %q, want %q", got, wantDigest)
	}
}

func TestRewriteGRPCLinuxSourcesCompileTheNonLinuxImplementations(t *testing.T) {
	moduleRoot := filepath.Join(pinnedModuleCache(t), "google.golang.org", "grpc@"+grpcVersion)
	for _, rewrite := range grpcLinuxRewrites {
		rewritten, err := rewriteModuleSource(grpcModulePath, moduleRoot, rewrite)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(string(rewritten), "//go:build linux\n") || strings.Contains(string(rewritten), "!linux") {
			t.Fatalf("%s replacement constraint = %q", rewrite.path, strings.SplitN(string(rewritten), "\n", 2)[0])
		}
		for _, removed := range []string{"\"golang.org/x/sys/unix\"", "SyscallConn", "rawConn.Control", "Getsockopt(int(fd)", "SetsockoptInt"} {
			if strings.Contains(string(rewritten), removed) {
				t.Fatalf("%s replacement retained %q", rewrite.path, removed)
			}
		}
		changedLinux := rewrite
		changedLinux.sourceSHA256 = rewrite.baseSHA256
		if _, err := rewriteModuleSource(grpcModulePath, moduleRoot, changedLinux); err == nil {
			t.Fatalf("rewriteModuleSource() accepted a changed %s", rewrite.path)
		}
		changedBase := rewrite
		changedBase.baseSHA256 = rewrite.sourceSHA256
		if _, err := rewriteModuleSource(grpcModulePath, moduleRoot, changedBase); err == nil {
			t.Fatalf("rewriteModuleSource() accepted a changed %s", rewrite.base)
		}
	}
}

func TestRewriteGRPCKeepaliveRejectsSourceIdentityDrift(t *testing.T) {
	source := append(readPinnedGRPCKeepalive(t), '\n')
	if _, err := rewriteAdapterSource(grpcModulePath, grpcKeepaliveRewrite, source); err == nil {
		t.Fatal("gRPC keepalive rewrite accepted changed source")
	}
}

func TestRewriteGRPCKeepaliveSourceRejectsChangedAnchor(t *testing.T) {
	source := strings.Replace(string(readPinnedGRPCKeepalive(t)), "Control: func", "Control:  func", 1)
	if _, err := applyAdapterAnchors(grpcModulePath, grpcKeepalivePath, grpcKeepaliveRewrite.rewrites, []byte(source)); err == nil {
		t.Fatal("gRPC keepalive anchors accepted a changed anchor")
	}
}

func TestRewriteGRPCKeepaliveSourceRejectsDuplicateAnchor(t *testing.T) {
	source := readPinnedGRPCKeepalive(t)
	anchor := []byte("\t\tControl: func(_, _ string, c syscall.RawConn) error {\n\t\t\treturn c.Control(func(fd uintptr) {\n\t\t\t\tunix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_KEEPALIVE, 1)\n\t\t\t})\n\t\t},\n")
	source = append(source, anchor...)
	if _, err := applyAdapterAnchors(grpcModulePath, grpcKeepalivePath, grpcKeepaliveRewrite.rewrites, source); err == nil {
		t.Fatal("gRPC keepalive anchors accepted a duplicate anchor")
	}
}

func TestPrepareGRPCRecordsExactPrivateReplacement(t *testing.T) {
	moduleCache := pinnedModuleCache(t)
	root := t.TempDir()
	identity := gomadversion.AdapterIdentity{Module: "google.golang.org/grpc", Version: "v1.83.2", Sum: "h1:EManeRomTObA0BU7I8vXgg/78uE5MJ9M8B39EX2WscU="}
	prepared, err := prepareGRPC(moduleCache, root, identity)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.replacement != prepared.evidence.ReplacementRoot || prepared.evidence.Module != identity.Module || prepared.evidence.Version != identity.Version || prepared.evidence.Sum != identity.Sum {
		t.Fatalf("prepared adapter = %#v", prepared)
	}
	if prepared.evidence.OriginalSourceInventorySHA256 != grpcOriginalSourceInventorySHA256 || prepared.evidence.ReplacementSourceInventorySHA256 != grpcReplacementSourceInventorySHA256 || prepared.evidence.SourceSHA256 != grpcKeepaliveSourceSHA256 || prepared.evidence.ReplacementSHA256 != grpcKeepaliveReplacementSHA256 {
		t.Fatalf("adapter evidence = %#v", prepared.evidence)
	}
	if prepared.evidence.PreparedPackage != "google.golang.org/grpc/internal" || prepared.evidence.PreparedSourceSetSHA256 != grpcPreparedInternalSourceSetSHA256 {
		t.Fatalf("prepared package evidence = %#v", prepared.evidence)
	}
	contents, err := os.ReadFile(prepared.evidence.Replacement)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(contents), "Control:") || !strings.Contains(string(contents), "KeepAlive: time.Duration(-1)") {
		t.Fatalf("replacement source = %s", contents)
	}
}

func TestPrepareGRPCRejectsChangedIdentity(t *testing.T) {
	identity := gomadversion.AdapterIdentity{Module: "google.golang.org/grpc", Version: "v1.80.1", Sum: "h1:changed"}
	if _, err := prepareGRPC(pinnedModuleCache(t), t.TempDir(), identity); err == nil {
		t.Fatal("prepareGRPC() accepted a changed identity")
	}
}

func TestPrepareGRPCReturnsTypedInventoryCapacityError(t *testing.T) {
	moduleCache := t.TempDir()
	moduleRoot := filepath.Join(moduleCache, "google.golang.org", "grpc@"+grpcVersion)
	if err := os.MkdirAll(moduleRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	large := filepath.Join(moduleRoot, "large")
	if err := os.WriteFile(large, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(large, maximumModuleBytes+1); err != nil {
		t.Fatal(err)
	}
	_, err := prepareGRPC(moduleCache, t.TempDir(), gomadversion.AdapterIdentity{Module: grpcModulePath, Version: grpcVersion, Sum: grpcSum})
	var capacity *AdapterCapacityError
	if !errors.As(err, &capacity) || capacity.Resource != "bytes" {
		t.Fatalf("prepareGRPC() error = %#v", err)
	}
}

func TestVerifyGRPCModuleRejectsInventoryDrift(t *testing.T) {
	moduleRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(moduleRoot, "go.mod"), []byte("module google.golang.org/grpc\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyAdapterModuleInventory(grpcModulePath, moduleRoot, grpcOriginalSourceInventorySHA256); err == nil {
		t.Fatal("gRPC module inventory check accepted a changed module inventory")
	}
}

func readPinnedGRPCKeepalive(t *testing.T) []byte {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(pinnedModuleCache(t), "google.golang.org", "grpc@v1.83.2", "internal", "tcp_keepalive_unix.go"))
	if err != nil {
		t.Fatal(err)
	}
	return contents
}

func pinnedModuleCache(t *testing.T) string {
	t.Helper()
	toolchainRoot, err := filepath.Abs(filepath.Join("..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(context.Background(), filepath.Join(toolchainRoot, "bin", "go"), "env", "GOMODCACHE")
	moduleCache, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(moduleCache))
}
