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
	grpcModulePath                        = "google.golang.org/grpc"
	grpcVersion                           = "v1.83.2"
	grpcSum                               = "h1:EManeRomTObA0BU7I8vXgg/78uE5MJ9M8B39EX2WscU="
	grpcOriginalSourceInventorySHA256     = "sha256:53960aeb3f1d34cfe2340c30365456689710cd7bf32b6faf6e39d6f5306fc9a9"
	grpcKeepaliveSourceSHA256             = "sha256:e8bfe03234b391d24006a3a274590111f0f8705fc5b25d9a78391bfdde3df32c"
	grpcKeepaliveReplacementSHA256        = "sha256:8705566fa6ba58f69d8c8215227ddadad46794c333bca38fe6d5399d6be24e8c"
	grpcReplacementSourceInventorySHA256  = "sha256:564d80fd13bb88861c7f5e5da650dd4a82699aae2e40043a85cb5a06c38936a3"
	grpcKeepalivePath                     = "internal/tcp_keepalive_unix.go"
	grpcChannelzLinuxPath                 = "internal/channelz/syscall_linux.go"
	grpcChannelzNonLinuxPath              = "internal/channelz/syscall_nonlinux.go"
	grpcChannelzLinuxSourceSHA256         = "sha256:6b90796f288344124b635c90237961789ca452ddf78e4b7f9c0417d0b458f61c"
	grpcChannelzNonLinuxSourceSHA256      = "sha256:1be23873d86f82b13b2c7d8d2534b5380215120d15ce3e47bf8d25c9fbdf983c"
	grpcChannelzLinuxReplacementSHA256    = "sha256:caa7ee7b9b6e324d88ad72047b1a5ff1120d82ebb4256243b4910aebedc9e287"
	grpcSyscallLinuxPath                  = "internal/syscall/syscall_linux.go"
	grpcSyscallNonLinuxPath               = "internal/syscall/syscall_nonlinux.go"
	grpcSyscallLinuxSourceSHA256          = "sha256:91096efdaa61581ac7b9d2d8076635d8ca5ac7ef3064094dab2f8efb70f003b2"
	grpcSyscallNonLinuxSourceSHA256       = "sha256:3b74e0c3889ed8a826963f5f3bf9e7bf2bab3ee05f6d2739c9f1bb75cca8d490"
	grpcSyscallLinuxReplacementSHA256     = "sha256:94aa678a38c485bfcdeb79f3e43a4df6db3d2755eadb15101acdc06579d222f0"
	grpcReadyReaderLinuxPath              = "internal/transport/readyreader/raw_conn_linux.go"
	grpcReadyReaderNonLinuxPath           = "internal/transport/readyreader/raw_conn_nonlinux.go"
	grpcReadyReaderLinuxSourceSHA256      = "sha256:05a068012fd512ce4ad3c5471f9546349b4e9acdad30dde5df0f02917af58d3f"
	grpcReadyReaderNonLinuxSourceSHA256   = "sha256:b9f0cf777d60fa3acfb2c22d263738a361b30ed484885b039db1c555a8f53256"
	grpcReadyReaderLinuxReplacementSHA256 = "sha256:00d8188fbf97bb5b6d63c19a36dfce662d3486f344854c10b55b3041dac1d4ce"
)

// grpcLinuxRewrites names the gRPC files that exist only for Linux hosts and
// reach the kernel through raw connections or x/sys/unix: channelz socket
// introspection, the TCP user-timeout and CPU-time helpers, and the
// non-blocking ready reader. The adapter replaces each with the module's own
// non-Linux implementation under a Linux build constraint, so linux/amd64
// targets take the path darwin already takes.
var grpcLinuxRewrites = []grpcLinuxRewrite{
	{
		linuxPath: grpcReadyReaderLinuxPath, nonLinuxPath: grpcReadyReaderNonLinuxPath,
		linuxSHA256: grpcReadyReaderLinuxSourceSHA256, nonLinuxSHA256: grpcReadyReaderNonLinuxSourceSHA256, replacementSHA256: grpcReadyReaderLinuxReplacementSHA256,
		constraint: []byte("//go:build !linux\n"), replacement: []byte("//go:build linux\n"),
	},
	{
		linuxPath: grpcChannelzLinuxPath, nonLinuxPath: grpcChannelzNonLinuxPath,
		linuxSHA256: grpcChannelzLinuxSourceSHA256, nonLinuxSHA256: grpcChannelzNonLinuxSourceSHA256, replacementSHA256: grpcChannelzLinuxReplacementSHA256,
		constraint: []byte("//go:build !linux\n"), replacement: []byte("//go:build linux\n"),
	},
	{
		linuxPath: grpcSyscallLinuxPath, nonLinuxPath: grpcSyscallNonLinuxPath,
		linuxSHA256: grpcSyscallLinuxSourceSHA256, nonLinuxSHA256: grpcSyscallNonLinuxSourceSHA256, replacementSHA256: grpcSyscallLinuxReplacementSHA256,
		constraint: []byte("//go:build !linux\n// +build !linux\n"), replacement: []byte("//go:build linux\n// +build linux\n"),
	},
}

