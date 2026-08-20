package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"go.temporal.io/server/tests/umpire3/observation"
	"go.temporal.io/server/tests/umpire3/protocol"
)

type exportSpec struct {
	root       string
	sourceRoot string
	inputs     []string
}

var catalogSpec = exportSpec{
	root: "Umpire3CatalogExport.lean", sourceRoot: "Temporal/Catalog.lean",
}

var monitorSpec = exportSpec{
	root: "Umpire3MonitorExport.lean", sourceRoot: "Temporal/Monitors.lean",
}

var observationSpec = exportSpec{
	root: "Umpire3ObservationExport.lean", sourceRoot: "Temporal/Observation/Nexus.lean",
}

var compositionSpec = exportSpec{
	root: "Umpire3CompositionExport.lean", sourceRoot: "Temporal/Composition.lean",
}

var paritySpec = exportSpec{
	root: "Umpire3ParityExport.lean", sourceRoot: "Temporal/Parity.lean",
}

var coverageSpec = exportSpec{
	root: "Umpire3CoverageExport.lean", sourceRoot: "Temporal/Coverage.lean",
}

var firstOrderSpecs = map[string]exportSpec{
	"sound": {
		root:       "Umpire3NexusFirstOrderExport.lean",
		sourceRoot: "Temporal/Targets/NexusCancellationFencingFirstOrder.lean",
	},
	"mutated": {
		root:       "Umpire3NexusMutatedFirstOrderExport.lean",
		sourceRoot: "Temporal/Targets/NexusCancellationFencingFirstOrder.lean",
	},
}

var exportSpecs = map[string]exportSpec{
	"nexus": {
		root: "Umpire3Export.lean", sourceRoot: "Temporal/Experiments/NexusCancellation.lean",
		inputs: []string{"Temporal/API/selection.json"},
	},
	"update": {
		root: "Umpire3UpdateExport.lean", sourceRoot: "Temporal/Experiments/UpdateLifecycle.lean",
		inputs: []string{"Temporal/API/selection.json"},
	},
}

var proofSpecs = map[string]exportSpec{
	"nexus": {
		root: "Umpire3NexusProofExport.lean", sourceRoot: exportSpecs["nexus"].sourceRoot,
		inputs: exportSpecs["nexus"].inputs,
	},
	"update": {
		root: "Umpire3UpdateProofExport.lean", sourceRoot: exportSpecs["update"].sourceRoot,
		inputs: exportSpecs["update"].inputs,
	},
}

func main() {
	modelRoot := flag.String("model-root", "tests/umpire3/model", "path to the Umpire3 Lean model")
	artifact := flag.String("artifact", "experiment", "artifact to export")
	experiment := flag.String("experiment", "nexus", "experiment to export")
	variant := flag.String("variant", "sound", "model variant to export")
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
	case "observation-programs":
		err = exportObservationCatalog(*modelRoot, observationSpec, &encoded)
	case "composition":
		err = exportComposition(*modelRoot, compositionSpec, &encoded)
	case "parity-ledger":
		err = exportParityLedger(*modelRoot, paritySpec, &encoded)
	case "coverage-denominator":
		err = exportCoverageDenominator(*modelRoot, coverageSpec, &encoded)
	case "first-order-view":
		spec, ok := firstOrderSpecs[*variant]
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown first-order variant %q\n", *variant)
			os.Exit(1)
		}
		err = exportFirstOrderView(*modelRoot, spec, &encoded)
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
	semanticHash, err := semanticSourceHash(modelRoot, spec)
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
	semanticHash, err := semanticSourceHash(modelRoot, spec)
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

