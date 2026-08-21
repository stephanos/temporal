package main

import (
	"bytes"
	"context"
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

	"go.temporal.io/server/tests/umpire3/migration"
	"go.temporal.io/server/tests/umpire3/model-checkers/veil"
	"go.temporal.io/server/tests/umpire3/mutationaudit"
	"go.temporal.io/server/tests/umpire3/observation"
	"go.temporal.io/server/tests/umpire3/protocol"
	releaseassurance "go.temporal.io/server/tests/umpire3/release"
	"go.temporal.io/server/tests/umpire3/resilience"
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
	root: "Umpire3ObservationExport.lean", sourceRoot: "Temporal/Observation.lean",
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

var finiteReplaySpec = exportSpec{
	root: "Umpire3FiniteReplayExport.lean", sourceRoot: "Temporal/Targets/FiniteReplay.lean",
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

var attemptSpecs = map[string]exportSpec{
	"sound": {
		root:       "Umpire3NexusAttemptExport.lean",
		sourceRoot: "Temporal/Targets/NexusCancellationFencingAttempt.lean",
	},
	"mutated": {
		root:       "Umpire3NexusMutatedAttemptExport.lean",
		sourceRoot: "Temporal/Targets/NexusCancellationFencingAttempt.lean",
	},
}

var veilBindingSpecs = map[string]exportSpec{
	"sound": {
		root:       "Umpire3NexusVeilBindingExport.lean",
		sourceRoot: "Temporal/Veil/NexusCancellationFencing/Binding.lean",
		inputs:     []string{"lake-manifest.json", "lean-toolchain"},
	},
	"mutated": {
		root:       "Umpire3NexusMutatedVeilBindingExport.lean",
		sourceRoot: "Temporal/Veil/NexusCancellationFencing/MutatedBinding.lean",
		inputs:     []string{"lake-manifest.json", "lean-toolchain"},
	},
	"trusted": {
		root:       "Umpire3NexusTrustedVeilBindingExport.lean",
		sourceRoot: "Temporal/Veil/NexusCancellationFencing/TrustedBinding.lean",
		inputs:     []string{"lake-manifest.json", "lean-toolchain"},
	},
}

var temporalSpecs = map[string]exportSpec{
	"sound": {
		root:       "Umpire3TaskDeliveryTemporalExport.lean",
		sourceRoot: "Temporal/Targets/TaskDeliveryProgressTemporal.lean",
	},
	"delivery-fairness-removed": {
		root:       "Umpire3TaskDeliveryMutatedTemporalExport.lean",
		sourceRoot: "Temporal/Targets/TaskDeliveryProgressTemporal.lean",
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
	"nexus-mutation-exact": {
		root:       "Umpire3NexusExactMutationProofExport.lean",
		sourceRoot: "Temporal/Mutations/NexusCancellationFencing.lean",
	},
	"nexus-mutation-refinement": {
		root:       "Umpire3NexusMutationRejectionProofExport.lean",
		sourceRoot: "Temporal/Mutations/NexusCancellationFencing.lean",
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
	releaseTemplate := flag.String("release-template", "tests/umpire3/testdata/umpire3-1.2.json",
		"candidate release template")
	releaseExperiments := flag.String("release-experiments",
		"tests/umpire3/testdata/nexus-cancellation.json,tests/umpire3/testdata/update-lifecycle.json",
		"comma-separated release experiments")
	migrationLedger := flag.String("migration-ledger", "tests/umpire3/migration/ledger.json",
		"release migration ledger")
	checkerNativeCertificate := flag.String("checker-native-certificate",
		"tests/umpire3/model-checkers/native/results/nexus-cancellation-scale.certificate.json",
		"native certificate for checker coverage")
	checkerNativeReceipt := flag.String("checker-native-receipt",
		"tests/umpire3/model-checkers/native/results/nexus-cancellation-scale.receipt.json",
		"native receipt for checker coverage")
	checkerNativeBenchmark := flag.String("checker-native-benchmark",
		"tests/umpire3/model-checkers/native/results/nexus-cancellation-scale.benchmark.json",
		"native scale benchmark for checker coverage")
	checkerVeilBinding := flag.String("checker-veil-binding",
		"tests/umpire3/model-checkers/veil/bindings/nexus-cancellation-sound.json",
		"Veil binding for checker coverage")
	checkerVeilResults := flag.String("checker-veil-results",
		"tests/umpire3/model-checkers/veil/results/nexus-cancellation-sound-concrete.json,"+
			"tests/umpire3/model-checkers/veil/results/nexus-cancellation-sound-symbolic.json,"+
			"tests/umpire3/model-checkers/veil/results/nexus-cancellation-sound-invariant.json",
		"comma-separated Veil results for checker coverage")
	mutationExperiment := flag.String("mutation-experiment",
		"tests/umpire3/testdata/nexus-cancellation.json", "experiment for the semantic mutation audit")
	mutationFiniteReplay := flag.String("mutation-finite-replay-command",
		"tests/umpire3/model/.lake/build/bin/umpire3_trace_replay",
		"canonical Lean finite replay executable for the semantic mutation audit")
	mutationTemporalReplay := flag.String("mutation-temporal-replay-command",
		"tests/umpire3/model/.lake/build/bin/umpire3_temporal_lasso_replay",
		"canonical Lean temporal replay executable for the semantic mutation audit")
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
	case "release-candidate":
		err = exportReleaseCandidate(*releaseTemplate, strings.Split(*releaseExperiments, ","),
			*migrationLedger, &encoded)
	case "resilience-audit":
		err = exportResilienceAudit(context.Background(), &encoded)
	case "semantic-mutation-audit":
		err = exportSemanticMutationAudit(context.Background(), *mutationExperiment,
			*mutationFiniteReplay, *mutationTemporalReplay, &encoded)
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
	case "finite-replay-catalog":
		err = exportFiniteReplayCatalog(*modelRoot, finiteReplaySpec, &encoded)
	case "checker-coverage":
		err = exportCheckerCoverage(checkerCoverageInputs{
			nativeCertificate: *checkerNativeCertificate,
			nativeReceipt:     *checkerNativeReceipt,
			nativeBenchmark:   *checkerNativeBenchmark,
			veilBinding:       *checkerVeilBinding,
			veilResults:       strings.Split(*checkerVeilResults, ","),
		}, &encoded)
	case "family-dependencies":
		err = exportFamilyDependencies(*modelRoot, &encoded)
	case "first-order-view":
		spec, ok := firstOrderSpecs[*variant]
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown first-order variant %q\n", *variant)
			os.Exit(1)
		}
		err = exportFirstOrderView(*modelRoot, spec, &encoded)
	case "attempt-view":
		spec, ok := attemptSpecs[*variant]
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown attempt variant %q\n", *variant)
			os.Exit(1)
		}
		err = exportAttemptView(*modelRoot, spec, firstOrderSpecs[*variant], *variant, &encoded)
	case "veil-binding":
		spec, ok := veilBindingSpecs[*variant]
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown Veil binding variant %q\n", *variant)
			os.Exit(1)
		}
		firstOrderVariant := map[string]string{
			"sound": "sound", "mutated": "mutated", "trusted": "sound",
		}[*variant]
		err = exportVeilBinding(*modelRoot, spec, firstOrderSpecs[firstOrderVariant], &encoded)
	case "temporal-view":
		spec, ok := temporalSpecs[*variant]
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown temporal variant %q\n", *variant)
			os.Exit(1)
		}
		err = exportTemporalView(*modelRoot, spec, &encoded)
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

func exportResilienceAudit(ctx context.Context, writer io.Writer) error {
	report, err := resilience.RunAudit(ctx)
	if err != nil {
		return fmt.Errorf("run resilience audit: %w", err)
	}
	encoded, err := report.CanonicalJSON()
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if _, err := writer.Write(encoded); err != nil {
		return fmt.Errorf("write resilience audit: %w", err)
	}
	return nil
}

func exportSemanticMutationAudit(
	ctx context.Context,
	experimentPath string,
	finiteReplayCommand string,
	temporalReplayCommand string,
	writer io.Writer,
) error {
	experimentBytes, err := os.ReadFile(experimentPath)
	if err != nil {
		return fmt.Errorf("read semantic mutation experiment: %w", err)
	}
	experiment, err := protocol.DecodeExperiment(bytes.NewReader(experimentBytes), protocol.DefaultDecodeLimit)
	if err != nil {
		return fmt.Errorf("decode semantic mutation experiment: %w", err)
	}
	report, err := mutationaudit.Run(ctx, mutationaudit.Request{
		Experiment: experiment, FiniteReplayCommand: []string{finiteReplayCommand},
		TemporalReplayCommand: []string{temporalReplayCommand},
	})
	if err != nil {
		return fmt.Errorf("run semantic mutation audit: %w", err)
	}
	encoded, err := report.CanonicalJSON()
	if err != nil {
		return err
	}
	if _, err := writer.Write(append(encoded, '\n')); err != nil {
		return fmt.Errorf("write semantic mutation audit: %w", err)
	}
	return nil
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

func exportReleaseCandidate(
	templatePath string,
	experimentPaths []string,
	migrationPath string,
	writer io.Writer,
) error {
	template, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("read release template: %w", err)
	}
	release, err := protocol.DecodeReleaseManifest(template)
	if err != nil {
		return fmt.Errorf("decode release template: %w", err)
	}
	experiments := make([]protocol.Experiment, 0, len(experimentPaths))
	for _, path := range experimentPaths {
		encoded, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read release experiment %q: %w", path, readErr)
		}
		experiment, decodeErr := protocol.DecodeExperiment(
			bytes.NewReader(encoded), protocol.DefaultDecodeLimit)
		if decodeErr != nil {
			return fmt.Errorf("decode release experiment %q: %w", path, decodeErr)
		}
		experiments = append(experiments, experiment)
	}
	ledgerBytes, err := os.ReadFile(migrationPath)
	if err != nil {
		return fmt.Errorf("read migration ledger: %w", err)
	}
	ledger, err := migration.DecodeLedger(ledgerBytes)
	if err != nil {
		return err
	}
	release, err = releaseassurance.Bind(release, experiments, ledger, ledgerBytes)
	if err != nil {
		return fmt.Errorf("bind release manifest: %w", err)
	}
	encoded, err := release.CanonicalJSON()
	if err != nil {
		return err
	}
	if _, err := writer.Write(encoded); err != nil {
		return fmt.Errorf("write release manifest: %w", err)
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

func exportAttemptView(
	modelRoot string,
	spec exportSpec,
	firstOrderSpec exportSpec,
	variant string,
	writer io.Writer,
) error {
	semanticHash, err := semanticSourceHash(modelRoot, spec)
	if err != nil {
		return err
	}
	firstOrderSemanticHash, err := semanticSourceHash(modelRoot, firstOrderSpec)
	if err != nil {
		return err
	}
	stdout, err := runLeanWithDependency(
		modelRoot, spec.root, semanticHash, firstOrderSemanticHash, "")
	if err != nil {
		return err
	}
	view, err := protocol.DecodeAttemptView(bytes.NewReader(stdout), protocol.DefaultDecodeLimit)
	if err != nil {
		return fmt.Errorf("validate Lean attempt view: %w", err)
	}
	firstOrder, found, err := protocol.DefaultFirstOrderView(protocol.TargetIDNexusCancellation,
		map[string]string{"sound": "sound", "mutated": "stale-completion-guard-removed"}[variant])
	if err != nil {
		return err
	}
	if !found {
		return errors.New("matching first-order view is unavailable")
	}
	if view.SemanticHash != semanticHash || view.FirstOrderSemanticHash != firstOrderSemanticHash {
		return errors.New("Lean attempt view semantic hashes do not match their sources")
	}
	if err := view.ValidateAgainst(firstOrder); err != nil {
		return fmt.Errorf("validate Lean attempt view against first-order view: %w", err)
	}
	encoded, err := view.CanonicalJSON()
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if _, err := writer.Write(encoded); err != nil {
		return fmt.Errorf("write canonical attempt view: %w", err)
	}
	return nil
}

func exportVeilBinding(
	modelRoot string,
	spec exportSpec,
	firstOrderSpec exportSpec,
	writer io.Writer,
) error {
	if err := buildLeanTarget(modelRoot, leanModuleName(spec.sourceRoot)); err != nil {
		return err
	}
	sourceDigest, err := semanticSourceHash(modelRoot, spec)
	if err != nil {
		return err
	}
	firstOrderSemanticHash, err := semanticSourceHash(modelRoot, firstOrderSpec)
	if err != nil {
		return err
	}
	stdout, err := runLeanWithDependency(
		modelRoot, spec.root, sourceDigest, firstOrderSemanticHash, "")
	if err != nil {
		return err
	}
	binding, err := veil.DecodeBindingArtifact(bytes.NewReader(stdout), protocol.DefaultDecodeLimit)
	if err != nil {
		return fmt.Errorf("validate Lean Veil binding: %w", err)
	}
	if binding.SourceDigest != sourceDigest ||
		binding.Binding.View.SemanticHash != firstOrderSemanticHash {
		return errors.New("Lean Veil binding hashes do not match their sources")
	}
	encoded, err := binding.CanonicalJSON()
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if _, err := writer.Write(encoded); err != nil {
		return fmt.Errorf("write canonical Veil binding: %w", err)
	}
	return nil
}

func buildLeanTarget(modelRoot string, target string) error {
	command := exec.Command("mise", "exec", "--", "lake", "build", target)
	command.Dir = modelRoot
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build Lean target %q: %w: %s", target, err, output)
	}
	return nil
}

func leanModuleName(source string) string {
	return strings.TrimSuffix(strings.ReplaceAll(source, "/", "."), ".lean")
}

func exportTemporalView(modelRoot string, spec exportSpec, writer io.Writer) error {
	semanticHash, err := semanticSourceHash(modelRoot, spec)
	if err != nil {
		return err
	}
	stdout, err := runLean(modelRoot, spec.root, semanticHash, "")
	if err != nil {
		return err
	}
	view, err := protocol.DecodeTemporalView(stdout)
	if err != nil {
		return fmt.Errorf("validate Lean temporal view: %w", err)
	}
	if view.SemanticHash != semanticHash {
		return fmt.Errorf("lean temporal semantic hash %q does not match sources %q",
			view.SemanticHash, semanticHash)
	}
	encoded, err := view.CanonicalJSON()
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if _, err := writer.Write(encoded); err != nil {
		return fmt.Errorf("write canonical temporal view: %w", err)
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
	if err := resolveProofDependencies(raw.Assumptions); err != nil {
		return fmt.Errorf("resolve Lean proof dependencies: %w", err)
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

func resolveProofDependencies(dependencies []protocol.ProofDependency) error {
	composition, err := protocol.DefaultComposition()
	if err != nil {
		return err
	}
	guarantees := make(map[string]string)
	for _, module := range composition.Modules {
		for _, guarantee := range module.Provides {
			guarantees[guarantee.Identifier] = guarantee.StatementHash
		}
	}
	for index := range dependencies {
		if !strings.HasPrefix(dependencies[index].StatementHash, "derived:") {
			continue
		}
		identifier := strings.TrimPrefix(dependencies[index].StatementHash, "derived:")
		if identifier != dependencies[index].Identifier {
			return fmt.Errorf("derived proof dependency %q does not match %q", identifier, dependencies[index].Identifier)
		}
		statementHash, ok := guarantees[identifier]
		if !ok {
			return fmt.Errorf("proof dependency %q has no registered guarantee", identifier)
		}
		dependencies[index].StatementHash = statementHash
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
	dependencies, err := resolveSourceDependencies(modelRoot, spec.sourceRoot, spec.inputs)
	if err != nil {
		return err
	}
	semanticHash, dependencyHash, err := protocol.DigestSourceDependencies(dependencies)
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
	stdout, err := runLeanWithDependency(modelRoot, spec.root, semanticHash, dependencyHash, catalogHash)
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
	dependencies, err := resolveSourceDependencies(modelRoot, spec.sourceRoot, spec.inputs)
	if err != nil {
		return err
	}
	semanticHash, dependencyHash, err := protocol.DigestSourceDependencies(dependencies)
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
	stdout, err := runLeanWithDependency(modelRoot, spec.root, semanticHash, dependencyHash, catalogHash)
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

func exportFiniteReplayCatalog(modelRoot string, spec exportSpec, writer io.Writer) error {
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
	replayCatalog, err := protocol.DecodeFiniteReplayCatalog(stdout)
	if err != nil {
		return fmt.Errorf("validate Lean finite replay catalog: %w", err)
	}
	if replayCatalog.SemanticHash != semanticHash {
		return fmt.Errorf("Lean finite replay semantic hash %q does not match sources %q",
			replayCatalog.SemanticHash, semanticHash)
	}
	encoded, err := replayCatalog.CanonicalJSON()
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if _, err := writer.Write(encoded); err != nil {
		return fmt.Errorf("write canonical finite replay catalog: %w", err)
	}
	return nil
}

func runLean(modelRoot string, root string, semanticHash string, catalogHash string) ([]byte, error) {
	return runLeanWithDependency(modelRoot, root, semanticHash, "", catalogHash)
}

func runLeanWithDependency(
	modelRoot string,
	root string,
	semanticHash string,
	dependencyHash string,
	catalogHash string,
) ([]byte, error) {
	build := exec.Command("mise", "exec", "--", "lake", "build")
	build.Dir = modelRoot
	var buildOutput bytes.Buffer
	build.Stdout = &buildOutput
	build.Stderr = &buildOutput
	if err := build.Run(); err != nil {
		return nil, fmt.Errorf("build Lean model dependencies: %w: %s", err, buildOutput.String())
	}
	command := exec.Command("mise", "exec", "--", "lake", "env", "lean", "--run", root)
	command.Dir = modelRoot
	command.Env = append(os.Environ(), "UMPIRE3_SEMANTIC_HASH="+semanticHash)
	if dependencyHash != "" {
		command.Env = append(command.Env, "UMPIRE3_DEPENDENCY_HASH="+dependencyHash)
	}
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
