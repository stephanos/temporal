package generate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"

	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
)

type Kind string

const (
	KindCatalog               Kind = "catalog"
	KindExperiment            Kind = "experiment"
	KindProofManifest         Kind = "proof-manifest"
	KindReleaseCandidate      Kind = "release-candidate"
	KindResilienceAudit       Kind = "resilience-audit"
	KindSemanticMutationAudit Kind = "semantic-mutation-audit"
	KindGoIdentifiers         Kind = "go-identifiers"
	KindAuthorFacade          Kind = "author-facade"
	KindExperimentSchema      Kind = "experiment-schema"
	KindMonitorPrograms       Kind = "monitor-programs"
	KindObservationPrograms   Kind = "observation-programs"
	KindComposition           Kind = "composition"
	KindParityLedger          Kind = "parity-ledger"
	KindCoverageDenominator   Kind = "coverage-denominator"
	KindFiniteReplayCatalog   Kind = "finite-replay-catalog"
	KindCheckerCoverage       Kind = "checker-coverage"
	KindFamilyDependencies    Kind = "family-dependencies"
	KindFirstOrderView        Kind = "first-order-view"
	KindAttemptView           Kind = "attempt-view"
	KindVeilBinding           Kind = "veil-binding"
	KindTemporalView          Kind = "temporal-view"
)

type Inputs struct {
	ModelRoot                string
	Experiment               string
	ReleaseTemplate          string
	ReleaseExperiments       []string
	MigrationLedger          string
	CheckerNativeCertificate string
	CheckerNativeReceipt     string
	CheckerNativeBenchmark   string
	CheckerVeilBinding       string
	CheckerVeilResults       []string
	MutationExperiment       string
	MutationFiniteReplay     string
	MutationTemporalReplay   string
}

type Request struct {
	Kind    Kind
	Variant string
	Inputs  Inputs
}

type Artifact struct {
	Kind    Kind
	Encoded []byte
}

type LeanRequest struct {
	ModelRoot      string
	Root           string
	Target         string
	SemanticHash   string
	DependencyHash string
	CatalogHash    string
}

type LeanRunner interface {
	Run(context.Context, LeanRequest) ([]byte, error)
}

type ProcessLeanRunner struct{}

func (ProcessLeanRunner) Run(ctx context.Context, request LeanRequest) ([]byte, error) {
	if request.Target != "" {
		command := exec.CommandContext(ctx, "mise", "exec", "--", "lake", "build", request.Target)
		command.Dir = request.ModelRoot
		output, err := command.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("build Lean target %q: %w: %s", request.Target, err, output)
		}
		return nil, nil
	}
	build := exec.CommandContext(ctx, "mise", "exec", "--", "lake", "build")
	build.Dir = request.ModelRoot
	var buildOutput bytes.Buffer
	build.Stdout = &buildOutput
	build.Stderr = &buildOutput
	if err := build.Run(); err != nil {
		return nil, fmt.Errorf("build Lean model dependencies: %w: %s", err, buildOutput.String())
	}
	command := exec.CommandContext(ctx, "mise", "exec", "--", "lake", "env", "lean", "--run", request.Root)
	command.Dir = request.ModelRoot
	command.Env = append(os.Environ(), "UMPIRE3_SEMANTIC_HASH="+request.SemanticHash)
	if request.DependencyHash != "" {
		command.Env = append(command.Env, "UMPIRE3_DEPENDENCY_HASH="+request.DependencyHash)
	}
	if request.CatalogHash != "" {
		command.Env = append(command.Env, "UMPIRE3_CATALOG_HASH="+request.CatalogHash)
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

type InMemoryLeanRunner struct {
	Outputs  map[string][]byte
	Requests []LeanRequest
}

func (r *InMemoryLeanRunner) Run(ctx context.Context, request LeanRequest) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.Requests = append(r.Requests, request)
	key := request.Root
	if request.Target != "" {
		key = request.Target
	}
	encoded, ok := r.Outputs[key]
	if request.Target != "" && !ok {
		return nil, nil
	}
	if !ok {
		return nil, fmt.Errorf("no in-memory Lean output for %q", key)
	}
	return slices.Clone(encoded), nil
}

type Generator struct {
	Lean LeanRunner
}

