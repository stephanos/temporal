# TLC second opinion

Decision, 2026-10-19. **Yes.** Build the TLC backend under fn-212. Model authors
need a backend for larger state spaces as the Rust corpus grows; retain the native
checker for fast local checks and reference agreement on tractable Models. Use a
**build-time `syn` pass over authored source**, rather than recording tokens in
`#[machine]`. Keep fn-212 and its dependent fn-211 open.

The owner made the scaling requirement explicit on 2026-10-19:

> we'll need TLC for scaling our models; our own model checker is a simple, non-scalable version that won't hold

This requirement makes TLC a planned execution backend, beyond an optional second
opinion. Today's small corpus supplies translation and agreement controls; it
does not bound the Models the framework must eventually support.

This assessment uses the Rust tree at `654d6392449005e9cc0094128b0b64b1c5147ecd`,
after the stock WASM/Go proof, explicit-fairness progress and activity boundary
cleanup. [README](../README.md) describes the live workspace. The Scala tree in
`model/archived/` and the older Scala/IR paths in the architecture documents supply
design history, not evidence of implemented Rust capabilities.

## What exists

| Current source | Evidence relevant to a second checker |
| --- | --- |
| [Attribute Kettle](../umpire/dev/fixtures/fixtures/sample_kettle/model.rs) and [manual Kettle](../umpire/backends/native/tests/fixtures/sample_kettle_trait/kettle.rs) | Separate authoring implementations of the same small machine. The attribute [report](../umpire/dev/fixtures/fixtures/sample_kettle/expected/report.txt.snap) records 81 starts, 283 reached states, 424 edges, no cut and four verified claims. |
| [Kettle progress questions](../umpire/dev/fixtures/fixtures/kettle_progress/model.rs) | The [golden](../umpire/dev/fixtures/fixtures/kettle_progress/expected/report.txt.snap) distinguishes 81/141-step room/cold bounds, an unrestricted cycle, a filtered dead end, a missed deadline and an unknown budget cut. |
| [Fairness graph](../umpire/backends/native/tests/fixtures/progress_self_loop/model.rs) and its fixtures | Explicit fair actions cover continuous versus intermittent enabledness, excluded fair actions, multiple choices, self-loops, multiple starts and target barriers. [Native progress](../umpire/backends/native/src/progress.rs) searches target-avoiding components and checks weak-fairness obligations. |
| [Activity Model](../temporal/feature/activity/model.rs) | One real Temporal slice. Its [report](../temporal/feature/activity/expected/report.txt.snap) records six states and seven edges. It has atomic cancellation and fatal failure, with no retry branch or progress registration. Its terminal-preservation tests exercise the antecedent directly and reject a hypothetical reopening; the verified edge report alone is vacuous at terminal states. |
| [Choices fixture](../umpire/backends/native/tests/fixtures/choices/model.rs) and [Machine trait](../umpire/author/dsl/src/machine.rs) | The engine already permits several ordered choices, outcomes and facts, including repeated destinations. The attribute macro currently emits unit outcomes/facts and only changed-state edges. |
| [Brute-force oracle](../umpire/backends/native/tests/oracle.rs) | A separate finite-universe reference compares reached sets, rows, safety/progress verdicts and witnesses over 256 generated worlds per property-test run, including exclusions, explicit fairness and numeric bounds. It still executes Rust `Machine::step`. |

At the pinned tree above (`654d639`), the 24 shared fixtures were chiefly Kettle and checker
controls, not 24 independent product Models. The current native DSL/checker has no refinement-checking API;
this decision does not claim the parent spec's anticipated refinement coverage
has arrived. No current Rust Model here declares a refinement requiring TLC.

## What TLC buys

