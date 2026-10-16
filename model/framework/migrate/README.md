# The migration harness

For a lift-based system sweep, a DSL change ships its migration as a source rule, proven on a copy
of a feature before the sweep applies it to the real features. The rules are a system's and temporary (Temporal's:
`model/temporal/migrate`, [its README](../../temporal/migrate/README.md)); this harness is the
framework's and permanent.

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
test counts. Keep table/law compatibility helpers finite-only; do not run whole-state tables or
the lift-based harness as this checkpoint's proof. Full carrier retirement remains fn-173 .3.
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
