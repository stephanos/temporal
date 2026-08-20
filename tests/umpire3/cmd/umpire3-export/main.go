package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
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
	"Umpire3/Catalog.lean",
	"Umpire3/Transition.lean",
	"Umpire3/Executable.lean",
	"Umpire3/Property.lean",
	"Umpire3/Refinement.lean",
	"Umpire3/Experiment.lean",
	"Umpire3/Explore.lean",
	"Umpire3/Fault.lean",
	"Umpire3/Manifest.lean",
	"Umpire3/Value.lean",
}

var catalogSpec = exportSpec{
	root: "Umpire3CatalogExport.lean",
	sources: []string{
		"Umpire3/Catalog.lean",
		"Umpire3/Declaration.lean",
		"Umpire3/Fault.lean",
		"Umpire3/Value.lean",
		"Temporal/Catalog.lean",
		"Temporal/Inventory.lean",
		"Temporal/Product/NexusLifecycle.lean",
		"Temporal/Product/NexusClosure.lean",
		"Temporal/Product/NexusActivityLink.lean",
		"Temporal/Product/NexusTimeout.lean",
		"Temporal/Product/CallbackReference.lean",
		"Temporal/Product/CallbackResponse.lean",
		"Temporal/Product/WorkflowLineage.lean",
		"Temporal/Product/WorkflowRouting.lean",
		"Temporal/Product/WorkflowOwnership.lean",
		"Temporal/Product/SpeculativeTask.lean",
		"Temporal/Product/WorkflowProgress.lean",
		"Temporal/System/NexusClosure.lean",
		"Temporal/System/NexusActivityLink.lean",
		"Temporal/System/NexusTimeout.lean",
		"Temporal/System/CallbackReference.lean",
		"Temporal/System/CallbackResponse.lean",
		"Temporal/System/WorkflowLineage.lean",
		"Temporal/System/WorkflowRouting.lean",
		"Temporal/System/WorkflowOwnership.lean",
		"Temporal/System/SpeculativeTask.lean",
		"Temporal/System/WorkflowProgress.lean",
		"Temporal/Refinement/NexusClosure.lean",
		"Temporal/Refinement/NexusActivityLink.lean",
		"Temporal/Refinement/NexusTimeout.lean",
		"Temporal/Refinement/CallbackReference.lean",
		"Temporal/Refinement/CallbackResponse.lean",
		"Temporal/Refinement/WorkflowLineage.lean",
		"Temporal/Refinement/WorkflowRouting.lean",
		"Temporal/Refinement/WorkflowOwnership.lean",
		"Temporal/Refinement/SpeculativeTask.lean",
		"Temporal/Refinement/WorkflowProgress.lean",
		"Temporal/Product/TaskAck.lean",
	},
}

var monitorSpec = exportSpec{
	root: "Umpire3MonitorExport.lean",
	sources: []string{
		"Umpire3/Manifest.lean",
		"Umpire3/Monitor.lean",
		"Temporal/Monitors.lean",
		"Temporal/Product/NexusClosure.lean",
		"Temporal/Product/NexusActivityLink.lean",
		"Temporal/Product/NexusTimeout.lean",
		"Temporal/Product/CallbackReference.lean",
		"Temporal/Product/CallbackResponse.lean",
		"Temporal/Product/WorkflowLineage.lean",
		"Temporal/Product/WorkflowRouting.lean",
		"Temporal/Product/WorkflowOwnership.lean",
		"Temporal/Product/SpeculativeTask.lean",
		"Temporal/Product/WorkflowProgress.lean",
	},
}

var compositionSpec = exportSpec{
	root: "Umpire3CompositionExport.lean",
	sources: []string{
		"Umpire3/Composition.lean",
		"Temporal/Composition.lean",
		"Temporal/Product/Nexus.lean",
		"Temporal/Product/NexusLifecycle.lean",
		"Temporal/Product/NexusClosure.lean",
		"Temporal/Product/NexusActivityLink.lean",
		"Temporal/Product/NexusTimeout.lean",
		"Temporal/Product/CallbackReference.lean",
		"Temporal/Product/CallbackResponse.lean",
		"Temporal/Product/WorkflowLineage.lean",
		"Temporal/Product/WorkflowRouting.lean",
		"Temporal/Product/WorkflowOwnership.lean",
		"Temporal/Product/SpeculativeTask.lean",
		"Temporal/Product/WorkflowProgress.lean",
		"Temporal/Product/TaskAck.lean",
		"Temporal/Product/Update.lean",
		"Temporal/System/NexusTasks.lean",
		"Temporal/System/NexusClosure.lean",
		"Temporal/System/NexusActivityLink.lean",
		"Temporal/System/NexusTimeout.lean",
		"Temporal/System/CallbackReference.lean",
		"Temporal/System/CallbackResponse.lean",
		"Temporal/System/WorkflowLineage.lean",
		"Temporal/System/WorkflowRouting.lean",
		"Temporal/System/WorkflowOwnership.lean",
		"Temporal/System/SpeculativeTask.lean",
		"Temporal/System/WorkflowProgress.lean",
		"Temporal/Refinement/NexusClosure.lean",
		"Temporal/Refinement/NexusActivityLink.lean",
		"Temporal/Refinement/NexusTimeout.lean",
		"Temporal/Refinement/CallbackReference.lean",
		"Temporal/Refinement/CallbackResponse.lean",
		"Temporal/Refinement/WorkflowLineage.lean",
		"Temporal/Refinement/WorkflowRouting.lean",
		"Temporal/Refinement/WorkflowOwnership.lean",
		"Temporal/Refinement/SpeculativeTask.lean",
		"Temporal/Refinement/WorkflowProgress.lean",
		"Temporal/System/TaskDelivery.lean",
		"Temporal/System/UpdateTasks.lean",
	},
}

