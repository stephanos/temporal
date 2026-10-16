package transcript

import gomadversion "go.temporal.io/server/tools/gomad/internal/backends/native/toolchain/version"

const (
	sprigModulePath                       = "github.com/Masterminds/sprig/v3"
	sprigVersion                          = "v3.3.0"
	sprigSum                              = "h1:mQh0Yrg1XPo6vjYXgtf5OtijNAKJRNcTdOOGZe3tPhs="
	sprigOriginalSourceInventorySHA256    = "sha256:0fd7e47099215b4b1f89700512adb2f468adb21d573d5a0ba7b567a06a684059"
	sprigReplacementSourceInventorySHA256 = "sha256:4073913c5c77980e1cae908337e24fa775746bd3bbd19866d8440b6e78cbe32d"
	sprigNetworkPath                      = "network.go"
	sprigNetworkSourceSHA256              = "sha256:ffc5b88a9aa83cb4ac331cadd1792c2d493ad65629eec462241104c0afc225ca"
	sprigNetworkReplacementSHA256         = "sha256:7783a960bbefa4e8bb28849af27de06c20e045bfb56c37dd9711f33030a4fc9e"
)

var sprigPreparedSourceSetSHA256ByHost = map[string]string{
	"darwin/arm64": "sha256:b6619946a45458e6ca507769f6e3b30f3e0edf0e9ca8c1631319b4238cb4a00d",
	"linux/amd64":  "sha256:b6619946a45458e6ca507769f6e3b30f3e0edf0e9ca8c1631319b4238cb4a00d",
}

var sprigPreparedSourceSetSHA256 = hostPin(sprigPreparedSourceSetSHA256ByHost)

var sprigRewrites = []sourceRewrite{
	{
		path: sprigNetworkPath, sourceSHA256: sprigNetworkSourceSHA256, replacementSHA256: sprigNetworkReplacementSHA256,
		rewrites: []anchorRewrite{
			{
				anchor: []byte(`import (
	"math/rand"
	"net"
)

`),
				replacement: []byte(``),
			},
			{
				anchor: []byte(`func getHostByName(name string) string {
	addrs, _ := net.LookupHost(name)
	//TODO: add error handing when release v3 comes out
	return addrs[rand.Intn(len(addrs))]
}`),
				replacement: []byte(`func getHostByName(name string) string {
	//TODO: add error handing when release v3 comes out
	panic("gomad: Sprig host lookup is unsupported")
}`),
			},
		},
	},
}

var sprigAdapter = rewrittenModule{
	module: sprigModulePath, version: sprigVersion, sum: sprigSum,
	cacheElements:                 []string{"github.com", "!masterminds", "sprig", "v3@" + sprigVersion},
	replacementDirectory:          "sprig",
	originalInventorySHA256:       sprigOriginalSourceInventorySHA256,
	replacementInventorySHA256:    sprigReplacementSourceInventorySHA256,
	preparedPackage:               sprigModulePath,
	preparedSourceSetSHA256ByHost: sprigPreparedSourceSetSHA256ByHost,
	rewrites:                      sprigRewrites,
}

func prepareSprig(moduleCache, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
	return prepareRewrittenModule(moduleCache, root, identity, sprigAdapter)
}
