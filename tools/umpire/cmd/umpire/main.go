// Command umpire runs the one host loop over a Model guest in one of its modes: question,
// explore, reduce, replay or repeat. It exits 0 when the Run is Explained, 1 when Unexplained,
// 2 when Refused or a Driver faulted, and 3 when the loop could not run. See
// model/umpire/hosts/go/README.md.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"umpire.local/model/umpire/runner"
)

// drivers are the Drivers this build acts through, by name. Without one, only question and
// replay run.
var drivers = map[string]runner.NewDriver{}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := runner.Main(ctx, os.Args[1:], os.Stdout, os.Stderr, drivers)
	stop()
	os.Exit(code)
}