The native [explorer](../umpire/backends/native/src/explore.rs) retains states, completed
action rows, transitions and predecessor information in memory. Its progress
checker also builds graph structures for claim evaluation. TLC offers a separate
engine with [disk-backed reachable-state storage](https://nightly.tlapl.us/doc/model/tlc-options-page.html),
providing a scaling route without expanding the simple native checker into a
second general-purpose backend. This is an architectural reason to implement it,
not a benchmark proving TLC faster on the current Models. TLC still needs a
finite admitted configuration, and state-space explosion remains possible.

A source lift followed by TLC would interpret guards, updates and predicates
through a different evaluator and explore through another engine. It could expose
a bad macro expansion or native exploration/progress defect that the existing
Rust oracle misses. Comparing decoded reachable **sets**, rather than just Kettle's
283-state count, would catch missing and spurious states. Comparing action/choice
rows and replaying counterexamples would additionally catch relation errors that
leave the reached set unchanged.

TLC also supplies an independent temporal-property implementation. Its
[documented specification form](https://dl.tlapl.us/tlatoolbox/doc/model/overview-page.html)
admits temporal properties and explicit fairness. That is useful validation of
the native fair-cycle checker; it does not fill a missing `LeadsTo`, dead-end or
step-bound facility in today's code. Broader temporal formulas would require a
new authored claim surface before TLC could answer them from these Models.

These benefits require an independent reading of source. Exporting the native
checker's already computed graph to TLC would share the very transition
interpretation being checked. Source translation still shares the author's
omissions, and agreement would establish neither correctness of the Temporal
implementation nor support beyond the admitted subset and completed graph.

## Machinery and semantic work

`umpire-tla` would own source admission, a typed internal tree, local name/type
resolution and translation of ordered mutations into one next-state relation.
Kettle already needs guarded enum matches, `+=`, multiple sequential assignments,
`check!`, before-state postconditions, starts and assumption builders. Activity
adds `matches!` and an enum helper method outside `impl Model`. The trait fixture
adds iterator-built starts and closure registrations; the choices fixture needs
outcomes, facts and nondeterminism. A parseable Rust expression is not evidence
that the proposed small subset can translate it.

Every admitted integer/collection operation needs an explicit semantic mapping.
An overflow, predicate hole or unsupported expression must retain its defect or
unknown/refusal status instead of being pruned into a passing TLA+ model. Future
subset warnings constrain otherwise valid Rust authors, and refusal diagnostics,
the method table, SSA lowering and source positions need maintained controls.

There is also a path-semantics obligation in the current proposal. In
`kettle_progress`, `room_never_switched_off` uses the original `boils` trigger
`heating_on`, target `beeping`, and exclusion of `User::DepressButton`, with no
explicit fair action. Native maximal emitted-edge paths must advance from
temperature 20 through Heat and Failover, so the golden verifies with longest
distance 81. After PressButton, plain TLA+ `Init /\ [][Next]_vars` can instead
stutter forever with heating on and beeping false. The empty list of explicit
`WF_vars(A)` conjuncts rules out none of that behavior. TLA+'s
[liveness tutorial](https://conf.tlapl.us/202110/StephanMerzTLA.pdf) illustrates this
infinite-stuttering distinction.

This is an unresolved translation obligation, not a native correctness defect.
The implementation design must specify how it represents maximal emitted-edge paths,
finite dead ends and counting only Model transitions for numeric bounds while
preserving `fair(A) -> WF_vars(A)`. It must distinguish an authored state-preserving
edge from idle stuttering. Adding implicit action fairness or changing native
answers to make TLC agree would change the Model contract. Reachable-set agreement
alone cannot detect this temporal mismatch.

The preferred implementation candidate is an observer encoding. Toggle a private
bit on every emitted Model edge, including authored self-loops, and hold it fixed
on idle stuttering. Define maximality as either infinitely many toggles or eventual
permanent residence at a scoped dead end. Check the progress property conditional
on that maximality predicate. Emit only the author's explicit `WF` conjuncts;
maximality defines which paths the native question quantifies over and grants no
individual action scheduling guarantee. Count bounded obligations with a capped
observer updated only on Model edges. Represent terminal completion explicitly
for TLC's deadlock check, without counting completion as a Model edge. Project
observer fields and completion steps out of state agreement and replay.
Reserve `__umpire_edge_tick`, `__umpire_pending` and `__umpire_steps` for this
candidate, refusing collisions with authored names. Decode by removing exactly
the declared observer fields and deduplicating projected Model states before the
283-state comparison; never discard arbitrary fields by a name-prefix rule.

This is a design candidate requiring the controls below, not a verified encoding.
For scoped claims, preserve unrestricted reachable prefixes before the trigger
and apply action filtering to the obligation suffix, as the native checker does.
An alternative is an explicit-edge temporal monitor with a progress-checking
algorithm over the TLC-produced graph; it can preserve maximality directly, but
reuses more checker machinery and gives less independent liveness validation.
Plain stuttering temporal properties and automatically adding `WF(Next)` or
`WF` for internal actions are rejected resolutions under the authored-fairness
contract. Prefer the observer route and validate its projected semantics before
expanding the supported subset.

`umpire-tlc` would additionally own pinned-jar admission, subprocess limits and
cleanup, complete dumps, typed decoding, trace replay, per-property answers and
deterministic agreement goldens. The tools' [integration guide](https://github.com/tlaplus/tlaplus/blob/master/USE.md)
requires a Java process and documents graph/trace exports. The repository already
pins a JDK in `mise.toml`; the new cost is making Java and a pinned jar part of
this backend's execution path. The current Rust [model gate](../../Makefile)
does not run a JVM. No TLC runtime or gate overhead was measured for this note.
Batching or `UMPIRE_BACKENDS=require` could protect the 30-second gate target,
but an opt-in check gives no agreement evidence when it is not run.

## Lifting source

Choose a build-time `syn` pass. It can read both current Kettles and the manual
`Machine` controls through one source route, retain file locations, and leave
`#[machine]` registration-only. The existing [fixture discovery entry point](../umpire/dev/fixtures/build.rs)
already identifies selected source roots; lifting should track those inputs and
their admitted helper dependencies as Cargo build inputs. `syn` is already a
workspace dependency. Reading source does not imply that every manual trait body
belongs to the liftable subset.

Token recording has a real advantage for the preferred attribute surface. The
[macro](../umpire/author/dsl/macros/src/machine.rs) already receives its inline module's
tokens, so capture avoids rediscovering that module. It still supplies no resolved
types and cannot capture the unannotated manual Kettle. Extending it with source
metadata publication and a separate manual-source path would add coupling and two
coverage routes. The build-time choice pays for explicit module/import/`cfg`
admission once, outside the native/WASM registration path. Unsupported resolution
must refuse by location; neither source choice licenses a second Rust compiler.
Keep the small typer and fail-closed subset in the proposed scope, and reassess
`ra_ap_hir` only against an actual unsupported Model.

## Implementation and assessment points

Proceed with [fn-212's lifting task](../../.flow/specs/fn-212-tla-lifting-and-the-tlc-backend-for.md).
Build the source reader, small typer and internal lifted tree in `umpire-tla`,
then the TLA+ module/config emitter and `umpire/backends/tlc` runner. Keep each
supported source dependency explicit. Warn by location on the subset boundary
and refuse emission for unsupported behavior; never silently omit a declaration.
Keep the lifted tree internal and review emitted `.tla`, `.cfg` and `tlc.txt`
goldens through the shared fixture consumer registry.

Before claiming liveness agreement, settle the maximal-path encoding above with
controls for no-fairness acyclic progress, finite dead ends, explicit fair cycles,
intermittent enabledness and authored accepted/rejecting self-loops versus idle
stuttering. Contrast a finite dead end with idling at a state that has enabled
actions. The encoding must preserve
authored fairness separately from path maximality. Numeric bounds must count
Model transitions. Match checked integer behavior and named assumptions before
admitting the corresponding fixture. Preserve expected termination while leaving
TLC's deadlock check enabled; the activity's terminal states are deliberate.

Use Kettle's complete 283-state set, each property and its negative control as the
first agreement proof, followed by the shared fixtures the subset admits and the
activity slice. Decode values and replay backend counterexamples through the
native Model. Preserve action identity, choices, outcomes and facts where used.
A missing/mis-versioned jar, unsupported expression, incomplete dump or disagreed
answer must fail with its reason. A native budget cut is unknown and cannot
certify TLC agreement. On larger Models, native reference checks may cover only
an explicitly comparable smaller configuration; report that scope separately
from TLC's result.

Measure the actual backend and gate before choosing batching versus a required
backend gate using `UMPIRE_BACKENDS=require`. Keeping JVM startup outside the
fastest local edit loop is compatible with required backend verification; a run
that omits TLC must report no TLC evidence. The current corpus provides no
TLC/native scaling measurement, so the first larger Model must record state/edge
counts, memory, time and completion scope for both engines where feasible.

Keep [fn-211](../../.flow/specs/fn-211-apalache-backend-for-umpire-in-rust.md) open
over the shared emitter. Its bounded symbolic route has a separate implementation
and agreement contract; this TLC decision claims no delivered Apalache result.

Reassess coverage and scaling against the corpus after fn-221's rejecting
outcomes/`Phased` and fn-213/fn-215's ModelSuite/`path!` work. These are planned,
so this note counts none as implementation evidence. That assessment should name
the first larger Temporal claim, supported dependency closure, measured limits,
required backend runs and any semantic gaps. It refines the required backend's
scope and operation rather than reopening the owner's decision to provide TLC.

## Implemented proof scope

Fn-212.2 implements the selected build-time source reader and pinned real TLC runner.
The observer proposal above now has executable controls for maximality, scoped dead ends,
explicit weak fairness, self-loops with outcomes/facts and real-edge bounds. The admitted
shared corpus compares complete sets and verdicts, including both 283-state Kettles and
negative trace replay. [DESIGN.md](../DESIGN.md#tlc-source-backend-fn-212) records the
supported source forms and remaining refusals. The standalone typed Module entry point
does not explore natively; a cut native reference supplies no agreement evidence.

This proof remains source-dependent. New `Phased`, rejection and imported law syntax
must either be admitted with new agreement evidence or retain its located refusal;
earlier activity evidence cannot certify a changed Model. The required backend run is
explicitly selected with `UMPIRE_BACKENDS=require`, and an omitted run reports no TLC
evidence. No larger-Model scaling benchmark or Apalache result is claimed here.
