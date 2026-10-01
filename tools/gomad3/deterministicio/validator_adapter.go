package deterministicio

import gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"

const (
	validatorModulePath                       = "github.com/go-playground/validator/v10"
	validatorVersion                          = "v10.30.1"
	validatorSum                              = "h1:f3zDSN/zOma+w6+1Wswgd9fLkdwy06ntQJp0BBvFG0w="
	validatorOriginalSourceInventorySHA256    = "sha256:95a2dc71c5712205d7897b73acaf2cf6d92b89fd488cbaa71bd9470921c07b17"
	validatorReplacementSourceInventorySHA256 = "sha256:69ae6078c526dc97d7e3457b0eab49a2fe9331979183fe2455c16aa512be96b1"
	validatorBakedInPath                      = "baked_in.go"
	validatorBakedInSourceSHA256              = "sha256:dd7e010f211a92030f57ed297c4bbc3a7951cedc4c195e828fa59ad8de25787d"
	validatorBakedInReplacementSHA256         = "sha256:1f3bd33bd351fa7822a5b1ddbf6a7e72b9dc486c343f80c51e926083f93363ea"
)

var validatorPreparedSourceSetSHA256 = hostPin(map[string]string{
	"darwin/arm64": "sha256:c0ed49ccbbb194fc15427fea32759d496ba7112aba9dd8d9936883bba6fe6943",
	"linux/amd64":  "sha256:c0ed49ccbbb194fc15427fea32759d496ba7112aba9dd8d9936883bba6fe6943",
})

var validatorRewrites = []sourceRewrite{
	{
		path: validatorBakedInPath, sourceSHA256: validatorBakedInSourceSHA256, replacementSHA256: validatorBakedInReplacementSHA256,
		rewrites: []anchorRewrite{
			{
				anchor: []byte(`func isTCP4AddrResolvable(fl FieldLevel) bool {
	if !isIP4Addr(fl) {
		return false
	}

	_, err := net.ResolveTCPAddr("tcp4", fl.Field().String())
	return err == nil
}`),
				replacement: []byte(`func isTCP4AddrResolvable(fl FieldLevel) bool {
	panic("gomad: Validator address resolution is unsupported")
}`),
			},
			{
				anchor: []byte(`func isTCP6AddrResolvable(fl FieldLevel) bool {
	if !isIP6Addr(fl) {
		return false
	}

	_, err := net.ResolveTCPAddr("tcp6", fl.Field().String())

	return err == nil
}`),
				replacement: []byte(`func isTCP6AddrResolvable(fl FieldLevel) bool {
	panic("gomad: Validator address resolution is unsupported")
}`),
			},
			{
				anchor: []byte(`func isTCPAddrResolvable(fl FieldLevel) bool {
	if !isIP4Addr(fl) && !isIP6Addr(fl) {
		return false
	}

	_, err := net.ResolveTCPAddr("tcp", fl.Field().String())

	return err == nil
}`),
				replacement: []byte(`func isTCPAddrResolvable(fl FieldLevel) bool {
	panic("gomad: Validator address resolution is unsupported")
}`),
			},
			{
				anchor: []byte(`func isUDP4AddrResolvable(fl FieldLevel) bool {
	if !isIP4Addr(fl) {
		return false
	}

	_, err := net.ResolveUDPAddr("udp4", fl.Field().String())

	return err == nil
}`),
				replacement: []byte(`func isUDP4AddrResolvable(fl FieldLevel) bool {
	panic("gomad: Validator address resolution is unsupported")
}`),
			},
			{
				anchor: []byte(`func isUDP6AddrResolvable(fl FieldLevel) bool {
	if !isIP6Addr(fl) {
		return false
	}

	_, err := net.ResolveUDPAddr("udp6", fl.Field().String())

	return err == nil
}`),
				replacement: []byte(`func isUDP6AddrResolvable(fl FieldLevel) bool {
	panic("gomad: Validator address resolution is unsupported")
}`),
			},
			{
				anchor: []byte(`func isUDPAddrResolvable(fl FieldLevel) bool {
	if !isIP4Addr(fl) && !isIP6Addr(fl) {
		return false
	}

	_, err := net.ResolveUDPAddr("udp", fl.Field().String())

	return err == nil
}`),
				replacement: []byte(`func isUDPAddrResolvable(fl FieldLevel) bool {
	panic("gomad: Validator address resolution is unsupported")
}`),
			},
			{
				anchor: []byte(`func isIP4AddrResolvable(fl FieldLevel) bool {
	if !isIPv4(fl) {
		return false
	}

	_, err := net.ResolveIPAddr("ip4", fl.Field().String())

	return err == nil
}`),
				replacement: []byte(`func isIP4AddrResolvable(fl FieldLevel) bool {
	panic("gomad: Validator address resolution is unsupported")
}`),
			},
			{
				anchor: []byte(`func isIP6AddrResolvable(fl FieldLevel) bool {
	if !isIPv6(fl) {
		return false
	}

	_, err := net.ResolveIPAddr("ip6", fl.Field().String())

	return err == nil
}`),
				replacement: []byte(`func isIP6AddrResolvable(fl FieldLevel) bool {
	panic("gomad: Validator address resolution is unsupported")
}`),
			},
			{
				anchor: []byte(`func isIPAddrResolvable(fl FieldLevel) bool {
	if !isIP(fl) {
		return false
	}

	_, err := net.ResolveIPAddr("ip", fl.Field().String())

	return err == nil
}`),
				replacement: []byte(`func isIPAddrResolvable(fl FieldLevel) bool {
	panic("gomad: Validator address resolution is unsupported")
}`),
			},
			{
				anchor: []byte(`func isUnixAddrResolvable(fl FieldLevel) bool {
	_, err := net.ResolveUnixAddr("unix", fl.Field().String())

	return err == nil
}`),
				replacement: []byte(`func isUnixAddrResolvable(fl FieldLevel) bool {
	panic("gomad: Validator address resolution is unsupported")
}`),
			},
		},
	},
}

func prepareValidator(moduleCache, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
	return prepareRewrittenModule(moduleCache, root, identity, rewrittenModule{
		module: validatorModulePath, version: validatorVersion, sum: validatorSum,
		cacheElements:              []string{"github.com", "go-playground", "validator", "v10@" + validatorVersion},
		replacementDirectory:       "validator",
		originalInventorySHA256:    validatorOriginalSourceInventorySHA256,
		replacementInventorySHA256: validatorReplacementSourceInventorySHA256,
		preparedPackage:            validatorModulePath,
		preparedSourceSetSHA256:    validatorPreparedSourceSetSHA256,
		rewrites:                   validatorRewrites,
	})
}
