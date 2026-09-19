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

### Authoring shape, decided 2026-09-15 at the start of delivery

`DESIGN.md` section 2.3 and the section 3 machines are written in the **row grammar**, which the
user's 2026-09-12 decision replaced with step functions (`UMPIRE_CMP_FIZZBEE.md` section 4.1, proved
by task `.3`). The design record therefore does not settle how a step function meets a declared
`action`, and this task cannot start without an answer. The answer taken:

- **`state:` names a declared `structure`**, not a field block. The step functions take that
  structure as their argument, so it has to exist before the `machine` command names them; a block
  the command synthesized could not be referred to by a function written above it.
- **One step function per declared action**, its action input curried after the state:
  `transportFault : State → List (Step …)` for an action with no `input:`,
  `handlerReply : State → Reply → List (Step …)` for one, and
  `schedule : State → Timeout → Timeout → Timeout → List (Step …)` for three. This is the shape the
  FizzBee comparison sketches and the one a Go `switch` reader writes without being taught anything.
- **The command synthesizes the machine's Action domain**, one constructor per action carrying that
  action's input fields, plus one per timer, and a total dispatching step function over it. That is
  what `.3`'s `enumerate` walks, so this task still adds no enumeration logic; the synthesis is what
  makes several per-action functions into the one `State → Action → List (Step …)` the enumerator and
  `declareModel` already take.
- **`ends:`, `setup:`, `timers:`, `evidence:` keep their `DESIGN.md` meaning**; only the rows become
  functions. `refines:`/`map:` parse here and are checked in `.6`.

`DESIGN.md` section 2.3 is amended with a dated note recording this, the way section 2.2 was amended
in `.2`.

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
- [x] `machine` elaborates into a checked `Umpire.Machine` record through `declareModel`, on rows enumerated from the author's step functions: `for:`, `state:`, `starts:`, `ends:`, `timers:`, `setup:`, `evidence:` and `steps:`, with the Action domain and the dispatching step function synthesized from the actions it steps on
- [x] `DESIGN.md` section 3's `nexusProduct` machine is written as step functions and elaborates; its table, its end states and its action classes are pinned
- [x] an undeclared entity or action, a state that is not a finite structure, a duplicate step, a step function of another signature, a timer no step names, an evidence line naming a fact no step returns, and a machine whose enumeration exceeds the bound each reject in place
- [x] a state the machine reaches, does not end in, and can take no step from is rejected with that concrete state as the witness
- [x] `lake build` green; `make lint-model` green at the 163 baseline

**Split on 2026-09-18.** The protocol machine, the Limits accounting, the tree-wide migration of 18 `model` declarations and 33 specimens, retiring the `model` spelling, and the byte-identical fixture regeneration moved to task `.16`, for the reason plan review round 1 split the former single `.3`: one task was carrying the command, its diagnostics, the specimen, the accounting, the migration and a fixture regeneration at once. The command half is reviewable on its own; the migration half touches every Model in the tree and must keep fixtures byte-identical, which is a different risk and a different review.
## Done summary
`machine` is a command. It names the entity a machine tracks, the structure it keeps per instance,
the states it begins and ends in, its timers, its setup parameters, its evidence lines, and one step
function per action it steps on — and it turns those into a checked `Umpire.Machine` on rows
enumerated from the author's own functions.

The author writes one function per action, over that action's own inputs. The enumerator walks one
`State → Action → List (Step …)`. The command is what turns the first into the second: an Action
domain with one constructor per action it steps on, carrying that action's input fields and one
constructor per timer, and a total step function dispatching to what the author wrote. Nothing
downstream knows there was more than one. `DESIGN.md` section 3's `nexusProduct` is written this way
and its table, reachable set, end states and action classes are pinned.

### Decisions

- **Outcome and Fact are read off the step functions**, not declared. A function returns
  `List (Step State Outcome Fact)`, so asking for them as keys would admit two answers that disagree.
  Read by unifying against the shape rather than matching a head constant, because `Umpire.Step` is
  an abbreviation and a match on what it reduces to would name a type no author wrote.
- **A timer is an action of the machine's domain**, not a separate kind: `system` behaviour with no
  input, which no realization can drive. A timer no `steps:` line names never fires.
- **`starts:` and `ends:` name values of one field**, which the values themselves identify; a value
  carried by two fields is rejected as undecided rather than resolved silently. Every other field of
  a start takes its first enumerated value — zero for a count, the first constructor for an enum.
- **`setup:` is parsed and recorded, not varied over.** `declareModel` takes one setup value, and
  varying it is task `.5`'s.
- **The enumeration has its own bound**, larger than `elaborationBound`. That one bounds rows an
  author writes; this one bounds evaluations nobody writes or reads.

### Review

Three rounds, each finding something real; round three had no Critical.

Round one: a machine with no `starts:` began nowhere, so nothing was reachable, the stuck check had
nothing to check, and every Property would have held vacuously. `ends:` picked the first field whose
values spelled the name, giving a machine a terminal set its author never wrote. A count field could
not be written at all — `Fin`'s members are spelled `0`, `1`, `2` and each rendered as an identifier
no scope declares. And my own specimen guard could not fail: it claimed the Model reaches every end
state while asserting only `stuck == none`, on a machine where `workerStop` self-loops from every
state so nothing could be stuck.

Round two found the worst one. Past about two dozen states the canonical-table law exceeded the
recursion limit, and `elabCommand` logs rather than throws — so the Model was declared anyway,
carrying `sorryAx` and reading as complete, with everything downstream resting on that law. The same
trap as the `enum` deriving handler in `.2`. Generated declarations now carry the limits their length
needs, and the command reads its own axioms and refuses a Model whose law did not check. The same
round found that round one's stuck fix was wrong: stopping the walk at terminal states assumed a
Search that does not take a row out of one, and Search takes every row the table carries, so the
guard was hiding genuinely stuck states.

Round three: the state binder was never type-checked, so a function over another type was reported
as a machine too large to prove — a verdict about size on a type error. And the axiom guard inspected
whatever a name already held, so a machine reusing a declared name checked someone else's constant,
found it clean, and registered a machine whose declaration is a `Nat`.

### What this task does not deliver, and where it went

The protocol machine, the Limits accounting, the migration of 18 `model` declarations and 33
specimens, retiring the `model` spelling, and the byte-identical fixture regeneration are task `.16`,
split out on 2026-09-18 for the reason plan review round 1 split the former single `.3`.

`.16` also carries two things this task measured or accepted rather than solved. The design's own
protocol machine does not elaborate — 112 states over 11 actions proves in ten seconds, 112 over 23
does not prove at all, so the cost is the canonical-table law forcing catalogs as long as the machine
rather than the enumeration, and halving the retry bound does not help. And a state with two fields
of the same enum can name neither a start nor an end, which `DESIGN.md` section 3's three `Timeout`
fields reach immediately.

Implementer and reviewer are the same session — no second backend is installed in a cloud session —
so this owes a cross-model re-review before the spec's completion review, as `.1`, `.2` and `.3` do.
## Evidence
- Commits: ac8592630, a689e3e1d, feef0e569, 521e417bd, 952672cab, bbcfc588b, c3ebca85e, 7a25f681c, 446c57ed4, a40aefef8, f40e14a9c, 7629df026
- Tests: cd model && lake build, LEAN_NUM_THREADS=1 make lint-model
- PRs: