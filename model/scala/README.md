# model/scala

A Scala 3 implementation of the Umpire model layer, beside the Lean one in `model/lean/` and the Go
one in `model/go/`. It ports the same Models, checks them against the same Lean evidence, and adds
Stainless proofs over the Nexus caller's step functions. The plan is
[`.plans/UMPIRE_SCALA.md`](../../.plans/UMPIRE_SCALA.md); the results are in [RESULTS.md](RESULTS.md).

## Run

```sh
model/scala/run.sh            # generate protos if stale, compile with -Werror, test, prove
model/scala/run.sh --views    # also render the views and diff them against goldens/views
model/scala/run.sh --no-prove # skip Stainless for a quick loop
```

The tools are `scala-cli` (installed by mise), JDK 21, `protoc` (mise), and Stainless, which
`tools.sh` downloads into `UMPIRE_SCALA_TOOLS` (default `/tmp/umpire-scala-tools`).

## Layout

The framework and the Temporal Models are separate trees. `umpire/` knows nothing of Temporal's
features and builds on its own; `run.sh` compiles it alone first, so a dependency from the framework
on a Model fails the gate.

| Path | What it holds |
| --- | --- |
| `project.scala` | Build directives: Scala version, `-Werror`, dependencies |
| `umpire/` | The framework, package `umpire`: finite domains, actions, machines, tables, refinement, composition, claims, search, coverage, sets, canonical JSON, fingerprints, lowering |
| `umpire/prelude/` | The runtime half of the kernel prelude: `Steps` is `scala.List` |
| `umpire/caseproducer/` | The Case producer, over the generated Testpilot Java classes |
| `umpire/views/` | Table, diagram, summary and diff views of any Model |
| `temporal/worker/`, `temporal/nexuscaller/`, `temporal/standaloneactivity/` | The three Models, package `temporal.*`, and the Nexus realization |
| `temporal/nexuscaller/kernel/` | The Nexus domains and step functions, in the Scala subset Stainless reads |
| `temporal/views/` | Which views of the Temporal Models are rendered |
| `temporal/test/` | Pins, parity against the Lean dumps, Case bytes, kernel agreement, views |
| `proofs/umpire/` | The Stainless half of the prelude |
| `proofs/temporal/` | The Nexus lemmas |
| `goldens/views/` | The rendered views |
| `scala.sh` | `scala-cli` with an exit code that reflects `-Werror` failures |
| `gen-proto.sh` | `protoc` to Java to `gen/testpilot-proto.jar` (gitignored) |

## The kernel and Stainless

Stainless verifies a subset of Scala and rejects `scala.List`, and its own library is compiled with
a nightly Scala, so it cannot be linked into the normal build. The kernel is written against a
prelude that each side supplies: `umpire/prelude/Prelude.scala` makes `Steps` a `scala.List` of the
framework's `Step`, `proofs/umpire/Prelude.scala` makes it Stainless's list. The same file,
`temporal/nexuscaller/kernel/Nexus.scala`, is what the machines enumerate, what the Cases are produced from, and what
the lemmas in `proofs/temporal/NexusLemmas.scala` are proved about.

`temporal/nexuscaller/kernel/NexusActions.scala` gathers every action into one dispatch, which the lemmas quantify
over; `temporal/test/NexusKernel.test.scala` checks that the dispatch gives every table row.

## Oracle

The parity tests read the Lean dumps in place, from `model/go/parity/testdata/lean/`, so the Go and
Scala experiments compare against one copy. `model/go/leandump/dump.sh` refreshes them.
