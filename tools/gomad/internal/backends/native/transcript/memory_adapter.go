package transcript

import gomadversion "go.temporal.io/server/tools/gomad/internal/backends/native/toolchain/version"

const (
	memoryModulePath                       = "modernc.org/memory"
	memoryVersion                          = "v1.11.0"
	memorySum                              = "h1:o4QC8aMQzmcwCK3t3Ux/ZHmwFPzE6hf2Y5LbkRs+hbI="
	memoryOriginalSourceInventorySHA256    = "sha256:360b00bba0f7a6fb8213804e7a344989eba5d0be18308dfc6d281ebca205b542"
	memoryMmapSourceSHA256                 = "sha256:d487e0d7f447b25397874a79e53c0e42b8568ed0503b562c959c83e8ef47f0a7"
	memoryMmapReplacementSHA256            = "sha256:c8a86dca80f526b39f0a855f59552d1085a352fb7fef47b86da827b854ab88ad"
	memoryReplacementSourceInventorySHA256 = "sha256:5d90ad2f458515ae2100121553e5cabbe3b908770a0095661df9b1bce972e2b6"
	memoryMmapPath                         = "mmap_unix.go"
)

var memoryPreparedSourceSetSHA256ByHost = map[string]string{
	"darwin/arm64": "sha256:7b3101da7f776587a8cf192c0757df9b7a8d2cdafdefd0e31446dc1a0250696f",
	"linux/amd64":  "sha256:7b3101da7f776587a8cf192c0757df9b7a8d2cdafdefd0e31446dc1a0250696f",
}

var memoryPreparedSourceSetSHA256 = hostPin(memoryPreparedSourceSetSHA256ByHost)

var memoryRewrites = []sourceRewrite{
	{
		path: memoryMmapPath, sourceSHA256: memoryMmapSourceSHA256, replacementSHA256: memoryMmapReplacementSHA256,
		rewrites: []anchorRewrite{
			{
				anchor:      []byte("var (\n\tosPageMask = osPageSize - 1\n\tosPageSize = os.Getpagesize()\n)\n"),
				replacement: []byte("var (\n\tosPageMask = osPageSize - 1\n\tosPageSize = os.Getpagesize()\n)\n\n//go:linkname gomadMemoryEnabled internal/gomadio.Enabled\nfunc gomadMemoryEnabled() bool\n\n//go:linkname gomadMemoryMap internal/gomadio.AnonymousMap\nfunc gomadMemoryMap(size, alignment uintptr) uintptr\n\n//go:linkname gomadMemoryUnmap internal/gomadio.AnonymousUnmap\nfunc gomadMemoryUnmap(address, size uintptr) bool\n"),
			},
			{
				anchor:      []byte("func unmap(addr uintptr, size int) error {\n\treturn unix.MunmapPtr(unsafe.Pointer(addr), uintptr(size))\n}\n"),
				replacement: []byte("func unmap(addr uintptr, size int) error {\n\tif gomadMemoryEnabled() {\n\t\tif !gomadMemoryUnmap(addr, uintptr(size)) {\n\t\t\treturn unix.EINVAL\n\t\t}\n\t\treturn nil\n\t}\n\treturn unix.MunmapPtr(unsafe.Pointer(addr), uintptr(size))\n}\n"),
			},
			{
				anchor:      []byte("\tsize = roundup(size, osPageSize)\n"),
				replacement: []byte("\tsize = roundup(size, osPageSize)\n\tif gomadMemoryEnabled() {\n\t\tp := gomadMemoryMap(uintptr(size), pageSize)\n\t\tif p == 0 {\n\t\t\treturn 0, 0, unix.ENOMEM\n\t\t}\n\t\treturn p, size, nil\n\t}\n"),
			},
		},
	},
}

var memoryAdapter = rewrittenModule{
	module: memoryModulePath, version: memoryVersion, sum: memorySum,
	cacheElements:                 []string{"modernc.org", "memory@" + memoryVersion},
	replacementDirectory:          "modernc-memory",
	originalInventorySHA256:       memoryOriginalSourceInventorySHA256,
	replacementInventorySHA256:    memoryReplacementSourceInventorySHA256,
	preparedPackage:               memoryModulePath,
	preparedSourceSetSHA256ByHost: memoryPreparedSourceSetSHA256ByHost,
	rewrites:                      memoryRewrites,
}

func prepareModerncMemory(moduleCache, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
	return prepareRewrittenModule(moduleCache, root, identity, memoryAdapter)
}
