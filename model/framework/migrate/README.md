# The migration harness

A DSL change ships its migration as a source rule, proven on a copy of a feature before the sweep
applies it to the real features. The rules are a system's and temporary (Temporal's:
`model/temporal/migrate`, [its README](../../temporal/migrate/README.md)); this harness is the
framework's and permanent.

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
