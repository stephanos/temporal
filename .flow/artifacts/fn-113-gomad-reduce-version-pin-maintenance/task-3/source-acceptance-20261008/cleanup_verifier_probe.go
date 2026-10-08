package main

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"go.temporal.io/server/tools/gomad3/deterministicio"
)

func main() {
	source := "/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/github.com/!masterminds/sprig/v3@v3.3.0"
	for iteration := 0; iteration < 8; iteration++ {
		err := deterministicio.VerifyRegisteredAdapter(context.Background(), "github.com/Masterminds/sprig/v3", source, filepath.Join(runtime.GOROOT(), "bin", "go"))
		fmt.Printf("actual unchanged verifier iteration %d source %s result %v\n", iteration, source, err)
	}
}
