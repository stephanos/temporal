package deterministicio

import gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"

const (
	sprigModulePath                       = "github.com/Masterminds/sprig/v3"
	sprigVersion                          = "v3.3.0"
	sprigSum                              = "h1:mQh0Yrg1XPo6vjYXgtf5OtijNAKJRNcTdOOGZe3tPhs="
	sprigOriginalSourceInventorySHA256    = "sha256:f02251fca63e5c1c142a1551f6705ba59cdd5e5a226f405a8a6d5e01f36db2ec"
	sprigReplacementSourceInventorySHA256 = "sha256:9d81749741355e3043004fe415f906556aaf4d6da4867676e8c2d7042607637d"
	sprigNetworkPath                      = "network.go"
	sprigNetworkSourceSHA256              = "sha256:ffc5b88a9aa83cb4ac331cadd1792c2d493ad65629eec462241104c0afc225ca"
	sprigNetworkReplacementSHA256         = "sha256:7783a960bbefa4e8bb28849af27de06c20e045bfb56c37dd9711f33030a4fc9e"
)

var sprigPreparedSourceSetSHA256ByHost = map[string]string{
	"darwin/arm64": "sha256:8f08780f8a874f591ad9c945e49cc1662008b33114fb0e80f3167dd9283165fd",
	"linux/amd64":  "sha256:8f08780f8a874f591ad9c945e49cc1662008b33114fb0e80f3167dd9283165fd",
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
