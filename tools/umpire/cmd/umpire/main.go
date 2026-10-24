// Command umpire runs the one host loop over a Model guest in one of its modes: question,
// explore, reduce, replay or repeat. It exits 0 when the Run is Explained, 1 when Unexplained,
// 2 when Refused or a Driver faulted, and 3 when the loop could not run. See
// model/umpire/hosts/go/README.md.
//
// Explore, reduce and repeat act through the Temporal Driver, configured by the UMPIRE_TEMPORAL_*
// environment (tools/umpire/README.md).
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"go.temporal.io/server/common/testing/testpilot/temporal/act"
	"go.temporal.io/server/tools/umpire/temporaldriver"
	"umpire.local/model/runner"
)

// cleanupTimeout bounds removing what the Temporal Driver provisioned.
const cleanupTimeout = time.Minute

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Getenv, os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) (code int) {
	drivers := map[string]runner.NewDriver{}
	opts, err := act.Configure(getenv)
	if err == nil {
		factory, factoryErr := temporaldriver.NewFactory(opts)
		if factoryErr != nil {
			err = factoryErr
		} else {
			drivers["temporal"] = factory.New
			defer func() {
				cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
				defer cancel()
				if cleanupErr := factory.Close(cleanup); cleanupErr != nil {
					_, _ = fmt.Fprintf(stderr, "umpire: temporal driver cleanup: %v\n", cleanupErr)
					if code == runner.ExitExplained {
						code = runner.ExitError
					}
				}
			}()
		}
	}
	if err != nil && len(args) > 0 && slices.Contains([]string{"explore", "reduce", "repeat"}, args[0]) {
		_, _ = fmt.Fprintf(stderr, "umpire: temporal driver: %v\n", err)
		return runner.ExitError
	}
	return runner.Main(ctx, args, stdout, stderr, drivers)
}
