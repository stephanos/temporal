# Modalities: MAY, MUST and MUST NOT in a Model

Design study, 2026-10-03. Question: should a Model distinguish permitted, required and prohibited
behavior (`may`, `must`, `mustNot`), generate a per-state and a per-operation modality report, and
call "neither MAY, MUST nor MUST NOT" a specification hole? Grounded in `model/SEMANTICS.md`,
`model/umpire`, `model/temporal/standaloneactivity`, `.plans/{TEMPORAL_PATTERNS,SEMANTIC_PROTOCOLS,
DSL_OPERATORS,UMPIRE4_VISION}.md`, fn-112 (API Contracts) and fn-120 (.3 lint, .4 explorer). Section 6
has numbers from a read-only evaluator over `model/ir/activity.json` (scratch script, not checked in).
Nothing here edits a spec; proposed edits are in section 5.

**Verdict.** The three modalities are already in the IR, spread over four constructs. The powerful
part is the *report*, and it is derivable from the IR today with no schema change. A `may`/`must`/
`mustNot` authoring vocabulary adds no meaning the step function and the claim patterns do not have,
and `must(...).eventually` would be a second, misleading spelling of `leadsTo`. Recommended: make the
explorer and lint speak the modalities; keep the step function the one place behavior is written;
add one lint family (holes); add one refinement check for #ZOOM (the "must" half of modal refinement).

## 1. Theory, briefly, and what Umpire already is

- **Modal transition systems** (Larsen & Thomsen 1988): each transition is *may* or *must* (must ⊆
  may). Modal refinement: an implementation keeps every must transition and may drop may transitions.
  An Umpire machine table is an MTS with **may transitions only**: a row is a may transition with its
  results fixed; a disabled pair is the absence of a may transition. Nothing at the table level is a
  must transition. The "must" lives one level up, in claims over the table.
- **Interface automata** (de Alfaro & Henzinger 2001): actions are inputs the environment controls
  or outputs the component controls; a component need not be input-enabled, and a disabled input
  means "the environment must not do this here". In Umpire the party of an action (`caller`,
  `worker`, `system` timers, `internal`) is this split. A disabled *system* pair (timer, internal)
  reads correctly as MUST NOT: the server does not do it. A disabled *party* pair (caller `control`)
  reads as "the caller cannot try", which is false of an RPC: the caller can always send it and the
  server answers. For a party action, MUST NOT is an enabled row with a rejecting outcome
  (`Outcome.notFound` on a closed activity is exactly that), and a disabled pair is the Model being
  silent on the answer.
- **Deontic reading** (permitted / obligatory / forbidden): matches the owner's three words one to
  one. The deontic hole is a state-action pair that is neither permitted nor forbidden; in a
  closed-world table that cannot happen (every pair is enabled or disabled), so the hole must be
  defined by *how* the pair got its modality (section 3), not by whether it has one.

## 2. Mapping: what each existing construct already says

| Construct | Modality | Of what | Where it differs from the owner's reading |
| --- | --- | --- | --- |
| Enabled row (step function returns steps) | MAY | this class, in this state, with exactly these results | MAY with the results fixed; a `choose` (fn-120 A) names the alternatives, which are all MAY |
| Disabled pair (`Nil` / `disabled`) | MUST NOT (system action) or *silence* (party action) | the class never happens here | the table is closed-world, so every non-row is MUST NOT by construction, whether the author decided it or a `case _ => Nil` did |
| Hole row (`hole(...).reached`, unmatched `match`) | neither | declared unknown | the only honest "no modality" today; SEMANTICS Holes |
| `when c holds p` (same-step Property) | MUST | the result of a MAY row of class `c` | a postcondition on the result, not on enabledness: it is read only on steps that exist, so it says nothing where `c` is disabled and is vacuous for a class no verify asks (section 6: two of eight are false on rows nobody checks) |
| `holdsAcross`, `never(to).from(p)`, `once(p).keeps(x)`, `stays(p).unless(r)` (transition Property) | MUST NOT (a transition shape) | every step | restates what the table already forbids; its value is that it survives Model edits and carries a name and reason; it cannot say "class `c` is disabled" (a stutter of `c` passes `never(started).from(paused)`) |
| `leadsTo(from, to, within, under)` | MUST eventually | reaching `to` from `from`, within `n` steps, on paths the fairness assumption admits | this is the only MUST that is not a postcondition; bounded and under weak fairness, not TLA `~>` (TEMPORAL_PATTERNS 3: do not rename it `eventually`) |
| Monitor (`violated` after steps) | MUST NOT (a history shape) | paths | history-scoped MUST NOT; never disables a row |
| Assumption `fair` | makes a MAY into a MUST eventually | classes enabled forever on a path | weak fairness: the only mechanism that turns permission into obligation |
| Refinement (Machines 6) | protocol MAY ⊆ product MAY | every protocol row is carried by a product row or is a stutter | the "may may be removed" half of modal refinement; the "must is preserved" half is absent (section 4) |
| `because` on a `Step` | prose | an enabled result | there is no `because` on a disabled pair |

