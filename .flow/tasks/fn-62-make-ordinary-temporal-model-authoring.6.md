---
satisfies: [R1, R7, R8]
---
# fn-62-make-ordinary-temporal-model-authoring.6 Carry model-owned Known Gaps end to end

## Description
Implement R1, R7, and R8 by placing optional checked model-owned Known Gaps at the narrowest established authoring boundary and carrying every declared gap deterministically through planning, portable lowering, Run Evaluation, and results.

**Size:** M
**Files:** `model/Umpire/Query/Language.lean`, `model/Umpire/Artifact/Planning.lean`, `model/Umpire/Planning/Tests/KnownGaps.lean`, `model/Temporal/Tool/PortableEvaluationContractTests.lean`, `model/Temporal/Tool/RunEvaluationTests.lean`
**Touches:** [model/Umpire/Query/Language.lean, model/Umpire/Artifact/Planning.lean, model/Umpire/Planning/Tests/KnownGaps.lean, model/Temporal/Tool/PortableEvaluationContractTests.lean, model/Temporal/Tool/RunEvaluationTests.lean]

### Approach
- Trace the existing `KnownGap`, `KnownGapSet`, canonical planner gaps, portable compiler, Run Evaluation, and result path; place optional authored gaps on the narrowest checked Query-owned seam that reaches `ExperimentSpec` without becoming behavior.
- Reuse `KnownGapSet.ofUnordered`, `checkCanonical`, `union`, and `toList` in `model/Umpire/Planning/Types.lean:8-148`; do not add another gap vocabulary, required-gap inference, category-specific binding scheme, or mutable registry.
- Define an explicit empty set as no authored gaps. Carry each declared row exactly once, union it deterministically with phase-owned gaps, and preserve its kind, code, optional subject, detail, source identity, and cardinality.
- Add one generic model-authored gap fixture that is compiled into a portable contract and reaches `ResultArtifact` through the normal Run Evaluation path; do not invent unsupported Temporal behavior solely to populate it.
- Keep unknown wire categories in the strict decoder boundary and existing `KnownGapError` precedence for malformed identifiers, duplicates, conflicting details, and noncanonical external order.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Planning/Types.lean:8-148` — exact optional row schema, error vocabulary, and canonical set operations.
- `model/Umpire/Artifact/Planning.lean:300-350` — planning artifact construction and current phase-owned gap injection.
- `model/Temporal/Tool/PortableEvaluationContract.lean` — lowering from `ExperimentSpec` Known Gaps into the portable contract.
- `model/Temporal/Tool/RunEvaluation.lean` — request/Run gap aggregation into `ResultArtifact`.
- `model/Temporal/Tool/RunEvaluationTests.lean:595-734` — existing raw/conflicting gap and result assertions.

**Optional** (reference as needed):
- `model/Umpire/Planning/Tests/KnownGaps.lean` — deterministic planner-gap regressions.
- `.flow/memory/bug/integration/portable-model-plans-need-exact-2026-09-03.md:16-25` — exact artifact and explicit-obligation constraint.

### Acceptance
- [ ] Model authors can optionally declare capability/input/interpretation/claim gaps as checked existing Known Gap rows; `KnownGapSet.empty` remains a valid explicit declaration of no gaps.
- [ ] Every declared gap reaches checked Query, plan, `ExperimentSpec`, portable contract, Run Evaluation, and `ResultArtifact` exactly once alongside deterministic phase-owned unions.
- [ ] Malformed codes/subjects, duplicates, conflicting details, and noncanonical external order retain existing `KnownGapError` failures; unknown wire categories retain strict decoder rejection.
- [ ] No checker claims it can infer an omitted real-world limitation or category-specific binding not represented by `KnownGap`; truthful completeness remains the author's responsibility.
- [ ] A declared gap never establishes success, changes behavior, or disappears from partial/failure result output.
## Acceptance
- [ ] R1, R7, and R8 are satisfied with one optional checked Known Gap vocabulary and no inferred required-gap mechanism.
- [ ] `cd model && mise exec -- lake build Umpire.Planning.Tests Temporal.Tool.PortableEvaluationContractTests Temporal.Tool.RunEvaluationTests` passes.
- [ ] `make umpire-check-portable-evaluation-fixtures` passes with no unexplained portable/result gap delta.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
