---
satisfies: [R3, R4]
---
# fn-85-model-side-effects-as-typed-actions-and.16 Retire model: the protocol machine, Limits accounting, and the tree-wide migration

## Description
Retire `model` (R3, R4) once `machine` exists: the protocol machine of `DESIGN.md` section 3, the Limits accounting timer firings and fault actions need, the migration of every `model` declaration and command specimen in the tree, and the vocabulary gate's retirement of the spelling.

Split out of `.14` on 2026-09-18, for the reason plan review round 1 split the former single `.3`: that task was carrying the command, its diagnostics, the specimen, the Limits accounting, the migration of 18 declarations and 33 specimens, and a byte-identical fixture regeneration at once. The command half is done and reviewable on its own; this half touches every Model in the tree and must keep fixtures byte-identical, which is a different risk and a different review.

`.14` delivered: the `machine` command with `for:`, `state:`, `starts:`, `ends:`, `timers:`, `setup:`, `evidence:` and `steps:`; the synthesized Action domain and dispatching step function; enumeration into `declareModel` with a bound of its own; the stuck-state witness diagnostic; and `DESIGN.md` section 3's `nexusProduct` written as step functions.

### Measured before starting, 2026-09-19

**The protocol machine as `DESIGN.md` section 3 writes it does not elaborate, and shrinking it is not
the answer.** The canonical-table law `declareModel` proves by `⟨rfl, rfl, rfl⟩` forces the whole
`declaredTable` application, whose state and action catalogs are `zip`/`map`s as long as the machine.
Measured on this session's 4-core cloud box:

| machine | states | action classes | result |
| --- | --- | --- | --- |
| a 3-field state, 11 actions | 112 | 11 | elaborates, `[propext]`, 9.7 s |
| the design's protocol machine | 224 | 23 | refuses at 1,000,000 heartbeats, 2 m 6 s |
| the same with `attempts : Fin 2` | 112 | 23 | refuses at 1,000,000 heartbeats, 2 m 9 s |

Halving the retry bound does not help, so the cost is not the state count alone: 112 states over 11
actions proves in ten seconds and 112 over 23 does not prove at all. The work is in the law, not in
the walk -- the enumeration itself is cheap.

So this task's first job is the law, not the machine.

### Resolved, 2026-09-19

It was neither the law's statement nor its transparency: it was
`Umpire.Command.satisfiesTransitionRequirement`, which searched every required row against every
authored row with a structural comparison at each step. A Model enumerated from its own step
functions passes its own transition list as the requirement, so the search concluded what a single
list comparison already says -- quadratically. Short-circuiting on `rows == required` made the
design's machine elaborate.

Measured after the change, on the same box:

| machine | states | action classes | rows | result |
| --- | --- | --- | --- | --- |
| `nexusProduct` | 6 | 11 | 34 | `[propext]`, about 10 s |
| `nexusProtocol` | 192 | 23 | 1152 | `[propext]`, about 40 s |

`nexusProtocol` is 192 rather than the 224 measured before because it is written with an
`unscheduled` phase and a retry bound of two: the design's `none + schedule` row is a step from a
phase, since a machine's state structure has no "no instance yet" member, and it is what makes the
three deadline fields reachable at anything but their first value.

### Decisions taken while delivering, 2026-09-19

**An `evidence:` line may name a fact constructor.** A fact that carries fields has one member per
assignment of them and each member's key spells that assignment out, but the mapping to a catalogued
event is the constructor's: `DESIGN.md` section 3 records every `nexusOperationTimedOut` under one
event name whichever of the three timers fired. It is also the only spelling an identifier admits.

**A `setup:` parameter ranges over any finite domain.** It configures the implementation under test
rather than being an input an action carries, so no class of it is written out and no example is
stored against one; the design's own parameters are `Bool`.

**A count's bound is reduced before it is read**, so a bound written as a name works -- which is what
the bound will be once it comes from the Limits rather than from the field.

