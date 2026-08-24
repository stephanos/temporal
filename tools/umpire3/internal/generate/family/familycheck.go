package family

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"

	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
)

type Plan struct {
	RepositoryRoot string
	LeanModules    []string
	MakeTargets    []string
	GoPackages     []string
}

type Command struct {
	Directory string
	Name      string
	Arguments []string
}

type Runner interface {
	Run(context.Context, Command) error
}

func PlanFor(graph protocolcatalog.FamilyDependencyGraph, target protocolcatalog.TargetID, repositoryRoot string) (Plan, error) {
	family, found := graph.Family(target)
	if !found {
		return Plan{}, fmt.Errorf("unknown Umpire3 model family %q", target)
	}
	leanModules := append(slices.Clone(family.BuildModules), family.LeanTests...)
	slices.Sort(leanModules)
	leanModules = slices.Compact(leanModules)
	makeTargets := []string{
		"umpire3-check-author-facade",
		"umpire3-check-observation",
		"umpire3-check-composition",
		"umpire3-check-coverage",
		"umpire3-check-finite-replay",
	}
	if target == protocolcatalog.TargetIDNexusCancellation || target == protocolcatalog.TargetIDWorkflowUpdateLifecycle {
		makeTargets = append(makeTargets, "umpire3-check-proof")
	}
	goPackages := []string{
		"./tools/umpire3/protocol/...",
		"./tools/umpire3/scenario/...",
		"./tools/umpire3/exploration",
	}
	for _, checker := range family.Checkers {
		switch checker {
		case "exact":
		case "lean-temporal":
			makeTargets = append(makeTargets, "umpire3-check-temporal")
		case "native":
			makeTargets = append(makeTargets, "umpire3-check-native-results")
			goPackages = append(goPackages, "./tools/umpire3/checker/finite")
		case "veil":
			makeTargets = append(makeTargets, "umpire3-check-veil-results")
			goPackages = append(goPackages, "./tools/umpire3/checker/veil")
		default:
			return Plan{}, fmt.Errorf("model family %q has unknown checker %q", target, checker)
		}
	}
	return Plan{
		RepositoryRoot: repositoryRoot,
		LeanModules:    leanModules,
		MakeTargets:    makeTargets,
		GoPackages:     goPackages,
	}, nil
}

func Run(ctx context.Context, plan Plan, runner Runner) error {
	if plan.RepositoryRoot == "" || len(plan.LeanModules) == 0 || len(plan.MakeTargets) == 0 ||
		len(plan.GoPackages) == 0 || runner == nil {
		return errors.New("complete family check plan and runner are required")
	}
	leanArguments := []string{"exec", "--", "lake", "build"}
	leanArguments = append(leanArguments, plan.LeanModules...)
	if err := runner.Run(ctx, Command{
		Directory: filepath.Join(plan.RepositoryRoot, "tools", "umpire3", "model"),
		Name:      "mise", Arguments: leanArguments,
	}); err != nil {
		return fmt.Errorf("build selected Lean family: %w", err)
	}
	if err := runner.Run(ctx, Command{
		Directory: plan.RepositoryRoot, Name: "make", Arguments: plan.MakeTargets,
	}); err != nil {
		return fmt.Errorf("run generated-artifact gates: %w", err)
	}
	goArguments := []string{"test", "-count=1", "-tags", "test_dep"}
	goArguments = append(goArguments, plan.GoPackages...)
	if err := runner.Run(ctx, Command{
		Directory: plan.RepositoryRoot, Name: "go", Arguments: goArguments,
	}); err != nil {
		return fmt.Errorf("run selected Go family tests: %w", err)
	}
	return nil
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, command Command) error {
	process := exec.CommandContext(ctx, command.Name, command.Arguments...)
	process.Dir = command.Directory
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr
	return process.Run()
}
