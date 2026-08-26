---
satisfies: [R2, R3, R4, R5, R6, R7]
---
# fn-12-lean-test-suite-decomposition-follow-up.1 Split Umpire Core tests by concern

## Description
Split the largest reusable suite into the approved fixtures and concern modules (R2-R7). Keep the existing root as the stable import-only facade; this task owns only the Core test tree.

**Size:** M
**Files:** `model/Umpire/CoreTests.lean`; new `model/Umpire/CoreTests/{Fixtures,Composition,Validation,KernelSoundness,Canonicalization,Trace}.lean`
**Touches:** [model/Umpire/CoreTests.lean, model/Umpire/CoreTests/**]

## Approach
- Confirm the owned tree matches the fn-10 closure baseline, then record a declaration/comment-to-destination map for all 31 assertions before moving anything.
- Follow the approved concern ownership at `docs/superpowers/specs/2026-08-25-lean-test-suite-structure-design.md:47-67`.
- Put only vocabulary shared across multiple concerns in `Fixtures`; keep derived variants with their consumer.
- Give every new module a short module comment, move existing explanatory comments verbatim, and replace the root body with direct imports for every concern leaf.
- Ensure fixtures and concern leaves import public production modules or sibling fixtures rather than the test facade; preserve both serializer-availability checks and all semantic source strings exactly.

## Investigation targets
**Required** (read before coding):
- `model/Umpire/CoreTests.lean:1-142` — shared constructors, kernels, providers, connector, and baseline target to classify into fixtures.
- `model/Umpire/CoreTests.lean:143-388` — composition, validation, kernel-soundness, canonicalization fixtures, and the assertion preceding the canonicalization block.
- `model/Umpire/CoreTests.lean:389-554` — canonicalization, digest, documentation, serializer, and trace assertions.
- `docs/superpowers/specs/2026-08-25-lean-test-suite-structure-design.md:47-67` — approved Core layout and ownership.

**Optional** (reference as needed):
- `model/Umpire/CoreTests.lean:189` — attached explanatory comment to preserve with its assertion.
- `model/Umpire/CoreTests.lean:366` — attached explanatory comment to preserve with its assertion.

## Key context
Lean modules are compilation and visibility boundaries. Validate every new fixtures and concern module directly before building the unchanged aggregate. This is a fresh-agent, serial current-branch task: stop for human direction on baseline drift, do not commit, and do not use a worktree.

## Acceptance
- [ ] `CoreTests.lean` is import-only and directly imports `Composition`, `Validation`, `KernelSoundness`, `Canonicalization`, and `Trace`; no fixtures or concern module imports the facade.
- [ ] A declaration-level evidence map accounts for all 31 Core assertions and existing explanatory comments exactly once, including both serializer-availability checks; semantic fixture strings are unchanged.
- [ ] Shared fixtures contain only vocabulary used by multiple concern modules, while consumer-specific variants remain local; every new file has a short module comment.
- [ ] `Fixtures` and every concern module pass direct Lean elaboration, then `cd model && mise exec -- lake build UmpireTests` passes.
- [ ] No production Umpire module, public API, dependency, build target, documentation, generated file, commit, or worktree is introduced.

## Done summary
Split the reusable Core regression suite behind its stable import-only facade into shared fixtures plus Composition, Validation, KernelSoundness, Canonicalization, and Trace concerns. The pre-move evidence map at `.flow/tmp/fn12-1-core-declaration-map.md` assigns all 31 assertions and both explanatory comments exactly once (2/12/3/13/1 assertions by concern), while definition, comment, and semantic-string comparisons remain identical to the fn-10 closure baseline.

stage: impl-review - ran [2026-08-26T02:28:59Z..2026-08-26T02:33:43Z]
## Evidence
- Commits: a92a9d40922cfafc6ec04abe378eb7fd36d955c5
- Tests: baseline: green ((cd model && mise exec -- lake build UmpireTests TemporalModelTests); make umpire-check-regression; git diff --check), cd model && mise exec -- lake env lean Umpire/CoreTests/Fixtures.lean, cd model && mise exec -- lake env lean Umpire/CoreTests/Composition.lean, cd model && mise exec -- lake env lean Umpire/CoreTests/Validation.lean, cd model && mise exec -- lake env lean Umpire/CoreTests/KernelSoundness.lean, cd model && mise exec -- lake env lean Umpire/CoreTests/Canonicalization.lean, cd model && mise exec -- lake env lean Umpire/CoreTests/Trace.lean, cd model && mise exec -- lake build UmpireTests, Core assertion/comment/definition/semantic-string and import-boundary inventory checks, (cd model && mise exec -- lake build UmpireTests TemporalModelTests), make umpire-check-regression, git diff --check 33f36d8e93e69ad666b008129cc3ebf892f196ad..HEAD
- PRs: