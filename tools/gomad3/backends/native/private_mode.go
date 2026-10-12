package native

import (
	"io"

	"go.temporal.io/server/tools/gomad3/backends/native/execution"
)

func DispatchPrivateMode(mode string, _ io.Reader, _ io.Writer) (bool, error) {
	switch mode {
	case "__supervisor":
		return true, execution.SupervisorMain()
	case "__target_bootstrap":
		return true, execution.BootstrapMain()
	default:
		return false, nil
	}
}
