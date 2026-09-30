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

| Path | What it holds |
| --- | --- |
| `src/project.scala` | Build directives: Scala version, `-Werror`, dependencies |
| `src/umpire/` | The framework: finite domains, actions, machines, tables, refinement, composition, claims, search, coverage, sets, canonical JSON, fingerprints, lowering |
| `src/kernel/` | The Nexus domains and step functions, in the Scala subset Stainless reads |
| `src/prelude/` | The runtime half of the kernel prelude: `Steps` is `scala.List` |
| `proofs/` | The Stainless half of the prelude and the lemmas |
| `src/worker/`, `src/nexuscaller/`, `src/standaloneactivity/` | The three Models, and the Nexus realization |
| `src/caseproducer/` | The Case producer, over the generated Testpilot Java classes |
| `src/views/` | Table, diagram, summary and diff views |
| `src/test/` | Pins, parity against the Lean dumps, Case bytes, kernel agreement, views |
| `goldens/views/` | The rendered views |
| `gen-proto.sh` | `protoc` to Java to `gen/testpilot-proto.jar` (gitignored) |

## The kernel and Stainless

Stainless verifies a subset of Scala and rejects `scala.List`, and its own library is compiled with
a nightly Scala, so it cannot be linked into the normal build. The kernel is written against a
prelude that each side supplies: `src/prelude/Prelude.scala` makes `Steps` a `scala.List` of the
framework's `Step`, `proofs/Prelude.scala` makes it Stainless's list. The same file,
`src/kernel/Nexus.scala`, is what the machines enumerate, what the Cases are produced from, and what
the lemmas in `proofs/NexusLemmas.scala` are proved about.

`src/kernel/NexusActions.scala` gathers every action into one dispatch, which the lemmas quantify
over; `src/test/NexusKernel.test.scala` checks that the dispatch gives every table row.

## Oracle

The parity tests read the Lean dumps in place, from `model/go/parity/testdata/lean/`, so the Go and
Scala experiments compare against one copy. `model/go/leandump/dump.sh` refreshes them.
