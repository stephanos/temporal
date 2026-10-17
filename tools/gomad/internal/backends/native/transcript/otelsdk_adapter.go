package transcript

import (
	gomadversion "go.temporal.io/server/tools/gomad/internal/backends/native/toolchain/version"
)

const (
	otelSDKModulePath                       = "go.opentelemetry.io/otel/sdk"
	otelSDKVersion                          = "v1.45.0"
	otelSDKSum                              = "h1:4VVSMgQ83dUgW2aoX5f6JgLvHwIvzcuLnF9lUdCSpCw="
	otelSDKOriginalSourceInventorySHA256    = "sha256:eb3224a421187a27fbc7270eacab0c1370ab2422f9bcd1b0c5c2fb42f43fc278"
	otelSDKReplacementSourceInventorySHA256 = "sha256:ad8b2c0454c56565e87a834d724dce6e0fd7547ffc7bd332e1c97eb8d4109693"
	otelSDKProcessPath                      = "resource/process.go"
	otelSDKOSUnixPath                       = "resource/os_unix.go"
	otelSDKHostIDExecPath                   = "resource/host_id_exec.go"
	otelSDKProcessSourceSHA256              = "sha256:cc58934674d5e12424ae43a3c63c2b59812c7b141af48e38561e6fd580d8f7b4"
	otelSDKProcessReplacementSHA256         = "sha256:39518e6b36396093da73a0ab96cb6174a71a7193eafced9243eacb035a60f779"
	otelSDKOSUnixSourceSHA256               = "sha256:56d383a5b108e0037049989dbb00c358c7ca8239fb9f9d5fad672eb3ef697100"
	otelSDKOSUnixReplacementSHA256          = "sha256:77a54c07215e60fedfdb3b26a6ff34196a93710f7ecf119485b86ab90d7b353b"
	otelSDKHostIDExecSourceSHA256           = "sha256:cc31e43da26ad90ad8a869baa7f5617137963fc3ea3ef179b88a34f69a102d33"
	otelSDKHostIDExecReplacementSHA256      = "sha256:73a4d92b3433f6ab793b9002b2d6ee5cf9fadb1b54d66168e5edf94cfc0e67fe"
)

var otelSDKPreparedSourceSetSHA256ByHost = map[string]string{
	"darwin/arm64": "sha256:d8bd182d665e7fe7fe297592c8d3600758496a8e61a4c421343662d0f2f9192c",
	"linux/amd64":  "sha256:10880f920b4c28d03806b53f0655e0116ef26e80a114455d139efb20d2c11866",
}

var otelSDKPreparedSourceSetSHA256 = hostPin(otelSDKPreparedSourceSetSHA256ByHost)

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

var otelSDKAdapter = rewrittenModule{
	module: otelSDKModulePath, version: otelSDKVersion, sum: otelSDKSum,
	cacheElements:                 []string{"go.opentelemetry.io", "otel", "sdk@" + otelSDKVersion},
	replacementDirectory:          "otel-sdk",
	originalInventorySHA256:       otelSDKOriginalSourceInventorySHA256,
	replacementInventorySHA256:    otelSDKReplacementSourceInventorySHA256,
	preparedPackage:               otelSDKModulePath + "/resource",
	preparedSourceSetSHA256ByHost: otelSDKPreparedSourceSetSHA256ByHost,
	rewrites:                      otelSDKRewrites,
}

func prepareOtelSDK(moduleCache, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
	return prepareRewrittenModule(moduleCache, root, identity, otelSDKAdapter)
}
