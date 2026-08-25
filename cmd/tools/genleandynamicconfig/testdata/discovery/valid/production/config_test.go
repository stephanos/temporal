package production

import "go.temporal.io/server/common/dynamicconfig"

var testOnly = dynamicconfig.NewGlobalBoolSetting("test.only", false, "test")