type grpcLinuxRewrite struct {
	linuxPath, nonLinuxPath                        string
	linuxSHA256, nonLinuxSHA256, replacementSHA256 string
	constraint, replacement                        []byte
}

// grpcSyscallRewrites names the portable gRPC files that reach the syscall
// package on every platform: errno classification for the disconnect metric
// label, the syscall.Conn wrapper credentials install around TLS connections,
// and the raw-connection reader. Each rewrite keeps the exported surface and
// takes the path the module already takes when the connection is not a
// syscall.Conn, so the adapted module never asks the kernel for a descriptor.
var grpcSyscallRewrites = []sourceRewrite{
	{
		path:              "clientconn_disconnect_reason_noplan9.go",
		sourceSHA256:      "sha256:23f1d780ca5185a832b94dc81111af6ea8916360631b450268adf288fad0e68a",
		replacementSHA256: "sha256:308091ae5dc8440b748302f3ba47442bec3f0e203b0d7707b444764b9bff5a6e",
		rewrites: []anchorRewrite{
			{anchor: []byte("\t\"syscall\"\n")},
			{anchor: []byte("\tvar sysErr syscall.Errno\n")},
			{anchor: []byte("\tcase errors.Is(err, syscall.ECONNRESET):\n\t\treturn \"connection reset\"\n")},
			{
				anchor:      []byte("\tcase errors.Is(err, syscall.ETIMEDOUT), errors.Is(err, context.DeadlineExceeded), errors.Is(err, os.ErrDeadlineExceeded):\n"),
				replacement: []byte("\tcase errors.Is(err, context.DeadlineExceeded), errors.Is(err, os.ErrDeadlineExceeded):\n"),
			},
			{anchor: []byte("\tcase errors.Is(err, syscall.ECONNABORTED):\n\t\treturn \"connection aborted\"\n")},
			{anchor: []byte("\tcase errors.As(err, &sysErr):\n\t\treturn \"socket error\"\n")},
		},
	},
	{
		path:              "internal/credentials/syscallconn.go",
		sourceSHA256:      "sha256:47cb93c2b159a3d1e9373a49a44dc50744e51f40ff3f16414cd15c7ceb390ef7",
		replacementSHA256: "sha256:47138dcc61cefa213a4172c426bbf8230e22f0a6f59d8eb2cdef5d763aded021",
		rewrites: []anchorRewrite{
			{anchor: []byte("import (\n\t\"net\"\n\t\"syscall\"\n)\n"), replacement: []byte("import (\n\t\"net\"\n)\n")},
			{anchor: []byte("type sysConn = syscall.Conn\n\n")},
			{anchor: []byte("type syscallConn struct {\n\tnet.Conn\n\t// sysConn is a type alias of syscall.Conn. It's necessary because the name\n\t// `Conn` collides with `net.Conn`.\n\tsysConn\n}\n\n")},
			{
				anchor:      []byte("func WrapSyscallConn(rawConn, newConn net.Conn) net.Conn {\n\tsysConn, ok := rawConn.(syscall.Conn)\n\tif !ok {\n\t\treturn newConn\n\t}\n\treturn &syscallConn{\n\t\tConn:    newConn,\n\t\tsysConn: sysConn,\n\t}\n}\n"),
				replacement: []byte("func WrapSyscallConn(_, newConn net.Conn) net.Conn {\n\treturn newConn\n}\n"),
			},
		},
	},
	{
		path:              "internal/transport/readyreader/ready_reader.go",
		sourceSHA256:      "sha256:aefa4f5b55e10c0ff86d5cf5b5d77a44eb79ca00c412dc9218087f23537fdb00",
		replacementSHA256: "sha256:0d7d71fc3c499a71b9489fe1ebbe6f970af76636e2a8f4100341a965059e1510",
		rewrites: []anchorRewrite{
			{anchor: []byte("\t\"net\"\n\t\"syscall\"\n"), replacement: []byte("\t\"net\"\n")},
			{anchor: []byte("\traw syscall.RawConn\n"), replacement: []byte("\traw interface {\n\t\tRead(f func(fd uintptr) (done bool)) error\n\t}\n")},
			{
				anchor:      []byte("\tsysConn, ok := r.(syscall.Conn)\n\tif !ok {\n\t\treturn nil\n\t}\n\traw, err := sysConn.SyscallConn()\n\tif err != nil {\n\t\treturn nil\n\t}\n\trr := &nonBlockingReader{raw: raw}\n\trr.doRead = func(fd uintptr) bool {\n\t\ts := &rr.state\n\n\t\ts.buf = s.pool.Get(s.bufSize)\n\t\ts.bytesRead, s.readError = sysRead(fd, *s.buf)\n\n\t\tif s.readError != nil {\n\t\t\ts.pool.Put(s.buf)\n\t\t\ts.buf = nil\n\t\t}\n\t\treturn !wouldBlock(s.readError)\n\t}\n\treturn rr\n}\n"),
				replacement: []byte("\treturn nil\n}\n"),
			},
		},
	},
}

