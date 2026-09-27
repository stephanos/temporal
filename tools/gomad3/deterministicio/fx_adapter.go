package deterministicio

import (
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

const (
	fxModulePath                       = "go.uber.org/fx"
	fxVersion                          = "v1.24.0"
	fxSum                              = "h1:wE8mruvpg2kiiL1Vqd0CC+tr0/24XIB10Iwp2lLWzkg="
	fxOriginalSourceInventorySHA256    = "sha256:4c4dea77b4c4aecf00624b18ad647b15ff29d07369124691016e4dc0db5a7249"
	fxReplacementSourceInventorySHA256 = "sha256:06047d99904f2cd2852cc0fe130c14fd8d9e94343b8fd95cfbe23f3d000e6226"
	fxSignalPath                       = "signal.go"
	fxSignalNamesPath                  = "app_unixes.go"
	fxSignalSourceSHA256               = "sha256:434face276b5ffa00dfb5076245bbefa807dcc7c7b642fdf7dadf09d10d0a7f2"
	fxSignalReplacementSHA256          = "sha256:5b83ea57866379bb9ba7b40473e2fff28f29ee19836cfb18bf5a85c2478b79c6"
	fxSignalNamesSourceSHA256          = "sha256:89db3c6b738470b9447d58c7aa96bf504c75b259bb27bac36adc6e0f67c79096"
	fxSignalNamesReplacementSHA256     = "sha256:9437126ec05010e763ba427171b9aeb835fc3ad15667865523c23461cd579a45"
)

var fxPreparedSourceSetSHA256 = hostPin(map[string]string{
	"darwin/arm64": "sha256:d8b6580641c5ead1685c31e2e3c2d10ac899513dee52a808f4762e34d90313c1",
	"linux/amd64":  "sha256:d8b6580641c5ead1685c31e2e3c2d10ac899513dee52a808f4762e34d90313c1",
})

// fxRewrites detaches the fx shutdowner from the host: the signal relay keeps
// its channel plumbing but never registers with os/signal, and the shutdown
// signal names come from a local type instead of x/sys/unix so a
// Shutdowner-triggered ShutdownSignal still prints "terminated".
var fxRewrites = []sourceRewrite{
	{
		path:              fxSignalPath,
		sourceSHA256:      fxSignalSourceSHA256,
		replacementSHA256: fxSignalReplacementSHA256,
		rewrites: []anchorRewrite{
			{anchor: []byte("\t\"os\"\n\t\"os/signal\"\n\t\"sync\"\n"), replacement: []byte("\t\"os\"\n\t\"sync\"\n")},
			{
				anchor:      []byte("\t\tnotify:     signal.Notify,\n\t\tstopNotify: signal.Stop,\n"),
				replacement: []byte("\t\tnotify:     func(chan<- os.Signal, ...os.Signal) {},\n\t\tstopNotify: func(chan<- os.Signal) {},\n"),
			},
		},
	},
	{
		path:              fxSignalNamesPath,
		sourceSHA256:      fxSignalNamesSourceSHA256,
		replacementSHA256: fxSignalNamesReplacementSHA256,
		rewrites: []anchorRewrite{
			{
				anchor: []byte("import \"golang.org/x/sys/unix\"\n\nconst (\n\t_sigINT  = unix.SIGINT\n\t_sigTERM = unix.SIGTERM\n)\n"),
				replacement: []byte("// hostSignal names the shutdown signals without consulting the host: the\n" +
					"// deterministic build never installs signal handlers, so only the names\n" +
					"// survive into ShutdownSignal.\n" +
					"type hostSignal string\n\n" +
					"func (sig hostSignal) String() string { return string(sig) }\n\n" +
					"func (hostSignal) Signal() {}\n\n" +
					"const (\n\t_sigINT  hostSignal = \"interrupt\"\n\t_sigTERM hostSignal = \"terminated\"\n)\n"),
			},
		},
	},
}

func prepareFx(moduleCache, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
	return prepareRewrittenModule(moduleCache, root, identity, rewrittenModule{
		module: fxModulePath, version: fxVersion, sum: fxSum,
		cacheElements:              []string{"go.uber.org", "fx@" + fxVersion},
		replacementDirectory:       "uber-fx",
		originalInventorySHA256:    fxOriginalSourceInventorySHA256,
		replacementInventorySHA256: fxReplacementSourceInventorySHA256,
		preparedPackage:            fxModulePath,
		preparedSourceSetSHA256:    fxPreparedSourceSetSHA256,
		rewrites:                   fxRewrites,
	})
}
