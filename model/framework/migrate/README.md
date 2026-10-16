# The migration harness

For a lift-based system sweep, a DSL change ships its migration as a source rule, proven on a copy
of a feature before the sweep applies it to the real features. The rules are a system's and temporary (Temporal's:
`model/temporal/migrate`, [its README](../../temporal/migrate/README.md)); this harness is the
framework's and permanent.

## Native sample maps: fn-173 .13–.15

Before the first sample rewrite, retain all six complete sources as non-Scala text at one
committed baseline. The [fixture guide](../samples/testdata/native-migration/README.md) fixes
that baseline and the discovery exclusions; later pairs consume the same bytes unchanged.
Maintain its [state-map and answer inventory](../samples/testdata/native-migration/inventory.md)
as each pair migrates. Bank's map omits zero denominations/empty accounts; TwoPC's map keeps
all three managers and distinguishes received members from ever-sent prepared messages.

Preserve existing finite input domains and effect branch order. In particular, Bank's amount
domain remains 0..3 and insufficient funds precedes self-transfer; do not copy the older design
illustration's negative amounts or zero-send refusal. Use final record-field domains for native
Int inputs. Pin sparse cleanup, refusal/self-transfer, persistent messages and normal/mutant
commit behavior with literal states through ordinary binding dispatch and independent Properties.

Voting maps each acceptor to a vote set and max ballot (-1 for old None), keeping all
three keys. Preserve record fields and ascending ballot input domain 0..1. Remove only
the two normal inv effect conjuncts; keep refused Steps, remaining conditions and the
unsafe vote's othersAgree check. Its inherited max-ballot effect changes too. Keep the
invariant Property and record that inconsistent chosen sets can contain both values,
outside the old Decision image; decides retains v1 membership truth on that conflict.

For ReliableBroadcast remove the manual state/Inbox catalog assembly, retaining its
selected message domain and channel policies. Declare inbox(_.network, wire.network)
on Model and test exact ownership through public illustration attachment, including
the actual Derived Naive. Keep authored crash classification/reset maps and effects.
Label ordinary receive callbacks and supplied post-channel states explicitly: native
binding dispatch does not dequeue the Inbox. Channel eligibility/removal stays with
the interpreter. Record structural unordered ordering and repeated equal messages.

Map Kettle's offset reading r to signed Celsius r-40 (0/60/140 -> -40/20/100),
then read temperature directly and heat by one. Retain heat<100/failover>=100,
button guards, beep resets and the neverBeeps negative Property on Model.
Map Cache values/num to Int without changing its num<3 write guard, including
the effect-only mutant rebound. Each monitor keeps a separate immutable ordered
List[Observation] of original Event and after-L1 Option[Int], with writes first,
one append per relevant step, repeated observations retained and silent expiry absent.
Keep after-every-step sticky stale-read and strict increasing-write verdicts.
Pin the actual mutant initial miss path separately from a supplied seeded older-L2
example. Do not replace history with a summary or add a history cap.

All six maps and all 25 intended Query answers are now inventoried. Cache's old Seen
projection is many-to-one: repeated reads separate product states the old summary merges.
No product-graph bijection follows. Original .6 must reconcile graph scope, bounds and
erased history; Voting's guard/projection deltas and interpreted channel work remain open.

This sample source migration requires no task-specific Scalafix or new equivalence harness.
Focused native suites and lint have nonzero counts; snapshots and maps specify held proof inputs.
They do not close graph isomorphism, complete Query answers, refinement or checked arithmetic.
Do not run the lift-based harness below as evidence for this checkpoint.

## Native catalog retirement: fn-173 .16

Remove the native `StepTables` producer/`writeStepTables` command and producer-only tests, the
`MachineLaws` whole-state evaluator and `Refinement.unclosed`. Keep typed refinement targets and
declaration discovery. Migrate law callers to direct assertions of actual binding order/identity,
duplicate declarations, discovery/ownership and explicitly supplied callback results. Preserve
Hole object identity and ordinary-exception distinctions without scanning state domains or
returning law pass/skip results. Construction must stay inert even with throwing state catalogs
and model callbacks.

The retired walks supplied universal refinement totality/target membership, phase-role coverage
and closedness, ordered repeatability and state/outcome/fact membership, including unknown-row
handling. Original .3/R3/R4 retains every replacement obligation. Concrete examples supply none of
those universal guarantees. Source-state tables and their historical text seal are unavailable;
ChangeCheck/Makefile and old consumers remain held for reconciliation. Keep pure comparison,
RenameMap, lift comparison and synthetic negative controls, and keep Samples' lift-and-Go harness
held rather than disabling it. Private Query total and exporter carriers remain integration debt.

