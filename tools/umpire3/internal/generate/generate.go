package generate

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
	"path/filepath"
	"slices"
	"strings"

	mutationaudit "go.temporal.io/server/tools/umpire3/assurance/audit/mutation"
	"go.temporal.io/server/tools/umpire3/assurance/audit/resilience"
	"go.temporal.io/server/tools/umpire3/assurance/migration"
	releaseassurance "go.temporal.io/server/tools/umpire3/assurance/release"
	"go.temporal.io/server/tools/umpire3/checker/finite"
	"go.temporal.io/server/tools/umpire3/checker/veil"
	"go.temporal.io/server/tools/umpire3/execution/observation"
	"go.temporal.io/server/tools/umpire3/internal/artifactio"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tools/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
	protocolmonitor "go.temporal.io/server/tools/umpire3/protocol/monitor"
	protocolrelease "go.temporal.io/server/tools/umpire3/protocol/release"
)

type exportSpec struct {
	root       string
	sourceRoot string
	inputs     []string
}

var catalogSpec = exportSpec{
	root: "Umpire3/Command/Export/Catalog.lean", sourceRoot: "Temporal/Catalog.lean",
}

var monitorSpec = exportSpec{
	root: "Umpire3/Command/Export/Monitor.lean", sourceRoot: "Temporal/Monitors.lean",
}

var observationSpec = exportSpec{
	root: "Umpire3/Command/Export/Observation.lean", sourceRoot: "Temporal/Observation.lean",
}

var compositionSpec = exportSpec{
	root: "Umpire3/Command/Export/Composition.lean", sourceRoot: "Temporal/Composition.lean",
}

var paritySpec = exportSpec{
	root: "Umpire3/Command/Export/Parity.lean", sourceRoot: "Temporal/Parity.lean",
}

var coverageSpec = exportSpec{
	root: "Umpire3/Command/Export/Coverage.lean", sourceRoot: "Temporal/Coverage.lean",
}

var finiteReplaySpec = exportSpec{
	root: "Umpire3/Command/Export/FiniteReplay.lean", sourceRoot: "Temporal/Targets/FiniteReplay.lean",
}

var firstOrderSpecs = map[string]exportSpec{
	"sound": {
		root:       "Umpire3/Command/Export/NexusFirstOrder.lean",
		sourceRoot: "Temporal/Families/NexusCancellation/Targets/FirstOrder.lean",
	},
	"mutated": {
		root:       "Umpire3/Command/Export/NexusMutatedFirstOrder.lean",
		sourceRoot: "Temporal/Families/NexusCancellation/Targets/FirstOrder.lean",
	},
}

var attemptSpecs = map[string]exportSpec{
	"sound": {
		root:       "Umpire3/Command/Export/NexusAttempt.lean",
		sourceRoot: "Temporal/Families/NexusCancellation/Targets/Attempt.lean",
	},
	"mutated": {
		root:       "Umpire3/Command/Export/NexusMutatedAttempt.lean",
		sourceRoot: "Temporal/Families/NexusCancellation/Targets/Attempt.lean",
	},
}

var veilBindingSpecs = map[string]exportSpec{
	"sound": {
		root:       "Umpire3/Command/Export/NexusVeilBinding.lean",
		sourceRoot: "Temporal/Families/NexusCancellation/Targets/Veil/Binding.lean",
		inputs:     []string{"lake-manifest.json", "lean-toolchain"},
	},
	"mutated": {
		root:       "Umpire3/Command/Export/NexusMutatedVeilBinding.lean",
		sourceRoot: "Temporal/Families/NexusCancellation/Targets/Veil/MutatedBinding.lean",
		inputs:     []string{"lake-manifest.json", "lean-toolchain"},
	},
	"trusted": {
		root:       "Umpire3/Command/Export/NexusTrustedVeilBinding.lean",
		sourceRoot: "Temporal/Families/NexusCancellation/Targets/Veil/TrustedBinding.lean",
		inputs:     []string{"lake-manifest.json", "lean-toolchain"},
	},
}

var temporalSpecs = map[string]exportSpec{
	"sound": {
		root:       "Umpire3/Command/Export/TaskDeliveryTemporal.lean",
		sourceRoot: "Temporal/Families/WorkflowProgress/Targets/Temporal.lean",
	},
	"delivery-fairness-removed": {
		root:       "Umpire3/Command/Export/TaskDeliveryMutatedTemporal.lean",
		sourceRoot: "Temporal/Families/WorkflowProgress/Targets/Temporal.lean",
	},
}

