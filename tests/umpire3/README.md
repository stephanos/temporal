# Umpire3

Umpire3 is an independent Lean-first semantic modeling and implementation-conformance system. Lean
owns product meaning, selected distributed-system behavior, executable exploration, and refinement.
The Go runtime consumes versioned experiments and never reimplements their state machines.

Run the complete focused gate from the repository root:

```sh
make umpire3-check
```

`make umpire3-integration` additionally runs the exported Nexus cancellation trace through a real
Temporal matching/frontend task exchange. It checks both the conforming worker and a controlled
faulty worker that reports stale success; the resulting cluster response is the checkpoint evidence.

Use `make umpire3-gen-experiment` after an intentional semantic-source change and
`make umpire3-gen-api` after a selected Temporal protobuf descriptor changes. Review generated
diffs as semantic changes; do not hand-edit generated files.

## Authoring a model

1. State the user-visible contract in `model/Temporal/Product` without task, shard, persistence, or
   ownership mechanics.
2. Add the smallest threatening mechanism under `model/Temporal/System` and name every assumption.
3. Represent executable transitions with a proved equivalence to the relational `Step`.
4. Carry product traces in system transition results, then prove the system-to-product relation.
5. Add permitted examples, forbidden examples, and a mutation that fails a theorem or emits a
   counterexample.
6. Export only well-formed traces with bounds, assumptions, theorem identity, source hash, and tool
   version.
7. Add a Go adapter only for stable semantic actions and normalize evidence independently of the
   expected checkpoint.

Proof, bounded exploration, and one implementation run are different claims. Missing identity,
causality, ordering, capability, cleanup, or source evidence must remain unsupported or
inconclusive.
