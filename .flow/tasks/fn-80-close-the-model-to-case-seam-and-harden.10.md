---
satisfies: [R2]
---
# fn-80-close-the-model-to-case-seam-and-harden.10 Generalize the Nexus3 success model to ordered lists

## Description
Step (a) of the four-step sequence recorded in task .5's block, which escalated SCOPE_EXCEEDED: R2 is a coordinated rewrite of the arity-fixed `Authoring.successModel`, not a macro edit.

**Size:** L
**Files:** model/Temporal/Feature/Nexus3/Authoring.lean; model/Temporal/Feature/Nexus3/Tests.lean; model/Temporal/Feature/Nexus3/Testpilot.lean
**Touches:** [model/Temporal/Feature/Nexus3/Authoring.lean, model/Temporal/Feature/Nexus3/Tests.lean, model/Temporal/Feature/Nexus3/Testpilot.lean]

### Scope
Generalize `Authoring.lean`'s data model to ordered lists and mechanically port `Tests.lean` and `Testpilot.lean` to positional access. **No syntax change at all in this step** — the existing model's elaborated output and every `native_decide` theorem stay unchanged, so `make umpire-check-case-runtime-conformance` proves byte-identity on its own.

Blast radius measured in task .5's block, which you must read first:
- `SuccessModelNames` (14 scalar name fields) becomes an ordered declaration: state, action, outcome and fact key lists plus a transition-row list.
- `SuccessModel`'s 10 named value fields and 12 named `DefinitionId` fields become five parallel lists.
- `successTable`, `successModel`, `SuccessLawStatement`, `satisfiesSuccessRequirement`, `hasExactTransition`, `modelVocabulary`, `propertySpec`, `behaviorSpec` and `check` are each written against the fixed arity. `identity.stateId`/`actionId`/`outcomeId`/`factId` are if-then-else chains and become catalog lookups with a fallback.
- `ModelVocabulary`'s nine named fields become four ordered `ModelValue` lists.
- `Tests.lean` (498 lines) reads the named fields in ~30 places, including inside `native_decide` theorems — the block lists them with line numbers.
- `Testpilot.lean:176-186` `supportsSuccessProperty` compares the checked Property clause-for-clause. Task .4 was going to delete it; with .4 blocked it must be ported to the general vocabulary and its output must stay byte-identical, because the async-nexus fixture and six conformance trees diff on it.

## Acceptance
- [ ] `Authoring.lean`'s success model is ordered lists rather than fixed-arity named fields, and `Tests.lean` and `Testpilot.lean` use positional access.
- [ ] Zero syntax change: the Nexus3 success declaration is character-for-character as it is today.
- [ ] `make umpire-check-case-runtime-conformance` is green with NO fixture bytes moved, proving elaborated output is byte-identical.
- [ ] Every `native_decide` theorem in `Tests.lean` still holds, none weakened or deleted.
- [ ] `supportsSuccessProperty` produces byte-identical output through the general vocabulary.
- [ ] `make lint-model` adds nothing to the 169-finding generated-API baseline; `Umpire.Lint` stays clean.

## Done summary
Generalized the Nexus3 success model from fixed arity to ordered member lists. `SuccessModelNames`'
14 scalar name fields and `SuccessModel`'s 22 named value and Definition ID fields became parallel
`states`/`actions`/`outcomes`/`facts`/`relations` lists with positional accessors; the four
`identity` if-then-else chains became catalog-position lookups; `SuccessLawStatement` and
`satisfiesTransitionRequirement` now range over any number of declared rows; and the terminal test
reads a declared terminal list instead of one hard-coded state. `propertySpec`, `behaviorSpec`,
`modelVocabulary`, the `model` macro expansion and `Tests.lean` were ported to positional access.

Zero syntax change: `Nexus.lean` is untouched, and `make umpire-check-case-runtime-conformance`
passes with no fixture bytes moved, proving the elaborated output is byte-identical. Every
`native_decide` theorem still holds; the axiom inventory is unchanged (`successModel` [propext],
`check` [propext, Classical.choice, Quot.sound]).

Scope notes: `Syntax.lean` was edited (the macro expansion targets the changed owner) although the
task's Files list named only Authoring/Tests/Testpilot. `Testpilot.lean` needed no change — task
.15 already deleted `supportsSuccessProperty`, so that acceptance bullet was satisfied before this
task started.

Follow-ups for task .11 (raised by review, all P3, none blocking): the parallel name/value lists
can silently truncate under `zip` for a member no transition references; the positional accessors
are total over a sentinel rather than `Option`/`Fin`; and an empty transition list now discharges
the canonical-table law vacuously. The constructor-derived elaborator is the natural place to close
all three.

stage: impl-review - ran | verdict SHIP (model: claude-fable-5-1 at high); 3 P3 findings, recorded
above as follow-ups for .11 rather than expanded into this task's scope.
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 11600cf64
- Tests: cd model && lake build Temporal TemporalModelTests UmpireTests TestpilotTests, make umpire-check-case-runtime-conformance (green, no fixture bytes moved), make lint-model (169 errors, all generated Temporal/API; Umpire.Lint and Shared clean; unchanged from baseline), CGO_ENABLED=0 go test -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/..., GATE_SKIPPED:live-integration:go test -tags 'test_dep integration' ./tests -run TestTestpilot needs a live cluster; the conformance gate proves the generated Case bytes are unmoved
- PRs: