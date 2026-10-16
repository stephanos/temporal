package transcript

import gomadversion "go.temporal.io/server/tools/gomad/internal/backends/native/toolchain/version"

const (
	xnetModulePath                       = "golang.org/x/net"
	xnetVersion                          = "v0.58.0"
	xnetSum                              = "h1:ynWG7rqYi4ccpTEuPZ2QGWHktVEM9DMCj9yzDE0Q7To="
	xnetOriginalSourceInventorySHA256    = "sha256:9b77aba70ef5ccc576db020dbb5c6e9af41a24f8adc30368d8a734e93d1ab45b"
	xnetSocketSourceSHA256               = "sha256:facf54b3bc8b1e36552241cdf5bf3f5cd1010cf864f995cb0cf2ed3830036d6c"
	xnetEmptySourceSHA256                = "sha256:0d09f2c52fc60c2d411818b538de77927fbf43ad530214066e26315922f5bdd6"
	xnetSocketReplacementSHA256          = "sha256:f7469c5b887c0c443d55bf7e03add926e55bacd5cfed76870c180635ca9d2bb8"
	xnetEmptyReplacementSHA256           = "sha256:a65c7dc68b8dded19a6cbd605e7e0f54a7b3e7d2638702e509ffae4592da1cff"
	xnetReplacementSourceInventorySHA256 = "sha256:11cc0769431cdecaa993a499890c4978b339805cdc50c0c320535c158e7170f1"
	xnetSocketPath                       = "internal/socket/sys_unix.go"
	xnetEmptyPath                        = "internal/socket/empty.s"
)

var xnetPreparedSocketSourceSetSHA256ByHost = map[string]string{
	"darwin/arm64": "sha256:d1e648a87394dcbe3a845b9db0ea092be4c17ba310090e315cffe8d8a46c2418",
	"linux/amd64":  "sha256:0da8d514d26715d2fd68d2c7e3222b635dfad7963d33a0113245084cbb8029ce",
}

var xnetPreparedSocketSourceSetSHA256 = hostPin(xnetPreparedSocketSourceSetSHA256ByHost)

// xnetRewrites deny raw socket options: the socket package's linknamed
// getsockopt and setsockopt become ENOTSUP, and the assembly stub that
// enabled the linkname on darwin is excluded from every build.
var xnetRewrites = []sourceRewrite{
	{
		path: xnetSocketPath, sourceSHA256: xnetSocketSourceSHA256, replacementSHA256: xnetSocketReplacementSHA256,
		rewrites: []anchorRewrite{
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
		},
	},
	{
		path: xnetEmptyPath, sourceSHA256: xnetEmptySourceSHA256, replacementSHA256: xnetEmptyReplacementSHA256,
		rewrites: []anchorRewrite{{anchor: []byte("//go:build darwin"), replacement: []byte("//go:build ignore")}},
	},
}

var xnetAdapter = rewrittenModule{
	module: xnetModulePath, version: xnetVersion, sum: xnetSum,
	cacheElements:                 []string{"golang.org", "x", "net@" + xnetVersion},
	replacementDirectory:          "golang-x-net",
	originalInventorySHA256:       xnetOriginalSourceInventorySHA256,
	replacementInventorySHA256:    xnetReplacementSourceInventorySHA256,
	preparedPackage:               xnetModulePath + "/internal/socket",
	preparedSourceSetSHA256ByHost: xnetPreparedSocketSourceSetSHA256ByHost,
	rewrites:                      xnetRewrites,
}

func prepareXNet(moduleCache, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
	return prepareRewrittenModule(moduleCache, root, identity, xnetAdapter)
}
