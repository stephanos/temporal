---
satisfies: [R5, R6, R8, R9]
---
# fn-78-typed-temporal-authoring-and-checked.7 Lower correlated obligations into the portable Contract

## Description
Nexus operation cancellation is deferred to fn-79. Implement and qualify this generic task with non-cancellation, multi-operation fixtures; no cancellation adapter or capability is a prerequisite.

Compile the supported correlated-obligation fragment into a closed, versioned Testpilot Contract capability and evaluate it through the generic Go runtime. The portable encoding must preserve the checked clock, key, endpoint, limits, projection identity, and clause provenance without embedding Lean callbacks or Temporal/Nexus branches.

**Size:** L
**Files:** `model/lakefile.lean` for the focused correlated-fixture renderer executable in the existing conformance generation route, `model/Umpire/Case.lean` for default-empty correlated provenance metadata, `model/Shared/SemanticData.lean`, `model/Umpire/Core.lean`, and `model/Umpire/Evidence/Evaluate/Types.lean` for inert shared representations with preserved Umpire facades, `model/Shared/Correlated{Projection,Obligation}.lean`, `model/Umpire/Case/Projection/Declaration.lean`, `model/Umpire/Property/Correlated/{Kernel,Reference}.lean` for the shared data-table kernel and preserved facades, `model/Umpire/Case/Projection.lean` and `model/Umpire/Property/Correlated.lean` for checked declaration/semantic accessors, `model/Umpire/Case/Correlated*.lean` for portable lowering and correspondence, `proto/internal/temporal/server/api/testpilot/v1/{contract,run}.proto`, generated Testpilot Go/Lean protocol files, `model/Testpilot/{Authoring,ProtoJSON}.lean`, `model/Umpire/Case/{Compiler,CompilerTests}.lean and model/Umpire/Provenance.lean`, `common/testing/testpilot/internal/{verification,execution}/**`, protocol/conformance fixtures and tests, `Makefile`
**Touches:** [model/lakefile.lean, model/Umpire/Case.lean, model/Shared/SemanticData.lean, model/Umpire/Core.lean, model/Umpire/Evidence/Evaluate/Types.lean, model/Shared/CorrelatedProjection.lean, model/Shared/CorrelatedObligation.lean, model/Umpire/Case/Projection/Declaration.lean, model/Umpire/Property/Correlated/Kernel.lean, model/Umpire/Property/Correlated/Reference.lean, model/Umpire/Case/Projection.lean, model/Umpire/Property/Correlated.lean, proto/internal/temporal/server/api/testpilot/v1/**, api/testpilot/v1/**, model/Testpilot/**, model/Umpire/Case/**, common/testing/testpilot/internal/verification/**, common/testing/testpilot/internal/execution/**, common/testing/testpilot/testdata/case-runtime-conformance/**, tests/testcore/testpilot/**, tools/umpire/cmd/umpire-gen-case-runtime-conformance/**, Makefile]

### Approach
- Share only inert identifier/value/transition-result/evidence-scalar representations needed by both interpreters. Keep Umpire validation, meanings, metadata, fingerprints, and authoring policy in Umpire; preserve facade constructors, projections, record syntax, and existing comments. Avoid broad caller rewrites.
- Extract the projection and countdown algorithms into neutral `Shared` modules consumed by both Umpire and Testpilot. Keep Shared independent of either consumer; preserve existing Umpire interfaces, comments, and semantics. Umpire derives complete admitted finite transition tables and proves row authority against the checked Target; Testpilot decodes and executes only table data, without serialized callbacks. Reject unsupported finite-domain lowering rather than silently omitting reachable transitions.
- Consume task 4 generic projection and task 6 semantics directly. Former dependencies on tasks 5 and 9 now refer only to scope-transfer records; no cancellation implementation is required.
- Add one closed correlated-obligation Contract capability version inside exact Case 1.0; preserve existing Contract rules and unchanged artifacts byte-for-byte.
- Reject unknown capability versions, enums, fields, unsupported clauses, overflow, and limit combinations during static Prepare before any Driver I/O.
- Generate the portable monitor from the checked clause/projection declarations and implement Go evaluation against the same transition contract, including inclusive semantic deadlines despite expiry-before-transition processing.
- Prove successful lowering and execution of the actual encoded Contract correspond to task 6's checked obligation transitions over admitted projected steps, including inclusive deadlines and both close modes; use fixtures only as supplementary evidence.
- Keep prepared definitions immutable and create fresh bounded evaluator state per Run. Preserve earlier violations across incomplete execution, evaluator failure, and cleanup disposition.
- Regenerate through established proto/Lean tooling and use focused fixture staleness checks; do not add broad generated API drift or CI gates.

### Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/testpilot/v1/contract.proto` — current finite Contract schema
- `model/Testpilot/Authoring.lean:183-330` — closed authoring constructors
- `model/Umpire/Case/Compiler.lean` — checked Case lowering and overflow boundary
- `common/testing/testpilot/internal/verification/prepare.go` — static Contract admission
- `common/testing/testpilot/internal/verification/evaluator.go` — run-local staged evaluation

### Key context
- ProtoJSON is not intrinsically canonical; preserve this repository's owned canonical policy and fixtures.
- Work exhaustion makes execution incomplete and cannot erase an already proved violation.
## Acceptance
- [ ] Checked correlated rules lower to a closed versioned Testpilot capability carrying exact key, clock, bound, endpoint, projection, limits, and Umpire provenance.
- [ ] Unknown/stale capability versions, unknown semantic enums/fields, unsupported formulas, overflow, malformed limits, and incompatible Program/Contract inputs reject during Prepare before Driver I/O.
- [ ] Existing Case 1.0 artifacts and unchanged Contract rules retain exact bytes, IDs, and fingerprints; any changed semantic artifact has an explicit migration fixture and stale-reader rejection.
- [ ] Lean and Go evaluators agree with independently derived fixtures for bound-zero, trigger, deadline, late response, multiple obligations, interleaved operations, self-loops, closed traces, and incomplete prefixes.
- [ ] Checked theorems connect successful lowering and actual portable Contract execution to task 6's obligation semantics over task 4's admitted projection, including inclusive deadlines and deliberate versus incomplete close.
- [ ] Expiry-before-transition runtime ordering still accepts an eligible response at the inclusive semantic deadline and never treats wall time as an operation-transition horizon.
- [ ] Every Run gets isolated buffers, obligations, captures, counters, and immutable results; buffer/work/capture/support exhaustion fails closed without automatic retry or manufactured deadline.
- [ ] Incremental and offline Run evaluation agree across causal-evidence chunk boundaries and preserve a prior violation through later projection/evaluator/cleanup failure.
- [ ] Proto/Lean generation, focused staleness/conformance fixtures, affected Go tests with `-tags test_dep`, and focused lints pass without new third-party dependencies.
## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: