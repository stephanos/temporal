//go:build !unix

package execution

import (
	"context"
	"fmt"
	"runtime"
)

func Run(context.Context, Spec) (Result, error) {
	return Result{}, fmt.Errorf("gomad process supervision is unsupported on %s", runtime.GOOS)
}

func SupervisorMain() error {
	return fmt.Errorf("gomad process supervision is unsupported on %s", runtime.GOOS)
}
