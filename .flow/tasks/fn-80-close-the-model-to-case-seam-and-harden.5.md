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
Blocked:
BLOCKED: SCOPE_EXCEEDED — R2 is a redesign of the Nexus3 authoring layer, not a macro edit.

The task is sized **M** and its Approach reads as "delete four whitelists and elaborate over the
parsed identifiers". The whitelists are the small half. The elaborators expand into
`Authoring.successModel`, whose entire data model is arity-fixed at three states, two actions, two
outcomes, two facts and two transitions, and that shape is load-bearing in three other modules.
Generalizing it is a coordinated rewrite that cannot land half-done, and this session could not
land it green.

Exact blast radius, measured against the tree at this commit:

1. `model/Temporal/Feature/Nexus3/Authoring.lean`
   - `SuccessModelNames` (14 scalar name fields) becomes an ordered declaration: state, action,
     outcome and fact key lists plus a transition-row list.
   - `SuccessModel` carries 10 named value fields (`scheduledState`, `startedState`,
     `awaitStartAction`, …) and 12 named `DefinitionId` fields (`scheduledStateId`,
     `startRelationId`, …). All 22 become five parallel lists.
   - `successTable`, `successModel`, `SuccessLawStatement`, `satisfiesSuccessRequirement`,
     `hasExactTransition`, `modelVocabulary`, `propertySpec`, `behaviorSpec` and `check` are each
     written against the fixed arity. `identity.stateId`/`actionId`/`outcomeId`/`factId` are
     if-then-else chains over the three states / two actions and become catalog lookups with a
     fallback.
   - `ModelVocabulary`'s nine named fields become four ordered `ModelValue` lists.

2. `model/Temporal/Feature/Nexus3/Tests.lean` (498 lines) reads the named fields in roughly 30
   places, including inside `native_decide` theorems: `checkedWitnessIsExact` (:167-172),
   `modelMemberIds` (:302-306), `malformedDefinition` / `wrongKindDefinition`
   (:325-335, uses `lifecycle.scheduledStateId`), `renamedIdentitiesAreCoherent` (:376-377, uses
   `awaitStartActionId`/`awaitSuccessActionId`), `outgoingTerminalTable` and
   `extraSuccessResultTable` (:187-197, use `lifecycle.startedResult`/`succeededResult`),
   `changedSuccessRelationIsRejected` (:222-227, calls `satisfiesSuccessRequirement` directly),
   and eight `values.awaitSuccessAction`-style reads.

3. `model/Temporal/Feature/Nexus3/Testpilot.lean:176-186` — `supportsSuccessProperty` compares the
   checked Property clause-for-clause against `values.awaitSuccessAction`, `values.succeededState`,
   `values.completedOutcome`, `values.succeededFact`. Task .4 was going to delete this function
   outright; with .4 blocked it has to be ported to the general vocabulary instead, and its output
   must stay byte-identical because the async-nexus fixture and six conformance trees diff on it.

4. The elaborator itself is new ground: this repo has no `Lean.Elab.Command` elaborator that reads
   an inductive's constructors (`getConstInfoInduct`), and the five `#guard_msgs` error classes
   (constructor with arguments, unknown constructor, duplicate `before + action`, unreachable
   terminal, transition count over 256) need their message text designed and pinned.

5. `RaceSyntaxTests.lean` (new, second lifecycle) and `Nexus.md` are the small remainder.

What is NOT blocked: nothing in R2 depends on the R1 lowering, and the finding recorded on task .4
(the scoped clause form can carry state, outcome and fact predicates) does not constrain it. R2 is
blocked purely on size.

Suggested resolution — re-plan into a sequence that can each land green:
  a. Generalize `Authoring.lean`'s data model to ordered lists and mechanically port `Tests.lean`
     and `Testpilot.lean` to positional access, with the existing model's elaborated output and
     every `native_decide` theorem unchanged. No syntax change at all in this step, so
     `make umpire-check-case-runtime-conformance` proves byte-identity on its own.
  b. Replace the four spelling whitelists with constructor-derived elaboration over that data
     model, keeping the Nexus3 success declaration character-for-character as it is today.
  c. Add the located diagnostics and their `#guard_msgs`.
  d. Add `RaceSyntaxTests.lean` and the `Nexus.md` wording.

Impact: task .6 (R8, the `query ... all ...` verify form) depends on this task and stays blocked;
its own Approach is small and would follow step (b) directly. Task .9's R2/R8 documentation
bullets are deferred with them.
## Evidence
- Commits:
- Tests:
- PRs:
