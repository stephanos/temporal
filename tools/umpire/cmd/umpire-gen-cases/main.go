package main

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	_ "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/tools/umpire/lower"
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
	"model": {directory: "model/cases", target: "umpire-gen-model"},
	"functional": {directory: "tests/testcore/testpilot/testdata/generated", target: "umpire-gen-fixtures", pinned: []lower.Selected{
		{Model: "nexus-caller.json", Query: "asyncCompletion"},
		{Model: "nexus-caller.json", Query: "asyncFailure"},
		{Model: "nexus-caller.json", Query: "handlerError"},
		{Model: "nexus-caller.json", Query: "retry"},
		{Model: "nexus-caller.json", Query: "scheduleToStartTimeout"},
		{Model: "nexus-caller.json", Query: "startToCloseTimeout"},
		{Model: "nexus-caller.json", Query: "syncCompletion"},
		{Model: "nexus-control.json", Query: "forgedCompletion"},
	}},
	"canary": {directory: "tools/canary/casebinding/testdata", target: "canary-gen-case", pinned: []lower.Selected{
		{Model: "nexus-caller.json", Query: "syncCompletion"},
	}},
}

// sync narrows the complete tree to what the kind pins and checks or publishes the kind's
// directory under root.
func sync(root string, complete map[string][]byte, selected kind, update bool) error {
	files := complete
	if len(selected.pinned) != 0 {
		var err error
		if files, err = lower.SelectCases(complete, selected.pinned); err != nil {
			return err
		}
	}
	return lower.SyncCases(filepath.Join(root, filepath.FromSlash(selected.directory)), files, update)
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
	complete, err := lower.GenerateCases("model/ir")
	if err == nil {
		err = sync(".", complete, selected, *update)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v; make %s publishes it\n", selected.directory, err, selected.target)
		os.Exit(1)
	}
}
