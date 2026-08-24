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

func TestPinnedGRPCModuleInventory(t *testing.T) {
	toolchainRoot, err := filepath.Abs(filepath.Join("..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(context.Background(), filepath.Join(toolchainRoot, "bin", "go"), "env", "GOMODCACHE")
	moduleCache, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	moduleRoot := filepath.Join(strings.TrimSpace(string(moduleCache)), "google.golang.org", "grpc@v1.80.0")
	got, err := target.DigestAdapterSourceInventory(moduleRoot)
	if err != nil {
		t.Fatal(err)
	}
	const want = "sha256:8b2fee8f36a0554c6652dccb95641983d2bf57c2efaa7af584c0ecc08ce6c1aa"
	if got != want {
		t.Fatalf("gRPC module inventory = %q, want %q", got, want)
	}
}

func TestRewriteGRPCKeepalivePreservesDialerWithoutHostControl(t *testing.T) {
	source := readPinnedGRPCKeepalive(t)
	rewritten, err := rewriteGRPCKeepalive(source)
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

func TestRewriteGRPCKeepaliveRejectsSourceIdentityDrift(t *testing.T) {
	source := append(readPinnedGRPCKeepalive(t), '\n')
	if _, err := rewriteGRPCKeepalive(source); err == nil {
		t.Fatal("rewriteGRPCKeepalive() accepted changed source")
	}
}

func TestRewriteGRPCKeepaliveSourceRejectsChangedAnchor(t *testing.T) {
	source := strings.Replace(string(readPinnedGRPCKeepalive(t)), "Control: func", "Control:  func", 1)
	if _, err := rewriteGRPCKeepaliveSource([]byte(source)); err == nil {
		t.Fatal("rewriteGRPCKeepaliveSource() accepted a changed anchor")
	}
}

func TestRewriteGRPCKeepaliveSourceRejectsDuplicateAnchor(t *testing.T) {
	source := readPinnedGRPCKeepalive(t)
	anchor := []byte("\t\tControl: func(_, _ string, c syscall.RawConn) error {\n\t\t\treturn c.Control(func(fd uintptr) {\n\t\t\t\tunix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_KEEPALIVE, 1)\n\t\t\t})\n\t\t},\n")
	source = append(source, anchor...)
	if _, err := rewriteGRPCKeepaliveSource(source); err == nil {
		t.Fatal("rewriteGRPCKeepaliveSource() accepted a duplicate anchor")
	}
}

func TestPrepareGRPCRecordsExactPrivateReplacement(t *testing.T) {
	moduleCache := pinnedModuleCache(t)
	root := t.TempDir()
	identity := gomadversion.AdapterIdentity{Module: "google.golang.org/grpc", Version: "v1.80.0", Sum: "h1:Xr6m2WmWZLETvUNvIUmeD5OAagMw3FiKmMlTdViWsHM="}
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

func TestProfileRejectsUnsupportedGRPCVersion(t *testing.T) {
	workingDirectory := t.TempDir()
	if err := os.WriteFile(filepath.Join(workingDirectory, "go.mod"), []byte("module example.test\n\ngo 1.26.4\n\nrequire google.golang.org/grpc v1.80.1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := Default().PrepareBuildAdapters(target.Spec{PreparationRoot: t.TempDir(), WorkingDir: workingDirectory}, t.TempDir())
	if err == nil || !IsInvalidBuildAdapterConfiguration(err) || !strings.Contains(err.Error(), "unsupported google.golang.org/grpc version") {
		t.Fatalf("PrepareBuildAdapters() error = %v", err)
	}
}

func TestProfileRejectsExistingGRPCReplacement(t *testing.T) {
	workingDirectory := t.TempDir()
	moduleFile := "module example.test\n\ngo 1.26.4\n\nrequire google.golang.org/grpc v1.80.0\n\nreplace google.golang.org/grpc => ./grpc\n"
	if err := os.WriteFile(filepath.Join(workingDirectory, "go.mod"), []byte(moduleFile), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := Default().PrepareBuildAdapters(target.Spec{PreparationRoot: t.TempDir(), WorkingDir: workingDirectory}, t.TempDir())
	if err == nil || !IsInvalidBuildAdapterConfiguration(err) || !strings.Contains(err.Error(), "already replaces google.golang.org/grpc") {
		t.Fatalf("PrepareBuildAdapters() error = %v", err)
	}
}

func TestProfileRejectsExistingGRPCReplacementBlock(t *testing.T) {
	workingDirectory := t.TempDir()
	moduleFile := "module example.test\n\ngo 1.26.4\n\nrequire google.golang.org/grpc v1.80.0\n\nreplace (\n\tgoogle.golang.org/grpc => ./grpc\n)\n"
	if err := os.WriteFile(filepath.Join(workingDirectory, "go.mod"), []byte(moduleFile), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := Default().PrepareBuildAdapters(target.Spec{PreparationRoot: t.TempDir(), WorkingDir: workingDirectory}, t.TempDir())
	if err == nil || !IsInvalidBuildAdapterConfiguration(err) || !strings.Contains(err.Error(), "already replaces google.golang.org/grpc") {
		t.Fatalf("PrepareBuildAdapters() error = %v", err)
	}
}

func TestProfileRejectsChangedGRPCModuleSum(t *testing.T) {
	workingDirectory := t.TempDir()
	if err := os.WriteFile(filepath.Join(workingDirectory, "go.mod"), []byte("module example.test\n\ngo 1.26.4\n\nrequire google.golang.org/grpc v1.80.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workingDirectory, "go.sum"), []byte("google.golang.org/grpc v1.80.0 h1:changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := Default().PrepareBuildAdapters(target.Spec{PreparationRoot: t.TempDir(), WorkingDir: workingDirectory}, pinnedModuleCache(t))
	if err == nil || !IsInvalidBuildAdapterConfiguration(err) || !strings.Contains(err.Error(), "module sum") {
		t.Fatalf("PrepareBuildAdapters() error = %v", err)
	}
}

func TestProfileRejectsBuildModFileWithGRPCAdapter(t *testing.T) {
	workingDirectory := t.TempDir()
	if err := os.WriteFile(filepath.Join(workingDirectory, "go.mod"), []byte("module example.test\n\ngo 1.26.4\n\nrequire google.golang.org/grpc v1.80.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := Default().PrepareBuildAdapters(target.Spec{
		PreparationRoot: t.TempDir(), WorkingDir: workingDirectory, BuildModFile: filepath.Join(workingDirectory, "existing.mod"),
	}, t.TempDir())
	if err == nil || !IsInvalidBuildAdapterConfiguration(err) || !strings.Contains(err.Error(), "existing build modfile") {
		t.Fatalf("PrepareBuildAdapters() error = %v", err)
	}
}

func TestVerifyGRPCModuleRejectsInventoryDrift(t *testing.T) {
	moduleRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(moduleRoot, "go.mod"), []byte("module google.golang.org/grpc\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyGRPCModule(moduleRoot); err == nil {
		t.Fatal("verifyGRPCModule() accepted changed module inventory")
	}
}

func readPinnedGRPCKeepalive(t *testing.T) []byte {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(pinnedModuleCache(t), "google.golang.org", "grpc@v1.80.0", "internal", "tcp_keepalive_unix.go"))
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
