---
satisfies: [R3, R4]
---
# fn-85-model-side-effects-as-typed-actions-and.14 The machine command: state fields, setup, timers, step functions; Limits accounting; model retired

## Description
Replace the `model` command with `machine` (R3) on the enumerator task `.3` proved: the entity it tracks, `ends:`, a finite state `structure`, `setup:`, `timers:`, its step functions and its evidence lines, with the command enumerating each step function at elaboration so the fingerprint, Search and lowering never see a function. Timers fire only while their step returns a successor; faults are actions; timer firings and fault actions count toward the step and action Limits (R4). The 18 `model` declarations and the 33 command specimens move to `machine`, and `model` is retired.

Split out of the former single `.3` by plan review round 1 (finding F2), which found that record carrying the enumerator, the command, the diagnostics, the Limits accounting, the specimen migration and the `property` change at once.

**Size:** M (two commits: the command and its diagnostics; Limits accounting and specimen migration)
**Files:** `model/Umpire/Command/Syntax.lean` (`machine` command with `for:`, `refines:`/`map:` reserved for task .6, `ends:`, `state:`, `setup:`, `timers:`, `steps:`, `evidence:`; `model` removed), `model/Umpire/Command/Authoring.lean` (`declareMachine` over `.3`'s enumerator; a reserved non-drivable action for `system` and timer steps; stuck-state witness diagnostics), `model/Umpire/Command/Registry.lean`, `model/Umpire/Search.lean` and `Search/Admission.lean` (Limits accounting for timer firings and fault actions; instance bound), `model/Umpire/Query.lean` (instance bound in Limits if needed), `model/Temporal/Feature/Nexus/Success/Model.lean` (the success Model as a step function, still producing the async-Nexus Case through `case` until task .11), `model/Temporal/Feature/Nexus/Success/Tests.lean` (specimens respelled; new witness specimens), `tools/umpire/internal/retiredvocabulary/check.go` (retire the `model` command spelling as a compound token if SEM-20 admits one; else record)
**Touches:** [model/Umpire/Command/**, model/Umpire/Search.lean, model/Umpire/Search/**, model/Umpire/Query.lean, model/Temporal/Feature/Nexus/Success/**, tools/umpire/internal/retiredvocabulary/check.go]

### Approach
- Read `.plans/LEAN_GUIDELINES.md` first. Start from `.3`'s enumerator and `Finite`; this task adds no enumeration logic of its own.
- Diagnostics with witnesses: a state outside `ends:` from which some action has no successor is reported with that concrete state and action at the `steps:` line; an evidence line naming an outcome the step never returns rejects at that line; a redundant `match` arm is Lean's own error (pin one specimen to show it surfaces at the function).
- `system` steps and timers: a step with no input gets a synthesized reserved action value the realization cannot bind `driven`; it counts toward the step Limit, and timer firings count toward the action Limit as R4 states.
- Evidence keyed by (action class pattern, outcome) with an optional `when` guard over the pre-state; `unobservable` becomes a Known Gap at Case production.
- `refines:`/`map:` are parsed here as reserved keys and checked in task .6.
- Respell `model:` keys as `machine:` in `property`, `scenario` and `query`; keep every fn-83 .14 diagnostic message that still applies. The `property` body stays in its keyed form here — task `.15` changes it to a predicate — so this task's specimen migration is a rename, not a rewrite of the Properties.

### Investigation targets
**Required:**
- `model/Umpire/Command/Syntax.lean:54-57,73-75,97,144-166,195-345,548-566` — the row grammar, `transitionBound`, `relationKey`, reachability, `elabModel`, the `evalExpr` bridge
- `model/Umpire/Command/Authoring.lean:138-168,215-323` — `DeclaredModel`, `step`, `declaredTable`, `declareModel`
- `model/Umpire/Core.lean:311,367-390` — vocabulary materialization and `Machine`
- `model/Temporal/Feature/Nexus/DESIGN.md` section 2.3 and the section 3 `nexusProtocol` machine — the rows to express as `match` arms
- `.plans/UMPIRE_CMP_FIZZBEE.md` section 4.1 — the chosen form and its rule reading (AUT-05, AUT-07a, AUT-09)

**Optional:**
- `model/Umpire/Property/Evaluate.lean:71` — `naturalAtMost` parses one state value today (task .4 gives fields their own values)

### Key context
- AUT-09 today admits enum-like inductives; task .13 drafts the amendment admitting a structure of finite fields and an enumerated step function. Until approved the command is drafted under that rule the way AUT-07a's commands were.
- Depends on `.3`: if `.3`'s fingerprint pin failed, this task is re-planned on the row grammar before it starts.

## Acceptance
- [ ] `machine` elaborates the DESIGN.md section 3 `nexusProduct` and `nexusProtocol` machines (without the cancel rows) written as step functions into checked `Umpire.Machine` records; `model` no longer elaborates
- [ ] a non-finite state or input field, a step with another signature, a stuck non-terminal state (with its witness), `terminal` without `ends:`, a timer no step names, an evidence outcome the step never returns, an evidence name that is neither in the realization's catalog nor a declared `observation` (deferred here from task .2, which has no `evidence:` line to reject on), and a system or timer step without evidence or `unobservable` each reject in place, pinned by `#guard_msgs`; a redundant `match` arm surfaces as Lean's error at the function
- [ ] a Search over a machine with a timer fires it only while its step returns a successor; timer firings and fault actions count toward the Limits, pinned by a `#guard` on `limitReached`
- [ ] all 18 `model` declarations and 33 command specimens are respelled; `model` is retired in the vocabulary gate or the reason it cannot be is recorded
- [ ] the success Model regenerates the async-Nexus fixture byte-identical through `case`; `lake build TemporalModelTests UmpireTests` green; `make lint-model` green

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
