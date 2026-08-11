//go:build !unix

package process

import (
	"context"
	"fmt"
	"runtime"
)

func Run(context.Context, Request) (Result, error) {
	return Result{}, fmt.Errorf("gomadv3 process supervision is unsupported on %s", runtime.GOOS)
}

func SupervisorMain() error {
	return fmt.Errorf("gomadv3 process supervision is unsupported on %s", runtime.GOOS)
}
