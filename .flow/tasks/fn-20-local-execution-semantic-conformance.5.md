---
satisfies: [R2, R3, R4, R5, R6, R8]
---
# fn-20-local-execution-semantic-conformance.5 Prove cross-layer fail-closed semantic interpretation

## Description
### Umpire4 reconciliation (normative)

The fail-closed matrix must independently mutate and classify raw evidence admission, System observation qualification, refinement correspondence/derivation, Feature trace identity, and Property evaluation. A valid runtime or SDK history replay cannot substitute for checked refinement.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Join the real Lean checker and Go controller with independent corruption/ambiguity oracles before exposing the command (R2-R6/R8).

**Size:** M
**Files:** `model/Temporal/Tool/ConformanceMutationTests.lean`, `model/TemporalModelTests.lean`, `tools/umpire/conformance/integration_test.go`, `tools/umpire/conformance/mutation_test.go`, `tools/umpire/conformance/testdata/**`
**Touches:** [model/Temporal/Tool/ConformanceMutationTests.lean, model/TemporalModelTests.lean, tools/umpire/conformance/integration_test.go, tools/umpire/conformance/mutation_test.go, tools/umpire/conformance/testdata/**]

### Approach
- Author literal expected qualification/verdict/status/identity outcomes independently of the checker and controller implementations.
- Mutate one layer at a time: artifact binding/bytes, request/response protocol, compiled semantic references, source schema/closure/order, causal/correlation edges, semantic duplicates/contradictions, dispositions, facts N/N+1, derivation bijection, query partition, status matrix, and qualified-outcome exclusions.
- Prove malformed artifact/protocol inputs produce no semantic artifacts, while valid partial/ambiguous/conflicting/unsupported evidence produces an admitted status-2-shaped semantic result.
- Permute facts where ordering is intentionally non-semantic and assert identical outputs; remove required ordering and assert unknown rather than a chosen trace.
- Run the actual fixed sibling checker through the Go bridge, including cancellation and repeated deterministic invocations.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Property/Tests/Evaluation.lean:9-96` — independent record-update mutation style
- `.flow/tasks/fn-4-umpire-observation-and-semantic-verdicts.5.md` — layer-specific mutation assurance
- `.flow/tasks/fn-18-versioned-umpire-artifact-boundary.6.md` — status/identity corruption cases
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.7.md` — source closure/capacity mutations
- `model/Temporal/Feature/Nexus/CallerClosureTests.lean` — current literal scenario expectations
## Acceptance
- [ ] Every R2-R6 invalid edge is diagnosed at its owning layer and no mutation oracle calls implementation logic under test.
- [ ] Valid partial, ambiguous, conflicting, unsupported, violated, and operationally failed/incomplete controls publishable-in-memory non-success Results rather than tooling errors.
- [ ] Required-field redaction/hash behavior cannot leak clear values or establish an unauthorized observation.
- [ ] Ordering permutations and missing-order cases prove deterministic semantics without first-match behavior.
- [ ] The real sibling integration is byte-deterministic, bounded, cancellable, and leaves no child or partial set.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
