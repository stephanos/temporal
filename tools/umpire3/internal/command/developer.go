package command

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"go.temporal.io/server/tools/umpire3/assurance/migration"
	"go.temporal.io/server/tools/umpire3/internal/generate"
	"go.temporal.io/server/tools/umpire3/internal/generate/api"
	"go.temporal.io/server/tools/umpire3/internal/generate/family"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
)

func RunDeveloper(ctx context.Context, arguments []string, stdout io.Writer) error {
	if len(arguments) == 0 {
		return errors.New("developer operation is required: api, export, family, manifest, or migration")
	}
	switch arguments[0] {
	case "api":
		return api.Run(arguments[1:])
	case "export":
		return generate.Run(arguments[1:], stdout)
	case "family":
		return runFamilyCheck(ctx, arguments[1:])
	case "manifest":
		return writeManifest(arguments[1:], stdout)
	case "migration":
		return writeMigration(arguments[1:])
	default:
		return fmt.Errorf("unknown developer operation %q", arguments[0])
	}
}

func runFamilyCheck(ctx context.Context, arguments []string) error {
	flags := flag.NewFlagSet("family", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	familyName := flags.String("family", "", "catalog model-family identifier")
	repositoryRoot := flags.String("repository-root", ".", "Temporal repository root")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional family arguments")
	}
	if *familyName == "" {
		return errors.New("umpire3 model family is required")
	}
	graph, err := protocolcatalog.DefaultFamilyDependencyGraph()
	if err != nil {
		return err
	}
	plan, err := family.PlanFor(graph, protocolcatalog.TargetID(*familyName), *repositoryRoot)
	if err != nil {
		return err
	}
	return family.Run(ctx, plan, family.ExecRunner{})
}

func writeManifest(arguments []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("manifest", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	leanVersion := flags.String("lean-version", "", "Lean toolchain version")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional manifest arguments")
	}
	if *leanVersion == "" {
		return errors.New("lean-version is required")
	}
	return protocolcatalog.WriteManifest(stdout, protocolcatalog.NewEmptyManifest(*leanVersion))
}

func writeMigration(arguments []string) error {
	flags := flag.NewFlagSet("migration", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	testsRoot := flags.String("tests-root", "tests", "root tests directory")
	output := flags.String("output", "tools/umpire3/assurance/migration/testdata/generated/ledger.json",
		"checked ledger output")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional migration arguments")
	}
	ledger, err := migration.Build(*testsRoot)
	if err != nil {
		return err
	}
	return migration.Write(*output, ledger)
}
