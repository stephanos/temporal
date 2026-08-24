package checker

import (
	"io"
	"strings"

	"go.temporal.io/server/tools/umpire3/protocol/catalog"
	"go.temporal.io/server/tools/umpire3/protocol/experiment"
	"go.temporal.io/server/tools/umpire3/protocol/internal/codec"
)

const DefaultDecodeLimit = codec.DefaultDecodeLimit

type ActionKind = catalog.ActionKind
type EntityKind = catalog.EntityKind
type FaultKind = catalog.FaultKind
type PropertyID = catalog.PropertyID
type TargetID = catalog.TargetID
type Catalog = catalog.Catalog
type TargetProjection = catalog.TargetProjection
type ResolvedDeclaration = catalog.ResolvedDeclaration
type ResultClass = catalog.ResultClass
type TrustBadge = catalog.TrustBadge
type Experiment = experiment.Experiment
type ActionOutcome = experiment.ActionOutcome

const (
	ActionOutcomeApplied          = experiment.ActionOutcomeApplied
	ActionOutcomeSuppressed       = experiment.ActionOutcomeSuppressed
	ActionOutcomeRejected         = experiment.ActionOutcomeRejected
	ActionOutcomeRetried          = experiment.ActionOutcomeRetried
	ActionOutcomeFaultIntercepted = experiment.ActionOutcomeFaultIntercepted

	ResultClassTraceWitness             = catalog.ResultClassTraceWitness
	ResultClassSampledNoCounterexample  = catalog.ResultClassSampledNoCounterexample
	ResultClassBoundedSafe              = catalog.ResultClassBoundedSafe
	ResultClassFiniteExhaustive         = catalog.ResultClassFiniteExhaustive
	ResultClassExternalNoCounterexample = catalog.ResultClassExternalNoCounterexample
	ResultClassInvariantProved          = catalog.ResultClassInvariantProved
	ResultClassTemporalProved           = catalog.ResultClassTemporalProved
	ResultClassRefinementProved         = catalog.ResultClassRefinementProved
	ResultClassImplementationConforming = catalog.ResultClassImplementationConforming
	ResultClassMetadataValidated        = catalog.ResultClassMetadataValidated
	ResultClassUnknown                  = catalog.ResultClassUnknown

	TrustBadgeKernel                   = catalog.TrustBadgeKernel
	TrustBadgeKernelWithDeclaredAxioms = catalog.TrustBadgeKernelWithDeclaredAxioms
	TrustBadgeReconstructedSolverProof = catalog.TrustBadgeReconstructedSolverProof
	TrustBadgeTrustedSolver            = catalog.TrustBadgeTrustedSolver
	TrustBadgeCheckedCertificate       = catalog.TrustBadgeCheckedCertificate
	TrustBadgeTestedInstance           = catalog.TrustBadgeTestedInstance

	ActionKindAcquireOwnership     = catalog.ActionKindAcquireOwnership
	ActionKindDispatchTask         = catalog.ActionKindDispatchTask
	ActionKindPersistSuccess       = catalog.ActionKindPersistSuccess
	ActionKindProgressEntity       = catalog.ActionKindProgressEntity
	ActionKindRecoverOwner         = catalog.ActionKindRecoverOwner
	ActionKindWorkerReturnsSuccess = catalog.ActionKindWorkerReturnsSuccess

	EntityKindNexusOperation = catalog.EntityKindNexusOperation

	PropertyIDNexusCancellationWonExcludesSuccess = catalog.PropertyIDNexusCancellationWonExcludesSuccess
	TargetIDFoundationBacklogAck                  = catalog.TargetIDFoundationBacklogAck
	TargetIDNexusCancellation                     = catalog.TargetIDNexusCancellation
)

func DefaultCatalog() (catalog.Catalog, error) {
	return catalog.DefaultCatalog()
}

func DefaultComposition() (catalog.Composition, error) {
	return catalog.DefaultComposition()
}

func decodeStrictJSON(reader io.Reader, limit int64, kind string, destination any) error {
	return codec.DecodeStrictJSON(reader, limit, kind, destination)
}

func validHash(value string) bool {
	return codec.ValidHash(value)
}

func stringCompare(left, right string) int {
	return strings.Compare(left, right)
}

func compareStrings(left, right string) int {
	return strings.Compare(left, right)
}
