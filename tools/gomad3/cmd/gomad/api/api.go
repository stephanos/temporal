// Package api exposes Gomad's command application to backend-specific launchers.
package api

import (
	"context"
	"io"

	"go.temporal.io/server/tools/gomad3/cmd/gomad/internal/cli"
	"go.temporal.io/server/tools/gomad3/runner"
	runnerbackend "go.temporal.io/server/tools/gomad3/runner/backend"
)

type BackendInstallation struct {
	Name          string
	Provider      runnerbackend.Provider
	ReplayCommand []string
}

func RunWithBackend(arguments []string, stdout, stderr io.Writer, resolve func() (BackendInstallation, error)) int {
	return RunWithBackendReplay(arguments, stdout, stderr, resolve, nil)
}

func RunWithBackendReplay(arguments []string, stdout, stderr io.Writer, resolve func() (BackendInstallation, error), replay func(context.Context, runner.ReplaySpec) (runner.ReplayResult, bool, error)) int {
	if resolve == nil {
		return cli.RunWithBackendReplay(arguments, stdout, stderr, nil, replay)
	}
	return cli.RunWithBackendReplay(arguments, stdout, stderr, func() (cli.BackendInstallation, error) {
		installed, err := resolve()
		return cli.BackendInstallation{Name: installed.Name, Provider: installed.Provider, ReplayCommand: append([]string(nil), installed.ReplayCommand...)}, err
	}, replay)
}