var exportSpecs = map[string]exportSpec{
	"nexus": {
		root: "Umpire3/Command/Export/Experiment.lean", sourceRoot: "Temporal/Families/NexusCancellation/Experiment.lean",
		inputs: []string{"Temporal/API/testdata/fixtures/selection.json"},
	},
	"update": {
		root: "Umpire3/Command/Export/Update.lean", sourceRoot: "Temporal/Families/UpdateLifecycle/Experiment.lean",
		inputs: []string{"Temporal/API/testdata/fixtures/selection.json"},
	},
}

var proofSpecs = map[string]exportSpec{
	"nexus": {
		root: "Umpire3/Command/Export/NexusProof.lean", sourceRoot: exportSpecs["nexus"].sourceRoot,
		inputs: exportSpecs["nexus"].inputs,
	},
	"nexus-mutation-exact": {
		root:       "Umpire3/Command/Export/NexusExactMutationProof.lean",
		sourceRoot: "Temporal/Families/NexusCancellation/Mutation.lean",
	},
	"nexus-mutation-refinement": {
		root:       "Umpire3/Command/Export/NexusMutationRejectionProof.lean",
		sourceRoot: "Temporal/Families/NexusCancellation/Mutation.lean",
	},
	"update": {
		root: "Umpire3/Command/Export/UpdateProof.lean", sourceRoot: exportSpecs["update"].sourceRoot,
		inputs: exportSpecs["update"].inputs,
	},
}

func Run(arguments []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("umpire3-export", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	modelRoot := flags.String("model-root", "tools/umpire3/model", "path to the Umpire3 Lean model")
	artifact := flags.String("artifact", "experiment", "artifact to export")
	experiment := flags.String("experiment", "nexus", "experiment to export")
	variant := flags.String("variant", "sound", "model variant to export")
	releaseTemplate := flags.String("release-template", "tools/umpire3/assurance/release/testdata/generated/umpire3-1.3.json",
		"candidate release template")
	releaseExperiments := flags.String("release-experiments",
		"tools/umpire3/testdata/generated/nexus-cancellation.json,tools/umpire3/testdata/generated/update-lifecycle.json",
		"comma-separated release experiments")
	migrationLedger := flags.String("migration-ledger", "tools/umpire3/assurance/migration/testdata/generated/ledger.json",
		"release migration ledger")
	checkerNativeCertificate := flags.String("checker-native-certificate",
		"tools/umpire3/checker/finite/testdata/generated/nexus-cancellation-scale.certificate.json",
		"native certificate for checker coverage")
	checkerNativeReceipt := flags.String("checker-native-receipt",
		"tools/umpire3/checker/finite/testdata/generated/nexus-cancellation-scale.receipt.json",
		"native receipt for checker coverage")
	checkerNativeBenchmark := flags.String("checker-native-benchmark",
		"tools/umpire3/checker/finite/testdata/retained/nexus-cancellation-scale.benchmark.json",
		"native scale benchmark for checker coverage")
	checkerVeilBinding := flags.String("checker-veil-binding",
		"tools/umpire3/checker/veil/testdata/generated/nexus-cancellation-sound.json",
		"Veil binding for checker coverage")
	checkerVeilResults := flags.String("checker-veil-results",
		"tools/umpire3/checker/veil/testdata/retained/nexus-cancellation-sound-concrete.json,"+
			"tools/umpire3/checker/veil/testdata/retained/nexus-cancellation-sound-symbolic.json,"+
			"tools/umpire3/checker/veil/testdata/retained/nexus-cancellation-sound-invariant.json",
		"comma-separated Veil results for checker coverage")
	mutationExperiment := flags.String("mutation-experiment",
		"tools/umpire3/testdata/generated/nexus-cancellation.json", "experiment for the semantic mutation audit")
	mutationFiniteReplay := flags.String("mutation-finite-replay-command",
		"tools/umpire3/model/.lake/build/bin/umpire3_trace_replay",
		"canonical Lean finite replay executable for the semantic mutation audit")
	mutationTemporalReplay := flags.String("mutation-temporal-replay-command",
		"tools/umpire3/model/.lake/build/bin/umpire3_temporal_lasso_replay",
		"canonical Lean temporal replay executable for the semantic mutation audit")
	output := flags.String("output", "", "optional output path")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}

	generated, err := (Generator{}).Generate(context.Background(), Request{
		Kind: Kind(*artifact), Variant: *variant,
		Inputs: Inputs{
			ModelRoot: *modelRoot, Experiment: *experiment,
			ReleaseTemplate: *releaseTemplate, ReleaseExperiments: strings.Split(*releaseExperiments, ","),
			MigrationLedger:          *migrationLedger,
			CheckerNativeCertificate: *checkerNativeCertificate, CheckerNativeReceipt: *checkerNativeReceipt,
			CheckerNativeBenchmark: *checkerNativeBenchmark, CheckerVeilBinding: *checkerVeilBinding,
			CheckerVeilResults: strings.Split(*checkerVeilResults, ","),
			MutationExperiment: *mutationExperiment, MutationFiniteReplay: *mutationFiniteReplay,
			MutationTemporalReplay: *mutationTemporalReplay,
		},
	})
	if err != nil {
		return err
	}
	if *output == "" {
		if _, err := stdout.Write(generated.Encoded); err != nil {
			return fmt.Errorf("write exported artifact: %w", err)
		}
		return nil
	}
	if err := artifactio.Publish(*output, generated.Encoded); err != nil {
		return fmt.Errorf("write exported artifact: %w", err)
	}
	return nil
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
	experiment, err := protocolexperiment.DecodeExperiment(bytes.NewReader(experimentBytes), protocolexperiment.DefaultDecodeLimit)
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
	return exportExperimentWith(context.Background(), ProcessLeanRunner{}, modelRoot, spec, writer)
}

