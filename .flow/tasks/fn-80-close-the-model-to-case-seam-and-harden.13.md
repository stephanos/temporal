---
satisfies: [R2]
---
# fn-80-close-the-model-to-case-seam-and-harden.13 Add the second-lifecycle syntax test and Nexus3 wording

## Description
Step (d) of task .5's recorded sequence, the small remainder. Depends on step (c).

**Size:** S
**Files:** model/Temporal/Feature/Nexus3/RaceSyntaxTests.lean; model/Temporal/Feature/Nexus3/Nexus.md
**Touches:** [model/Temporal/Feature/Nexus3/RaceSyntaxTests.lean, model/Temporal/Feature/Nexus3/Nexus.md]

### Scope
Add `RaceSyntaxTests.lean` — a genuinely second enum-like finite lifecycle elaborated through the generalized syntax, which is the real proof that the whitelists are gone rather than widened — and update `Nexus.md` to describe the generalized form.

Check for a conflict with fn-67 before editing `Nexus.md`: fn-67 has an open documentation task on that file and on `Integration.md`.

## Acceptance
- [ ] `RaceSyntaxTests.lean` elaborates a second, structurally different lifecycle through the same syntax, wired into the test aggregate.
- [ ] `Nexus.md` describes the generalized form; any overlap with fn-67's open documentation task is named rather than silently overwritten.
- [ ] Full model build and `make umpire-check-case-runtime-conformance` green.

## Done summary
Added `RaceSyntaxTests.lean`, a second lifecycle authored through the same five commands and wired
into the model test aggregate: role `handler`, five states, four Actions, four Model Outcomes,
three Facts, four transition rows, two terminal states, a losing row that records no Fact, a
two-clause Property, and a three-occurrence Behavior over a four-Action model. Its witness Query
checks, so the whole five-block surface is exercised on a lifecycle that shares no state, Action or
Fact spelling with the success slice.

Two grammar generalizations were needed to author it and landed with it: `property` now accepts one
or more `require` clauses, each naming its own member through `Authoring.PropertyRequirement`
(R2's two-clause Property was previously inexpressible — the grammar was fixed at three); and the
`model` elaborator now rejects an unsorted Action catalog and an unsorted multi-state `initial` list
at those identifiers, because the planner admits only canonically ordered ones and the failure
otherwise surfaced late as `invalidPlanner` at Query check. Both are pinned by `#guard_msgs`.

`Nexus.md` describes the generalized grammar: what it accepts, the seven located rejections, the two
ordering requirements, the implicit one-constructor `Setup`, and what genuinely remains proposed
(`oneOf` alternatives, an omitted `facts` field, a Property with no `when`, `eventually ... within`,
`finalState`, `verify ... search exhaustive`, and the compatibility ID). The draft's own Action
constructors were sorted so the teaching file does not show a shape the grammar rejects.

fn-67 overlap: fn-67.2 and its completion review had already landed, so there was no open task to
conflict with. This task retouched two sentences fn-67.2 authored — "success-only syntax" became
"syntax", and the `cancellationResolves` docstring was left exactly as fn-67 wrote it, including its
"delivered generic capability, qualified" wording and the cancellation-specific rejection.

Shape note: R2 and task .5 name "four states, three transitions" for the second lifecycle. The
delivered one has five states and four transitions — the extra `queued`/`initiate` prefix and the
losing `complete` row are what let the race actually race without the still-proposed `oneOf` form.

stage: impl-review - ran | round 1 NEEDS_WORK, round 2 NEEDS_WORK, round 3 SHIP (model:
claude-fable-5-1 at high). Round 1 caught a false rename claim in Nexus.md and the silent
three-clause Property limit; round 2 caught the undocumented Action-order requirement the test had
worked around. Both were fixed in the code rather than in the prose. All findings addressed.
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: a14a3b6b6, 6d0a4b6e5, e78b4764e, addd7242d
- Tests: cd model && lake build Temporal TemporalModelTests UmpireTests TestpilotTests, make umpire-check-case-runtime-conformance (green, no fixture bytes moved), make lint-model (169 errors, all generated Temporal/API; Umpire.Lint and Shared clean; unchanged from baseline), CGO_ENABLED=0 go test -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/..., GATE_SKIPPED:live-integration:go test -tags 'test_dep integration' ./tests -run TestTestpilot needs a live cluster; the conformance gate proves the generated Case bytes are unmoved
- PRs: