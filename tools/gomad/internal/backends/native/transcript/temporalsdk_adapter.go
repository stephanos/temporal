package transcript

import (
	gomadversion "go.temporal.io/server/tools/gomad/internal/backends/native/toolchain/version"
)

const (
	temporalSDKModulePath                       = "go.temporal.io/sdk"
	temporalSDKVersion                          = "v1.48.0"
	temporalSDKSum                              = "h1:WDctKDVuh0Z8Nf7euAyqs/EwcPg1JTIIq1Fut8Tq118="
	temporalSDKOriginalSourceInventorySHA256    = "sha256:6926d4cf41adf86a33470ada5055f2a4c569ea9a6b115ba6eb757841cc81ac65"
	temporalSDKReplacementSourceInventorySHA256 = "sha256:121ceca590af1b635205ba2674baa072e545dbe3e909bbc4b88dac6fd9ab9eb1"
	temporalSDKUtilsPath                        = "internal/internal_utils.go"
	temporalSDKUtilsSourceSHA256                = "sha256:7e7549f7b790779d45f1c8042ceb42753cf87d0cec97f37c5391d4229b5e1eb7"
	temporalSDKUtilsReplacementSHA256           = "sha256:32a474075bd195aebbc147d64d573a198a149562100dcc7d2c583c5193e3034d"
)

var temporalSDKPreparedSourceSetSHA256ByHost = map[string]string{
	"darwin/arm64": "sha256:0071fcadbc4f15cdc22a845abc404ecfe5d1b54dde74ecbc6160f00ce5421cdd",
	"linux/amd64":  "sha256:0071fcadbc4f15cdc22a845abc404ecfe5d1b54dde74ecbc6160f00ce5421cdd",
}

var temporalSDKPreparedSourceSetSHA256 = hostPin(temporalSDKPreparedSourceSetSHA256ByHost)

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

var temporalSDKAdapter = rewrittenModule{
	module: temporalSDKModulePath, version: temporalSDKVersion, sum: temporalSDKSum,
	cacheElements:                 []string{"go.temporal.io", "sdk@" + temporalSDKVersion},
	replacementDirectory:          "temporal-sdk",
	originalInventorySHA256:       temporalSDKOriginalSourceInventorySHA256,
	replacementInventorySHA256:    temporalSDKReplacementSourceInventorySHA256,
	preparedPackage:               temporalSDKModulePath + "/internal",
	preparedSourceSetSHA256ByHost: temporalSDKPreparedSourceSetSHA256ByHost,
	rewrites:                      temporalSDKRewrites,
}

func prepareTemporalSDK(moduleCache, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
	return prepareRewrittenModule(moduleCache, root, identity, temporalSDKAdapter)
}
