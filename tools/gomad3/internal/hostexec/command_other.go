//go:build !unix

package hostexec

import (
	"context"
	"fmt"
	"runtime"
)

func Run(context.Context, Request) (Result, error) {
	return Result{}, fmt.Errorf("gomad3 command supervision is unsupported on %s", runtime.GOOS)
}