So MAY = rows, MUST NOT = non-rows plus the safety claims that pin them, MUST = postconditions,
progress and fairness. The report in section 3 is a join of these, per state and per class.

## 3. The specification hole, computed from the IR

**Inputs, all present in the IR today:** the table (SEMANTICS Machines 1-3), the branch decisions the
evaluator takes per pair (fn-120 R9's "why", whose last decision is a `match` case whose `pattern` is
`wildcard` or not), the action's `party`/`timer`/`internal` flags, `Property.when_class`/`when_action`/
`transition`, `Query.form` and whether its Scenario is `free`, `Progress.from/to`, and the lifted
named predicates (`terminalPhase`, `running`, `attemptHeld`: ordinary `functions`). No schema change.

**Hole kinds** (each a lint finding with kind, machine, class, state set, Scala position):

- **H1 `disabled-by-default`:** a pair whose empty result comes from a wildcard `match` arm (or an
  `if` whose condition names no field of the state). The author did not decide this pair; the
  language's fallthrough did. Reported per class × phase, never per state.
- **H2 `silent-rejection`:** a disabled pair of a party action (not `timer`, not `internal`) in a
  reachable, non-end state. An RPC can be sent here; the Model does not say what it answers. The
  fix is a row with a rejecting outcome and a `because`, never a wildcard. (A worker action whose
  delivery the system decides, `attemptStart`, is the known exception; the finding is accepted with
  a reason in fn-120 R7's file, or the Model separates dispatch from the worker's step as the
  admission Model does.)
- **H3 `unconstrained-result`:** an enabled pair of a class that no same-step Property names, no
  transition Property or monitor of the machine (or of its product, read through the refinement)
  constrains, and no progress claim reaches. MAY with nothing said about the result.
- **H4 `witness-only`:** a same-step Property asked only by `find` Queries over pinned Scenarios.
  It is a MUST on one path, not on the table; the table may contradict it elsewhere (section 6).
  fn-120 R5 already has "a Property no Query names" and "a verify whose Property never fired"; H4
  is the third member of that family.
- **H5 `must-not-pinned`:** a disabled pair of a system action that no `never`/transition Property
  pins. Lowest severity: the table forbids it, nothing says it is intended. Off by default.

**Per-state rendering** (explorer, `> state paused-1-unset-unset-unset`), from the real table:

```text
State: paused (attempts 1, no deadline)                     count: 24 states of this phase
attemptStart            MUST NOT   guard: phase != scheduled                    Model.scala:323
control-pause           ?          disabled by `case _ => Nil`                  Model.scala:386   H1 H2
control-unpause         MAY        accepted -> scheduled [statusScheduled]     no Property       H3 H4
control-requestCancel   MAY        accepted -> cancelRequested [...]            MUST cancelRequestedWhileStarted (find only)  H4
control-terminate       MAY        accepted -> terminated [...]                 MUST terminated (find only)                   H4
workerStop              MAY        accepted -> paused []                        no Property       H3
scheduleToClose         MUST NOT   guard: !running(phase) || scheduleToClose == unset
start-*                 MUST NOT   guard: phase != unstarted
pausedIsNotDispatched   MUST NOT   no step lands in started (verify, pinned path only)
```

**Per-operation rendering** (lint and `> rules attemptStart`): the same table grouped by class and
by the named predicates the step function itself evaluated, which is the owner's `rules(Poll)` table
derived rather than written (section 4c).

**Provenance.** None is required for H1-H5. If a reason on a disabled pair is wanted later (so an
accepted H2 can carry its text in the Model rather than in the acceptance file), the one field is an
optional inert `because` on the `list` expression node, the only node that yields `[]`; it would be
metadata like fn-120's choice names and enter no fingerprint. Not proposed now: the acceptance file
(fn-120 R7) already holds reasons, and a party-action rejection should be a row, which has `because`.

## 4. Author surface: meaning or sugar?

