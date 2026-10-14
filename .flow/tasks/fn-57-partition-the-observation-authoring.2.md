---
satisfies: [R2, R3, R4, R5, R6]
---
# fn-57-partition-the-observation-authoring.2 Extract the Observation compiler behind Language

## Description
Move the check context, Target meaning resolution, typed diagnostics, checked expression and plan contracts, canonical identity machinery, and checker pipeline intact into Compiler. Keep Language as the stable aggregation and explicit-proof authoring seam.

**Size:** M
**Files:** `model/Umpire/Observation/Compiler.lean`, `model/Umpire/Observation/Language.lean`, `model/Umpire/Observation/Evaluation.lean`, `model/Umpire/Observation/Tests/Compilation.lean`, `model/Umpire/Observation/ImportTests.lean`
**Touches:** [model/Umpire/Observation/Compiler.lean, model/Umpire/Observation/Language.lean, model/Umpire/Observation/Evaluation.lean, model/Umpire/Observation/Tests/Compilation.lean, model/Umpire/Observation/ImportTests.lean]

### Approach
- Move `ObservationCheckContext`, resolved Target meaning selection, `ObservationError`, checked expression/plan contracts, canonical rendering, fingerprint construction, and `checkObservation` into Compiler without reordering checks.
- Keep canonical sorting, identity, and private validation helpers single-owned and private to Compiler; do not create another checked-plan or helper module.
- Leave `checkedObservation` in Language as the explicit-proof convenience over the same `checkObservation`.
- Narrow Evaluation's import to Compiler while retaining the checked-plan identity operations it consumes.
- Extend import checks only where needed to pin raw checking, proof-taking checking, errors, checked plans, and complete facade visibility.
- Preserve every existing comment, source fallback, diagnostic field, and first-failure precedence.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Language.lean:192-445` — context, error vocabulary, and checked contracts
- `model/Umpire/Observation/Language.lean:447-705` — canonical identity and exact rendering
- `model/Umpire/Observation/Language.lean:707-1105` — deterministic checking pipeline
- `model/Umpire/Observation/Tests/Compilation.lean:34-117` — proof-taking authoring, identity, typed plans, and reconciliation
- `model/Umpire/Observation/Tests/Compilation.lean:238-453` — failure matrix and exact diagnostics
- `model/Umpire/Shared/DefinitionGraph.lean:147-205` — structural checking contract
- `model/Umpire/Observation/Evaluation.lean:2052-2124` — checked-plan consumer and entry point

### Key context
- Preserve checker statement order; diagnostic precedence and exact related-identity order are contracts.
- `checkedObservation` must not gain hidden native evaluation, an unchecked constructor, or a recovery path.
- Fn-54 will decompose Evaluation afterward; this task only narrows its single upstream import.

## Acceptance
- [ ] R2-R4 are satisfied by the same `checkObservation` and `checkedObservation` interfaces with no partial checked plan, hidden evaluation, bypass, or trust change.
- [ ] Every existing error kind retains exact kind, definition ID, source fallback, offending value, canonical related identities, rendered form, and first-failure precedence.
- [ ] Agreeing and conflicting provider meaning, explicit connector reconciliation, unauthorized semantics, equivalent reordering, and behavior-affecting fingerprint changes remain exact.
- [ ] Compiler owns canonical ordering and fingerprint machinery once; no duplicate helper or shallow checked-plan module is introduced.
- [ ] Evaluation imports Compiler directly, and Language, Observation, and Umpire facade tests retain the complete public surface.
- [ ] Existing comments are preserved.
- [ ] `cd model && mise exec -- lake build Umpire.Observation.Tests Umpire.Observation.ImportTests Temporal.Feature.Nexus.ObservationTests Temporal.System.Nexus.Tests` passes.

## Done summary
Moved Observation checking, Target meaning resolution, typed diagnostics, checked contracts, canonical identity, and fingerprint construction intact into Compiler; retained explicit-proof checkedObservation in Language and narrowed Evaluation to Compiler.
## Evidence
- Commits: bd4609a34
- Tests: cd model && mise exec -- lake build Umpire.Observation.Tests Umpire.Observation.ImportTests Temporal.Feature.Nexus.ObservationTests Temporal.System.Nexus.Tests, cd model && mise exec -- lake build UmpireTests TemporalModelTests, git diff --check
- PRs: