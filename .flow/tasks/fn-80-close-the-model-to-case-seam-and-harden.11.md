---
satisfies: [R2]
---
# fn-80-close-the-model-to-case-seam-and-harden.11 Elaborate Nexus3 identifiers from constructors, not whitelists

## Description
Step (b) of task .5's recorded sequence. Depends on step (a).

**Size:** M
**Files:** model/Temporal/Feature/Nexus3/Syntax.lean; model/Temporal/Feature/Nexus3/Authoring.lean
**Touches:** [model/Temporal/Feature/Nexus3/Syntax.lean, model/Temporal/Feature/Nexus3/Authoring.lean]

### Scope
Replace the four spelling whitelists with constructor-derived elaboration over the ordered data model step (a) delivered. Keep the Nexus3 success declaration character-for-character as it is today, so this step also moves no fixture bytes.

Task .5's block flags this as new ground: the repo has no `Lean.Elab.Command` elaborator that reads an inductive's constructors via `getConstInfoInduct`. Design that reader here; its diagnostics land in step (c).

This is what R8 (task .6, the `query ... all ...` verify form) follows directly — .6's own approach is small and sits on top of this step.

## Acceptance
- [ ] The four identifier whitelists are gone; elaboration derives admissible spellings from the declaring inductive's constructors.
- [ ] A renamed Action or an added transition elaborates instead of producing a whitelist miss.
- [ ] The Nexus3 success declaration is unchanged character-for-character and no fixture bytes move.
- [ ] `make umpire-check-case-runtime-conformance` green; `make lint-model` adds nothing to the 169 baseline.

## Done summary
Replaced the four Nexus3 identifier whitelists with constructor-derived elaboration. The `model`
command is now a `Lean.Elab.Command` elaborator that reads each declared inductive through
`getConstInfoInduct`, derives the ordered state, Action, Model Outcome and Fact domains from
constructor order, and resolves every authored spelling against them with a located error for an
unknown one. The grammar generalized to any number of initial states, terminal states, transition
rows, per-transition Facts and Behavior occurrences. `property` and `behavior` carry the spellings
they select — including the role — into the Property and Behavior specs, so a `require` clause names
its own member instead of the last declared one, and the `limits` whitelist is gone.

The Nexus3 success declaration is unchanged character-for-character and
`make umpire-check-case-runtime-conformance` passes with no fixture bytes moved.

The four whitelist `#guard_msgs` blocks were replaced by a probe that renames the role to `worker`,
adds a third transition, selects the start Action rather than the completion one, and declares its
own limits — every spelling a whitelist used to reject — plus negative guards for a misspelled
Property member, a misspelled Behavior role, an unknown Action in a transition row, and an unknown
initial state. Task .13's second lifecycle supersedes the probe's positive half with the full
five-block surface.

Deferred to .12 by the recorded plan: the located diagnostics for a constructor with arguments, a
duplicate `before + action` pair, an unreachable terminal, and a transition count over 256. Also
noted for a later task: the `Setup` type is still implicit — the elaborator resolves `Setup` in the
model name's scope while the other four domains are named in the grammar; a second lifecycle with
its own Setup will want a `setup <Type>` clause.

Follow-ups from .10 that this task closed: the parallel name/value lists are still zipped, and the
positional accessors are still total over a sentinel, but the sentinel is now load-bearing in a good
way — an unresolvable member spelling deliberately resolves to an id no Target provides, so
admission rejects it by name.

stage: impl-review - ran | verdict SHIP (model: claude-fable-5-1 at high); 1 P2 and 4 P3 findings,
all five addressed in a follow-up commit.
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: b114db22b, 4c8932e2e
- Tests: cd model && lake build Temporal TemporalModelTests UmpireTests TestpilotTests, make umpire-check-case-runtime-conformance (green, no fixture bytes moved), make lint-model (169 errors, all generated Temporal/API; Umpire.Lint and Shared clean; unchanged from baseline), CGO_ENABLED=0 go test -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/..., GATE_SKIPPED:live-integration:go test -tags 'test_dep integration' ./tests -run TestTestpilot needs a live cluster; the conformance gate proves the generated Case bytes are unmoved
- PRs: