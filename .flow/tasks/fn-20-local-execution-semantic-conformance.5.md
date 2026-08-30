---
satisfies: [R2, R3, R4, R5, R6, R8]
---

# fn-20-local-execution-semantic-conformance.5 Prove cross-layer fail-closed semantic interpretation

## Description
### Umpire4 reconciliation (normative)

The fail-closed matrix must independently mutate and classify raw evidence admission, System Observation Evaluation, Implementation Link correspondence/Evidence Link, Feature Model Trace identity, and Property evaluation. A valid runtime or SDK history replay cannot substitute for checked Implementation Link.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Join the real Lean checker and Go controller with independent corruption/ambiguity oracles before exposing the command (R2-R6/R8).

**Size:** M
**Files:** `model/Temporal/Tool/RunEvaluationMutationTests.lean`, `model/TemporalModelTests.lean`, `tools/umpire/runevaluation/integration_test.go`, `tools/umpire/runevaluation/mutation_test.go`, `tools/umpire/runevaluation/testdata/**`
**Touches:** [model/Temporal/Tool/RunEvaluationMutationTests.lean, model/TemporalModelTests.lean, tools/umpire/runevaluation/integration_test.go, tools/umpire/runevaluation/mutation_test.go, tools/umpire/runevaluation/testdata/**]

### Approach
- Author literal expected Observation Evaluation/verdict/status/identity outcomes independently of the checker and controller implementations.
- Mutate one layer at a time: artifact binding/bytes, request/response protocol, compiled semantic references, source schema/closure/order, causal/correlation edges, semantic duplicates/contradictions, dispositions, facts N/N+1, Evidence Link bijection, query partition, status matrix, and accepted-outcome exclusions.
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
Implemented independent fail-closed mutation oracles from raw admission through System Observation, checked Implementation/Evidence Links, Feature trace identity, and Property evaluation, with deterministic real-sibling execution, cancellation, partial-result admission, and exact canonical wire closure. Split runtime configuration identities from checked mapping/profile authority, pinned both checked bindings, preserved raw origin ordinals and nonaccepted Results, and regenerated only dependent canonical fixtures.

Verification passed for focused Go/Lean suites, artifact cross-language closure, race, fuzz, scoped Go lint/errortype, Lean lint, and the full Umpire regression; repository-wide `lint-code` remains inherited-red only at `tools/umpire/runtime/errors.go:60:9` (`et:unw+`).

stage: impl-review - ran [2026-08-30T04:52:03Z..2026-08-30T05:04:21Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 0c6f1024be33c836dcd46cddac7f15b35753e142, a7b041cef14044475d3ac15295c20d2289468c33, c31e6ba4ba77451239e683750fcae8cd38857c32
- Tests: cd model && mise exec -- lake build Umpire.Observation.Tests.Check, cd model && mise exec -- lake build Temporal.Tool.RunEvaluationTests temporal-run-evaluation-checker, go test -count=1 ./tools/umpire/runevaluation/..., go test -tags test_dep -count=1 ./tools/umpire/runevaluation/... ./tools/umpire/artifact/..., go test -tags test_dep -race -count=1 ./tools/umpire/runevaluation, go test -tags test_dep -run '^$' -fuzz '^FuzzDecodeCheckerResponse$' -fuzztime=2s ./tools/umpire/runevaluation, cd model && mise exec -- lake build Umpire.Artifact.Tests.Goldens Umpire.Artifact.Tests.Set Umpire.Artifact.Tests.Result Temporal.Tool.RunEvaluationMutationTests Temporal.Tool.RunEvaluationTests temporal-run-evaluation-checker TemporalModelTests, make lint-model, .bin/golangci-lint-v2.13.1 run --build-tags disable_grpc_modules,,test_dep, --timeout 10m --fix=false --new-from-rev=b5e87c26eab2bb05236ea75a9b0256c032e2f94f --config=.github/.golangci.yml ./tools/umpire/artifact/... ./tools/umpire/internal/artifactv2/... ./tools/umpire/runevaluation/..., .bin/errortype -style-check=false ./tools/umpire/artifact/... ./tools/umpire/internal/artifactv2/... ./tools/umpire/runevaluation/..., make umpire-check-regression, TDD_RED: go test -tags test_dep -count=1 -run '^TestCheckerResponseRejectsConsistentCheckedProfileDriftAtTheProtocolBoundary$' ./tools/umpire/runevaluation (failed at construction before the checked-profile protocol binding), INHERITED_RED: make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=b5e87c26eab2bb05236ea75a9b0256c032e2f94f lint-code (tools/umpire/runtime/errors.go:60:9 et:unw+)
- PRs:
