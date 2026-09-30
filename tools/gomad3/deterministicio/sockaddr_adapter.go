package deterministicio

import (
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

const (
	sockaddrModulePath                       = "github.com/hashicorp/go-sockaddr"
	sockaddrVersion                          = "v1.0.7"
	sockaddrSum                              = "h1:G+pTkSO01HpR5qCxg7lxfsFEZaG+C0VssTy/9dbT+Fw="
	sockaddrOriginalSourceInventorySHA256    = "sha256:ffbeb087a82c83f3e04f3c7f85317e40096a21854b6ce86ce3edc9c5809357c1"
	sockaddrReplacementSourceInventorySHA256 = "sha256:8b26b980151c28eb5fbfdc1e2cde5ef7cb213d36e37271cccd0fa2304ac6087b"
	sockaddrRouteBSDPath                     = "route_info_bsd.go"
	sockaddrRouteLinuxPath                   = "route_info_linux.go"
	sockaddrRouteBSDSourceSHA256             = "sha256:729003d9a6d1bf3ee2b59bc181f5d5737db07688cb53e8ba854a62bb64af8903"
	sockaddrRouteBSDReplacementSHA256        = "sha256:05d693058b991d8ad6c33bcfcfdde27e7bbe8cfe2aa744833aee1d3582179284"
	sockaddrRouteLinuxSourceSHA256           = "sha256:84afc4064c753bebbbbd406a7011e7af4353bbd4184e42bacbd2719c45c6ecae"
	sockaddrRouteLinuxReplacementSHA256      = "sha256:3f93d047bf59e363e5fdb68bc030b457dc21d4d05336285273b963794dbada94"
)

// sockaddrPreparedSourceSetSHA256 pins darwin/arm64 only: the module is reached
// through a downstream module's gossip membership, and no linux/amd64 target
// has needed it yet, so a linux build fails closed.
var sockaddrPreparedSourceSetSHA256 = hostPin(map[string]string{
	"darwin/arm64": "sha256:fc302a373437ca8cef4fe978e6fe963222b0cb6393728d71d43603f119741954",
})

const sockaddrRouteRefusal = "\n// routeCommandOutput refuses: the deterministic build has no host routing\n" +
	"// table to read, so the default interface is unknown rather than a subprocess.\n" +
	"func routeCommandOutput([]string) ([]byte, error) {\n" +
	"\treturn nil, errors.New(\"default route lookup is unavailable without host access\")\n}\n"

// sockaddrRewrites keep the address library's default-route lookup away from
// the host: it ran /sbin/route or ip through os/exec, and it now refuses with an
// error, which the library already reports when no default route exists.
// Interface enumeration and address parsing are unchanged.
var sockaddrRewrites = []sourceRewrite{
	{
		path:              sockaddrRouteBSDPath,
		sourceSHA256:      sockaddrRouteBSDSourceSHA256,
		replacementSHA256: sockaddrRouteBSDReplacementSHA256,
		rewrites: []anchorRewrite{
			{anchor: []byte("import \"os/exec\"\n"), replacement: []byte("import \"errors\"\n")},
			{
				anchor:      []byte("\tout, err := exec.Command(cmds[\"route\"][0], cmds[\"route\"][1:]...).Output()\n"),
				replacement: []byte("\tout, err := routeCommandOutput(cmds[\"route\"])\n"),
			},
			{anchor: []byte("\treturn ifName, nil\n}\n"), replacement: []byte("\treturn ifName, nil\n}\n" + sockaddrRouteRefusal)},
		},
	},
	{
		path:              sockaddrRouteLinuxPath,
		sourceSHA256:      sockaddrRouteLinuxSourceSHA256,
		replacementSHA256: sockaddrRouteLinuxReplacementSHA256,
		rewrites: []anchorRewrite{
			{anchor: []byte("import (\n\t\"errors\"\n\t\"os/exec\"\n)\n"), replacement: []byte("import (\n\t\"errors\"\n)\n")},
			{
				anchor:      []byte("\tpath, _ := exec.LookPath(\"ip\")\n\tif path == \"\" {\n\t\tpath = \"/sbin/ip\"\n\t}\n"),
				replacement: []byte("\tpath := \"/sbin/ip\"\n"),
			},
			{
				anchor:      []byte("\tout, err := exec.Command(ri.cmds[\"ip\"][0], ri.cmds[\"ip\"][1:]...).Output()\n"),
				replacement: []byte("\tout, err := routeCommandOutput(ri.cmds[\"ip\"])\n"),
			},
			{anchor: []byte("\treturn ifName, nil\n}\n"), replacement: []byte("\treturn ifName, nil\n}\n" + sockaddrRouteRefusal)},
		},
	},
}

func prepareSockaddr(moduleCache, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
	return prepareRewrittenModule(moduleCache, root, identity, rewrittenModule{
		module: sockaddrModulePath, version: sockaddrVersion, sum: sockaddrSum,
		cacheElements:              []string{"github.com", "hashicorp", "go-sockaddr@" + sockaddrVersion},
		replacementDirectory:       "go-sockaddr",
		originalInventorySHA256:    sockaddrOriginalSourceInventorySHA256,
		replacementInventorySHA256: sockaddrReplacementSourceInventorySHA256,
		preparedPackage:            sockaddrModulePath,
		preparedSourceSetSHA256:    sockaddrPreparedSourceSetSHA256,
		rewrites:                   sockaddrRewrites,
	})
}
