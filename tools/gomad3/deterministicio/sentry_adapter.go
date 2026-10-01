package deterministicio

import gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"

const (
	sentryModulePath                       = "github.com/getsentry/sentry-go"
	sentryVersion                          = "v0.46.0"
	sentrySum                              = "h1:mbdDaarbUdOt9X+dx6kDdntkShLEX3/+KyOsVDTPDj0="
	sentryOriginalSourceInventorySHA256    = "sha256:d8028d24e1a6c27fc5619d349fa58685d2855629a76877cb639a23afbbf67cde"
	sentryReplacementSourceInventorySHA256 = "sha256:b60d63bd862365df7195c5dcf6ae7fa70a97916f3538809773fe42241ab61c32"
	sentryUtilPath                         = "util.go"
	sentryUtilSourceSHA256                 = "sha256:5aee08ef9700b0ebf958c586dd5b71a5db51169d6c3d1c17b6e9b6b2a5644b91"
	sentryUtilReplacementSHA256            = "sha256:21639baea4b81631670f430529d9461d91f79475a3f72844568b02f207d6cabe"
)

var sentryPreparedSourceSetSHA256 = hostPin(map[string]string{
	"darwin/arm64": "sha256:fa2f95c22392afb742711ed26ce02fb79d80005c1af486f4ec71e5e4675c45cb",
	"linux/amd64":  "sha256:fa2f95c22392afb742711ed26ce02fb79d80005c1af486f4ec71e5e4675c45cb",
})

// sentryRewrites leave optional release metadata unknown when discovering it
// would require Git. Explicit, environment, and build-info releases survive.
var sentryRewrites = []sourceRewrite{
	{
		path: sentryUtilPath, sourceSHA256: sentryUtilSourceSHA256, replacementSHA256: sentryUtilReplacementSHA256,
		rewrites: []anchorRewrite{
			{anchor: []byte("\texec \"golang.org/x/sys/execabs\"\n"), replacement: nil},
			{
				anchor: []byte(`	if _, err := exec.LookPath("git"); err == nil {
		cmd := exec.Command("git", "describe", "--long", "--always", "--dirty")
		b, err := cmd.Output()
		if err != nil {
			// Either Git is not available or the current directory is not a
			// Git repository.
			var s strings.Builder
			fmt.Fprintf(&s, "Release detection failed: %v", err)
			if err, ok := err.(*exec.ExitError); ok && len(err.Stderr) > 0 {
				fmt.Fprintf(&s, ": %s", err.Stderr)
			}
			debuglog.Print(s.String())
		} else {
			release = strings.TrimSpace(string(b))
			debuglog.Printf("Using release from Git: %s", release)
			return release
		}
	}
`),
				replacement: []byte(`	// Either Git is not available or the current directory is not a
	// Git repository.
	debuglog.Print("gomad: Git release discovery is unavailable; set ClientOptions.Release or SENTRY_RELEASE")
`),
			},
		},
	},
}

func prepareSentry(moduleCache, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
	return prepareRewrittenModule(moduleCache, root, identity, rewrittenModule{
		module: sentryModulePath, version: sentryVersion, sum: sentrySum,
		cacheElements:              []string{"github.com", "getsentry", "sentry-go@" + sentryVersion},
		replacementDirectory:       "sentry-go",
		originalInventorySHA256:    sentryOriginalSourceInventorySHA256,
		replacementInventorySHA256: sentryReplacementSourceInventorySHA256,
		preparedPackage:            sentryModulePath,
		preparedSourceSetSHA256:    sentryPreparedSourceSetSHA256,
		rewrites:                   sentryRewrites,
	})
}
