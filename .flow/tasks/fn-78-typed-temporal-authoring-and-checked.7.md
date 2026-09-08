---
satisfies: [R5, R6, R8, R9]
---
# fn-78-typed-temporal-authoring-and-checked.7 Lower scoped obligations into the portable Contract

## Description
Compile the supported scoped-obligation fragment into a closed, versioned Testpilot Contract capability and evaluate it through the generic Go runtime. The portable encoding must preserve the checked clock, key, endpoint, limits, projection identity, and clause provenance without embedding Lean callbacks or Temporal/Nexus branches.

**Size:** L
**Files:** `proto/internal/temporal/server/api/testpilot/v1/{contract,run}.proto`, generated Testpilot Go/Lean protocol files, `model/Testpilot/{Authoring,ProtoJSON}.lean`, `model/Umpire/Case/{Compiler,Provenance,CompilerTests}.lean`, `common/testing/testpilot/internal/{verification,execution}/**`, protocol/conformance fixtures and tests, `Makefile`
**Touches:** [proto/internal/temporal/server/api/testpilot/v1/**, api/testpilot/v1/**, model/Testpilot/**, model/Umpire/Case/**, common/testing/testpilot/internal/verification/**, common/testing/testpilot/internal/execution/**, common/testing/testpilot/testdata/case-runtime-conformance/**, tests/testcore/testpilot/**, tools/umpire/cmd/umpire-gen-case-runtime-conformance/**, Makefile]

### Approach
- Add one closed scoped-obligation Contract capability version inside exact Case 1.0; preserve existing Contract rules and unchanged artifacts byte-for-byte.
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
- [ ] Checked scoped clauses lower to a closed versioned Testpilot capability carrying exact key, clock, bound, endpoint, projection, limits, and Umpire provenance.
- [ ] Unknown/stale capability versions, unknown semantic enums/fields, unsupported formulas, overflow, malformed limits, and incompatible Program/Contract inputs reject during Prepare before Driver I/O.
- [ ] Existing Case 1.0 artifacts and unchanged Contract rules retain exact bytes, IDs, and fingerprints; any changed semantic artifact has an explicit migration fixture and stale-reader rejection.
- [ ] Lean and Go evaluators agree with independently derived fixtures for bound-zero, trigger, deadline, late response, multiple obligations, interleaved operations, self-loops, closed traces, and incomplete prefixes.
- [ ] Checked theorems connect successful lowering and actual portable Contract execution to task 6's obligation semantics over task 4's admitted projection, including inclusive deadlines and deliberate versus incomplete close.
- [ ] Expiry-before-transition runtime ordering still accepts an eligible response at the inclusive semantic deadline and never treats wall time as an operation-transition horizon.
- [ ] Every Run gets isolated buffers, obligations, captures, counters, and immutable results; buffer/work/capture/support exhaustion fails closed without automatic retry or manufactured deadline.
- [ ] Incremental and offline Run evaluation agree across causal-evidence chunk boundaries and preserve a prior violation through later projection/evaluator/cleanup failure.
- [ ] Proto/Lean generation, focused staleness/conformance fixtures, affected Go tests with `-tags test_dep`, and scoped lints pass without new third-party dependencies.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