var paritySpec = exportSpec{
	root: "Umpire3ParityExport.lean",
	sources: []string{
		"Temporal/Parity.lean",
		"Temporal/Product/NexusClosure.lean",
		"Temporal/Product/NexusActivityLink.lean",
		"Temporal/Product/NexusTimeout.lean",
		"Temporal/Product/CallbackReference.lean",
		"Temporal/Product/CallbackResponse.lean",
		"Temporal/Product/WorkflowLineage.lean",
		"Temporal/Product/WorkflowRouting.lean",
		"Temporal/Product/WorkflowOwnership.lean",
		"Temporal/Product/SpeculativeTask.lean",
		"Temporal/Product/WorkflowProgress.lean",
		"Temporal/System/NexusClosure.lean",
		"Temporal/System/NexusActivityLink.lean",
		"Temporal/System/NexusTimeout.lean",
		"Temporal/System/CallbackReference.lean",
		"Temporal/System/CallbackResponse.lean",
		"Temporal/System/WorkflowLineage.lean",
		"Temporal/System/WorkflowRouting.lean",
		"Temporal/System/WorkflowOwnership.lean",
		"Temporal/System/SpeculativeTask.lean",
		"Temporal/System/WorkflowProgress.lean",
		"Temporal/Refinement/NexusClosure.lean",
		"Temporal/Refinement/NexusActivityLink.lean",
		"Temporal/Refinement/NexusTimeout.lean",
		"Temporal/Refinement/CallbackReference.lean",
		"Temporal/Refinement/CallbackResponse.lean",
		"Temporal/Refinement/WorkflowLineage.lean",
		"Temporal/Refinement/WorkflowRouting.lean",
		"Temporal/Refinement/WorkflowOwnership.lean",
		"Temporal/Refinement/SpeculativeTask.lean",
		"Temporal/Refinement/WorkflowProgress.lean",
		"Temporal/Monitors.lean",
		"Temporal/Product/TaskAck.lean",
	},
}

var coverageSpec = exportSpec{
	root: "Umpire3CoverageExport.lean",
	sources: []string{
		"Umpire3/Transition.lean",
		"Umpire3/Executable.lean",
		"Umpire3/Property.lean",
		"Temporal/Product/NexusLifecycle.lean",
		"Temporal/Coverage.lean",
	},
}

var exportSpecs = map[string]exportSpec{
	"nexus": {
		root: "Umpire3Export.lean",
		sources: append(append([]string{}, semanticKernelSources...), []string{
			"Temporal/API/selection.json",
			"Temporal/API/Generated/Wire.lean",
			"Temporal/API/Interpretation.lean",
			"Temporal/API/Nexus.lean",
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
			"Temporal/API/selection.json",
			"Temporal/API/Generated/Wire.lean",
			"Temporal/API/Interpretation.lean",
			"Temporal/API/Update.lean",
			"Temporal/Product/Update.lean",
			"Temporal/System/UpdateTasks.lean",
			"Temporal/Refinement/UpdateTasks.lean",
			"Temporal/Experiments/UpdateLifecycle.lean",
			"Temporal/System/TaskDelivery.lean",
		}...),
	},
}

var proofSpecs = map[string]exportSpec{
	"nexus": {
		root:    "Umpire3NexusProofExport.lean",
		sources: exportSpecs["nexus"].sources,
	},
	"update": {
		root:    "Umpire3UpdateProofExport.lean",
		sources: exportSpecs["update"].sources,
	},
}

