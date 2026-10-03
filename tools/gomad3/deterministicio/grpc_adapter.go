package deterministicio

import (
	"slices"

	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

const (
	grpcModulePath                        = "google.golang.org/grpc"
	grpcVersion                           = "v1.83.2"
	grpcSum                               = "h1:EManeRomTObA0BU7I8vXgg/78uE5MJ9M8B39EX2WscU="
	grpcOriginalSourceInventorySHA256     = "sha256:53960aeb3f1d34cfe2340c30365456689710cd7bf32b6faf6e39d6f5306fc9a9"
	grpcKeepaliveSourceSHA256             = "sha256:e8bfe03234b391d24006a3a274590111f0f8705fc5b25d9a78391bfdde3df32c"
	grpcKeepaliveReplacementSHA256        = "sha256:8705566fa6ba58f69d8c8215227ddadad46794c333bca38fe6d5399d6be24e8c"
	grpcReplacementSourceInventorySHA256  = "sha256:9098668e7b66e1d82f5ecbb302556cf9723775b19aa2db68ede45aa1a166adbf"
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
var grpcLinuxRewrites = []sourceRewrite{
	{
		path: grpcReadyReaderLinuxPath, sourceSHA256: grpcReadyReaderLinuxSourceSHA256,
		base: grpcReadyReaderNonLinuxPath, baseSHA256: grpcReadyReaderNonLinuxSourceSHA256,
		replacementSHA256: grpcReadyReaderLinuxReplacementSHA256,
		rewrites:          []anchorRewrite{{anchor: []byte("//go:build !linux\n"), replacement: []byte("//go:build linux\n")}},
	},
	{
		path: grpcChannelzLinuxPath, sourceSHA256: grpcChannelzLinuxSourceSHA256,
		base: grpcChannelzNonLinuxPath, baseSHA256: grpcChannelzNonLinuxSourceSHA256,
		replacementSHA256: grpcChannelzLinuxReplacementSHA256,
		rewrites:          []anchorRewrite{{anchor: []byte("//go:build !linux\n"), replacement: []byte("//go:build linux\n")}},
	},
	{
		path: grpcSyscallLinuxPath, sourceSHA256: grpcSyscallLinuxSourceSHA256,
		base: grpcSyscallNonLinuxPath, baseSHA256: grpcSyscallNonLinuxSourceSHA256,
		replacementSHA256: grpcSyscallLinuxReplacementSHA256,
		rewrites:          []anchorRewrite{{anchor: []byte("//go:build !linux\n// +build !linux\n"), replacement: []byte("//go:build linux\n// +build linux\n")}},
	},
}

// grpcKeepaliveRewrite drops the dialer's host socket-option control, so the
// virtual network's connections keep the module's keepalive defaults.
var grpcKeepaliveRewrite = sourceRewrite{
	path: grpcKeepalivePath, sourceSHA256: grpcKeepaliveSourceSHA256, replacementSHA256: grpcKeepaliveReplacementSHA256,
	rewrites: []anchorRewrite{
		{anchor: []byte("\t\"syscall\"\n")},
		{anchor: []byte("\n\t\"golang.org/x/sys/unix\"\n"), replacement: []byte("\n")},
		{anchor: []byte("\t\tControl: func(_, _ string, c syscall.RawConn) error {\n\t\t\treturn c.Control(func(fd uintptr) {\n\t\t\t\tunix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_KEEPALIVE, 1)\n\t\t\t})\n\t\t},\n")},
	},
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

var grpcDNSRewrites = []sourceRewrite{
	{
		path:              "internal/resolver/dns/dns_resolver.go",
		sourceSHA256:      "sha256:fae3828426c9cb90d31a0feafce53892434cfce8ec4ceb2f5cd7f823c2d67136",
		replacementSHA256: "sha256:990c975b23005dd89b766c501b1a07a1c9735431a5cb29d2da83a70ecca3c7d5",
		rewrites: []anchorRewrite{
			{anchor: []byte("var newNetResolver = func(authority string) (internal.NetResolver, error) {\n\tif authority == \"\" {\n\t\treturn net.DefaultResolver, nil\n\t}\n\n\thost, port, err := parseTarget(authority, defaultDNSSvrPort)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\tauthorityWithPort := net.JoinHostPort(host, port)\n\n\treturn &net.Resolver{\n\t\tPreferGo: true,\n\t\tDial:     internal.AddressDialer(authorityWithPort),\n\t}, nil\n}\n"), replacement: []byte("var newNetResolver = func(string) (internal.NetResolver, error) {\n\treturn nil, fmt.Errorf(\"gomad: DNS resolution is unavailable; use a literal IP target\")\n}\n")},
		},
	},
}

var grpcPreparedInternalSourceSetSHA256ByHost = map[string]string{
	"darwin/arm64": "sha256:59a97baa8db98487dac40abe058ac89865e1ea2cf3d66c6d94cb622e7119d2a7",
	"linux/amd64":  "sha256:59a97baa8db98487dac40abe058ac89865e1ea2cf3d66c6d94cb622e7119d2a7",
}

var grpcPreparedInternalSourceSetSHA256 = hostPin(grpcPreparedInternalSourceSetSHA256ByHost)

// grpcAdapter names the keepalive rewrite first: the build evidence records
// it as the adapter's source.
var grpcAdapter = rewrittenModule{
	module: grpcModulePath, version: grpcVersion, sum: grpcSum,
	cacheElements:                 []string{"google.golang.org", "grpc@" + grpcVersion},
	replacementDirectory:          "google-grpc",
	originalInventorySHA256:       grpcOriginalSourceInventorySHA256,
	replacementInventorySHA256:    grpcReplacementSourceInventorySHA256,
	preparedPackage:               grpcModulePath + "/internal",
	preparedSourceSetSHA256ByHost: grpcPreparedInternalSourceSetSHA256ByHost,
	rewrites:                      slices.Concat([]sourceRewrite{grpcKeepaliveRewrite}, grpcLinuxRewrites, grpcSyscallRewrites, grpcDNSRewrites),
}

func prepareGRPC(moduleCache, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
	return prepareRewrittenModule(moduleCache, root, identity, grpcAdapter)
}
