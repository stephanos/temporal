package runner

import (
	"fmt"
	"io"

	"go.temporal.io/server/tools/gomadv3/runner/internal/execution"
)

const (
	MinimumChoiceTraceBytes = execution.MinimumChoiceTraceBytes
	MaximumChoiceTraceBytes = execution.MaximumChoiceTraceBytes
)

func DispatchPrivateMode(mode string, stdin io.Reader, stdout io.Writer) error {
	switch mode {
	case "__coordinator":
		return CoordinatorMain(stdin, stdout)
	case "__target_bootstrap":
		return execution.BootstrapMain()
	case "__supervisor":
		return execution.SupervisorMain()
	default:
		return fmt.Errorf("unknown private runner mode %q", mode)
	}
}