func main() {
	modelRoot := flag.String("model-root", "tests/umpire3/model", "path to the Umpire3 Lean model")
	artifact := flag.String("artifact", "experiment", "artifact to export")
	experiment := flag.String("experiment", "nexus", "experiment to export")
	output := flag.String("output", "", "optional output path")
	flag.Parse()

	var encoded bytes.Buffer
	var err error
	switch *artifact {
	case "catalog":
		err = exportCatalog(*modelRoot, catalogSpec, &encoded)
	case "experiment":
		spec, ok := exportSpecs[*experiment]
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown experiment %q\n", *experiment)
			os.Exit(1)
		}
		err = exportExperiment(*modelRoot, spec, &encoded)
	case "proof-manifest":
		spec, ok := proofSpecs[*experiment]
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown experiment %q\n", *experiment)
			os.Exit(1)
		}
		err = exportProofManifest(*modelRoot, spec, &encoded)
	case "go-identifiers":
		var catalog protocol.Catalog
		catalog, err = protocol.DefaultCatalog()
		if err == nil {
			err = exportGoIdentifiers(catalog, &encoded)
		}
	case "author-facade":
		var catalog protocol.Catalog
		catalog, err = protocol.DefaultCatalog()
		if err == nil {
			err = exportAuthorFacade(catalog, &encoded)
		}
	case "experiment-schema":
		err = exportExperimentSchema(&encoded)
	case "monitor-programs":
		err = exportMonitorCatalog(*modelRoot, monitorSpec, &encoded)
	case "composition":
		err = exportComposition(*modelRoot, compositionSpec, &encoded)
	case "parity-ledger":
		err = exportParityLedger(*modelRoot, paritySpec, &encoded)
	case "coverage-denominator":
		err = exportCoverageDenominator(*modelRoot, coverageSpec, &encoded)
	default:
		fmt.Fprintf(os.Stderr, "unknown artifact %q\n", *artifact)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *output == "" {
		if _, err := os.Stdout.Write(encoded.Bytes()); err != nil {
			fmt.Fprintln(os.Stderr, fmt.Errorf("write exported artifact: %w", err))
			os.Exit(1)
		}
		return
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("create artifact directory: %w", err))
		os.Exit(1)
	}
	if err := os.WriteFile(*output, encoded.Bytes(), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("write exported artifact: %w", err))
		os.Exit(1)
	}
}

func exportExperiment(modelRoot string, spec exportSpec, writer io.Writer) error {
	semanticHash, err := hashSources(modelRoot, spec.sources)
	if err != nil {
		return err
	}
	catalog, err := protocol.DefaultCatalog()
	if err != nil {
		return fmt.Errorf("load semantic catalog: %w", err)
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return fmt.Errorf("digest semantic catalog: %w", err)
	}
	stdout, err := runLean(modelRoot, spec.root, semanticHash, catalogHash)
	if err != nil {
		return err
	}

	experiment, err := protocol.DecodeExperiment(bytes.NewReader(stdout), protocol.DefaultDecodeLimit)
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
	if _, err := writer.Write(encoded); err != nil {
		return fmt.Errorf("write canonical experiment: %w", err)
	}
	return nil
}

func exportCatalog(modelRoot string, spec exportSpec, writer io.Writer) error {
	semanticHash, err := hashSources(modelRoot, spec.sources)
	if err != nil {
		return err
	}
	stdout, err := runLean(modelRoot, spec.root, semanticHash, "")
	if err != nil {
		return err
	}
	catalog, err := protocol.DecodeCatalog(bytes.NewReader(stdout), protocol.DefaultDecodeLimit)
	if err != nil {
		return fmt.Errorf("validate Lean catalog: %w", err)
	}
	if catalog.SemanticHash != semanticHash {
		return fmt.Errorf("lean catalog semantic hash %q does not match sources %q", catalog.SemanticHash, semanticHash)
	}
	encoded, err := catalog.CanonicalJSON()
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if _, err := writer.Write(encoded); err != nil {
		return fmt.Errorf("write canonical catalog: %w", err)
	}
	return nil
}

func exportProofManifest(modelRoot string, spec exportSpec, writer io.Writer) error {
	semanticHash, err := hashSources(modelRoot, spec.sources)
	if err != nil {
		return err
	}
	stdout, err := runLean(modelRoot, spec.root, semanticHash, "")
	if err != nil {
		return err
	}
	manifest, err := protocol.DecodeProofManifest(bytes.NewReader(stdout), protocol.DefaultDecodeLimit)
	if err != nil {
		return fmt.Errorf("validate Lean proof manifest: %w", err)
	}
	if manifest.SemanticHash != semanticHash {
		return fmt.Errorf("lean proof semantic hash %q does not match sources %q", manifest.SemanticHash, semanticHash)
	}
	encoded, err := manifest.CanonicalJSON()
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if _, err := writer.Write(encoded); err != nil {
		return fmt.Errorf("write canonical proof manifest: %w", err)
	}
	return nil
}

