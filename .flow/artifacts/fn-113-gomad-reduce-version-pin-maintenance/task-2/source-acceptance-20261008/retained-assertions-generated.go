package deterministicio

import (
"encoding/json"
"go.temporal.io/server/tools/gomad3/target"
"go/parser"
"go/token"
"os"
"os/exec"
"path/filepath"
"reflect"
"slices"
"strings"
"testing"
compatibility "go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

func TestPortableRetainedRewriteAdapterSourceRejectsDrift(t *testing.T) {
	moduleCache := portableRetainedModuleCache(t)
	moduleRoot := filepath.Join(moduleCache, "go.uber.org", "fx@"+fxVersion)
	source, err := readAdapterSource(fxModulePath, moduleRoot, fxSignalPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rewriteAdapterSource(fxModulePath, fxRewrites[0], append(source, '\n')); err == nil {
		t.Fatal("rewriteAdapterSource() accepted changed source")
	}
	duplicated := sourceRewrite{path: fxSignalPath, sourceSHA256: fxSignalSourceSHA256, replacementSHA256: fxSignalReplacementSHA256, rewrites: []anchorRewrite{{anchor: []byte("\trecv.m.Lock()\n")}}}
	if _, err := rewriteAdapterSource(fxModulePath, duplicated, source); err == nil || !strings.Contains(err.Error(), "anchor mismatch") {
		t.Fatalf("rewriteAdapterSource() ambiguous anchor error = %v", err)
	}
	wrongOutput := sourceRewrite{path: fxSignalPath, sourceSHA256: fxSignalSourceSHA256, replacementSHA256: fxSignalSourceSHA256, rewrites: fxRewrites[0].rewrites}
	if _, err := rewriteAdapterSource(fxModulePath, wrongOutput, source); err == nil || !strings.Contains(err.Error(), "replacement identity mismatch") {
		t.Fatalf("rewriteAdapterSource() replacement digest error = %v", err)
	}
}

// TestPinnedAdapterModuleInventories digests each adapted module in the
// pinned toolchain's module cache against its reviewed source inventory.
func TestPortableRetainedPinnedAdapterModuleInventories(t *testing.T) {
	moduleCache := portableRetainedModuleCache(t)
	for _, test := range []struct{ name, moduleDirectory, want string }{
		{name: "gRPC", moduleDirectory: "google.golang.org/grpc@v1.83.2", want: "sha256:53960aeb3f1d34cfe2340c30365456689710cd7bf32b6faf6e39d6f5306fc9a9"},
		{name: "x/net", moduleDirectory: "golang.org/x/net@v0.58.0", want: xnetOriginalSourceInventorySHA256},
		{name: "modernc memory", moduleDirectory: "modernc.org/memory@v1.11.0", want: memoryOriginalSourceInventorySHA256},
	} {
		got, err := digestAdapterSourceInventory(filepath.Join(moduleCache, filepath.FromSlash(test.moduleDirectory)))
		if err != nil {
			t.Fatal(err)
		}
		if got != test.want {
			t.Fatalf("%s module inventory = %q, want %q", test.name, got, test.want)
		}
	}
}

func TestPortableRetainedRewriteGRPCKeepalivePreservesDialerWithoutHostControl(t *testing.T) {
	source := portableRetainedReadPinnedGRPCKeepalive(t)
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

func TestPortableRetainedRewriteGRPCLinuxSourcesCompileTheNonLinuxImplementations(t *testing.T) {
	moduleRoot := filepath.Join(portableRetainedModuleCache(t), "google.golang.org", "grpc@"+grpcVersion)
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

func TestPortableRetainedRewriteGRPCKeepaliveRejectsSourceIdentityDrift(t *testing.T) {
	source := append(portableRetainedReadPinnedGRPCKeepalive(t), '\n')
	if _, err := rewriteAdapterSource(grpcModulePath, grpcKeepaliveRewrite, source); err == nil {
		t.Fatal("gRPC keepalive rewrite accepted changed source")
	}
}

func TestPortableRetainedRewriteGRPCKeepaliveSourceRejectsChangedAnchor(t *testing.T) {
	source := strings.Replace(string(portableRetainedReadPinnedGRPCKeepalive(t)), "Control: func", "Control:  func", 1)
	if _, err := applyAdapterAnchors(grpcModulePath, grpcKeepalivePath, grpcKeepaliveRewrite.rewrites, []byte(source)); err == nil {
		t.Fatal("gRPC keepalive anchors accepted a changed anchor")
	}
}

func TestPortableRetainedRewriteGRPCKeepaliveSourceRejectsDuplicateAnchor(t *testing.T) {
	source := portableRetainedReadPinnedGRPCKeepalive(t)
	anchor := []byte("\t\tControl: func(_, _ string, c syscall.RawConn) error {\n\t\t\treturn c.Control(func(fd uintptr) {\n\t\t\t\tunix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_KEEPALIVE, 1)\n\t\t\t})\n\t\t},\n")
	source = append(source, anchor...)
	if _, err := applyAdapterAnchors(grpcModulePath, grpcKeepalivePath, grpcKeepaliveRewrite.rewrites, source); err == nil {
		t.Fatal("gRPC keepalive anchors accepted a duplicate anchor")
	}
}

func TestPortableRetainedPrepareGRPCRecordsExactPrivateReplacement(t *testing.T) {
	moduleCache := portableRetainedModuleCache(t)
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

func TestPortableRetainedPrepareGRPCRejectsChangedIdentity(t *testing.T) {
	identity := gomadversion.AdapterIdentity{Module: "google.golang.org/grpc", Version: "v1.80.1", Sum: "h1:changed"}
	if _, err := prepareGRPC(portableRetainedModuleCache(t), t.TempDir(), identity); err == nil {
		t.Fatal("prepareGRPC() accepted a changed identity")
	}
}

func portableRetainedReadPinnedGRPCKeepalive(t *testing.T) []byte {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(portableRetainedModuleCache(t), "google.golang.org", "grpc@v1.83.2", "internal", "tcp_keepalive_unix.go"))
	if err != nil {
		t.Fatal(err)
	}
	return contents
}

func TestPortableRetainedGRPCDNSRewrite(t *testing.T) {
	portableRetainedDownloadPinnedModule(t, grpcModulePath, grpcVersion)
	rewrite := grpcDNSRewrites[0]
	moduleRoot := filepath.Join(portableRetainedModuleCache(t), "google.golang.org", "grpc@"+grpcVersion)
	source, err := readAdapterSource(grpcModulePath, moduleRoot, rewrite.path)
	if err != nil {
		t.Fatal(err)
	}
	replacement, err := rewriteAdapterSource(grpcModulePath, rewrite, source)
	if err != nil {
		t.Fatal(err)
	}
	comments := func(contents []byte) []string {
		t.Helper()
		file, err := parser.ParseFile(token.NewFileSet(), rewrite.path, contents, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		var result []string
		for _, group := range file.Comments {
			for _, comment := range group.List {
				result = append(result, comment.Text)
			}
		}
		return result
	}
	if !reflect.DeepEqual(comments(source), comments(replacement)) {
		t.Fatal("gRPC DNS adapter changed original comments")
	}
	for _, test := range []struct {
		name, want string
		change     func(*sourceRewrite, *[]byte)
	}{
		{name: "source", want: "source identity mismatch", change: func(_ *sourceRewrite, source *[]byte) { *source = append(*source, '\n') }},
		{name: "missing-anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) {
			rewrite.rewrites[0].anchor = []byte("absent resolver factory")
		}},
		{name: "ambiguous-anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) { rewrite.rewrites[0].anchor = []byte("return nil") }},
		{name: "replacement", want: "replacement identity mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) { rewrite.replacementSHA256 = rewrite.sourceSHA256 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := rewrite
			changed.rewrites = append([]anchorRewrite(nil), rewrite.rewrites...)
			contents := append([]byte(nil), source...)
			test.change(&changed, &contents)
			if _, err := rewriteAdapterSource(grpcModulePath, changed, contents); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("gRPC DNS rewrite: %v, want %s", err, test.want)
			}
		})
	}
}

func TestPortableRetainedGRPCDNSConsumer(t *testing.T) {
	portableRetainedDownloadPinnedModule(t, grpcModulePath, grpcVersion)
	prepared, err := prepareGRPC(portableRetainedModuleCache(t), t.TempDir(), gomadversion.AdapterIdentity{
		Module: grpcModulePath, Version: grpcVersion, Sum: grpcSum,
	})
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := os.ReadFile(filepath.Join("testdata", "grpcdns", "dns_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(prepared.replacement, "internal", "resolver", "dns", "gomad_dns_test.go"), fixture, 0o600); err != nil {
		t.Fatal(err)
	}
	goCommand, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(t.Context(), goCommand, "test", "-tags=test_dep", "-v", "-p=2", "-mod=readonly", "-run=^TestGomadDNS", "./internal/resolver/dns")
	command.Dir = prepared.replacement
	command.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("prepared gRPC DNS consumer: %v\n%s", err, output)
	}
	t.Logf("prepared gRPC DNS consumer:\n%s", output)
}

func TestPortableRetainedRewriteModerncMemoryModelsOnlyAnonymousAllocatorMappings(t *testing.T) {
	source := portableRetainedReadPinnedModerncMemorySource(t)
	rewritten, err := rewriteAdapterSource(memoryModulePath, memoryRewrites[0], source)
	if err != nil {
		t.Fatal(err)
	}
	for _, retained := range []string{"return unix.MunmapPtr", "unix.MmapPtr(-1", "Ask for more so we can align"} {
		if !strings.Contains(string(rewritten), retained) {
			t.Fatalf("rewritten mmap_unix.go omitted %q", retained)
		}
	}
	for _, modeled := range []string{"gomadMemoryEnabled()", "gomadMemoryMap(uintptr(size), pageSize)", "gomadMemoryUnmap(addr, uintptr(size))"} {
		if !strings.Contains(string(rewritten), modeled) {
			t.Fatalf("rewritten mmap_unix.go omitted %q", modeled)
		}
	}
	if got := digestBytes(rewritten); got != memoryMmapReplacementSHA256 {
		t.Fatalf("rewritten mmap_unix.go digest = %q, want %q", got, memoryMmapReplacementSHA256)
	}
}

func TestPortableRetainedRewriteModerncMemoryRejectsSourceIdentityDrift(t *testing.T) {
	if _, err := rewriteAdapterSource(memoryModulePath, memoryRewrites[0], append(portableRetainedReadPinnedModerncMemorySource(t), '\n')); err == nil {
		t.Fatal("rewriteAdapterSource() accepted changed mmap_unix.go")
	}
}

func TestPortableRetainedModerncMemoryRewriteRejectsAnchorDrift(t *testing.T) {
	source := portableRetainedReadPinnedModerncMemorySource(t)
	for _, test := range []struct {
		name, want string
		change     func(*sourceRewrite)
	}{
		{name: "missing-anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite) { rewrite.rewrites[0].anchor = []byte("absent page size anchor") }},
		{name: "ambiguous-anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite) { rewrite.rewrites[0].anchor = []byte("osPageSize") }},
		{name: "replacement-digest", want: "replacement identity mismatch", change: func(rewrite *sourceRewrite) { rewrite.replacementSHA256 = memoryMmapSourceSHA256 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			rewrite := memoryRewrites[0]
			rewrite.rewrites = append([]anchorRewrite(nil), rewrite.rewrites...)
			test.change(&rewrite)
			if _, err := rewriteAdapterSource(memoryModulePath, rewrite, source); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("modernc memory rewrite: %v, want %s", err, test.want)
			}
		})
	}
}

func TestPortableRetainedPrepareModerncMemoryRejectsModuleDrift(t *testing.T) {
	identity := gomadversion.AdapterIdentity{Module: memoryModulePath, Version: memoryVersion, Sum: memorySum}
	pinned := filepath.Join(portableRetainedModuleCache(t), "modernc.org", "memory@"+memoryVersion)
	for _, test := range []struct {
		name, want, wantRead string
		change               func(t *testing.T, path string, contents []byte)
	}{
		{name: "changed-source", want: "source inventory identity mismatch", change: func(t *testing.T, path string, contents []byte) {
			if err := os.WriteFile(path, append(contents, '\n'), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "non-regular-source", want: "symbolic link", wantRead: "not a regular file", change: func(t *testing.T, path string, contents []byte) {
			outside := filepath.Join(t.TempDir(), memoryMmapPath)
			if err := os.WriteFile(outside, contents, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, path); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			moduleCache := t.TempDir()
			moduleRoot := filepath.Join(moduleCache, "modernc.org", "memory@"+memoryVersion)
			if err := os.MkdirAll(filepath.Dir(moduleRoot), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := copyAdapterModule(pinned, moduleRoot, nil, defaultAdapterCopyLimits); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(moduleRoot, memoryMmapPath)
			contents, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			test.change(t, path, contents)
			if _, err := prepareModerncMemory(moduleCache, t.TempDir(), identity); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("changed modernc memory module: %v, want %s", err, test.want)
			}
			if test.wantRead == "" {
				return
			}
			if _, err := readAdapterSource(memoryModulePath, moduleRoot, memoryMmapPath); err == nil || !strings.Contains(err.Error(), test.wantRead) {
				t.Fatalf("changed modernc memory source: %v, want %s", err, test.wantRead)
			}
		})
	}
}

func TestPortableRetainedModerncMemoryRejectsChangedReplacementInventory(t *testing.T) {
	identity := gomadversion.AdapterIdentity{Module: memoryModulePath, Version: memoryVersion, Sum: memorySum}
	changed := memoryAdapter
	changed.replacementInventorySHA256 = memoryOriginalSourceInventorySHA256
	_, err := prepareRewrittenModule(portableRetainedModuleCache(t), t.TempDir(), identity, changed)
	if err == nil || !strings.Contains(err.Error(), "replacement inventory identity mismatch") {
		t.Fatalf("changed modernc memory replacement inventory: %v", err)
	}
}

func TestPortableRetainedPrepareModerncMemoryRecordsExactPrivateReplacement(t *testing.T) {
	identity := gomadversion.AdapterIdentity{Module: memoryModulePath, Version: memoryVersion, Sum: memorySum}
	prepared, err := prepareModerncMemory(portableRetainedModuleCache(t), t.TempDir(), identity)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.replacement != prepared.evidence.ReplacementRoot || prepared.evidence.Module != identity.Module || prepared.evidence.Version != identity.Version || prepared.evidence.Sum != identity.Sum {
		t.Fatalf("prepared adapter = %#v", prepared)
	}
	if prepared.evidence.PreparedPackage != memoryModulePath || prepared.evidence.PreparedSourceSetSHA256 != memoryPreparedSourceSetSHA256 {
		t.Fatalf("prepared package evidence = %#v", prepared.evidence)
	}
	if prepared.evidence.SourceSHA256 != memoryMmapSourceSHA256 || prepared.evidence.ReplacementSHA256 != memoryMmapReplacementSHA256 || prepared.evidence.OriginalSourceInventorySHA256 != memoryOriginalSourceInventorySHA256 || prepared.evidence.ReplacementSourceInventorySHA256 != memoryReplacementSourceInventorySHA256 {
		t.Fatalf("prepared identity evidence = %#v", prepared.evidence)
	}
	contents, err := os.ReadFile(prepared.evidence.Replacement)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "gomadMemoryMap") || !strings.Contains(string(contents), "unix.MmapPtr(-1") {
		t.Fatalf("replacement source = %s", contents)
	}
}

func TestPortableRetainedPrepareModerncMemoryRejectsChangedIdentity(t *testing.T) {
	for _, identity := range []gomadversion.AdapterIdentity{
		{Module: memoryModulePath, Version: "v1.11.1", Sum: "h1:changed"},
		{Module: memoryModulePath, Version: "v1.11.1", Sum: memorySum},
		{Module: memoryModulePath, Version: memoryVersion, Sum: "h1:changed"},
	} {
		if _, err := prepareModerncMemory(portableRetainedModuleCache(t), t.TempDir(), identity); err == nil {
			t.Fatalf("prepareModerncMemory() accepted a changed identity %#v", identity)
		}
	}
}

func portableRetainedReadPinnedModerncMemorySource(t *testing.T) []byte {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(portableRetainedModuleCache(t), "modernc.org", "memory@v1.11.0", memoryMmapPath))
	if err != nil {
		t.Fatal(err)
	}
	return contents
}

func TestPortableRetainedSockaddrBoundaryConsumer(t *testing.T) {
	portableRetainedDownloadPinnedModule(t, sockaddrModulePath, sockaddrVersion)
	workingDirectory := t.TempDir()
	for _, name := range []string{"go.mod", "go.sum", "main.go", "boundary_test.go"} {
		contents, err := os.ReadFile(filepath.Join("testdata", "sockaddr", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(workingDirectory, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	registry, err := newAdapterRegistry([]gomadversion.AdapterIdentity{sockaddrBoundaryAdapterIdentity()}, []adapterImplementation{
		{module: sockaddrModulePath, prepare: prepareSockaddr},
	})
	if err != nil {
		t.Fatal(err)
	}
	spec, adapters, err := registry.prepare(target.Spec{
		Kind: target.KindGoTest, Source: ".", WorkingDir: workingDirectory, PreparationRoot: t.TempDir(),
	}, portableRetainedModuleCache(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(adapters) != 1 || adapters[0].Module != sockaddrModulePath {
		t.Fatalf("sockaddr consumer adapter selection = %#v", adapters)
	}
	goCommand, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	run := func(dir string, environment []string, args ...string) []byte {
		t.Helper()
		command := exec.CommandContext(t.Context(), goCommand, args...)
		command.Dir = dir
		command.Env = append(os.Environ(), append([]string{"GOWORK=off", "GOFLAGS="}, environment...)...)
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("prepared sockaddr consumer %v: %v\n%s", args, err, output)
		}
		return output
	}
	output := run(workingDirectory, nil, "test", "-tags=test_dep", "-v", "-p=2", "-buildvcs=false", "-mod=readonly", "-modfile="+spec.BuildModFile, ".")
	t.Logf("prepared stock sockaddr consumer:\n%s", output)
	output = run(adapters[0].ReplacementRoot, nil, "test", "-tags=test_dep", "-v", "-p=2", "-mod=readonly", "-run=^Test(SockAddr_IPv[46]Addr|IPv[46])", ".")
	t.Logf("prepared upstream address computations:\n%s", output)
	for _, platform := range []struct{ goos, goarch, pin string }{
		{"darwin", "arm64", "sha256:05bad9b7f5550962a542a4b8c7101d135b1d0fdf199f2c01ce4cc9f33ab96f45"}, {"linux", "amd64", "sha256:08ac1f34ac338d5d5090f13f6a9dbed65cadb17c473cf6ed1f9b6b4813990b43"},
	} {
		environment := []string{"GOOS=" + platform.goos, "GOARCH=" + platform.goarch, "CGO_ENABLED=0"}
		output := run(workingDirectory, environment, "list", "-mod=readonly", "-modfile="+spec.BuildModFile, "-json", sockaddrModulePath)
		var pkg struct {
			Dir     string
			GoFiles []string
		}
		if err := json.Unmarshal(output, &pkg); err != nil {
			t.Fatal(err)
		}
		slices.Sort(pkg.GoFiles)
		sources := make([]compatibility.Source, len(pkg.GoFiles))
		for index, name := range pkg.GoFiles {
			contents, err := os.ReadFile(filepath.Join(pkg.Dir, name))
			if err != nil {
				t.Fatal(err)
			}
			sources[index] = compatibility.Source{Name: name, SHA256: digestBytes(contents)}
		}
		if got := compatibility.DigestSources(sources); got != platform.pin {
			t.Fatalf("prepared sockaddr source set on %s/%s = %s, want %s", platform.goos, platform.goarch, got, platform.pin)
		}
		t.Logf("prepared %s/%s source set %s", platform.goos, platform.goarch, platform.pin)
	}
}

func TestPortableRetainedSockaddrBoundaryRewritesRejectDrift(t *testing.T) {
	portableRetainedDownloadPinnedModule(t, sockaddrModulePath, sockaddrVersion)
	moduleRoot := filepath.Join(portableRetainedModuleCache(t), "github.com", "hashicorp", "go-sockaddr@"+sockaddrVersion)
	for _, original := range sockaddrRewrites {
		t.Run(original.path, func(t *testing.T) {
			source, err := readAdapterSource(sockaddrModulePath, moduleRoot, original.path)
			if err != nil {
				t.Fatal(err)
			}
			for _, test := range []struct {
				name, want string
				change     func(*sourceRewrite, *[]byte)
			}{
				{name: "upstream-file", want: "source identity mismatch", change: func(_ *sourceRewrite, source *[]byte) { *source = append(*source, '\n') }},
				{name: "missing-anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) { rewrite.rewrites[0].anchor = []byte("absent boundary anchor") }},
				{name: "ambiguous-anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) { rewrite.rewrites[0].anchor = []byte("func") }},
				{name: "replacement-digest", want: "replacement identity mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) { rewrite.replacementSHA256 = original.sourceSHA256 }},
			} {
				t.Run(test.name, func(t *testing.T) {
					rewrite := original
					rewrite.rewrites = append([]anchorRewrite(nil), rewrite.rewrites...)
					contents := append([]byte(nil), source...)
					test.change(&rewrite, &contents)
					if _, err := rewriteAdapterSource(sockaddrModulePath, rewrite, contents); err == nil || !strings.Contains(err.Error(), test.want) {
						t.Fatalf("sockaddr rewrite: %v, want %s", err, test.want)
					}
				})
			}
		})
	}
}

func TestPortableRetainedSockaddrBoundaryPreservesOriginalComments(t *testing.T) {
	portableRetainedDownloadPinnedModule(t, sockaddrModulePath, sockaddrVersion)
	moduleRoot := filepath.Join(portableRetainedModuleCache(t), "github.com", "hashicorp", "go-sockaddr@"+sockaddrVersion)
	for _, rewrite := range sockaddrRewrites {
		t.Run(rewrite.path, func(t *testing.T) {
			source, err := readAdapterSource(sockaddrModulePath, moduleRoot, rewrite.path)
			if err != nil {
				t.Fatal(err)
			}
			replacement, err := rewriteAdapterSource(sockaddrModulePath, rewrite, source)
			if err != nil {
				t.Fatal(err)
			}
			comments := func(contents []byte) []string {
				t.Helper()
				file, err := parser.ParseFile(token.NewFileSet(), rewrite.path, contents, parser.ParseComments)
				if err != nil {
					t.Fatal(err)
				}
				var result []string
				for _, group := range file.Comments {
					for _, comment := range group.List {
						result = append(result, comment.Text)
					}
				}
				return result
			}
			remaining := comments(replacement)
			for _, want := range comments(source) {
				index := slices.Index(remaining, want)
				if index < 0 {
					t.Fatalf("sockaddr adapter removed or reordered original comment %q", want)
				}
				remaining = remaining[index+1:]
			}
		})
	}
}

func TestPortableRetainedRewriteXNetSocketDeniesRawSocketOptions(t *testing.T) {
	sysSource, emptySource := portableRetainedReadPinnedXNetSocketSources(t)
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

func TestPortableRetainedRewriteXNetSocketRejectsSourceIdentityDrift(t *testing.T) {
	sysSource, emptySource := portableRetainedReadPinnedXNetSocketSources(t)
	if _, err := rewriteAdapterSource(xnetModulePath, xnetRewrites[0], append(sysSource, '\n')); err == nil {
		t.Fatal("x/net rewrite accepted changed sys_unix.go")
	}
	if _, err := rewriteAdapterSource(xnetModulePath, xnetRewrites[1], append(emptySource, '\n')); err == nil {
		t.Fatal("x/net rewrite accepted changed empty.s")
	}
}

func TestPortableRetainedPrepareXNetRecordsExactPrivateReplacement(t *testing.T) {
	identity := gomadversion.AdapterIdentity{Module: xnetModulePath, Version: xnetVersion, Sum: xnetSum}
	prepared, err := prepareXNet(portableRetainedModuleCache(t), t.TempDir(), identity)
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

func TestPortableRetainedPrepareXNetRejectsChangedIdentity(t *testing.T) {
	identity := gomadversion.AdapterIdentity{Module: xnetModulePath, Version: "v0.57.1", Sum: "h1:changed"}
	if _, err := prepareXNet(portableRetainedModuleCache(t), t.TempDir(), identity); err == nil {
		t.Fatal("prepareXNet() accepted a changed identity")
	}
}

func portableRetainedReadPinnedXNetSocketSources(t *testing.T) ([]byte, []byte) {
	t.Helper()
	root := filepath.Join(portableRetainedModuleCache(t), "golang.org", "x", "net@v0.58.0", "internal", "socket")
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

func portableRetainedModuleCache(t *testing.T) string {
	t.Helper()
	_, cache := portableAdapterGo(t)
	return cache
}

func portableRetainedDownloadPinnedModule(t *testing.T, name, version string) {
	t.Helper()
	portableAdapterModule(t, portableRetainedModuleCache(t), name, version)
}
