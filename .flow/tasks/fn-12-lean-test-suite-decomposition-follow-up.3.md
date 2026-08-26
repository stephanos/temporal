---
satisfies: [R2, R3, R4, R5, R6, R7]
---
# fn-12-lean-test-suite-decomposition-follow-up.3 Split Umpire Property tests by concern

## Description
Split Property tests into evaluation, validation, logical-time, and canonicalization concerns behind the stable facade (R2-R7). Apply only the two approved assertion corrections for this suite.

**Size:** M
**Files:** `model/Umpire/Property/Tests.lean`; new `model/Umpire/Property/Tests/{Fixtures,Evaluation,Validation,LogicalTime,Canonicalization}.lean`
**Touches:** [model/Umpire/Property/Tests.lean, model/Umpire/Property/Tests/**]

## Approach
- Confirm the owned tree matches the fn-10 closure baseline, then map all 25 original assertions and attached comments to their destination, correction, or approved removal.
- Follow the approved layout at `docs/superpowers/specs/2026-08-25-lean-test-suite-structure-design.md:90-110`.
- Place only shared vocabulary, context, clauses, base property, positive trace, and result helpers in `Fixtures`.
- Make the negative uniqueness assertion evaluate `uniquenessProperty` itself, remove only the reflexive canonical self-comparison, and preserve the resulting 24 meaningful assertions exactly once.
- Keep the direct theorem proof and its explanatory comment unchanged, add a short module comment to every new file, and make the root directly import each concern.

## Investigation targets
**Required** (read before coding):
- `model/Umpire/Property/Tests.lean:1-204` — common fixtures and `uniquenessProperty`.
- `model/Umpire/Property/Tests.lean:205-388` — evaluation, theorem, visibility, validation, logical-time checks, and canonicalization fixtures preceding the final block.
- `model/Umpire/Property/Tests.lean:389-463` — canonicalization/digest checks and result evidence, including the reflexive comparison.
- `docs/superpowers/specs/2026-08-25-lean-test-suite-structure-design.md:90-110` — approved Property layout and corrections.

**Optional** (reference as needed):
- `model/Umpire/Property/Tests.lean:229-234` — direct theorem proof and attached comment.

## Key context
The focused uniqueness assertion must demonstrate failure by checking `uniquenessProperty`, not merely mention or reuse its fixture. This is a fresh-agent, serial current-branch task: stop for human direction on baseline drift, do not commit, and do not use a worktree.

## Acceptance
- [ ] `Property/Tests.lean` is import-only and directly imports `Evaluation`, `Validation`, `LogicalTime`, and `Canonicalization`; no fixtures or concern module imports the facade.
- [ ] The evidence map accounts for 25 originals: the negative uniqueness assertion now evaluates `uniquenessProperty`, the reflexive self-comparison alone is removed, and 24 meaningful assertions remain exactly once.
- [ ] Existing explanatory comments, including the direct theorem proof comment, remain verbatim and attached; all other semantic source strings are unchanged and every new file has a short module comment.
- [ ] `Fixtures` and every concern module pass direct Lean elaboration, then `cd model && mise exec -- lake build UmpireTests` passes.
- [ ] No production property behavior, public API, dependency, build target, documentation, generated file, commit, or worktree changes.

## Done summary
Split the reusable Property regression suite behind its stable import-only facade into shared fixtures plus Evaluation, Validation, LogicalTime, and Canonicalization concerns. The declaration map at `.flow/tmp/fn12-3-property-declaration-map.md` accounts for all 25 originals: 24 assertions remain exactly once, the reflexive comparison alone is removed, the negative case evaluates `uniquenessProperty`, and the theorem comment plus semantic strings remain unchanged.

stage: impl-review - ran [2026-08-26T03:03:15Z..2026-08-26T03:05:52Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 847545675cf52dd32e5fcd7b69564db7a2a238a2
- Tests: GATE_SKIPPED:build:green-receipt 31cd4043 - baseline reused from prior post-gate pass, GATE_SKIPPED:unittest:green-receipt 31cd4043 - baseline reused from prior post-gate pass, git diff --check, cd model && mise exec -- lake env lean Umpire/Property/Tests/Fixtures.lean, cd model && mise exec -- lake build Umpire.Property.Tests.Fixtures, cd model && mise exec -- lake env lean Umpire/Property/Tests/Evaluation.lean, cd model && mise exec -- lake env lean Umpire/Property/Tests/Validation.lean, cd model && mise exec -- lake env lean Umpire/Property/Tests/LogicalTime.lean, cd model && mise exec -- lake env lean Umpire/Property/Tests/Canonicalization.lean, cd model && mise exec -- lake build UmpireTests, Property assertion/comment/definition/semantic-string and import-boundary inventory checks, (cd model && mise exec -- lake build UmpireTests TemporalModelTests), make umpire-check-regression, git diff --check 8de97ffdc5bbedd195af434587f2d7b8fe18cead..HEAD
- PRs:
