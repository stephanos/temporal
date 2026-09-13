---
satisfies: [R3, R4]
---
# fn-85-model-side-effects-as-typed-actions-and.3 The machine command: state fields, setup, timers, ordered rows; model retired

## Description
Replace the `model` command with `machine` (R3): the entity it tracks, `ends:`, a finite state `structure`, `setup:`, `timers:`, its step functions and its evidence lines. A step function is ordinary Lean, `State → Input → List (State × Outcome)`; the command enumerates it over the derived finite domain at elaboration into the same finite table `model` produced, so the fingerprint, Search and lowering never see a function. Timers fire only while their step returns a successor; faults are actions; timer firings and fault actions count toward the step and action Limits (R4). The 33 command specimens move from `model` to `machine`; `model` is retired. Decided with the user on 2026-09-12 over the row grammar (see the spec's Planning decisions); the first commit is the prototype that proves the fingerprint equality.

**Size:** M (three commits: the finite-domain enumerator and prototype; the command and its diagnostics; Limits accounting and specimen migration)
**Files:** `model/Umpire/Command/Finite.lean` (new: a `Finite` class with instances for enum-like inductives, `Bool` and `Fin`, and a deriving handler for structures of finite fields; a located rejection for any other field type), `model/Umpire/Command/Syntax.lean` (`machine` command with `for:`, `refines:`/`map:` reserved for task .6, `ends:`, `state:`, `setup:`, `timers:`, `steps:`, `evidence:`; `model` removed), `model/Umpire/Command/Authoring.lean` (`declareMachine`: enumerate each step function through the `Meta.evalExpr` bridge `evalDiagnostic` already uses into `BehaviorTransitionRow`s; `count` fields as `Fin (bound+1)` saturating to `limitReached`; a reserved non-drivable action for `system` and timer steps; stuck-state witness diagnostics), `model/Umpire/Command/Registry.lean`, `model/Umpire/Search.lean` and `Search/Admission.lean` (Limits accounting for timer firings and fault actions; instance bound), `model/Umpire/Query.lean` (instance bound in Limits if needed), `model/Temporal/Feature/Nexus/Success/Model.lean` (the success Model as a step function, still producing the async-Nexus Case through `case` until task .11), `model/Temporal/Feature/Nexus/Success/Tests.lean` (specimens respelled; new witness specimens), `tools/umpire/internal/retiredvocabulary/check.go` (retire the `model` command spelling as a compound token if SEM-20 admits one; else record)
**Touches:** [model/Umpire/Command/**, model/Umpire/Search.lean, model/Umpire/Search/**, model/Umpire/Query.lean, model/Temporal/Feature/Nexus/Success/**, tools/umpire/internal/retiredvocabulary/check.go]

### Approach
- Read `.plans/LEAN_GUIDELINES.md` first. Commit 1, the prototype: `Finite` and its deriving handler; rewrite the success Model's two rows as a step function over `structure { phase : State }`; enumerate it and assert with `#guard` that the resulting `BehaviorTable` and fingerprint equal the row form's; measure elaboration with `lake env lean` three times against the Race baselines (6 to 12 ms per check) and record. If the fingerprint differs or elaboration is an order of magnitude slower, stop and report; the fallback is the row grammar.
- Enumeration: domain = product of the state structure's fields × the action's input class members (task .2's constructors with fields); evaluate the step for every pair; each successor becomes a transition row keyed by (state, action input, successor, outcome). The bound is the existing `transitionBound`, re-checked against the product state space and reported as a located error, never a truncation.
- Diagnostics with witnesses: a state outside `ends:` from which some action has no successor is reported with that concrete state and action at the `steps:` line; an evidence line naming an outcome the step never returns rejects at that line; a redundant `match` arm is Lean's own error (pin one specimen to show it surfaces at the function).
- `system` steps and timers: a step with no input gets a synthesized reserved action value the realization cannot bind `driven`; it counts toward the step Limit, and timer firings count toward the action Limit as R4 states.
- Evidence keyed by (action class pattern, outcome) with an optional `when` guard over the pre-state; `unobservable` becomes a Known Gap at Case production.
- `refines:`/`map:` are parsed here as reserved keys and checked in task .6.
- Respell `model:` keys as `machine:` in `property`, `scenario` and `query`; keep every fn-83 .14 diagnostic message that still applies.

### Investigation targets
**Required:**
- `model/Umpire/Command/Syntax.lean:54-57,73-75,97,144-166,195-345,548-566` — the row grammar, `transitionBound`, `relationKey`, reachability, `elabModel`, the `evalExpr` bridge
- `model/Umpire/Command/Authoring.lean:138-168,215-323` — `DeclaredModel`, `step`, `declaredTable`, `declareModel`
- `model/Umpire/Model/Types.lean:52-76` — `BehaviorTransitionRow`, `BehaviorTable` (the canonical form the enumeration fills)
- `model/Umpire/Core.lean:311,367-390` — vocabulary materialization and `Machine`
- `.plans/UMPIRE_CMP_FIZZBEE.md` section 4.1 — the chosen form and its rule reading (AUT-05, AUT-07a, AUT-09)
- `model/Temporal/Feature/Nexus/DESIGN.md` section 2.3 and the section 3 `nexusProtocol` machine — the rows to express as `match` arms

**Optional:**
- `model/Temporal/Feature/Nexus/Race/COVERAGE.md:28-30` — the elaboration baselines
- `model/Umpire/Property/Evaluate.lean:71` — `naturalAtMost` parses one state value today (task .4 gives fields their own values)

### Key context
- AUT-09 today admits enum-like inductives; task .13 drafts the amendment admitting a structure of finite fields and an enumerated step function. Until approved the command is drafted under that rule the way AUT-07a's commands were.
- Memory: work accounting in admission; bound instance count and `count` fields before enumeration, never enumerate eagerly beyond Limits.
- No Mathlib: `Finite` is a small local class, not `Fintype`.
- 2026-09-12: `property` follows the same rule as `machine` (see the spec's Planning decisions and API contracts): its body is a Lean predicate, `Step → Bool` or `Step → Step → Bool`, enumerated over the machine's table into the existing `PropertyClause` records; the keyed `require: state:/outcome:/fact:` form is retired with `model`. Task .10 writes the Nexus Properties in this form.
## Acceptance
- [ ] prototype: the success Model as a step function enumerates to a `BehaviorTable` and fingerprint equal to the row form's, pinned by `#guard`; elaboration time recorded against the Race baselines
- [ ] `machine` elaborates the DESIGN.md section 3 `nexusProduct` and `nexusProtocol` machines (without the cancel rows) written as step functions into checked `Umpire.Machine` records; `model` no longer elaborates
- [ ] a non-finite state or input field, a step with another signature, a stuck non-terminal state (with its witness), `terminal` without `ends:`, a timer no step names, an evidence outcome the step never returns, and a system or timer step without evidence or `unobservable` each reject in place, pinned by `#guard_msgs`; a redundant `match` arm surfaces as Lean's error at the function
- [ ] a Search over a machine with a timer fires it only while its step returns a successor; timer firings and fault actions count toward the Limits, pinned by a `#guard` on `limitReached`
- [ ] the success Model regenerates the async-Nexus fixture byte-identical through `case`; `lake build TemporalModelTests UmpireTests` green; `make lint-model` green
- [ ] `property` takes a `Step → Bool` or `Step → Step → Bool` predicate and enumerates it into `PropertyClause` records; `successfulResult` as a predicate has the fingerprint of its keyed form, pinned by `#guard`; a predicate over another machine's `Step` type and a non-decidable predicate reject in place, pinned by `#guard_msgs`
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
