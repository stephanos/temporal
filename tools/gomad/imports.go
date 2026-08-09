//go:build never

package gomad

// This file exists as a dependency for users of gomad. The CLI, through
// internal/translate, needs to be able to import all packages below when run
// _inside_ the user's module. By importing the gomad package their go.mod will
// get all the required dependencies.

// TODO: allow fixing these at a different, independent version? what if there
// are multiple go.mod because we run on multiple versions?

import (
	// Allow running the CLI.
	_ "github.com/temporalio/gomad/cmd/gomad"

	// Packages listed in internal/translate.TranslatedRuntimePackages.
	_ "github.com/temporalio/gomad/internal/hooks/go123"
	_ "github.com/temporalio/gomad/internal/reflect"
	_ "github.com/temporalio/gomad/internal/simulation"
	_ "github.com/temporalio/gomad/internal/testing"

	// Tools used by gomad. For this repository.
	_ "github.com/go-task/task/v3/cmd/task"
	_ "golang.org/x/tools/cmd/goimports"
	_ "mvdan.cc/gofumpt"
)