func exportFirstOrderView(modelRoot string, spec exportSpec, writer io.Writer) error {
	semanticHash, err := semanticSourceHash(modelRoot, spec)
	if err != nil {
		return err
	}
	stdout, err := runLean(modelRoot, spec.root, semanticHash, "")
	if err != nil {
		return err
	}
	view, err := protocol.DecodeFirstOrderView(bytes.NewReader(stdout), protocol.DefaultDecodeLimit)
	if err != nil {
		return fmt.Errorf("validate Lean first-order view: %w", err)
	}
	if view.SemanticHash != semanticHash {
		return fmt.Errorf("lean first-order semantic hash %q does not match sources %q",
			view.SemanticHash, semanticHash)
	}
	encoded, err := view.CanonicalJSON()
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if _, err := writer.Write(encoded); err != nil {
		return fmt.Errorf("write canonical first-order view: %w", err)
	}
	return nil
}

func exportProofManifest(modelRoot string, spec exportSpec, writer io.Writer) error {
	dependencies, err := resolveSourceDependencies(modelRoot, spec.sourceRoot, spec.inputs)
	if err != nil {
		return err
	}
	semanticHash, _, err := protocol.DigestSourceDependencies(dependencies)
	if err != nil {
		return err
	}
	stdout, err := runLean(modelRoot, spec.root, semanticHash, "")
	if err != nil {
		return err
	}
	raw, err := decodeLeanProofManifest(stdout)
	if err != nil {
		return fmt.Errorf("validate Lean proof manifest: %w", err)
	}
	manifest, err := protocol.NewProofManifest(raw.Identifier, raw.Theorem, raw.Statement, raw.ResultClass,
		raw.Axioms, raw.LeanVersion, raw.Assumptions, dependencies)
	if err != nil {
		return fmt.Errorf("bind Lean proof provenance: %w", err)
	}
	if manifest.SemanticHash != semanticHash {
		return fmt.Errorf("proof semantic hash %q does not match sources %q", manifest.SemanticHash, semanticHash)
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
	semanticHash, err := semanticSourceHash(modelRoot, spec)
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

func exportObservationCatalog(modelRoot string, spec exportSpec, writer io.Writer) error {
	semanticHash, err := semanticSourceHash(modelRoot, spec)
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
	programs, err := observation.DecodeCatalog(stdout)
	if err != nil {
		return fmt.Errorf("validate Lean observation catalog: %w", err)
	}
	if programs.SemanticHash != semanticHash {
		return fmt.Errorf("lean observation semantic hash %q does not match sources %q",
			programs.SemanticHash, semanticHash)
	}
	encoded, err := programs.CanonicalJSON()
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if _, err := writer.Write(encoded); err != nil {
		return fmt.Errorf("write canonical observation catalog: %w", err)
	}
	return nil
}

func exportComposition(modelRoot string, spec exportSpec, writer io.Writer) error {
	semanticHash, err := semanticSourceHash(modelRoot, spec)
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
	semanticHash, err := semanticSourceHash(modelRoot, spec)
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
	semanticHash, err := semanticSourceHash(modelRoot, spec)
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

type leanProofManifest struct {
	FormatVersion string                     `json:"formatVersion"`
	Identifier    string                     `json:"identifier"`
	Theorem       string                     `json:"theorem"`
	Statement     string                     `json:"statement"`
	ResultClass   protocol.ResultClass       `json:"resultClass"`
	Axioms        []string                   `json:"axioms"`
	LeanVersion   string                     `json:"leanVersion"`
	Assumptions   []protocol.ProofDependency `json:"assumptions"`
}

func decodeLeanProofManifest(encoded []byte) (leanProofManifest, error) {
	if int64(len(encoded)) > protocol.DefaultDecodeLimit {
		return leanProofManifest{}, fmt.Errorf("lean proof manifest exceeds %d bytes", protocol.DefaultDecodeLimit)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var manifest leanProofManifest
	if err := decoder.Decode(&manifest); err != nil {
		return leanProofManifest{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return leanProofManifest{}, errors.New("lean proof manifest must contain exactly one JSON value")
	}
	if manifest.FormatVersion != protocol.ProofManifestFormatVersion || manifest.Identifier == "" ||
		manifest.Theorem == "" || manifest.Statement == "" || manifest.LeanVersion == "" {
		return leanProofManifest{}, errors.New("complete resolved Lean proof identity is required")
	}
	return manifest, nil
}

func semanticSourceHash(modelRoot string, spec exportSpec) (string, error) {
	dependencies, err := resolveSourceDependencies(modelRoot, spec.sourceRoot, spec.inputs)
	if err != nil {
		return "", err
	}
	sourceHash, _, err := protocol.DigestSourceDependencies(dependencies)
	return sourceHash, err
}

func resolveSourceDependencies(
	modelRoot string,
	sourceRoot string,
	inputs []string,
) ([]protocol.SourceDependency, error) {
	if sourceRoot == "" {
		return nil, errors.New("semantic source root is required")
	}
	dependencies := make(map[string]protocol.SourceDependency)
	visiting := make(map[string]bool)
	var visit func(string) error
	visit = func(source string) error {
		source, err := cleanSourcePath(source)
		if err != nil {
			return err
		}
		if _, exists := dependencies[source]; exists {
			return nil
		}
		if visiting[source] {
			return fmt.Errorf("cyclic local Lean import through %q", source)
		}
		visiting[source] = true
		defer delete(visiting, source)
		content, err := os.ReadFile(filepath.Join(modelRoot, filepath.FromSlash(source)))
		if err != nil {
			return fmt.Errorf("read semantic source %q: %w", source, err)
		}
		imports, err := localLeanImports(modelRoot, content)
		if err != nil {
			return fmt.Errorf("resolve imports for %q: %w", source, err)
		}
		for _, imported := range imports {
			if err := visit(imported); err != nil {
				return err
			}
		}
		dependencies[source] = protocol.SourceDependency{
			Path: source, Digest: contentDigest(content), Imports: imports,
		}
		return nil
	}
	if err := visit(sourceRoot); err != nil {
		return nil, err
	}
	for _, input := range inputs {
		input, err := cleanSourcePath(input)
		if err != nil {
			return nil, err
		}
		if _, exists := dependencies[input]; exists {
			continue
		}
		content, err := os.ReadFile(filepath.Join(modelRoot, filepath.FromSlash(input)))
		if err != nil {
			return nil, fmt.Errorf("read semantic input %q: %w", input, err)
		}
		dependencies[input] = protocol.SourceDependency{Path: input, Digest: contentDigest(content), Imports: []string{}}
	}
	result := make([]protocol.SourceDependency, 0, len(dependencies))
	for _, dependency := range dependencies {
		result = append(result, dependency)
	}
	slices.SortFunc(result, func(left, right protocol.SourceDependency) int {
		return strings.Compare(left.Path, right.Path)
	})
	return result, nil
}

func localLeanImports(modelRoot string, content []byte) ([]string, error) {
	var imports []string
	for _, line := range strings.Split(string(content), "\n") {
		line, _, _ = strings.Cut(line, "--")
		fields := strings.Fields(line)
		importIndex := slices.Index(fields, "import")
		if importIndex < 0 || importIndex > 1 {
			continue
		}
		for _, module := range fields[importIndex+1:] {
			path := strings.ReplaceAll(module, ".", "/") + ".lean"
			info, err := os.Stat(filepath.Join(modelRoot, filepath.FromSlash(path)))
			switch {
			case err == nil && !info.IsDir():
				imports = append(imports, path)
			case err == nil:
				return nil, fmt.Errorf("local import %q resolves to a directory", module)
			case os.IsNotExist(err):
				continue
			default:
				return nil, fmt.Errorf("stat local import %q: %w", module, err)
			}
		}
	}
	slices.Sort(imports)
	return slices.Compact(imports), nil
}

func cleanSourcePath(source string) (string, error) {
	source = filepath.ToSlash(filepath.Clean(source))
	if source == "." || filepath.IsAbs(source) || source == ".." || strings.HasPrefix(source, "../") {
		return "", fmt.Errorf("semantic source path %q must remain under the model root", source)
	}
	return source, nil
}

func contentDigest(content []byte) string {
	digest := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(digest[:])
}
