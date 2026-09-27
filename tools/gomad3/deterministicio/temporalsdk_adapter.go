package deterministicio

import (
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

const (
	temporalSDKModulePath                       = "go.temporal.io/sdk"
	temporalSDKVersion                          = "v1.48.0"
	temporalSDKSum                              = "h1:WDctKDVuh0Z8Nf7euAyqs/EwcPg1JTIIq1Fut8Tq118="
	temporalSDKOriginalSourceInventorySHA256    = "sha256:01c242d55449ccbc7bbbccbaf45e857a9c8cf44c9335b19901bc56d4400285fe"
	temporalSDKReplacementSourceInventorySHA256 = "sha256:71d51e848794ac4b8981994fcc9d9b04f21ca20f1ca27ec6c20c35c4b8357aa6"
	temporalSDKUtilsPath                        = "internal/internal_utils.go"
	temporalSDKUtilsSourceSHA256                = "sha256:7e7549f7b790779d45f1c8042ceb42753cf87d0cec97f37c5391d4229b5e1eb7"
	temporalSDKUtilsReplacementSHA256           = "sha256:32a474075bd195aebbc147d64d573a198a149562100dcc7d2c583c5193e3034d"
)

var temporalSDKPreparedSourceSetSHA256 = hostPin(map[string]string{
	"darwin/arm64": "sha256:45cd84114a3b540d08926dbfefc61619f1a1762a2c10572054ae585f356c57b6",
	"linux/amd64":  "sha256:45cd84114a3b540d08926dbfefc61619f1a1762a2c10572054ae585f356c57b6",
})

// temporalSDKRewrites removes the SDK's interrupt channel from the host: the
// deterministic build never delivers SIGINT or SIGTERM, so InterruptCh returns
// a channel that never fires and the package stops importing os/signal and
// syscall.
var temporalSDKRewrites = []sourceRewrite{
	{
		path:              temporalSDKUtilsPath,
		sourceSHA256:      temporalSDKUtilsSourceSHA256,
		replacementSHA256: temporalSDKUtilsReplacementSHA256,
		rewrites: []anchorRewrite{
			{
				anchor:      []byte("\t\"os\"\n\t\"os/signal\"\n\t\"strings\"\n\t\"sync\"\n\t\"syscall\"\n\t\"time\"\n"),
				replacement: []byte("\t\"os\"\n\t\"strings\"\n\t\"sync\"\n\t\"time\"\n"),
			},
			{
				anchor: []byte("func InterruptCh() <-chan interface{} {\n\tc := make(chan os.Signal, 1)\n\tsignal.Notify(c, os.Interrupt, syscall.SIGTERM)\n\n" +
					"\tret := make(chan interface{}, 1)\n\tgo func() {\n\t\ts := <-c\n\t\tret <- s\n\t\tclose(ret)\n\t}()\n\n\treturn ret\n}\n"),
				replacement: []byte("func InterruptCh() <-chan interface{} {\n\treturn make(chan interface{})\n}\n"),
			},
		},
	},
}

func prepareTemporalSDK(moduleCache, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
	return prepareRewrittenModule(moduleCache, root, identity, rewrittenModule{
		module: temporalSDKModulePath, version: temporalSDKVersion, sum: temporalSDKSum,
		cacheElements:              []string{"go.temporal.io", "sdk@" + temporalSDKVersion},
		replacementDirectory:       "temporal-sdk",
		originalInventorySHA256:    temporalSDKOriginalSourceInventorySHA256,
		replacementInventorySHA256: temporalSDKReplacementSourceInventorySHA256,
		preparedPackage:            temporalSDKModulePath + "/internal",
		preparedSourceSetSHA256:    temporalSDKPreparedSourceSetSHA256,
		rewrites:                   temporalSDKRewrites,
	})
}
