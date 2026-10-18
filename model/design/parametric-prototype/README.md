# Parametric Bank probe

Run from any directory, with the repository's `mise` tools and existing
`model/build/ir-scalapb.jar`:

```sh
bash model/design/parametric-prototype/run.sh
bash model/design/parametric-prototype/run.sh refusals
```

The first command runs only `parametricprototype.ParametricBankTest` against the product
`framework.checker.Checker`. The second requires the desired generic forms to fail compilation
with their observed schema diagnostics. Both return nonzero on an unexpected result. Logs and
fresh refusal compilation workspaces are retained under `.flow/tmp/fn1941/`.

`project.scala` pins Scala 3.9.0, JVM 27, munit 1.2.0, ScalaPB 0.11.20 and all current native strict
flags, including `-Werror`, `-Wsafe-init`, strict equality and `-Yretain-trees`. The script lists
DSL, realization and checker main sources explicitly; it does not include their tests, export,
Temporal sources or `model/project.scala`. Conversely, the normal native framework roots do not
discover this design directory or the refusal snapshots (`.scala.txt`). No custom explorer or
prototype framework implementation is used.

The runnable alternative stores address indices, not `Addr`, in its state and input record.
For the ordered external enum `Addr`, the representation map replaces each address by its index
in the selected `Finite[Addr].values`. `holder` is mapped by that same function; public `balance`
maps back through it. `Denom`, `Balances` and `Send` belong to the `Bank` companion. The input
domains contain exactly the selected indices and amounts `-1..supply+1`. Transfers retain the
sample's sparse zero cleanup, rejection ordering, self-send and signed-balance bounds.

For the two-address world, each denomination has `supply+1` distributions; the independent
`opening` start var has `supply+1` fixed values. Thus the expected reached-state counts are
`(supply+1)^3`: 8 and 27 for supplies 1 and 2. Public checker completion at exactly that budget,
and a cut one state below it, test those counts without accessing a private graph or substituting
an explorer. `opening` is deliberately inert: it isolates parameter-dependent start registration.

This is partial runtime evidence. It does not prove the intended direct generic field types,
instance-nested vocabulary, or automatic instance names work. Those refusals and the concrete
production alternatives are recorded in [the verdict](../parametric-models.md). The index
representation is a probe fallback, not the proposed authoring API for .2.
