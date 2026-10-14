---
satisfies: [R6]
---
# fn-85-model-side-effects-as-typed-actions-and.6 Refinement: refines and map checked by the forward simulation

## Description
A machine that declares `refines:` and `map:` is checked against the product machine by the bounded forward simulation inside `Umpire.ImplementationLink`, with same-named values mapped by default, fields the product lacks mapped to `hidden`, and the step mapping derived (a mapped row is a product step, a stutter when the mapped states are equal, or a rejection). A Property on the product machine is checked on the protocol machine's paths (R6).

**Size:** M
**Files:** `model/Umpire/Command/Syntax.lean` (`refines:`, `map:` keys on `machine`), `model/Umpire/Command/Authoring.lean` (derive the value and step mappings; build the `StepPreservation` witness by `decide` over the finite vocabulary), `model/Umpire/ImplementationLink/Language.lean` (a refinement entry point reusing `initialForward`/`stepForward` without the Feature-to-System naming), `model/Umpire/Search/Admission.lean` (a Property on the product machine admits against the refining machine's paths), `model/Umpire/ImplementationLink/Tests/*.lean`, the Nexus command specimens (`#guard_msgs` for the four rejections and a `#guard` on the checked refinement)
**Touches:** [model/Umpire/Command/**, model/Umpire/ImplementationLink/**, model/Umpire/Search/**, model/Temporal/Feature/Nexus/**]

### Approach
- The forward simulation's `initialForward`/`stepForward` are `Prop` fields; the command synthesizes the witness by `decide` over the finite vocabulary (the product and protocol state spaces from task .3 are finite and bounded), never by an authored proof; if `decide` is too slow on the section 3 machines, use `native_decide` as `Property.checked` does and record the measured time.
- `ImplementationLinkRequiredCoverage` demands a mapping or a Known Gap for every action, outcome and observation: derive those maps by name too, and reject an unmapped one at the refinement with its name.
- A refinement is not an Implementation Link (SEM-08); name the entry point `Refinement.check` and keep `checkImplementationLink` for Feature-to-System.
- Rejections: mapping to an undeclared product value; a protocol value with no same-named product value and no map entry; a row whose mapped states are neither a product step nor equal; `map:` without `refines:`.

### Investigation targets
**Required:**
- `model/Umpire/ImplementationLink/Language.lean:22,81,408-457,520-620,681-696,1159` — mappings, `StepPreservation`, the simulation, obligations, `checkImplementationLink`
- `model/Umpire/ImplementationLink/Application.lean:20-147,731-830` — status and result shapes
- `model/Temporal/Feature/Nexus/DESIGN.md` section 2.5 and the section 3 `map:` block
- `model/Umpire/Search/Admission.lean:180-205` — `admit`

**Optional:**
- `model/Temporal/System/Nexus/ImplementationLink.lean` — the only Implementation Link today (kept by fn-86; do not edit)

### Key context
- fn-86 later re-anchors the Feature-to-System link on the product machine this task checks; keep the product machine's element definitions addressable by name.

## Acceptance
- [x] `nexusProtocol refines: nexusProduct` with the section 3 `map:` checks with a derived step mapping; the witness is synthesized (no authored proof in the Model file), pinned by `#guard`
- [x] the four R6 rejections reject in place, pinned by `#guard_msgs`
- [x] `terminalIsFinal` declared on `nexusProduct` is checked on `nexusProtocol`'s paths and a Query over the protocol machine may `find:` it
- [x] `lake build Umpire.ImplementationLink.Tests TemporalModelTests` green; elaboration time recorded; `make lint-model` green


## Done summary

A machine declares `refines:` and `map:` -- the product machine, and a Lean function from this
machine's state to the product's, by the rule that replaced rows with step functions -- and the
command walks every row through the map and derives the step mapping `DESIGN.md` section 2.5 asks
for: a row whose mapped states are a product step is that step, a row whose mapped states are equal
is a stutter, and any other row rejects at the `map:` line naming the row, both readings and what
the product lacks. `nexusProtocol refines: nexusProduct map: productOf` checks: 1152 rows, of which
264 stutter (every schedule command, every retry out of `scheduled`, every backoff), pinned by
`#guard` on the read-back mapping.

### The witness

`Umpire.ImplementationLink.Refinement` is the forward simulation `StepPreservation` states, with
the two allowances a refinement needs: a step may stutter, and the product may record less than the
protocol step it carries. `RefinementMorphism` maps the setup and the state totally and outcomes and
facts by name, `RefinedStep` is the step-or-stutter disjunction over `authoritativeStep`,
`StutteringSimulation` reuses the `initialForward`/`stepForward` obligations, and `traceForward`
carries every admitted protocol trace to an admitted product trace through the states the product
sees (stutters dropped). `StepPreservation.toStutteringSimulation` embeds the exact simulation.
Over tables, `FiniteTable.refines` is the `Bool` the kernel decides, `TableRefinement.ofChecked`
reads it back as the obligations, and `TableRefinement.simulation` carries them to the kernels the
tables derive, through two lemmas added to `Umpire.Model.Table` (`machine_initialStates_mem`,
`machine_steps_mem`). The `machine` command emits `name.abstraction` (the morphism), `name.refinement`
(the report, evaluated and reported in place) and `name.refines : TableRefinement … :=
.ofChecked (by decide +kernel)`, and refuses a machine whose witness did not decide. No Model file
writes a proof; `#print axioms nexusProtocol.refines` is `[propext, Quot.sound]`.

Elaboration time: `Temporal.Feature.Nexus.Tests.Machines` went from about 40 seconds to about 75
with the refinement decided, and to about 150 with the two protocol Queries and the small pair of
machines the rejections are pinned on -- the Queries because a Search over the 192-state machine
explores thousands of candidates for a sequence of four (`search: 8192`).

### What maps by name, and what the design's machines needed

Outcomes and facts read as the product's value of the same key, a fact's constructor covering its
members the way an `evidence:` line does (`nexusOperationTimedOut (timeoutType := …)` reads as the
product's bare `nexusOperationTimedOut`). A fact the product does not name is hidden
(`pendingAttempts`); an outcome it does not name rejects. A product step carries a protocol step
when it reaches the mapped state with the mapped outcome and records a subset of the mapped facts:
a completion before the start records the Started event first, and the product's completion, which
records it alone, still carries it. The product action of the row's own name is preferred, else any
that carries it. `unscheduled` reads as `scheduled`, so the schedule command stutters and the
protocol begins where the product does.

The product machine gained one timer, `timeout`, out of `scheduled` and `started`: a deadline
firing is neither a stutter nor a step a product without a timer could take, and `DESIGN.md`'s
product rows reach `timedOut`. `#print axioms nexusProduct` is unchanged.

### A product Property on protocol paths

A refining machine carries the product state each of its states reads as in a state field named
after the product machine (`nexusProduct`, id `…state-field.nexusProtocol.nexusProduct`), computed
by the map over the members. `Umpire.Command.refinedProperty` reads a product Property on the
refining machine: a prior-state trigger and a fixed state become claims about that field, read
apart from the state the way `.4` made every field readable; outcomes, facts and trigger Actions are
the refining machine's values of the same name. The Property keeps its own identity. `query`
accepts a Property on the machine its Scenario's machine refines, rejects at `find:` naming an
Action, outcome or fact the refining machine lacks (`timesOut`, about the product's `timeout`, is
the pin), and rejects a lift over several instances. `terminalIsFinal` on `nexusProduct` is found
by `asyncCompletion` over a protocol Scenario and verified by `asyncCompletionHolds`; `attemptEnds`
on `attemptLoop` is found over `retryLoop`.

A stutter is checked like any step: a product transition claim that requires the state to change
fails on a stutter, which the spec records; the Nexus Properties are stutter-invariant.

### Rejections, pinned on a small pair

`retryLoop refines: attemptLoop` is the positive small specimen; on it, `#guard_msgs` pins `map:`
without `refines:`, `refines:` without `map:`, `refines:` naming no machine, a `map:` into another
machine's state (the map's type is checked at the line that named it), an outcome the refined
machine has no name for, and a row that is neither a step nor a stutter.

### What the protocol Query needed

Three things stood in the way of any Query over a structured-state machine, none of them this task's
rule. A Scenario names classed actions with their inputs (`complete (succeeded)`, keyed the way
`when:` keys them) and `starts:` by the phase, resolved to the one start state holding it. A
machine's setup constructor was named by its first start *key*, which a structured state punctuates
into something no catalog key admits; it is named by the start phase as the `starts:` line spells
it, which is the same name for a one-field state. And `checkFiniteTarget` refused a step out of an
end state -- a `model`-era authoring rule -- where the design writes completions that arrive after
the operation is over as `notFound`; the refusal is retired, and the Success test that pinned it
pins the table as noncanonical, the reason that remains. The admission diagnostic for a table that
was not admitted now names the table error.

### Gates

`lake build` green; `make umpire-check-testpilot-authoring`, `umpire-check-case-runtime-conformance`
and `umpire-check-goldens` exit 0; `go test -tags test_dep ./common/testing/testpilot/...` green;
`LEAN_NUM_THREADS=1 make lint-model` at the 163 baseline. `DESIGN.md` section 2.5 carries a dated
amendment with the map as a function.

Self-review: no second backend is installed in this cloud session, so this owes a cross-model
re-review before the completion review, as the tasks before it do.

## Evidence
- Commits: 2c45251
- Tests: cd model && lake build; make umpire-check-testpilot-authoring; make umpire-check-case-runtime-conformance; make umpire-check-goldens; go test -tags test_dep ./common/testing/testpilot/...; LEAN_NUM_THREADS=1 make lint-model
- PRs:
