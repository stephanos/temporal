# The migration harness

A DSL change ships its migration as a source rule, proven on a copy of a feature before the sweep
applies it to the real features. The rules are a system's and temporary (Temporal's:
`model/temporal/migrate`, [its README](../../temporal/migrate/README.md)); this harness is the
framework's and permanent.

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
