package main

import (
	"errors"
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	_ "go.temporal.io/api/workflowservice/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/ir"
	"go.temporal.io/server/tools/umpire/lower"
	"google.golang.org/protobuf/encoding/protojson"
)

// kind is one managed tree of lowered Cases: where it is published, the target that publishes it,
// and the Queries it pins. No pinned Query means every Query of every checked Model.
type kind struct {
	directory string
	target    string
	pinned    []lower.Selected
}

// The functional and canary trees pin Queries of the complete tree, so a consumer's Case is the
// model tree's Case byte for byte. A Query enters a pinned tree only when its Model lowers it.
var kinds = map[string]kind{
	"model": {directory: ir.FeatureDirectory, target: "umpire-gen-model"},
	"functional": {directory: "tests/testcore/testpilot/testdata/generated", target: "umpire-gen-fixtures", pinned: []lower.Selected{
		{Model: "nexus-workflow.json", Query: "asyncCompletion"},
		{Model: "nexus-workflow.json", Query: "asyncFailure"},
		{Model: "nexus-workflow.json", Query: "handlerError"},
		{Model: "nexus-workflow.json", Query: "retry"},
		{Model: "nexus-workflow.json", Query: "scheduleToStartTimeout"},
		{Model: "nexus-workflow.json", Query: "startToCloseTimeout"},
		{Model: "nexus-workflow.json", Query: "syncCompletion"},
		{Model: "nexus-workflow-control.json", Query: "forgedCompletion"},
	}},
	"canary": {directory: "tools/canary/casebinding/testdata", target: "canary-gen-case", pinned: []lower.Selected{
		{Model: "nexus-workflow.json", Query: "syncCompletion"},
	}},
}

// sync narrows the complete tree to what the kind pins and checks or publishes the kind's
// directory under root.
func sync(root string, complete map[string][]byte, selected kind, update bool) error {
	if len(selected.pinned) != 0 {
		manifest, err := lower.DecodeManifest(complete["manifest.json"])
		if err != nil {
			return err
		}
		models := map[string]bool{}
		for _, entry := range manifest.Queries {
			models[entry.Model] = true
		}
		pins := slices.DeleteFunc(slices.Clone(selected.pinned), func(pin lower.Selected) bool { return !models[pin.Model] })
		if len(pins) == 0 {
			return nil
		}
		return syncPins(root, complete, selected, pins, update)
	}
	return lower.SyncCases(filepath.Join(root, filepath.FromSlash(selected.directory)), complete, update)
}

func syncPins(root string, complete map[string][]byte, selected kind, pins []lower.Selected, update bool) error {
	if len(pins) == 0 {
		return nil
	}
	files, err := lower.SelectCases(complete, pins)
	if err != nil {
		return err
	}
	directory := filepath.Join(root, filepath.FromSlash(selected.directory))
	if len(pins) != len(selected.pinned) {
		return lower.SyncSelectedCases(directory, files, update)
	}
	return lower.SyncCases(directory, files, update)
}

// requireTotals refuses each Query of the current IR under irDirectory that declares no static
// combination total. Lowering admits IR without totals, as lifted before the assertion existed;
// Cases are generated only from a current Model, whose every Query declares its total.
func requireTotals(irDirectory string) error {
	paths, err := ir.IRPaths(irDirectory)
	if err != nil {
		return err
	}
	var errs []error
	for _, path := range paths {
		encoded, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		m := &umpirespb.Model{}
		if err := protojson.Unmarshal(encoded, m); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if err := ir.RequireTotals(m); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", path, err))
		}
	}
	return errors.Join(errs...)
}

func main() {
	update := flag.Bool("update", false, "rewrite generated Cases and manifest")
	name := flag.String("kind", "model", "the managed tree: "+strings.Join(slices.Sorted(maps.Keys(kinds)), ", "))
	flag.Parse()
	selected, known := kinds[*name]
	if !known || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}
	scope := ir.FeatureRoot(".")
	if *name == "model" {
		if err := modelCases(scope, *update); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	root, pins, err := pinnedScope(scope, selected.pinned)
	if err == nil && len(pins) == 0 {
		return
	}
	var complete map[string][]byte
	if err == nil {
		complete, err = lower.GenerateCases(scope)
	}
	if err == nil {
		if _, err = lower.SelectCases(complete, pins); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err = requireTotals(scope); err != nil {
			fmt.Fprintf(os.Stderr, "%v; every Query declares its total\n", err)
			os.Exit(1)
		}
		err = syncPins(root, complete, selected, pins, *update)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v; make %s publishes it\n", selected.directory, err, selected.target)
		os.Exit(1)
	}
}

func pinnedScope(scope string, pins []lower.Selected) (string, []lower.Selected, error) {
	absolute, err := filepath.Abs(scope)
	if err != nil {
		return "", nil, err
	}
	physical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", nil, err
	}
	info, err := os.Stat(physical)
	if err != nil {
		return "", nil, err
	}
	if !info.IsDir() {
		return "", nil, errors.New("pinned Case input must be a feature folder")
	}
	for features := absolute; features != filepath.Dir(features); features = filepath.Dir(features) {
		if !strings.HasSuffix(filepath.ToSlash(features), "/"+ir.FeatureDirectory) {
			continue
		}
		resolved, err := filepath.EvalSymlinks(features)
		if err != nil {
			return "", nil, err
		}
		relative, err := filepath.Rel(resolved, physical)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return "", nil, fmt.Errorf("feature scope escapes %s", features)
		}
		root := filepath.Dir(filepath.Dir(filepath.Dir(features)))
		if relative == "." {
			return root, pins, nil
		}
		for _, part := range strings.Split(relative, string(filepath.Separator)) {
			if part == "generated" {
				return "", nil, errors.New("pinned Case input must be a source feature folder, not generated output")
			}
		}
		owner := strings.TrimSuffix(ir.ModelName(filepath.Join(absolute, "generated", "ir.json")), ".json")
		selected := slices.DeleteFunc(slices.Clone(pins), func(pin lower.Selected) bool {
			model := strings.TrimSuffix(pin.Model, ".json")
			return model != owner && !strings.HasPrefix(model, owner+"-")
		})
		return root, selected, nil
	}
	return "", nil, fmt.Errorf("pinned Case input must be a feature scope under %s", ir.FeatureDirectory)
}

func modelCases(scope string, update bool) error {
	paths, err := ir.IRPaths(scope)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("no managed IR tree in %s", scope)
	}
	managed := 0
	for _, path := range paths {
		if filepath.Base(path) != "ir.json" {
			continue
		}
		directory := filepath.Dir(path)
		managed++
		if err := requireTotals(directory); err != nil {
			return fmt.Errorf("%s: %w", directory, err)
		}
		files, err := lower.GenerateCases(directory)
		if err != nil {
			return fmt.Errorf("%s: %w", directory, err)
		}
		if err := lower.SyncCases(directory, files, update); err != nil {
			return fmt.Errorf("%s: %w", directory, err)
		}
	}
	if managed == 0 {
		return fmt.Errorf("no primary managed IR tree in %s", scope)
	}
	return nil
}