The six retained text fixtures, all six maps and all 25 intended answers stay unchanged. They
remain proof inputs under the many-to-one Cache history projection and the recorded Voting and
Broadcast distinctions. No replacement graph, complete Query answer or backend proof follows
from retiring the producer. This step needs no Scalafix, new checker or proof manifest.

## Native collection state: fn-173 .10

1. Use native `Set`, partial `Map`, `List` and `Int` fields. Let `Data[S]` supply structural
   comparison and `StateData[S]` prove supported state leaves; remove whole-state `Finite[S]`
   catalog scaffolding, including a local `Finite[Inbox[M]]` used only for that catalog.
2. Keep finite input/outcome/fact witnesses and each channel's selected message domain. Preserve
   exact `Inbox` owners through construction and derivation. String has `Data` for text use but
   direct/nested String state, functions and unsupported leaves must still refuse early, including
   through custom finite witnesses.
   Keep Inbox fields outside collections, including records stored inside collections; recursive
   state types also refuse the native structural proof.
3. Pin catalog and structural order independently: `Finite.of(2, 0)` retains action classes
   `[2, 0]`, while an unordered Inbox uses numeric type order `[0, 2]`. Record this representation
   delta without claiming IR key parity. Retain FIFO, capacity, duplicate and owner controls.
4. Keep guards and effects unevaluated during rule registration. Exercise callbacks explicitly in
   native tests; missing partial-map lookup throws a host exception. Use `get` or `getOrElse` when
   absence is expected. Registration supplies no overlap or reachability verdict.