var grpcPreparedInternalSourceSetSHA256 = hostPin(map[string]string{
	"darwin/arm64": "sha256:348f37231e8391fd9361eb84ed9d5a39b9cacc4136461ce103ccc828be7db250",
	"linux/amd64":  "sha256:59a97baa8db98487dac40abe058ac89865e1ea2cf3d66c6d94cb622e7119d2a7",
})

func prepareGRPC(moduleCache, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
	if identity.Module != grpcModulePath || identity.Version != grpcVersion || identity.Sum != grpcSum {
		return adapterPreparation{}, errors.New("gRPC adapter identity mismatch")
	}
	moduleSource, err := filepath.EvalSymlinks(filepath.Join(moduleCache, "google.golang.org", "grpc@"+identity.Version))
	if err != nil {
		return adapterPreparation{}, fmt.Errorf("resolve pinned gRPC module: %w", err)
	}
	if err := verifyGRPCModule(moduleSource); err != nil {
		return adapterPreparation{}, err
	}
	source := filepath.Join(moduleSource, filepath.FromSlash(grpcKeepalivePath))
	contents, err := readGRPCAdapterSource(moduleSource, grpcKeepalivePath)
	if err != nil {
		return adapterPreparation{}, err
	}
	rewritten, err := rewriteGRPCKeepalive(contents)
	if err != nil {
		return adapterPreparation{}, err
	}
	replacements := map[string][]byte{grpcKeepalivePath: rewritten}
	for _, rewrite := range grpcLinuxRewrites {
		linuxSource, err := readGRPCAdapterSource(moduleSource, rewrite.linuxPath)
		if err != nil {
			return adapterPreparation{}, err
		}
		nonLinuxSource, err := readGRPCAdapterSource(moduleSource, rewrite.nonLinuxPath)
		if err != nil {
			return adapterPreparation{}, err
		}
		replacements[rewrite.linuxPath], err = rewriteGRPCLinuxSource(rewrite, linuxSource, nonLinuxSource)
		if err != nil {
			return adapterPreparation{}, err
		}
	}
	for _, rewrite := range grpcSyscallRewrites {
		portableSource, err := readGRPCAdapterSource(moduleSource, rewrite.path)
		if err != nil {
			return adapterPreparation{}, err
		}
		replacements[rewrite.path], err = rewriteAdapterSource(grpcModulePath, rewrite, portableSource)
		if err != nil {
			return adapterPreparation{}, err
		}
	}
	moduleReplacement := filepath.Join(root, "google-grpc")
	if err := copyAdapterModule(moduleSource, moduleReplacement, replacements, defaultAdapterCopyLimits); err != nil {
		return adapterPreparation{}, fmt.Errorf("copy gRPC adapter module: %w", err)
	}
	replacementInventory, err := digestAdapterSourceInventory(moduleReplacement)
	if err != nil {
		return adapterPreparation{}, fmt.Errorf("hash gRPC replacement inventory: %w", err)
	}
	if replacementInventory != grpcReplacementSourceInventorySHA256 {
		return adapterPreparation{}, fmt.Errorf("gRPC replacement inventory identity mismatch: got %s, want %s", replacementInventory, grpcReplacementSourceInventorySHA256)
	}
	return adapterPreparation{
		replacement: moduleReplacement,
		evidence: BuildAdapter{
			Module: identity.Module, Version: identity.Version, Sum: identity.Sum,
			Source: source, ReplacementRoot: moduleReplacement, Replacement: filepath.Join(moduleReplacement, filepath.FromSlash(grpcKeepalivePath)),
			PreparedPackage:                  grpcModulePath + "/internal",
			SourceSHA256:                     grpcKeepaliveSourceSHA256,
			ReplacementSHA256:                grpcKeepaliveReplacementSHA256,
			OriginalSourceInventorySHA256:    grpcOriginalSourceInventorySHA256,
			ReplacementSourceInventorySHA256: replacementInventory,
			PreparedSourceSetSHA256:          grpcPreparedInternalSourceSetSHA256,
		},
	}, nil
}

