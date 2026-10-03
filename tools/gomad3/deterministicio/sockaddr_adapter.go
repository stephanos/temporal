package deterministicio

import (
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

const (
	sockaddrModulePath                       = "github.com/hashicorp/go-sockaddr"
	sockaddrVersion                          = "v1.0.7"
	sockaddrSum                              = "h1:G+pTkSO01HpR5qCxg7lxfsFEZaG+C0VssTy/9dbT+Fw="
	sockaddrOriginalSourceInventorySHA256    = "sha256:ffbeb087a82c83f3e04f3c7f85317e40096a21854b6ce86ce3edc9c5809357c1"
	sockaddrReplacementSourceInventorySHA256 = "sha256:470bd36489f5ac4873436f4a200d92ef28d7b36b2eb1c8bf55ee557105cec95e"
	sockaddrRouteBSDPath                     = "route_info_bsd.go"
	sockaddrRouteLinuxPath                   = "route_info_linux.go"
	sockaddrRouteBSDSourceSHA256             = "sha256:729003d9a6d1bf3ee2b59bc181f5d5737db07688cb53e8ba854a62bb64af8903"
	sockaddrRouteBSDReplacementSHA256        = "sha256:05d693058b991d8ad6c33bcfcfdde27e7bbe8cfe2aa744833aee1d3582179284"
	sockaddrRouteLinuxSourceSHA256           = "sha256:84afc4064c753bebbbbd406a7011e7af4353bbd4184e42bacbd2719c45c6ecae"
	sockaddrRouteLinuxReplacementSHA256      = "sha256:3f93d047bf59e363e5fdb68bc030b457dc21d4d05336285273b963794dbada94"
	sockaddrIPv4Path                         = "ipv4addr.go"
	sockaddrIPv4SourceSHA256                 = "sha256:9655d8a01b9bc8e7a46f38335a144e2f3e57d2757d72fcb2cc00514ac2d65a48"
	sockaddrIPv4ReplacementSHA256            = "sha256:0592b790ad0e9697af6c0d949ec44c5b5eb64addf90955f2873a255fece96776"
	sockaddrIPv6Path                         = "ipv6addr.go"
	sockaddrIPv6SourceSHA256                 = "sha256:ff59a903ea308a82b9c6fe62a0a47ffa952d130e3268641b9a22dcf25f231fae"
	sockaddrIPv6ReplacementSHA256            = "sha256:ba1cdb8434d5e9e6adb708cdf081dfcca74086d33d637298d2a7d61c07ca59c0"
	sockaddrIfAddrsPath                      = "ifaddrs.go"
	sockaddrIfAddrsSourceSHA256              = "sha256:69d92dce7cb9059ed8cba28b8ed596111b44105b7447125f8f1acf944b3e4ca2"
	sockaddrIfAddrsReplacementSHA256         = "sha256:a6bdf5ccd9a393a48b34b9db65125aaa641dd0dba6172c40fffc00f46c21634a"
	sockaddrIfAttrPath                       = "ifattr.go"
	sockaddrIfAttrSourceSHA256               = "sha256:926fa616231e2c04092a313a74d3782ac72ebc3127d2c1ae9c96eb63f336c6e2"
	sockaddrIfAttrReplacementSHA256          = "sha256:082fc52d4eb17b056399d0d36def4a9b9071ac6fe07dac4a6f9e2630a6e0b769"
)

// Historical restriction:
// sockaddrPreparedSourceSetSHA256 pins darwin/arm64 only: the module is reached
// through a downstream module's gossip membership, and no linux/amd64 target
// has needed it yet, so a linux build fails closed.
// Downstream workflow preparation now pins both qualified platforms.
var sockaddrPreparedSourceSetSHA256ByHost = map[string]string{
	"darwin/arm64": "sha256:05bad9b7f5550962a542a4b8c7101d135b1d0fdf199f2c01ce4cc9f33ab96f45",
	"linux/amd64":  "sha256:08ac1f34ac338d5d5090f13f6a9dbed65cadb17c473cf6ed1f9b6b4813990b43",
}

var sockaddrPreparedSourceSetSHA256 = hostPin(sockaddrPreparedSourceSetSHA256ByHost)

const sockaddrRouteRefusal = "\n// routeCommandOutput refuses: the deterministic build has no host routing\n" +
	"// table to read, so the default interface is unknown rather than a subprocess.\n" +
	"func routeCommandOutput([]string) ([]byte, error) {\n" +
	"\treturn nil, errors.New(\"default route lookup is unavailable without host access\")\n}\n"

// sockaddrRewrites keep the address library's default-route lookup away from
// the host: it ran /sbin/route or ip through os/exec, and it now refuses with an
// error, which the library already reports when no default route exists.
// Historical restriction before resolver and interface preparation:
// Interface enumeration and address parsing are unchanged.
// The prepared library now parses literal addresses without resolution and
// refuses interface discovery; supplied interface metadata remains usable.
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
	{
		path: sockaddrIPv4Path, sourceSHA256: sockaddrIPv4SourceSHA256, replacementSHA256: sockaddrIPv4ReplacementSHA256,
		rewrites: []anchorRewrite{
			{anchor: []byte(`	tcpAddr, err := net.ResolveTCPAddr("tcp4", ipv4Str)
`), replacement: []byte(`	tcpAddr, err := parseLiteralTCPAddr("tcp4", ipv4Str)
`)},
			{anchor: []byte(`// AddressBinString returns a string with the IPv4Addr's Address represented
`), replacement: []byte(`func parseLiteralTCPAddr(network, address string) (*net.TCPAddr, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("gomad: sockaddr address resolution requires a literal IP and numeric port: %w", err)
	}
	zone := ""
	if index := strings.LastIndexByte(host, '%'); index >= 0 {
		zone = host[index+1:]
		host = host[:index]
	}
	ip := net.ParseIP(host)
	if ip == nil || network == "tcp4" && (ip.To4() == nil || zone != "") || network == "tcp6" && ip.To4() != nil {
		return nil, fmt.Errorf("gomad: sockaddr address resolution requires a literal IP and numeric port")
	}
	if port == "" {
		port = "0"
	}
	number, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("gomad: sockaddr address resolution requires a literal IP and numeric port: %w", err)
	}
	return &net.TCPAddr{IP: ip, Port: int(number), Zone: zone}, nil
}

// AddressBinString returns a string with the IPv4Addr's Address represented
`)},
		},
	},
	{
		path: sockaddrIPv6Path, sourceSHA256: sockaddrIPv6SourceSHA256, replacementSHA256: sockaddrIPv6ReplacementSHA256,
		rewrites: []anchorRewrite{
			{anchor: []byte(`	tcpAddr, err := net.ResolveTCPAddr("tcp6", ipv6Str)
`), replacement: []byte(`	tcpAddr, err := parseLiteralTCPAddr("tcp6", ipv6Str)
	literalErr := err
`)},
			{anchor: []byte(`return IPv6Addr{}, fmt.Errorf("Unable to parse %+q to an IPv6 address: %v", ipv6Str, err)`), replacement: []byte(`return IPv6Addr{}, fmt.Errorf("Unable to parse %+q to an IPv6 address: %v", ipv6Str, literalErr)`)},
		},
	},
	{
		path: sockaddrIfAddrsPath, sourceSHA256: sockaddrIfAddrsSourceSHA256, replacementSHA256: sockaddrIfAddrsReplacementSHA256,
		rewrites: []anchorRewrite{
			{anchor: []byte(`func GetAllInterfaces() (IfAddrs, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	ifAddrs := make(IfAddrs, 0, len(ifs))
	for _, intf := range ifs {
		addrs, err := intf.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			var ipAddr IPAddr
			ipAddr, err = NewIPAddr(addr.String())
			if err != nil {
				return IfAddrs{}, fmt.Errorf("unable to create an IP address from %q", addr.String())
			}

			ifAddr := IfAddr{
				SockAddr:  ipAddr,
				Interface: intf,
			}
			ifAddrs = append(ifAddrs, ifAddr)
		}
	}

	return ifAddrs, nil
}`), replacement: []byte(`func GetAllInterfaces() (IfAddrs, error) {
	return nil, errors.New("gomad: sockaddr interface discovery is unsupported")
}`)},
		},
	},
	{
		path: sockaddrIfAttrPath, sourceSHA256: sockaddrIfAttrSourceSHA256, replacementSHA256: sockaddrIfAttrReplacementSHA256,
		rewrites: []anchorRewrite{
			{anchor: []byte(`type IfAddr struct {
	SockAddr
	net.Interface
}
`), replacement: []byte(`type IfAddr struct {
	SockAddr
	net.Interface
}

func (ifAddr IfAddr) Addrs() ([]net.Addr, error) {
	return nil, fmt.Errorf("gomad: sockaddr interface discovery is unsupported")
}

func (ifAddr IfAddr) MulticastAddrs() ([]net.Addr, error) {
	return nil, fmt.Errorf("gomad: sockaddr interface discovery is unsupported")
}
`)},
		},
	},
}

var sockaddrAdapter = rewrittenModule{
	module: sockaddrModulePath, version: sockaddrVersion, sum: sockaddrSum,
	cacheElements:                 []string{"github.com", "hashicorp", "go-sockaddr@" + sockaddrVersion},
	replacementDirectory:          "go-sockaddr",
	originalInventorySHA256:       sockaddrOriginalSourceInventorySHA256,
	replacementInventorySHA256:    sockaddrReplacementSourceInventorySHA256,
	preparedPackage:               sockaddrModulePath,
	preparedSourceSetSHA256ByHost: sockaddrPreparedSourceSetSHA256ByHost,
	rewrites:                      sockaddrRewrites,
}

func prepareSockaddr(moduleCache, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
	return prepareRewrittenModule(moduleCache, root, identity, sockaddrAdapter)
}
