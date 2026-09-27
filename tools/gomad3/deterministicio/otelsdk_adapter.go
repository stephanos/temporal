package deterministicio

import (
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

const (
	otelSDKModulePath                       = "go.opentelemetry.io/otel/sdk"
	otelSDKVersion                          = "v1.44.0"
	otelSDKSum                              = "h1:nHYwb9lK+fJPU/dnT6s7W7Z8itMWyqrnVfbheVYrZ58="
	otelSDKOriginalSourceInventorySHA256    = "sha256:ce46e14e165abf759ffb3fec32b9f955954819522bd31ae3cf423c465f0407da"
	otelSDKReplacementSourceInventorySHA256 = "sha256:a34d3421c87a2f4a6221132b849a56fd85069480df262a880a8a6027eb0610c4"
	otelSDKProcessPath                      = "resource/process.go"
	otelSDKOSUnixPath                       = "resource/os_unix.go"
	otelSDKHostIDExecPath                   = "resource/host_id_exec.go"
	otelSDKProcessSourceSHA256              = "sha256:44e977fb2bcab86d3dec9f74c5ae99613ed6df34462324235237014989978854"
	otelSDKProcessReplacementSHA256         = "sha256:62c9b39dc9f7c86bb7e1e1a788b2bd9e1f772d0f10148dd58e7b125bf7562577"
	otelSDKOSUnixSourceSHA256               = "sha256:c11a96200ddd84d22524ee6bae631f3ce454ab9b636294b38081ba46f65250de"
	otelSDKOSUnixReplacementSHA256          = "sha256:65a327c25dfea41778aeb598610dc47f5a9b80071bcd0478e1e970b95e7da5ef"
	otelSDKHostIDExecSourceSHA256           = "sha256:6086e8a93c1aef9ccc1a1d17fdb25f109692fe7b823290433573821e0041a256"
	otelSDKHostIDExecReplacementSHA256      = "sha256:630bd1c268665d1fcdac0fbf61ddbc1932a53389b1c88a3f18bdd5afc62de8fb"
)

var otelSDKPreparedSourceSetSHA256 = hostPin(map[string]string{
	"darwin/arm64": "sha256:796855abd6e097de9927ea5e57b82d691335b47e2a65dad02987bcf5f8e30afa",
	"linux/amd64":  "sha256:1ca44b7f7b5e9ace498db65a954cd5612a08617abbc0e48e3fe70112503d4bb2",
})

// otelSDKRewrites keeps the OpenTelemetry resource detectors away from the
// host: the process owner and uname detectors report "<unknown>" the way the
// module already does on unsupported platforms, and the BSD host-id reader's
// command runner refuses instead of reaching os/exec.
var otelSDKRewrites = []sourceRewrite{
	{
		path:              otelSDKProcessPath,
		sourceSHA256:      otelSDKProcessSourceSHA256,
		replacementSHA256: otelSDKProcessReplacementSHA256,
		rewrites: []anchorRewrite{
			{
				anchor:      []byte("\t\"context\"\n\t\"fmt\"\n\t\"os\"\n\t\"os/user\"\n\t\"path/filepath\"\n"),
				replacement: []byte("\t\"context\"\n\t\"fmt\"\n\t\"os\"\n\t\"path/filepath\"\n"),
			},
			{
				anchor: []byte("type (\n\tpidProvider            func() int\n"),
				replacement: []byte("// processOwner stands in for user.User: the deterministic build never\n" +
					"// consults the host account database.\n" +
					"type processOwner struct {\n\tUsername string\n}\n\n" +
					"func unknownProcessOwner() (*processOwner, error) {\n\treturn &processOwner{Username: \"<unknown>\"}, nil\n}\n\n" +
					"type (\n\tpidProvider            func() int\n"),
			},
			{
				anchor:      []byte("\townerProvider          func() (*user.User, error)\n"),
				replacement: []byte("\townerProvider          func() (*processOwner, error)\n"),
			},
			{
				anchor:      []byte("\tdefaultOwnerProvider          ownerProvider          = user.Current\n"),
				replacement: []byte("\tdefaultOwnerProvider          ownerProvider          = unknownProcessOwner\n"),
			},
		},
	},
	{
		path:              otelSDKOSUnixPath,
		sourceSHA256:      otelSDKOSUnixSourceSHA256,
		replacementSHA256: otelSDKOSUnixReplacementSHA256,
		rewrites: []anchorRewrite{
			{anchor: []byte("\t\"os\"\n\n\t\"golang.org/x/sys/unix\"\n"), replacement: []byte("\t\"os\"\n")},
			{
				anchor:      []byte("type unameProvider func(buf *unix.Utsname) (err error)\n"),
				replacement: []byte("type unameProvider func() (string, error)\n"),
			},
			{
				anchor:      []byte("var defaultUnameProvider unameProvider = unix.Uname\n"),
				replacement: []byte("var defaultUnameProvider unameProvider = func() (string, error) { return \"<unknown>\", nil }\n"),
			},
			{
				anchor: []byte("func uname() (string, error) {\n\tvar utsName unix.Utsname\n\n\terr := currentUnameProvider(&utsName)\n\tif err != nil {\n\t\treturn \"\", err\n\t}\n\n" +
					"\treturn fmt.Sprintf(\n\t\t\"%s %s %s %s %s\",\n" +
					"\t\tunix.ByteSliceToString(utsName.Sysname[:]),\n" +
					"\t\tunix.ByteSliceToString(utsName.Nodename[:]),\n" +
					"\t\tunix.ByteSliceToString(utsName.Release[:]),\n" +
					"\t\tunix.ByteSliceToString(utsName.Version[:]),\n" +
					"\t\tunix.ByteSliceToString(utsName.Machine[:]),\n\t), nil\n}\n"),
				replacement: []byte("func uname() (string, error) {\n\treturn currentUnameProvider()\n}\n"),
			},
		},
	},
	{
		path:              otelSDKHostIDExecPath,
		sourceSHA256:      otelSDKHostIDExecSourceSHA256,
		replacementSHA256: otelSDKHostIDExecReplacementSHA256,
		rewrites: []anchorRewrite{
			{
				anchor: []byte("import (\n\t\"context\"\n\t\"os/exec\"\n)\n\nfunc execCommand(name string, arg ...string) (string, error) {\n" +
					"\tcmd := exec.CommandContext(context.Background(), name, arg...)\n\tb, err := cmd.Output()\n\tif err != nil {\n\t\treturn \"\", err\n\t}\n\n\treturn string(b), nil\n}\n"),
				replacement: []byte("import \"errors\"\n\nfunc execCommand(string, ...string) (string, error) {\n" +
					"\treturn \"\", errors.New(\"host command execution is unavailable without host access\")\n}\n"),
			},
		},
	},
}

func prepareOtelSDK(moduleCache, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
	return prepareRewrittenModule(moduleCache, root, identity, rewrittenModule{
		module: otelSDKModulePath, version: otelSDKVersion, sum: otelSDKSum,
		cacheElements:              []string{"go.opentelemetry.io", "otel", "sdk@" + otelSDKVersion},
		replacementDirectory:       "otel-sdk",
		originalInventorySHA256:    otelSDKOriginalSourceInventorySHA256,
		replacementInventorySHA256: otelSDKReplacementSourceInventorySHA256,
		preparedPackage:            otelSDKModulePath + "/resource",
		preparedSourceSetSHA256:    otelSDKPreparedSourceSetSHA256,
		rewrites:                   otelSDKRewrites,
	})
}