func verifyGRPCModule(moduleRoot string) error {
	inventory, err := digestAdapterSourceInventory(moduleRoot)
	if err != nil {
		return fmt.Errorf("hash pinned gRPC source inventory: %w", err)
	}
	if inventory != grpcOriginalSourceInventorySHA256 {
		return fmt.Errorf("pinned gRPC source inventory identity mismatch: got %s, want %s", inventory, grpcOriginalSourceInventorySHA256)
	}
	return nil
}

func readGRPCAdapterSource(moduleRoot, relative string) ([]byte, error) {
	path := filepath.Join(moduleRoot, filepath.FromSlash(relative))
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("pinned gRPC source is not a regular file: %s", relative)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pinned gRPC source %s: %w", relative, err)
	}
	return contents, nil
}

func rewriteGRPCLinuxSource(rewrite grpcLinuxRewrite, linuxSource, nonLinuxSource []byte) ([]byte, error) {
	if digestBytes(linuxSource) != rewrite.linuxSHA256 || digestBytes(nonLinuxSource) != rewrite.nonLinuxSHA256 {
		return nil, fmt.Errorf("pinned gRPC source identity mismatch for %s", rewrite.linuxPath)
	}
	if bytes.Count(nonLinuxSource, rewrite.constraint) != 1 {
		return nil, fmt.Errorf("pinned gRPC build constraint anchor mismatch for %s", rewrite.nonLinuxPath)
	}
	rewritten := bytes.Replace(nonLinuxSource, rewrite.constraint, rewrite.replacement, 1)
	if got := digestBytes(rewritten); got != rewrite.replacementSHA256 {
		return nil, fmt.Errorf("gRPC replacement identity mismatch for %s: got %s, want %s", rewrite.linuxPath, got, rewrite.replacementSHA256)
	}
	return rewritten, nil
}

func rewriteGRPCKeepalive(contents []byte) ([]byte, error) {
	if digestBytes(contents) != grpcKeepaliveSourceSHA256 {
		return nil, errors.New("pinned gRPC keepalive source identity mismatch")
	}
	rewritten, err := rewriteGRPCKeepaliveSource(contents)
	if err != nil {
		return nil, err
	}
	if digestBytes(rewritten) != grpcKeepaliveReplacementSHA256 {
		return nil, errors.New("gRPC keepalive replacement identity mismatch")
	}
	return rewritten, nil
}

func rewriteGRPCKeepaliveSource(contents []byte) ([]byte, error) {
	rewrites := []struct {
		anchor      []byte
		replacement []byte
	}{
		{anchor: []byte("\t\"syscall\"\n")},
		{anchor: []byte("\n\t\"golang.org/x/sys/unix\"\n"), replacement: []byte("\n")},
		{anchor: []byte("\t\tControl: func(_, _ string, c syscall.RawConn) error {\n\t\t\treturn c.Control(func(fd uintptr) {\n\t\t\t\tunix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_KEEPALIVE, 1)\n\t\t\t})\n\t\t},\n")},
	}
	result := append([]byte(nil), contents...)
	for _, rewrite := range rewrites {
		if bytes.Count(result, rewrite.anchor) != 1 {
			return nil, fmt.Errorf("pinned gRPC keepalive rewrite anchor mismatch for %q", rewrite.anchor)
		}
		result = bytes.Replace(result, rewrite.anchor, rewrite.replacement, 1)
	}
	return result, nil
}