func exportMonitorCatalog(modelRoot string, spec exportSpec, writer io.Writer) error {
	semanticHash, err := hashSources(modelRoot, spec.sources)
	if err != nil {
		return err
	}
	catalog, err := protocol.DefaultCatalog()
	if err != nil {
		return fmt.Errorf("load semantic catalog: %w", err)
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return fmt.Errorf("digest semantic catalog: %w", err)
	}
	stdout, err := runLean(modelRoot, spec.root, semanticHash, catalogHash)
	if err != nil {
		return err
	}
	monitors, err := protocol.DecodeMonitorCatalog(stdout)
	if err != nil {
		return fmt.Errorf("validate Lean monitor catalog: %w", err)
	}
	if monitors.SemanticHash != semanticHash {
		return fmt.Errorf("lean monitor semantic hash %q does not match sources %q", monitors.SemanticHash, semanticHash)
	}
	encoded, err := monitors.CanonicalJSON()
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if _, err := writer.Write(encoded); err != nil {
		return fmt.Errorf("write canonical monitor catalog: %w", err)
	}
	return nil
}

func exportComposition(modelRoot string, spec exportSpec, writer io.Writer) error {
	semanticHash, err := hashSources(modelRoot, spec.sources)
	if err != nil {
		return err
	}
	catalog, err := protocol.DefaultCatalog()
	if err != nil {
		return fmt.Errorf("load semantic catalog: %w", err)
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return fmt.Errorf("digest semantic catalog: %w", err)
	}
	stdout, err := runLean(modelRoot, spec.root, semanticHash, catalogHash)
	if err != nil {
		return err
	}
	composition, err := protocol.DecodeComposition(stdout)
	if err != nil {
		return fmt.Errorf("validate Lean composition: %w", err)
	}
	if composition.SemanticHash != semanticHash {
		return fmt.Errorf("lean composition semantic hash %q does not match sources %q", composition.SemanticHash, semanticHash)
	}
	encoded, err := composition.CanonicalJSON()
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if _, err := writer.Write(encoded); err != nil {
		return fmt.Errorf("write canonical composition: %w", err)
	}
	return nil
}

func exportParityLedger(modelRoot string, spec exportSpec, writer io.Writer) error {
	semanticHash, err := hashSources(modelRoot, spec.sources)
	if err != nil {
		return err
	}
	catalog, err := protocol.DefaultCatalog()
	if err != nil {
		return fmt.Errorf("load semantic catalog: %w", err)
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return fmt.Errorf("digest semantic catalog: %w", err)
	}
	stdout, err := runLean(modelRoot, spec.root, semanticHash, catalogHash)
	if err != nil {
		return err
	}
	ledger, err := protocol.DecodeParityLedger(stdout)
	if err != nil {
		return fmt.Errorf("validate Lean parity ledger: %w", err)
	}
	if ledger.SemanticHash != semanticHash {
		return fmt.Errorf("lean parity semantic hash %q does not match sources %q", ledger.SemanticHash, semanticHash)
	}
	encoded, err := ledger.CanonicalJSON()
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if _, err := writer.Write(encoded); err != nil {
		return fmt.Errorf("write canonical parity ledger: %w", err)
	}
	return nil
}

func exportCoverageDenominator(modelRoot string, spec exportSpec, writer io.Writer) error {
	semanticHash, err := hashSources(modelRoot, spec.sources)
	if err != nil {
		return err
	}
	catalog, err := protocol.DefaultCatalog()
	if err != nil {
		return fmt.Errorf("load semantic catalog: %w", err)
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return fmt.Errorf("digest semantic catalog: %w", err)
	}
	stdout, err := runLean(modelRoot, spec.root, semanticHash, catalogHash)
	if err != nil {
		return err
	}
	denominator, err := protocol.DecodeCoverageDenominator(stdout)
	if err != nil {
		return fmt.Errorf("validate Lean coverage denominator: %w", err)
	}
	if denominator.SemanticHash != semanticHash {
		return fmt.Errorf("lean coverage semantic hash %q does not match sources %q",
			denominator.SemanticHash, semanticHash)
	}
	encoded, err := denominator.CanonicalJSON()
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if _, err := writer.Write(encoded); err != nil {
		return fmt.Errorf("write canonical coverage denominator: %w", err)
	}
	return nil
}

func runLean(modelRoot string, root string, semanticHash string, catalogHash string) ([]byte, error) {
	command := exec.Command("mise", "exec", "--", "lake", "env", "lean", "--run", root)
	command.Dir = modelRoot
	command.Env = append(os.Environ(), "UMPIRE3_SEMANTIC_HASH="+semanticHash)
	if catalogHash != "" {
		command.Env = append(command.Env, "UMPIRE3_CATALOG_HASH="+catalogHash)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("export Lean artifact: %w: %s", err, stderr.String())
	}
	if stderr.Len() != 0 {
		return nil, fmt.Errorf("export Lean artifact emitted diagnostics: %s", stderr.String())
	}
	return stdout.Bytes(), nil
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
