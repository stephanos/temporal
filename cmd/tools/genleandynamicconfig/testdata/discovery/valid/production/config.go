package production

import "go.temporal.io/server/common/dynamicconfig"

var Alpha = dynamicconfig.NewGlobalBoolSetting("Alpha.Key", false, "alpha")

func init() {
	_ = dynamicconfig.NewNamespaceIntSetting("beta.key", 1, "beta")
}

func runtimeOnly() {
	_ = dynamicconfig.NewGlobalBoolSetting("runtime.only", false, "not an initializer")
}
