package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "go.temporal.io/api/workflowservice/v1"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	"go.temporal.io/server/model/scalav2/explore"
	"go.temporal.io/server/model/scalav2/goir"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
func run() error {
	if len(os.Args) == 3 && os.Args[1] == "proposal" {
		data, err := os.ReadFile(os.Args[2])
		if err != nil {
			return err
		}
		c, err := explore.ReadProposal(data)
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(append(c.Bytes, '\n'))
		return err
	}
	if len(os.Args) != 1 {
		return errors.New("usage: umpire-ir-bridge [proposal <regression.json>]; bridge runs in model/scalav2")
	}
	paths, err := filepath.Glob("ir/*.json")
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return errors.New("no IR Models in ir/")
	}
	var models []*modelirspb.Model
	for _, path := range paths {
		m, err := goir.Load(path)
		if err != nil {
			return err
		}
		models = append(models, m)
	}
	return explore.Serve(os.Stdin, os.Stdout, models)
}