func (g Generator) Generate(ctx context.Context, request Request) (Artifact, error) {
	runner := g.Lean
	if runner == nil {
		runner = ProcessLeanRunner{}
	}
	var encoded bytes.Buffer
	switch request.Kind {
	case KindCatalog:
		if request.Inputs.ModelRoot == "" {
			return Artifact{}, errors.New("generator model root is required")
		}
		if err := exportCatalogWith(ctx, runner, request.Inputs.ModelRoot, catalogSpec, &encoded); err != nil {
			return Artifact{}, err
		}
	case KindExperiment:
		spec, ok := exportSpecs[request.Inputs.Experiment]
		if !ok {
			return Artifact{}, fmt.Errorf("unknown experiment %q", request.Inputs.Experiment)
		}
		if err := exportExperimentWith(ctx, runner, request.Inputs.ModelRoot, spec, &encoded); err != nil {
			return Artifact{}, err
		}
	case KindProofManifest:
		spec, ok := proofSpecs[request.Inputs.Experiment]
		if !ok {
			return Artifact{}, fmt.Errorf("unknown experiment %q", request.Inputs.Experiment)
		}
		if err := exportProofManifestWith(ctx, runner, request.Inputs.ModelRoot, spec, &encoded); err != nil {
			return Artifact{}, err
		}
	case KindReleaseCandidate:
		if err := exportReleaseCandidate(request.Inputs.ReleaseTemplate, request.Inputs.ReleaseExperiments,
			request.Inputs.MigrationLedger, &encoded); err != nil {
			return Artifact{}, err
		}
	case KindResilienceAudit:
		if err := exportResilienceAudit(ctx, &encoded); err != nil {
			return Artifact{}, err
		}
	case KindSemanticMutationAudit:
		if err := exportSemanticMutationAudit(ctx, request.Inputs.MutationExperiment,
			request.Inputs.MutationFiniteReplay, request.Inputs.MutationTemporalReplay, &encoded); err != nil {
			return Artifact{}, err
		}
	case KindGoIdentifiers:
		semanticCatalog, err := protocolcatalog.DefaultCatalog()
		if err != nil {
			return Artifact{}, err
		}
		if err := exportGoIdentifiers(semanticCatalog, &encoded); err != nil {
			return Artifact{}, err
		}
	case KindAuthorFacade:
		semanticCatalog, err := protocolcatalog.DefaultCatalog()
		if err != nil {
			return Artifact{}, err
		}
		if err := exportAuthorFacade(semanticCatalog, &encoded); err != nil {
			return Artifact{}, err
		}
	case KindExperimentSchema:
		if err := exportExperimentSchema(&encoded); err != nil {
			return Artifact{}, err
		}
	case KindMonitorPrograms:
		if err := exportMonitorCatalogWith(ctx, runner, request.Inputs.ModelRoot, monitorSpec, &encoded); err != nil {
			return Artifact{}, err
		}
	case KindObservationPrograms:
		if err := exportObservationCatalogWith(ctx, runner, request.Inputs.ModelRoot, observationSpec, &encoded); err != nil {
			return Artifact{}, err
		}
	case KindComposition:
		if err := exportCompositionWith(ctx, runner, request.Inputs.ModelRoot, compositionSpec, &encoded); err != nil {
			return Artifact{}, err
		}
	case KindParityLedger:
		if err := exportParityLedgerWith(ctx, runner, request.Inputs.ModelRoot, paritySpec, &encoded); err != nil {
			return Artifact{}, err
		}
	case KindCoverageDenominator:
		if err := exportCoverageDenominatorWith(ctx, runner, request.Inputs.ModelRoot, coverageSpec, &encoded); err != nil {
			return Artifact{}, err
		}
	case KindFiniteReplayCatalog:
		if err := exportFiniteReplayCatalogWith(ctx, runner, request.Inputs.ModelRoot, finiteReplaySpec, &encoded); err != nil {
			return Artifact{}, err
		}
	case KindCheckerCoverage:
		if err := exportCheckerCoverage(checkerCoverageInputs{
			nativeCertificate: request.Inputs.CheckerNativeCertificate,
			nativeReceipt:     request.Inputs.CheckerNativeReceipt,
			nativeBenchmark:   request.Inputs.CheckerNativeBenchmark,
			veilBinding:       request.Inputs.CheckerVeilBinding,
			veilResults:       request.Inputs.CheckerVeilResults,
		}, &encoded); err != nil {
			return Artifact{}, err
		}
	case KindFamilyDependencies:
		if err := exportFamilyDependencies(request.Inputs.ModelRoot, &encoded); err != nil {
			return Artifact{}, err
		}
	case KindFirstOrderView:
		spec, ok := firstOrderSpecs[request.Variant]
		if !ok {
			return Artifact{}, fmt.Errorf("unknown first-order variant %q", request.Variant)
		}
		if err := exportFirstOrderViewWith(ctx, runner, request.Inputs.ModelRoot, spec, &encoded); err != nil {
			return Artifact{}, err
		}
	case KindAttemptView:
		spec, ok := attemptSpecs[request.Variant]
		if !ok {
			return Artifact{}, fmt.Errorf("unknown attempt variant %q", request.Variant)
		}
		if err := exportAttemptViewWith(ctx, runner, request.Inputs.ModelRoot, spec,
			firstOrderSpecs[request.Variant], request.Variant, &encoded); err != nil {
			return Artifact{}, err
		}
	case KindVeilBinding:
		spec, ok := veilBindingSpecs[request.Variant]
		if !ok {
			return Artifact{}, fmt.Errorf("unknown Veil binding variant %q", request.Variant)
		}
		firstOrderVariant := map[string]string{"sound": "sound", "mutated": "mutated", "trusted": "sound"}[request.Variant]
		if err := exportVeilBindingWith(ctx, runner, request.Inputs.ModelRoot, spec,
			firstOrderSpecs[firstOrderVariant], &encoded); err != nil {
			return Artifact{}, err
		}
	case KindTemporalView:
		spec, ok := temporalSpecs[request.Variant]
		if !ok {
			return Artifact{}, fmt.Errorf("unknown temporal variant %q", request.Variant)
		}
		if err := exportTemporalViewWith(ctx, runner, request.Inputs.ModelRoot, spec, &encoded); err != nil {
			return Artifact{}, err
		}
	default:
		return Artifact{}, fmt.Errorf("unknown artifact %q", request.Kind)
	}
	return Artifact{Kind: request.Kind, Encoded: slices.Clone(encoded.Bytes())}, nil
}

func (g Generator) Check(ctx context.Context, request Request, expected []byte) error {
	artifact, err := g.Generate(ctx, request)
	if err != nil {
		return err
	}
	if !bytes.Equal(artifact.Encoded, expected) {
		return fmt.Errorf("generated %s artifact differs from checked output", request.Kind)
	}
	return nil
}