**(a) `may`/`mustNot` per state vs the step function.** The owner's `rules(Poll) { when(paused) ->
mustNot(deliver); when(dispatchable) -> may(deliver) }` is a step function organized by state class
with modalities instead of results. It cannot replace the step function: `may(deliver)` does not say
the next state or the facts, and a Model without those has no table, no evidence and no Case. As
sugar it is `match` over named predicates with `disabled` arms, which fn-112 R5 already gives
(`if s.phase.in(...) then accept(...) else disabled`). As a *claim* ("attemptStart is enabled in every
dispatchable state"), `may` adds meaning the IR cannot hold: no Property, monitor or progress claim
can see a disabled pair. That claim is worth having only as a lint kind over the table (H1/H2 cover
the practical cases), not as a Model declaration. Decision: no `may`/`mustNot` on the authoring
surface for steps; one way to write behavior, the step function; the rules table is a view.

**`must(p)` same-step** is `when c holds p` (100 of 171 claims). **`mustNot` of a transition shape** is
`never(to).from(before)` (fn-112.4). **`must(...).eventually`** is `leadsTo(from, to, within, under)`;
the word `eventually` is reserved for an unbounded TLA operator (fn-120 decision 4; TEMPORAL_PATTERNS
3 says explicitly not to rename `leadsTo`), so `.eventually` would be a second spelling with a wrong
meaning. Rejected. If a reader wants the deontic word, the generated view prints `MUST eventually`
beside a `leadsTo`; the source keeps `leadsTo`.

**(b) Coverage of a rules table.** With named guards, "do the `when` guards partition the state
space" is computable per class: for each state, the guards that hold; none = gap, two = overlap, and
a covered state whose guard's modality disagrees with the table = conflict. On `attemptStart` with the
owner's four guards (`paused`, `terminal`, `backingOff`, `dispatchable = scheduled`) over the 288-state
catalog: covered 192, **gaps 96** (phases `unstarted`, `started`, `pauseRequested`, `cancelRequested`,
all disabled by an explicit guard in the step function), overlaps 0, conflicts 0. Had `dispatchable`
been written `!terminal && !paused`, `backingOff` would overlap it with opposite modalities: a
conflict the derived view shows and a hand-written table hides until a Query fails.

**(c) Named guards as named state sets.** Yes: the guards are exactly the "status sets named on
vocabulary objects" of SEMANTIC_PROTOCOLS (`Product.terminal`, `Product.paused`, `Protocol.held`) and
the capability parameters (`Closable(terminal = …)`, `Pausable(paused = …)`, `Dispatchable(running =
…)`). They are lifted defs, so they are in `functions` and the explorer can evaluate them on every
state. The derived rules table groups by the predicates the step function *called* (its decision
trace names them with positions), and additionally by every capability parameter declared on the
machine. A protocol law then appears as a *row annotation*, not a step: the `paused -> mustNot(deliver)`
row is `pausedIsNotDispatched` (`never(running).from(paused)`) rendered beside the cells it pins;
`terminal -> mustNot(deliver)` is `closedIsRejectedUniformly`. Laws never write rows of the table;
they pin rows the step function wrote, and the view says which cells have no law on them (H3/H5).

**(d) Two renderings of one IR.** Per-state (what can happen here, with results, for the author who
asked "why") and per-operation (where this class is allowed, grouped by named set, with the laws that
pin each group, for review). Both are joins of table × decisions × claims; neither is authored.
**Lint prints the per-operation form** (holes aggregate by class × phase: the 168 H1 pairs below are
seven lines), **the explorer prints the per-state form** on `state` and the per-operation form on
`rules <class>`.

**Recommended author form.** Step functions as `match`/`if` over named state sets with `accept`,
`stay`, `disabled` and rejecting rows with `because`; no wildcard arm in a step function (H1 is a
lint finding; a wildcard that is intended is accepted with a reason); every claim through the fn-112
patterns and `leadsTo`. Then every modality in the report has a Scala line, and the report is the
owner's example with no new syntax.

**Refinement and #ZOOM.** Machines 6 checks the may half (protocol rows carried or stutter). The must
half of modal refinement, "every product MAY is realized by some protocol row from every state that
maps to its source", is not checked; today it happens to hold (section 6) by luck. Two checks to add
to the refinement result, both over the two tables and the map, no search: a product row that no
protocol row carries (a product MAY the implementation never exercises: a hole of the product), and
a product `leadsTo` read through the refinement as a protocol progress claim (mapped `from`/`to`, the
protocol's own fairness), which #ZOOM names as the thing state matching alone does not preserve.
MAY may be narrowed by refinement; MUST (postconditions through `through`, progress) must be re-read
on the refining machine, and the report should mark a product MUST the protocol has not been checked
against as `inherited, unchecked`.

## 5. Where it goes (proposed edits; none applied)

- **fn-120.3 (lint, R5).** Add kinds H1 `disabled-by-default`, H2 `silent-rejection`, H3
  `unconstrained-result`, H4 `witness-only`; H5 behind a flag. Acceptance: "each kind reports class,
  state set (by phase, never per state), position; fixtures per kind; the activity IR's first run
  lists 7 H1 lines, 7 H2 lines, 15 H3 classes and 8 H4 Properties, each fixed or accepted with a
  reason." Approach bullet: "the hole kinds join the table, the R9 decision trace and the claim
  index; no second evaluator."
- **fn-120.4 (explorer, R8/R9).** `state <key>` prints the per-state modality report (MAY with
  results and the Properties that pin them, MUST NOT with the guard at its line, `?` for H1/H2);
  `rules <class>` prints the per-operation table grouped by the predicates the decision trace called
  and by the machine's capability parameters, with gap/overlap/conflict lines. Acceptance: "both
  views are produced from the same table and trace; a fixture Model with a wildcard arm shows `?`."
- **fn-120 R13 (SEMANTICS levels).** One paragraph "Modalities" under Machines: a row is permission
  with fixed results, a disabled pair prohibition for a system action and silence for a party action,
  obligations are same-step Properties, progress claims and fairness; refinement narrows permission
  and does not by itself preserve obligation.
- **fn-112.** No new author words. `.3` acceptance gains: "a step function arm that is a wildcard is
  lifted as today; the pattern kind is in the IR already and lint reads it." Decision-context bullet
  "Modalities, 2026-10-03": `may`/`must`/`mustNot` rejected as authoring words (sugar for the step
  function, `when … holds`, `never/from` and `leadsTo`; `.eventually` conflicts with decision 4);
  adopted as the vocabulary of the generated views. Per the owner's constraint, any later sugar form
  lives in a `Syntax` file with lifter matching in `model/lifter/Syntax.scala` and an IR-equality
  fixture; none is proposed here because none lowers to anything new.
- **Capabilities spec (SEMANTIC_PROTOCOLS, being written).** R8's table view: "each law is rendered
  as the modality it pins on the cells of the per-operation table; the view lists cells of a
  capability's action that no law pins." Boundary: "a law pins rows; it never adds or removes one."
- **#ZOOM, later spec.** The two refinement checks of section 4 (uncarried product row; product
  progress read through). Not fn-120.

## 6. What the analysis reports today on `activityProtocol`

Catalog: 288 states (12 phases × 3 attempts × 8 deadline flags) × 22 classes = 6336 pairs; 1788
enabled, 4380 disabled by an explicit guard, **168 disabled by a wildcard arm** (H1), 0 hole rows.
Counts are over the catalog; each phase has 24 states.

- **H1/H2, all seven in `protocolControlStep` (`case _ => Nil`, Model.scala:386, 391):**
  `control-pause` in `paused`, `pauseRequested`, `cancelRequested`; `control-unpause` in `scheduled`,
  `backingOff`, `started`, `cancelRequested`. All seven are caller RPCs the server answers
  (`FailedPrecondition` "non-pausable state" / not paused; SEMANTIC_PROTOCOLS, Pause). The Model is
  silent. These are the owner's holes, and they are real.
- **H3, 15 of 22 classes** have enabled rows and no same-step Property: `attemptStart`,
  `attemptResult-failed-true`, `backoff`, `control-pause`, `control-unpause`, `scheduleToClose`,
  `workerStop`, and the 8 `start-*` classes. The two product transition Properties hold on all 1788
  results through `productOf` but are verified only on pinned paths of 3 and 5 steps.
- **H4, all 8 same-step Properties** are asked only by `find` over pinned Scenarios. Evaluated on
  every row of their class: `cancelRequestedWhileStarted` and `terminated` are **false on 120 results
  each** (the `notFound` stutter from the five terminal phases: `when control(terminate) holds phase
  == terminated` is not what the Model does there); `retryCompletes` is false on 69 of 72 (it pins
  one state, by design). The other five hold everywhere. Nobody sees this because no verify asks.
- **Rules-table coverage** for `attemptStart` with the owner's guards: 96 of 288 states in no rule
  (section 4b). The step function is total; the table as written was not.
- **Refinement, must half:** 22 product rows change state; all 22 are carried by some protocol row.
  Holds, unchecked.

Three things fall out before any new syntax: write the seven wildcard arms as rejecting rows with a
`because`; rephrase `terminated` and `cancelRequestedWhileStarted` as `when c holds (s.outcome == accepted
implies …)` or restrict their Scenario claim honestly; and give the two product laws a free-Scenario
verify so the MUST NOT is on the table, not on a path.