func exportExperimentWith(
	ctx context.Context,
	runner LeanRunner,
	modelRoot string,
	spec exportSpec,
	writer io.Writer,
) error {
	semanticHash, err := semanticSourceHash(modelRoot, spec)
	if err != nil {
		return err
	}
	catalog, err := protocolcatalog.DefaultCatalog()
	if err != nil {
		return fmt.Errorf("load semantic catalog: %w", err)
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return fmt.Errorf("digest semantic catalog: %w", err)
	}
	stdout, err := runner.Run(ctx, LeanRequest{
		ModelRoot: modelRoot, Root: spec.root, SemanticHash: semanticHash, CatalogHash: catalogHash,
	})
	if err != nil {
		return err
	}

	experiment, err := protocolexperiment.DecodeExperiment(bytes.NewReader(stdout), protocolexperiment.DefaultDecodeLimit)
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
	release, err := protocolrelease.DecodeReleaseManifest(template)
	if err != nil {
		return fmt.Errorf("decode release template: %w", err)
	}
	release.Documents = map[string]string{
		"authoring":         "docs/authoring.md",
		"incident-recovery": "docs/recovery.md",
		"modeling":          "docs/modeling.md",
		"operations":        "docs/operations.md",
		"security":          "docs/security.md",
		"support":           "docs/support.md",
	}
	for index := range release.ExternalQualifications {
		release.ExternalQualifications[index].Command = fmt.Sprintf(
			"go run -tags test_dep ./tools/umpire3/cmd/umpire3 qualify -profile %s "+
				"-release tools/umpire3/assurance/release/testdata/generated/umpire3-1.3.json "+
				"-experiment <experiment.json> -result <result.json> "+
				"-signing-key <authority.pem> -output <receipt.json>",
			release.ExternalQualifications[index].Profile,
		)
	}
	experiments := make([]protocolexperiment.Experiment, 0, len(experimentPaths))
	for _, path := range experimentPaths {
		encoded, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read release experiment %q: %w", path, readErr)
		}
		experiment, decodeErr := protocolexperiment.DecodeExperiment(
			bytes.NewReader(encoded), protocolexperiment.DefaultDecodeLimit)
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
	return exportCatalogWith(context.Background(), ProcessLeanRunner{}, modelRoot, spec, writer)
}

func exportCatalogWith(
	ctx context.Context,
	runner LeanRunner,
	modelRoot string,
	spec exportSpec,
	writer io.Writer,
) error {
	semanticHash, err := semanticSourceHash(modelRoot, spec)
	if err != nil {
		return err
	}
	stdout, err := runner.Run(ctx, LeanRequest{
		ModelRoot: modelRoot, Root: spec.root, SemanticHash: semanticHash,
	})
	if err != nil {
		return err
	}
	catalog, err := protocolcatalog.DecodeCatalog(bytes.NewReader(stdout), protocolexperiment.DefaultDecodeLimit)
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
	return exportFirstOrderViewWith(context.Background(), ProcessLeanRunner{}, modelRoot, spec, writer)
}

func exportFirstOrderViewWith(
	ctx context.Context,
	runner LeanRunner,
	modelRoot string,
	spec exportSpec,
	writer io.Writer,
) error {
	semanticHash, err := semanticSourceHash(modelRoot, spec)
	if err != nil {
		return err
	}
	stdout, err := runner.Run(ctx, LeanRequest{
		ModelRoot: modelRoot, Root: spec.root, SemanticHash: semanticHash,
	})
	if err != nil {
		return err
	}
	view, err := protocolchecker.DecodeFirstOrderView(bytes.NewReader(stdout), protocolexperiment.DefaultDecodeLimit)
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
	return exportAttemptViewWith(context.Background(), ProcessLeanRunner{}, modelRoot, spec, firstOrderSpec,
		variant, writer)
}

func exportAttemptViewWith(
	ctx context.Context,
	runner LeanRunner,
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
	stdout, err := runner.Run(ctx, LeanRequest{
		ModelRoot: modelRoot, Root: spec.root, SemanticHash: semanticHash,
		DependencyHash: firstOrderSemanticHash,
	})
	if err != nil {
		return err
	}
	view, err := protocolchecker.DecodeAttemptView(bytes.NewReader(stdout), protocolexperiment.DefaultDecodeLimit)
	if err != nil {
		return fmt.Errorf("validate Lean attempt view: %w", err)
	}
	firstOrder, found, err := finite.DefaultFirstOrderView(protocolcatalog.TargetIDNexusCancellation,
		map[string]string{"sound": "sound", "mutated": "stale-completion-guard-removed"}[variant])
	if err != nil {
		return err
	}
	if !found {
		return errors.New("matching first-order view is unavailable")
	}
	if view.SemanticHash != semanticHash || view.FirstOrderSemanticHash != firstOrderSemanticHash {
		return errors.New("lean attempt view semantic hashes do not match their sources")
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
	return exportVeilBindingWith(context.Background(), ProcessLeanRunner{}, modelRoot, spec, firstOrderSpec, writer)
}

func exportVeilBindingWith(
	ctx context.Context,
	runner LeanRunner,
	modelRoot string,
	spec exportSpec,
	firstOrderSpec exportSpec,
	writer io.Writer,
) error {
	if _, err := runner.Run(ctx, LeanRequest{ModelRoot: modelRoot, Target: leanModuleName(spec.sourceRoot)}); err != nil {
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
	stdout, err := runner.Run(ctx, LeanRequest{
		ModelRoot: modelRoot, Root: spec.root, SemanticHash: sourceDigest,
		DependencyHash: firstOrderSemanticHash,
	})
	if err != nil {
		return err
	}
	binding, err := veil.DecodeBindingArtifact(bytes.NewReader(stdout), protocolexperiment.DefaultDecodeLimit)
	if err != nil {
		return fmt.Errorf("validate Lean Veil binding: %w", err)
	}
	if binding.SourceDigest != sourceDigest ||
		binding.Binding.View.SemanticHash != firstOrderSemanticHash {
		return errors.New("lean Veil binding hashes do not match their sources")
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

func leanModuleName(source string) string {
	return strings.TrimSuffix(strings.ReplaceAll(source, "/", "."), ".lean")
}

func exportTemporalView(modelRoot string, spec exportSpec, writer io.Writer) error {
	return exportTemporalViewWith(context.Background(), ProcessLeanRunner{}, modelRoot, spec, writer)
}

func exportTemporalViewWith(
	ctx context.Context,
	runner LeanRunner,
	modelRoot string,
	spec exportSpec,
	writer io.Writer,
) error {
	semanticHash, err := semanticSourceHash(modelRoot, spec)
	if err != nil {
		return err
	}
	stdout, err := runner.Run(ctx, LeanRequest{
		ModelRoot: modelRoot, Root: spec.root, SemanticHash: semanticHash,
	})
	if err != nil {
		return err
	}
	view, err := protocolchecker.DecodeTemporalView(stdout)
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
	return exportProofManifestWith(context.Background(), ProcessLeanRunner{}, modelRoot, spec, writer)
}

func exportProofManifestWith(
	ctx context.Context,
	runner LeanRunner,
	modelRoot string,
	spec exportSpec,
	writer io.Writer,
) error {
	dependencies, err := resolveSourceDependencies(modelRoot, spec.sourceRoot, spec.inputs)
	if err != nil {
		return err
	}
	semanticHash, _, err := protocolchecker.DigestSourceDependencies(dependencies)
	if err != nil {
		return err
	}
	stdout, err := runner.Run(ctx, LeanRequest{
		ModelRoot: modelRoot, Root: spec.root, SemanticHash: semanticHash,
	})
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
	manifest, err := protocolchecker.NewProofManifest(raw.Identifier, raw.Theorem, raw.Statement, raw.ResultClass,
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

func resolveProofDependencies(dependencies []protocolchecker.ProofDependency) error {
	composition, err := protocolcatalog.DefaultComposition()
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
	return exportMonitorCatalogWith(context.Background(), ProcessLeanRunner{}, modelRoot, spec, writer)
}

func exportMonitorCatalogWith(
	ctx context.Context,
	runner LeanRunner,
	modelRoot string,
	spec exportSpec,
	writer io.Writer,
) error {
	semanticHash, err := semanticSourceHash(modelRoot, spec)
	if err != nil {
		return err
	}
	catalog, err := protocolcatalog.DefaultCatalog()
	if err != nil {
		return fmt.Errorf("load semantic catalog: %w", err)
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return fmt.Errorf("digest semantic catalog: %w", err)
	}
	stdout, err := runner.Run(ctx, LeanRequest{
		ModelRoot: modelRoot, Root: spec.root, SemanticHash: semanticHash, CatalogHash: catalogHash,
	})
	if err != nil {
		return err
	}
	monitors, err := protocolmonitor.DecodeMonitorCatalog(stdout)
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
	return exportObservationCatalogWith(context.Background(), ProcessLeanRunner{}, modelRoot, spec, writer)
}

func exportObservationCatalogWith(
	ctx context.Context,
	runner LeanRunner,
	modelRoot string,
	spec exportSpec,
	writer io.Writer,
) error {
	semanticHash, err := semanticSourceHash(modelRoot, spec)
	if err != nil {
		return err
	}
	catalog, err := protocolcatalog.DefaultCatalog()
	if err != nil {
		return fmt.Errorf("load semantic catalog: %w", err)
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return fmt.Errorf("digest semantic catalog: %w", err)
	}
	stdout, err := runner.Run(ctx, LeanRequest{
		ModelRoot: modelRoot, Root: spec.root, SemanticHash: semanticHash, CatalogHash: catalogHash,
	})
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
	return exportCompositionWith(context.Background(), ProcessLeanRunner{}, modelRoot, spec, writer)
}

func exportCompositionWith(
	ctx context.Context,
	runner LeanRunner,
	modelRoot string,
	spec exportSpec,
	writer io.Writer,
) error {
	dependencies, err := resolveSourceDependencies(modelRoot, spec.sourceRoot, spec.inputs)
	if err != nil {
		return err
	}
	semanticHash, dependencyHash, err := protocolchecker.DigestSourceDependencies(dependencies)
	if err != nil {
		return err
	}
	catalog, err := protocolcatalog.DefaultCatalog()
	if err != nil {
		return fmt.Errorf("load semantic catalog: %w", err)
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return fmt.Errorf("digest semantic catalog: %w", err)
	}
	stdout, err := runner.Run(ctx, LeanRequest{
		ModelRoot: modelRoot, Root: spec.root, SemanticHash: semanticHash,
		DependencyHash: dependencyHash, CatalogHash: catalogHash,
	})
	if err != nil {
		return err
	}
	composition, err := protocolcatalog.DecodeComposition(stdout)
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
	return exportParityLedgerWith(context.Background(), ProcessLeanRunner{}, modelRoot, spec, writer)
}

func exportParityLedgerWith(
	ctx context.Context,
	runner LeanRunner,
	modelRoot string,
	spec exportSpec,
	writer io.Writer,
) error {
	dependencies, err := resolveSourceDependencies(modelRoot, spec.sourceRoot, spec.inputs)
	if err != nil {
		return err
	}
	semanticHash, dependencyHash, err := protocolchecker.DigestSourceDependencies(dependencies)
	if err != nil {
		return err
	}
	catalog, err := protocolcatalog.DefaultCatalog()
	if err != nil {
		return fmt.Errorf("load semantic catalog: %w", err)
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return fmt.Errorf("digest semantic catalog: %w", err)
	}
	stdout, err := runner.Run(ctx, LeanRequest{
		ModelRoot: modelRoot, Root: spec.root, SemanticHash: semanticHash,
		DependencyHash: dependencyHash, CatalogHash: catalogHash,
	})
	if err != nil {
		return err
	}
	ledger, err := protocolcatalog.DecodeParityLedger(stdout)
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
	return exportCoverageDenominatorWith(context.Background(), ProcessLeanRunner{}, modelRoot, spec, writer)
}

func exportCoverageDenominatorWith(
	ctx context.Context,
	runner LeanRunner,
	modelRoot string,
	spec exportSpec,
	writer io.Writer,
) error {
	semanticHash, err := semanticSourceHash(modelRoot, spec)
	if err != nil {
		return err
	}
	catalog, err := protocolcatalog.DefaultCatalog()
	if err != nil {
		return fmt.Errorf("load semantic catalog: %w", err)
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return fmt.Errorf("digest semantic catalog: %w", err)
	}
	stdout, err := runner.Run(ctx, LeanRequest{
		ModelRoot: modelRoot, Root: spec.root, SemanticHash: semanticHash, CatalogHash: catalogHash,
	})
	if err != nil {
		return err
	}
	denominator, err := protocolcatalog.DecodeCoverageDenominator(stdout)
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
	return exportFiniteReplayCatalogWith(context.Background(), ProcessLeanRunner{}, modelRoot, spec, writer)
}

func exportFiniteReplayCatalogWith(
	ctx context.Context,
	runner LeanRunner,
	modelRoot string,
	spec exportSpec,
	writer io.Writer,
) error {
	semanticHash, err := semanticSourceHash(modelRoot, spec)
	if err != nil {
		return err
	}
	catalog, err := protocolcatalog.DefaultCatalog()
	if err != nil {
		return fmt.Errorf("load semantic catalog: %w", err)
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return fmt.Errorf("digest semantic catalog: %w", err)
	}
	stdout, err := runner.Run(ctx, LeanRequest{
		ModelRoot: modelRoot, Root: spec.root, SemanticHash: semanticHash, CatalogHash: catalogHash,
	})
	if err != nil {
		return err
	}
	replayCatalog, err := protocolchecker.DecodeFiniteReplayCatalog(stdout)
	if err != nil {
		return fmt.Errorf("validate Lean finite replay catalog: %w", err)
	}
	if replayCatalog.SemanticHash != semanticHash {
		return fmt.Errorf("lean finite replay semantic hash %q does not match sources %q",
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

type leanProofManifest struct {
	FormatVersion string                            `json:"formatVersion"`
	Identifier    string                            `json:"identifier"`
	Theorem       string                            `json:"theorem"`
	Statement     string                            `json:"statement"`
	ResultClass   protocolcatalog.ResultClass       `json:"resultClass"`
	Axioms        []string                          `json:"axioms"`
	LeanVersion   string                            `json:"leanVersion"`
	Assumptions   []protocolchecker.ProofDependency `json:"assumptions"`
}

func decodeLeanProofManifest(encoded []byte) (leanProofManifest, error) {
	if int64(len(encoded)) > protocolexperiment.DefaultDecodeLimit {
		return leanProofManifest{}, fmt.Errorf("lean proof manifest exceeds %d bytes", protocolexperiment.DefaultDecodeLimit)
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
	if manifest.FormatVersion != protocolchecker.ProofManifestFormatVersion || manifest.Identifier == "" ||
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
	sourceHash, _, err := protocolchecker.DigestSourceDependencies(dependencies)
	return sourceHash, err
}

func resolveSourceDependencies(
	modelRoot string,
	sourceRoot string,
	inputs []string,
) ([]protocolchecker.SourceDependency, error) {
	if sourceRoot == "" {
		return nil, errors.New("semantic source root is required")
	}
	dependencies := make(map[string]protocolchecker.SourceDependency)
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
		dependencies[source] = protocolchecker.SourceDependency{
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
		dependencies[input] = protocolchecker.SourceDependency{Path: input, Digest: contentDigest(content), Imports: []string{}}
	}
	result := make([]protocolchecker.SourceDependency, 0, len(dependencies))
	for _, dependency := range dependencies {
		result = append(result, dependency)
	}
	slices.SortFunc(result, func(left, right protocolchecker.SourceDependency) int {
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
