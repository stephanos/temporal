package monitor_test

import (
	testmonitor "go.temporal.io/server/tests/testcore/monitor"
	"go.temporal.io/server/tools/umpire1"
	"go.temporal.io/server/tools/umpire2"
)

var (
	_ testmonitor.Monitor = (*umpire1.Monitor)(nil)
	_ testmonitor.Monitor = (*umpire2.Monitor)(nil)
)
