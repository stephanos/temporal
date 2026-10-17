package transcript

import (
	gomadversion "go.temporal.io/server/tools/gomad/internal/backends/native/toolchain/version"
)

const (
	temporalSDKModulePath                       = "go.temporal.io/sdk"
	temporalSDKVersion                          = "v1.49.0"
	temporalSDKSum                              = "h1:CtGI0BUe/SCo3eoqTwuWWtXKueii9GBVus7KrKKH1Vo="
	temporalSDKOriginalSourceInventorySHA256    = "sha256:fc76ccf2b03a37cd42533b3af03fbe51ee127d0053c950ae43455ea8633c7644"
	temporalSDKReplacementSourceInventorySHA256 = "sha256:12eefbbe4a827127b9821bcfdc0963e158af410bb472d91e87457425e05e21c4"
	temporalSDKUtilsPath                        = "internal/internal_utils.go"
	temporalSDKUtilsSourceSHA256                = "sha256:502b4a346c0ddb6b6bf14c1fcf164f094597535f80f45e214c9399b711648d20"
	temporalSDKUtilsReplacementSHA256           = "sha256:2a62b1b424053fdf72e85c66d01ae0f86ad133d7c64faa0c8777d537f05516c6"
)

var temporalSDKPreparedSourceSetSHA256ByHost = map[string]string{
	"darwin/arm64": "sha256:4d8a18511e183959a759c12b6bf0c21951c7d47a23a1cc714d2fa92fe5344510",
	"linux/amd64":  "sha256:852c90534cc599fcc374bdadf372a6d35641f4fd7728c5be5793f686e550a452",
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
				anchor: []byte("func InterruptCh() <-chan any {\n\tc := make(chan os.Signal, 1)\n\tsignal.Notify(c, os.Interrupt, syscall.SIGTERM)\n\n" +
					"\tret := make(chan any, 1)\n\tgo func() {\n\t\ts := <-c\n\t\tret <- s\n\t\tclose(ret)\n\t}()\n\n\treturn ret\n}\n"),
				replacement: []byte("func InterruptCh() <-chan any {\n\treturn make(chan any)\n}\n"),
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
