package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"go.temporal.io/server/tools/gomadv3/runner"
)

type recoverDependencies struct {
	recover func(context.Context, string) (runner.Recovery, error)
}

func runRecover(arguments []string, stdout, stderr io.Writer) int {
	return runRecoverWith(arguments, stdout, stderr, recoverDependencies{recover: runner.Recover})
}

func runRecoverWith(arguments []string, stdout, stderr io.Writer, dependencies recoverDependencies) int {
	flags := flag.NewFlagSet("gomad recover", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 1 || flags.Arg(0) == "" {
		if _, err := fmt.Fprint(stderr, usage); err != nil {
			return 3
		}
		return 2
	}
	recovered, err := dependencies.recover(context.Background(), flags.Arg(0))
	if err != nil {
		if _, writeErr := fmt.Fprintf(stderr, "recover %s: %v\n", flags.Arg(0), err); writeErr != nil {
			return 3
		}
		if runner.IsInvalidRecoveryError(err) {
			return 2
		}
		return 3
	}
	if *jsonOutput {
		encoded, err := json.Marshal(recovered)
		if err != nil {
			if _, writeErr := fmt.Fprintf(stderr, "encode recovery result: %v\n", err); writeErr != nil {
				return 3
			}
			return 3
		}
		if _, err := fmt.Fprintf(stdout, "%s\n", encoded); err != nil {
			return 3
		}
		return 0
	}
	if _, err := fmt.Fprintf(stdout, "gomad recover: path=%s action=%s changed=%t state=%s stable=%s published=%t resumable=%t repairable=%t reason=%s\n", recovered.Path, recovered.Action, recovered.Changed, recovered.After.State, recovered.After.LastStableState, recovered.After.Published, recovered.After.Resumable, recovered.After.Repairable, recovered.After.Reason); err != nil {
		return 3
	}
	return 0
}
