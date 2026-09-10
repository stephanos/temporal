---
satisfies: [R2]
---
# fn-80-close-the-model-to-case-seam-and-harden.5 Generalize the Nexus3 model, property, behavior, and limits macros

## Description
Implements R2 (spec §R2). Replaces the spelling whitelists in the five-block syntax with elaboration over parsed identifiers, deriving `FiniteTable` domains and enumerators from enum-like inductive constructors, and proves it on a second lifecycle shaped like the Nexus2 cancellation race. The `all` query form is task .6.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus3/Syntax.lean`, `model/Temporal/Feature/Nexus3/Authoring.lean`, `model/Temporal/Feature/Nexus3/Tests.lean`, new `model/Temporal/Feature/Nexus3/RaceSyntaxTests.lean` (second lifecycle), `model/Temporal/Feature/Nexus3/Nexus.md`
**Touches:** [model/Temporal/Feature/Nexus3/**]

### Approach
- Whitelists to delete: `Syntax.lean:22-27` (model), `:76-80` (property), `:97-100` (behavior), `:115-117` (limits). The hard-coded `mkIdentFrom` block at `:42-51` becomes constructor-driven.
- Derive constructors with `getConstInfoInduct` in a `Lean.Elab.Command` elaborator (first such derivation in the repo; no existing helper). Reject a constructor with arguments with a located error.
- Target the existing owners: `SuccessModel`/`successTable`/`successModel` at `Authoring.lean:99-197` generalize to an N-transition table feeding `FiniteTable` (`Umpire/Target/FiniteTable.lean:17-137`: `FiniteCatalogEntry`, `validKey`, `encode?`, `validate`). Discharge enumeration completeness, domain membership, and Action executability by `decide`/`rfl` on enum-like inductives.
- Reachability: a terminal state must be reachable from some initial state over the transition table; decide it and report a located error.
- Located errors follow `Umpire/Property/Authoring.lean:259-285` (`throwErrorAt`).
- Second lifecycle: mirror `Temporal/Feature/Nexus2/Race.lean:36-58` (four states) in a new test module using only the five commands; do not import Race.
- Add `#guard_msgs` tests for each error class; keep the tested scale at ≤16 states.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus3/Syntax.lean` — all five macros
- `model/Temporal/Feature/Nexus3/Authoring.lean:36-197,274-422` — model record, table, vocabulary, check
- `model/Umpire/Target/FiniteTable.lean:17-137` — catalog and validation the derivation must satisfy
- `model/Umpire/Property/Authoring.lean:200-290` — located diagnostic path

**Optional** (reference as needed):
- `model/Temporal/Feature/Nexus2/Race.lean:22-60` — second lifecycle shape
- `.plans/LEAN_GUIDELINES.md:83-115` — legible proofs

### Key context
- AUT-08 says authors provide ordered domains and enumerators; task .9 drafts the rule text that macro-derived domains count as author-provided.
- fn-67.2 edits `Nexus.md` prose only; coordinate wording rather than fixtures.

## Acceptance
- [ ] The three `unsupported Nexus3 ... spelling` error strings no longer exist in `model/`
- [ ] `RaceSyntaxTests.lean` declares a lifecycle with a different role name, four states, three transitions, a two-clause `property`, a `behavior`, `limits`, and a witness `query` that checks
- [ ] `#guard_msgs` covers: constructor with arguments, unknown constructor, duplicate `before + action`, unreachable terminal, transition count over 256
- [ ] The existing Nexus3 success model still elaborates and `produce` (task .4) yields byte-identical fixture output; `make umpire-check-case-runtime-conformance` passes
- [ ] `lake build Temporal TemporalModelTests` and `make lint-model` pass; axiom inventory unchanged

## Done summary
Closed as SUPERSEDED, not implemented. No code was ever written against this task.

This task escalated `BLOCKED: SCOPE_EXCEEDED`: it was sized M and read as "delete four spelling
whitelists", but the elaborators expand into `Authoring.successModel`, whose data model was
arity-fixed at three states, two actions, two outcomes, two facts and two transitions — a shape that
was load-bearing across ~30 reads in `Tests.lean` (several inside `native_decide` theorems) and in
`Testpilot.lean`. Its block recorded the measured blast radius with line numbers and recommended a
four-step sequence that could each land green.

That recommendation was acted on. R2 is fully delivered by the four successor tasks:
- .10 generalized the data model to ordered lists with zero syntax change, so
  `make umpire-check-case-runtime-conformance` proved byte-identity on its own.
- .11 replaced the four whitelists with a `Lean.Elab.Command` elaborator reading each inductive's
  constructors, generalizing the grammar to N initial/terminal states, N transitions, N facts per
  row and N occurrences.
- .12 added five located diagnostics at the author's own coordinates, each pinned by `#guard_msgs`.
- .13 added a genuinely second lifecycle (`RaceSyntaxTests.lean`) and rewrote `Nexus.md`.

Task .6 (R8) was re-pointed to depend on .11 and has also shipped. The spec's completion review
records all eight R-IDs met.

This task is retained as the record of why R2 was split, and is closed only so the spec can close.

stage: impl-review - skipped(policy: superseded, no implementation to review)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs:
## Superseded
This task's SCOPE_EXCEEDED block was acted on: R2 is now carried by the four-step sequence its own
handover recommended, created as tasks .10 (ordered lists, no syntax change), .11 (constructor-derived
elaboration), .12 (located diagnostics), .13 (second lifecycle + Nexus.md), chained in that order.
Task .6 (R8) was re-pointed to depend on .11, since its approach follows step (b) directly.
Do not implement this task; it stays blocked as the record of why R2 was split.
