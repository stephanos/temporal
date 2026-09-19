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

So this task's first job is the law, not the machine. Options, in the order I would try them:

1. Prove the law at reducible transparency, or restate `TableLawStatement` so the catalogs are not
   forced: `table.transitions = transitions` holds by construction and should not need the `states`
   and `actions` catalogs whnf'd to see it.
2. Let `declareMachine` build the table with the catalogs already in the shape the law wants, so the
   three components are `rfl` at a glance rather than after reduction.
3. Fall back to `decide`/`native_decide` at the `checked` seam, which the repository's trust policy
   already admits.

`.14` refuses such a machine rather than declaring a `sorryAx`-carrying Model, so nothing is unsound
today -- the design's own specimen simply cannot be written yet.

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