Use focused native domain/schema/machine/Inbox/composition/refinement selections with positive
test counts. Native table/law producers are retired by fn-173 .16; no compatibility helper or
whole-state scan admits native state. The lift-based harness supplies no native checkpoint proof.
Full carrier retirement remains fn-173 .3.
The [author guide](../README.md#native-state) describes Scala `Int`'s 32-bit wrapping smoke boundary.
Checked signed-int64 endpoints, overflow holes, division by zero/minimum divided by `-1`, and
host constant-fold/initializer preservation or explicit refusal remain held with .4/.5. Native
arithmetic is no checking oracle; cross-interpreter parity and full R5/R6 remain open.

## Finite input domains and native guards: fn-173 .11

1. For a record with non-enumerated fields, pass typed bindings to the declaration:
   `.inputs[Request](inputField[Request](_.amount).in(domains.oneOf(2, 0)), ...)`.
   Bind each nonfinite field independently; remaining fields keep typed `Finite` fallback.
   Do not install a global numeric catalog or add domains after constructing the action.
2. Preserve authored domain order and defaults. Ranges expand in iteration order; collections
   remain individual values. New empty or duplicate `domains.oneOf` domains refuse immediately. Installing
   record domains runs no constructor/default getter; invalid omitted defaults refuse when used.
   Supply scalar domains with `.input[Int](name = "amount", domain = domains.oneOf(2, 0))`.
3. Keep a selected correlated record witness as `.inputs[Request](selectedWholeWitness)`;
   do not replace it with field projections' Cartesian product. Explicit whole-record selection
   combined with field overrides refuses. Ordinary selected witness metadata remains lazy.
4. Replace repeated per-class guards with whole-action `where((s, in) => ...)` only where the
   complete input type is retained. Keep pinned-class, heterogeneous and zero-input guards
   state-only. Preserve condition order, false-guard effect suppression and input object identity
   through facets and derivations. Binary callbacks read one whole record or scalar.
5. Pair valid declarations with malformed selector/domain/default and guard-shape refusals.
   Keep independent expected class values and `Step` results; test public author code against a
   fresh separately compiled framework jar as well as focused native dispatch tests.

[NativeInputs.scala](../samples/NativeInputs.scala) shows the final native surface alongside
expectation alternatives. This migration needs no source rewrite rule or copied Temporal proof.
It contributes native R6/R7 only: collection lifting/capture, checked numeric interpretation,
complete six-sample graph equivalence and backend proofs remain held with the original tasks.

## Native reach and Query configuration: fn-173 .12

1. Remove public Query and Witness `total` reads and writes, including infix `total n` clauses;
   there is no compatibility alias. Keep independent declaration pins for form, Property,
   Scenario, names and Limits instead of asserting a static total.
2. Use owner `reach: Limits` for machine/composition reach. The framework default is
   `frameworkReach` with steps 256, actions 0 and search 65536. Derived, built and replaced owners
   inherit source reach unless intentionally overridden. Queries use their Scenario owner even
   when the Property comes from a refined owner. Native reach consumption rejects negative
   steps/actions and nonpositive search; zero depth is valid. Ordinary Query Limits are unchanged.
3. State open acceptance with `.open(because = "reason")`; retain the original nonblank text.
   Select a primary with `.checkedBy(Checker.apalache)` or `.checkedBy(Checker.tlc)` when the
   default `Checker.umpire` is unsuitable. Open and primary assignments each reject repetition.
4. Add required confirmations with `.confirm(Checker.tlc, Checker.umpire)` after selecting the
   primary. Confirmation is one nonempty ordered assignment, without duplicates or the effective
   primary. It fixes even the default primary; `.checkedBy` after `.confirm` rejects.
5. Preserve `.live`, `.explore` and specialized Witness refinements in either unrelated modifier
   order. Pin the public metadata and verify construction, copying and inspection run no model
   callbacks. Use separate author/client compilations against a fresh native framework jar to
   prove typed checker controls and inaccessible total getters/writers for Query and Witness.

Use strict native warnings, including Werror/deprecation, and focused native suites with nonzero
test counts. This checkpoint proves declaration metadata only: no backend availability check,
coverage admission, search result or trace replay follows from checker selection. Full carrier and
engine retirement remain fn-173 .2/.3, and backend/replay obligations remain .7–.9. The lift-based
harness and system migration are separate held work.

## Native durability and assumptions: fn-123 .10

Replace an ordinary per-field crash reset with one retained classification and a generated binding:

```scala
// Before, inside Rules:
on(fault.restart):
  always ~> ((s: State) => List(Step(Outcome.internal, s.copy(pending = false))))

// After, inside the machine:
durability(durable(_.stored), ephemeral(_.pending, resetTo = false), durable(_.remaining))
// Inside Rules:
crashes(fault.restart.budgetedBy(_.remaining), Outcome.internal)
```

Use `inMemory(_.field)(from -> durableFallback, ...)` for per-value resets. Cover each field exactly
once, retaining durable budget fields. The generated budgeted crash disables on exhaustion and
consumes its own budget once; do not add an authored decrement around it. Keep a non-per-field crash
authored and use `rebind` for a deliberate override. Preserve existing guards, outcome, facts and
enabledness when adding classification to an authored crash, as ReliableBroadcast does.

Retain authored assumption occurrences in their original order. Remove a hand-authored storage-loss
assumption when the active binding now generates it; restrict away that fault to remove only its
generated contribution. Validate through completed native acquisition, then pin reset and budget
results against ordinary independent functions. Use focused native suites and fresh strict external
author/client compilation; no Scalafix, Temporal sweep or IR/Go parity is proved by this recipe.

## Native inputs: fn-165

Migrate framework authors directly to the native surface in [Framework authoring](../README.md):

1. Classify each declaration from its actual inputs. Multiple fields become one case-class record
   declared with `.inputs[In]`; a genuine scalar uses `.input[A](name = "label")`, even when `A` is
   a product or tuple. Preserve exact labels, field order and selected finite witnesses.
2. Put only previously declared defaults on record constructor parameters or the scalar's
   `default = value`. Required inputs remain required; never choose a catalog's first value.
   An explicit `None` is a default. Record omissions fail compilation; required scalar omissions
   fail when the empty class is requested, naming the missing input.
3. Replace field-argument class calls with a record value or checked named arguments. Supply scalar
   values directly and examples with `.example(value, text)`. Helpers taking `Action[In]` can
   forward values; retain `RecordAction[In]` or `ScalarAction[A]` for named or empty native calls.
4. Adapt input-reading whole-action effects, rules, facets and derivations to `(state, in)`, reading
   fields from that one input. State-only callbacks can ignore inputs; keep zero-input,
   class-specific and heterogeneous-target callbacks state-only.
   Capture structural field evidence with `action.field(_.field)` or `inputField[In](_.field)`
   at the concrete record boundary, then forward the `InputField[In, A]` descriptor.
5. Preserve independent core oracles: compare exact class restrictions with independently ordered
   `Class(decl, List(...))` values and keep separately authored predicates and expected steps.
   Remove standalone tokens, appended declarations and positional arity APIs rather than keeping
   compatibility aliases. Pair retirement refusals with valid native controls.

Run the native compile/test and lint recipe in the author guide, retaining nonzero executed tests
and command exit statuses. This framework caller migration needs neither a Scalafix rule nor a
copied-mini lift proof. It proves native declarations, callbacks and restrictions only; capture,
exact IR equivalence and Temporal's real-feature sweep remain separate held obligations. Do not
restore disabled Temporal sources or use the lift-based harness below as native proof.

## Native expectations: fn-164

The final native author surface is documented in [Framework authoring](../README.md). Replace
same-step `property.holds(predicate)` callers with `property.expect: check(predicate(after))`;
replace transition `property.holdsAcross(predicate)` with
`property.across.expect: check(predicate(before, after))`, laid out as colon blocks. Inline
conjunctions become ordered `check`/`records` requirements; explicit alternatives use `oneOf`
with `branch` blocks. Preserve action restrictions with `after` for an exact class or
`afterAction` for a whole action, and preserve explicit contextual parameter names.

Keep independent core predicate oracles independent. Public finishers and renamed/deprecated
compatibility routes are retired together. A reused composed predicate must retain its exact
field/machine evidence through the final `check` forwarding form, including attachment rejection
for invalid selectors in unreachable branches. Reordered members and repeated uses of one machine
are distinct cases, not interchangeable fact names.

The native compile/test recipe in the author guide excludes capture/exporter and system inputs.
It proves native truth values, short-circuiting, exceptions and attachment checks only. Exact IR
and exporter obligations in fn-164 .3/.7–.9 remain open, and the real-feature sweep stays held;
the [system migration guidance](../../temporal/migrate/README.md) is the separate system-owned
recipe. Do not run the lift-based harness below as evidence for this native checkpoint or restore
disabled Temporal sources to make it run.

## Optional native Property illustrations: fn-153 .7

Before: keep a completed final `expect` value as an ordinary Property. After: retain its inferred
`SameStepProperty[S, O, F]` or `TransitionProperty[S, O, F]` type and append
`.illustrate(label, because, IllustrationKind.positive, actionClass, step)`; use
`IllustrationKind.counterexample` for an illustrative violation. Transition calls supply the
before-state between the class and step. Keep metadata outside the expectation requirement block.
Widen to `Property[S]` after attachment when an ordinary consumer needs it.

Add `inbox(_.field, channel)` once inside the owning individual machine for each encountered Inbox;
use full nested selectors, optional `.get`, or `inboxCase[StateCase, Message]` for a root enum case.
Keep the actual channel reference; names/configuration or the supplied Inbox do not prove ownership.
Do not replace selected finite domains, enumerate state catalogs, call modeled functions to obtain
evidence, or sort/repair supplied deliveries. No-illustration declarations need no new annotations.

Keep independent predicate and callback tests. Successful attachment proves one-value admission
and metadata only, including hypothetical unreachable states and unselected owner actions. It does
not classify the illustration, lift it, render checked documentation, establish identity parity or
finish the pilot. Use the focused native tests and separately compiled public author/client proof;
no source rewrite rule or Temporal copy is needed for this checkpoint.

## Lift-based migration proofs

```sh
FEATURES=activity_mini model/framework/migrate/prove.sh <runner main class> [<target>]
```

`prove.sh` copies the sources the feature compiles from, as the change-scoped check selects them
(`framework.check.selectedSources`, `ModelSources.scala`), into a fresh `.flow/tmp/mini-migration.*`
directory with the API jar, and lifts the copy. It then runs the rule's proof runner on the copy's
`<target>` (a path of the repository, mapped into the copy; by default the feature's protocol
level), lifts the copy again from the copy's own directory, so positions and source checks read the
copied sources, and compares the two lifts with `framework.check.equivalence.compareLifts`
(`model/framework/check/equivalence/Lifts.scala`).

The lifts must be the same byte for byte, IR files and waivers, unless the runner declares a delta
in the copy's `delta.txt`. A delta names what the rewrite may change and nothing else:

```
# positions in Product.scala at or below line 23 move down by 4 lines
moved model/temporal/features/activity_mini/product/Product.scala 23 4
# an identity the rewrite renamed, as a rename map's entry
temporal.features.activity.mini.protocol.oldName temporal.features.activity.mini.protocol.newName
```

Every entry must explain a difference, and every difference must be explained. `RULES` and
`LIFTER` name the system's rule build and lifter halves (Temporal's by default).
