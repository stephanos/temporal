package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"go.temporal.io/server/tests/umpire3/protocol"
)

type exportSpec struct {
	root    string
	sources []string
}

var semanticKernelSources = []string{
	"Umpire3/Transition.lean",
	"Umpire3/Executable.lean",
	"Umpire3/Property.lean",
	"Umpire3/Refinement.lean",
	"Umpire3/Experiment.lean",
	"Umpire3/Manifest.lean",
}

var exportSpecs = map[string]exportSpec{
	"nexus": {
		root: "Umpire3Export.lean",
		sources: append(append([]string{}, semanticKernelSources...), []string{
			"Temporal/Product/Nexus.lean",
			"Temporal/System/NexusTasks.lean",
			"Temporal/Refinement/NexusTasks.lean",
			"Temporal/Experiments/NexusCancellation.lean",
			"Temporal/System/TaskDelivery.lean",
		}...),
	},
	"update": {
		root: "Umpire3UpdateExport.lean",
		sources: append(append([]string{}, semanticKernelSources...), []string{
			"Temporal/Product/Update.lean",
			"Temporal/System/UpdateTasks.lean",
			"Temporal/Refinement/UpdateTasks.lean",
			"Temporal/Experiments/UpdateLifecycle.lean",
			"Temporal/System/TaskDelivery.lean",
		}...),
	},
}

func main() {
	modelRoot := flag.String("model-root", "tests/umpire3/model", "path to the Umpire3 Lean model")
	experiment := flag.String("experiment", "nexus", "experiment to export")
	flag.Parse()

	spec, ok := exportSpecs[*experiment]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown experiment %q\n", *experiment)
		os.Exit(1)
	}
	if err := export(*modelRoot, spec); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func export(modelRoot string, spec exportSpec) error {
	semanticHash, err := hashSources(modelRoot, spec.sources)
	if err != nil {
		return err
	}

	command := exec.Command("mise", "exec", "--", "lake", "env", "lean", "--run", spec.root)
	command.Dir = modelRoot
	command.Env = append(os.Environ(), "UMPIRE3_SEMANTIC_HASH="+semanticHash)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("export Lean experiment: %w: %s", err, stderr.String())
	}
	if stderr.Len() != 0 {
		return fmt.Errorf("export Lean experiment emitted diagnostics: %s", stderr.String())
	}

	experiment, err := protocol.DecodeExperiment(&stdout, protocol.DefaultDecodeLimit)
	if err != nil {
		return fmt.Errorf("validate Lean experiment: %w", err)
	}
	if experiment.Model.SemanticHash != semanticHash {
		return fmt.Errorf("lean experiment semantic hash %q does not match sources %q", experiment.Model.SemanticHash, semanticHash)
	}
	encoded, err := experiment.CanonicalJSON()
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if _, err := os.Stdout.Write(encoded); err != nil {
		return fmt.Errorf("write canonical experiment: %w", err)
	}
	return nil
}

func hashSources(modelRoot string, sources []string) (string, error) {
	hash := sha256.New()
	for _, source := range sources {
		content, err := os.ReadFile(filepath.Join(modelRoot, source))
		if err != nil {
			return "", fmt.Errorf("read semantic source %q: %w", source, err)
		}
		if _, err := fmt.Fprintf(hash, "%d:%s:%d:", len(source), source, len(content)); err != nil {
			return "", fmt.Errorf("hash semantic source metadata: %w", err)
		}
		if _, err := hash.Write(content); err != nil {
			return "", fmt.Errorf("hash semantic source: %w", err)
		}
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}
