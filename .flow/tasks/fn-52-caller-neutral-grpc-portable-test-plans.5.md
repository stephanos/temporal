---
satisfies: [R3, R7, R10]
---
# fn-52-caller-neutral-grpc-portable-test-plans.5 Compile Lean tests into portable plans and prove parity

## Description
Make Lean a deterministic producer of the same typed plan accepted from external clients, retain exact existing model identities, and prove cross-language evaluation parity for R3 and R7.

**Size:** M
**Files:** `model/Umpire/Artifact/PortableEvaluationContract.lean`, `model/Temporal/Tool/PortableEvaluationContract.lean`, related Lean tests, generated portable fixtures, Go parity tests
**Touches:** [model/Umpire/Artifact/PortableEvaluationContract.lean, model/Temporal/Tool/PortableEvaluationContract.lean, model/Temporal/Tool/*PortableEvaluationContract*Tests.lean, tools/umpire/portableevaluation/testdata/**, tools/umpire/portableevaluation/*test.go]

### Approach
- Extend the existing closed Lean lowering rather than create another Property, Behavior, Query, or execution language.
- Emit the complete typed execution and verification programs plus exact ExperimentSpec/model bindings and compiler provenance.
- Lower every supported check to the shared finite vocabulary; emit explicit required/advisory external obligations for unsupported checks.
- Preserve existing ExperimentSpec checksums and fn-28 fixtures while adding separately versioned plan fixtures.
- Compare stable detailed Lean outcomes with Go results for every operator, branch, scope, obligation, and N/N+1 boundary.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Artifact/PortableEvaluationContract.lean:265-360` — reusable Lean contract data
- `model/Temporal/Tool/PortableEvaluationContract.lean:609-760` — current checked-Test lowering
- `model/Temporal/Tool/PortableEvaluationContractTests.lean` — lowering and failure fixtures
- `tools/umpire/portableevaluation/parity_test.go` — cross-language parity harness
- `tools/umpire/portableevaluation/testdata/**` — current generated contract/evidence fixtures

### Acceptance
- [ ] Lean emits deterministic typed plans accepted by the same path as external plans.
- [ ] Existing ExperimentSpec identities and fn-28 fixtures remain unchanged.
- [ ] Supported execution/verification operators and every result branch have Lean/Go parity.
- [ ] Unsupported checks are explicit obligations and cannot silently affect portable or model-bound success.
- [ ] Model bindings, compiler provenance, source locations, checksums, and obligation mutations reject deterministically.
- [ ] Focused Lean builds and Go parity tests pass.

## Acceptance
- [ ] R3 and R7 Lean production, identity retention, obligations, and parity are complete.
- [ ] Existing model/artifact identities remain stable.
- [ ] Focused Lean and Go tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
