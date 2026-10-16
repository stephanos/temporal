# Retained native migration inputs

All six `*.scala.txt` files here retain the complete source bytes at the single committed
pre-rewrite baseline `53f05f9bce8434ee069ec7717212700217358bbc` (fn-173 .13 admission).
Each file corresponds to `model/framework/samples/<same-name>.scala` at that commit:
Bank, TwoPhaseCommit, Voting, ReliableBroadcast, TwoLayeredCache and Kettle. This includes
ReliableBroadcast's already integrated fault and durability declarations. Later pairs consume
this same bundle unchanged; their implementation tips do not replace its baseline.

The `.scala.txt` suffix is intentional: recursive Scala source discovery admits `.scala`,
and `framework.check.Samples` discovers direct `.scala` children of `samples`, excluding
`.test.scala`. These retained texts are neither production sources nor additional sample roots.
They survive independently of the retired step-table producer and migration tooling.

The six files were compared byte for byte with the committed originals before any sample
rewrite. To repeat that check from the repository root, for each of the six names run:

```sh
git show 53f05f9bce8434ee069ec7717212700217358bbc:model/framework/samples/Bank.scala | cmp - model/framework/samples/testdata/native-migration/Bank.scala.txt
```

[Inventory](inventory.md) specifies representation maps, deliberate deltas and intended answers.
These are inputs to the held fn-173 .6 proof. Retention, map descriptions and handwritten native
examples establish no reached-graph isomorphism, complete Query answer, refinement, checked-int64
interpretation, operation lifting or backend equivalence. The full sample lift-and-Go check is
held until its native-state integration is implemented.