**`ends:` is required**, the way `starts:` already was. Without it no state is terminal, a Search
runs to its limit on every path and a Property that requires an instance to finish holds by never
being reached. This is the step-function reading of the acceptance item's "`terminal` without
`ends:`": the row grammar's `terminal` keyword is `terminalPhase` in ordinary Lean, and what it
needs from the machine is the `ends:` line.

**`unobservable:` names a timer, not a row.** `DESIGN.md` section 2.3 writes `unobservable` as an
evidence value on one row; per-action is the granularity a step function has, and a timer is the
only `system` step the command builds. A timer that fires and records nothing an `evidence:` line
names is rejected unless it is declared `unobservable:`, and one that does record evidence is
rejected if it is.

**The Action catalog is emitted in canonical order.** A Search admits a Model whose action catalog is
sorted by member Definition ID, and this cannot be left to the author: a classed action contributes
one member per assignment of its inputs in its domain's member order, so no arrangement of `steps:`
lines could sort `complete-succeeded`, `complete-failed` and `complete-canceled`. `enumerate` gained
`enumerateOver`, which walks an explicit action order, and the machine passes the sorted catalog to
it and to `declareModel` alike.

**A machine records a `ModelEntry`, not only a `MachineEntry`,** so `property`, `scenario` and
`query` see it. Members resolve by the key the table carries; a machine over a one-field state keeps
the bare spellings a `model` had, which is what makes the migration mechanical.

### Carried forward, not resolved

A `scenario` names its start state and its Actions with identifiers, and a structured machine's keys
are punctuated -- `schedule-unset-unset-expires`, and a state key naming all five fields. So
`DESIGN.md` section 3's own `asyncThenSucceeded` scenario and `asyncCompletion` query cannot be
written over `nexusProtocol` yet. Which surface a Scenario should use over a structured state is
task `.5`'s question; item 4's claims are pinned on `attemptLoop`, a machine small enough to be named
by identifiers.

### Accepted limitation, carried from `.14`'s review

A state structure with two fields of the same enum can name neither a `starts:` nor an `ends:` value:
the value is carried by both fields, the rejection is correct, and the grammar admits only a bare
identifier so there is no way to say which field is meant. `DESIGN.md` section 3's protocol state has
three `Timeout` fields, so this is reachable the moment `cancel` or a second phase field arrives.
Give `starts:`/`ends:` a `field: value` spelling here, or record why not.

### Approach
- The protocol machine is 224 states over 23 action classes. Measure its elaboration before assuming it is affordable, and record the number the way `.3` recorded the enumerator's.
- `setup:` parameters are parsed and recorded by `.14` but do not vary the table: `declareModel` takes one setup value. Varying it is task `.5`'s, so either depend on `.5` or carry the one-setup table and say so.
- Migration is mechanical but must be checked by the fixtures, not by reading: regenerate and diff rather than respelling and hoping.
- `model` is retired in `tools/umpire/internal/retiredvocabulary/check.go` if SEM-20 admits a compound token; if it does not, record why rather than leaving the spelling live.

## Acceptance
- [ ] `machine` elaborates the DESIGN.md section 3 `nexusProtocol` machine (without the cancel rows) into a checked `Umpire.Machine` record, and its elaboration cost is measured and recorded
- [ ] a step with another signature, `terminal` without `ends:`, and a system or timer step without evidence or `unobservable` each reject in place, pinned by `#guard_msgs`; a redundant `match` arm surfaces as Lean's error at the function
- [ ] an evidence name that is neither in the realization's catalog nor a declared `observation` rejects in place, pinned by `#guard_msgs` (deferred from task .2, which has no `evidence:` line to reject on)
- [ ] a Search over a machine with a timer fires it only while its step returns a successor; timer firings and fault actions count toward the Limits, pinned by a `#guard` on `limitReached`
- [ ] all 18 `model` declarations and 33 command specimens are respelled; `model` no longer elaborates; `model` is retired in the vocabulary gate or the reason it cannot be is recorded
- [ ] the success Model regenerates the async-Nexus fixture byte-identical through `case`; `lake build TemporalModelTests UmpireTests` green; `make lint-model` green; `make umpire-check-regression` exit 0


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
