package catalog

type ResultClass string

const (
	ResultClassTraceWitness             ResultClass = "trace-witness"
	ResultClassLassoWitness             ResultClass = "lasso-witness"
	ResultClassSampledNoCounterexample  ResultClass = "sampled-no-counterexample"
	ResultClassBoundedSafe              ResultClass = "bounded-safe"
	ResultClassFiniteExhaustive         ResultClass = "finite-exhaustive"
	ResultClassExternalNoCounterexample ResultClass = "external-no-counterexample"
	ResultClassInvariantProved          ResultClass = "invariant-proved"
	ResultClassTemporalProved           ResultClass = "temporal-proved"
	ResultClassRefinementProved         ResultClass = "refinement-proved"
	ResultClassCompositionProved        ResultClass = "composition-proved"
	ResultClassEvidenceResolved         ResultClass = "evidence-resolved"
	ResultClassImplementationConforming ResultClass = "implementation-conforming"
	ResultClassMetadataValidated        ResultClass = "metadata-validated"
	ResultClassUnknown                  ResultClass = "unknown"
)

func (r ResultClass) valid() bool {
	return r.Valid()
}

func (r ResultClass) Valid() bool {
	switch r {
	case ResultClassTraceWitness, ResultClassLassoWitness, ResultClassSampledNoCounterexample, ResultClassBoundedSafe,
		ResultClassFiniteExhaustive, ResultClassExternalNoCounterexample, ResultClassInvariantProved,
		ResultClassTemporalProved, ResultClassRefinementProved, ResultClassCompositionProved,
		ResultClassEvidenceResolved, ResultClassImplementationConforming,
		ResultClassMetadataValidated, ResultClassUnknown:
		return true
	default:
		return false
	}
}

type TrustBadge string

const (
	TrustBadgeKernel                   TrustBadge = "kernel"
	TrustBadgeKernelWithDeclaredAxioms TrustBadge = "kernel-with-declared-axioms"
	TrustBadgeReconstructedSolverProof TrustBadge = "reconstructed-solver-proof"
	TrustBadgeTrustedSolver            TrustBadge = "trusted-solver"
	TrustBadgeCheckedCertificate       TrustBadge = "checked-certificate"
	TrustBadgeExternalTool             TrustBadge = "external-tool"
	TrustBadgeTestedInstance           TrustBadge = "tested-instance"
	TrustBadgeSampled                  TrustBadge = "sampled"
	TrustBadgeHeuristic                TrustBadge = "heuristic"
)

func (b TrustBadge) valid() bool {
	return b.Valid()
}

func (b TrustBadge) Valid() bool {
	switch b {
	case TrustBadgeKernel, TrustBadgeKernelWithDeclaredAxioms, TrustBadgeReconstructedSolverProof,
		TrustBadgeTrustedSolver, TrustBadgeCheckedCertificate, TrustBadgeExternalTool,
		TrustBadgeTestedInstance, TrustBadgeSampled, TrustBadgeHeuristic:
		return true
	default:
		return false
	}
}

type MetadataStatus string

const (
	MetadataPresent MetadataStatus = "metadata-present"
	MetadataMissing MetadataStatus = "metadata-missing"
)
