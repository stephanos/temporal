# Results: the Scala model layer

Run on 2026-09-30 against commit `d08d20140` plus the uncommitted `model/` tree. Plan:
[`.plans/archive/UMPIRE_SCALA.md`](../../.plans/archive/UMPIRE_SCALA.md). The Go results are in
[`model/go/RESULTS.md`](../go/RESULTS.md); this report compares all three implementations.

## Outcome

The Scala model layer reproduces every answer the Lean model gives for the Nexus caller, and emits
the same seven Case files byte for byte. Its framework is half the size of Go's, its Models are a
quarter smaller, its edit loop is about three seconds, and it catches more authoring mistakes at
compile time than Go does. Stainless adds proofs over every state of the Nexus protocol machine in
about ten seconds, from the same source the machines run.

Two things cost more than the sample in `.plans/archive/cmp/scala` suggested. Stainless needs the step
functions in a restricted subset with a prelude per side. And `scala-cli` through its Bloop server
exits 0 on a `-Werror` failure, which a wrapper script has to correct.

## Parity with Lean

| What | Result |
| --- | --- |
| Nexus tables: product, protocol, worker, handler worker, composition | Equal, row for row |
| Definition IDs of the same five machines | Equal |
| Refinement rows, 1,152 with 456 stutters | Equal |
| Query outcomes, nine, and the seven witnesses | Equal |
| Exploration targets, 889 | Equal |
| Target semantic string, 590 KB | Byte-equal |
| Scenario, Property and Query canonical strings for the seven functional Queries | Byte-equal; 22 fingerprints equal |
| `nexusCallerTests-*-case.json` fixtures | Seven of seven byte-identical |
| Activity product table and IDs | Equal; the protocol machine is past Lean's 256-state bound |
| Translated pins | All pass, Nexus and activity |

Scala follows Go where Go and Lean differ. The free-path search over `terminalIsFinal` visits 111
product states where Veil visits 171, because Lean keeps a fired bit per lowered clause group.
`pausedIsNotDispatched` verifies, where Lean's lowering rejects it as claiming nothing.

## What Stainless proves

`proofs/temporal/NexusLemmas.scala` states five lemmas over the kernel in `temporal/nexuscaller/kernel/`, each for every state
and every action class. The machines enumerate the same file, and a runtime test checks that the
dispatch the lemmas quantify over gives every table row.

| Lemma | Says |
| --- | --- |
| `attemptsStayBounded` | No protocol step takes the attempt count out of `0..2` |
| `terminalIsFinal` | No protocol step changes a terminal phase |
| `productTerminalIsFinal` | The same on the product machine |
| `refines` | Every protocol step is a stutter or a step of the product action of its own name, with the same outcome and the product's facts |
| `silentStepsAreBackoffAndWorkerStop` | The only accepted steps that record nothing are the two a canary may not name |

Stainless discharges 79 verification conditions, and rejects a mutated kernel with the failing lemma
and a counterexample. The `refines` lemma is stricter than Lean's `refines:` check, which accepts any
product action as the carrier.

What the proofs buy over the table is generality, not a new answer: every lemma also holds on the
enumerated table, which the pins already check. They matter more as the domain grows: the table
grows with the product of the fields, and the proofs do not.

## Error corpus

The twelve mistakes of `model/go/corpus/cases.json`, applied one at a time by
`model/go/corpus/run.py` and reverted after each. Seconds run from applying the edit to the first
failure. These runs overlapped with builds of another side, so the seconds are indicative; the
measurements below were not contended.

| # | Mistake | Lean | Go | Scala | Stainless |
| --- | --- | --- | --- | --- | --- |
| 1 | A step names an undeclared action | compile, 7.6 s | compile, 0.3 s | compile, 0.6 s | not run |
| 2 | A step's input type disagrees with the action's input | compile, 4.7 s | compile, 0.4 s | compile, 0.6 s | not run |
| 3 | A Scenario passes an input of the wrong type | compile, 174 s | compile, 0.3 s | compile, 0.7 s | not run |
| 4 | A match or switch misses a case | compile, 30 s | lint, 3.4 s | compile, 0.8 s | catches |
| 5 | A step moves the attempt count past its bound | pins, 596 s | test, 8.2 s | test, 3.0 s | catches, at the line |
| 6 | A recorded fact has no evidence line | pins, 581 s | test, 7.5 s | compile, 0.9 s | not run |
| 7 | A protocol row has no product counterpart under the map | compile, 29 s | test, 7.3 s | test, 3.9 s | catches |
| 8 | A verify Query pairs a product claim with a protocol path and no refinement | compile, 180 s | compile, 0.3 s | compile, 0.9 s | not run |
| 9 | A Query's claim is unreachable on its path | compile, 188 s | test, 7.3 s | test, 3.7 s | not run |
| 10 | A machine has a stuck state | compile, 4.5 s | test, 6.2 s | test, 3.7 s | misses |
| 11 | A canary set names a path with a silent step | compile, 202 s | test, 7.1 s | test, 3.7 s | not run |
| 12 | Two declarations share a model name | compile, 200 s | test, 6.3 s | test, 3.4 s | not run |

"Not run" means the mistake is outside the kernel, where the lemmas do not look.

- **Scala catches six mistakes at compile time, Go four, Lean ten.** Scala's extra two are the
  missing match case, which Go needs a linter for, and the missing evidence line, because evidence
  is a total function over the facts.
- **Every Scala failure arrives within four seconds.** Lean's arrive in 5 to 600 seconds, and the two
  it does not catch at compile time cost a full pins build.
- **Lean still catches the most at compile time.** Its elaborator runs the Queries, the canary rule
  and the stuck-state check while the file compiles. In Scala those are tests, as in Go.
- **Messages point near the author's line.** Compile errors name the file and line; test failures
  name the pin and the declaration. The harness's line heuristic misses a few by a line or two, so
  locality is judged by reading the messages, which the JSON results keep.

## Loop and size

Measured by `model/go/measure.sh` on one machine, one side at a time.

| Metric | Lean | Go | Scala |
| --- | --- | --- | --- |
| Cold build and test | 20 min, whole workspace | 22.4 s | 8.1 s, JVM warm |
| Warm, nothing changed | 1.4 s | 3.4 s | 2.5 s |
| Edit a Nexus Model file, rebuild and test | 191 s | 2.7 s | 2.9 s |
| Edit, reaching the pins | about 590 s | same as above | same as above |
| Full gate after an edit | not measured | 10.6 s with lint | 19 s with proofs |
| Prove the kernel | not applicable | not applicable | 9.9 s |

| Lines | Lean | Go | Scala |
| --- | --- | --- | --- |
| Framework production | 57,088 authored, all of `model/lean` | 4,935 | 2,434 |
| Models production | 873 for Nexus caller and worker | 2,194 | 1,665, with the kernel |
| Nexus caller and worker Models | 873 | 940 | 694 |
| Nexus realization | in the framework count | 385 | 290 |
| Proofs | kernel `decide` in the Models | none | 89 |
| Tests | 41,101 | 1,411 | 653 |

The Scala cold number keeps the JVM and the Bloop server warm; a first run on a fresh machine also
downloads dependencies and starts the server. Lean's framework count is the whole authored
workspace, not only the parts this experiment ports.

## The authoring surface, as it compiled

The sample's design mostly survived:

- **Domains as enums and case classes** with `derives Finite`. Catalog order is declaration order,
  and keys fall out of case names, so no key overrides are needed.
- **Declaration blocks** as context functions: `machine[S, O, F](family, name) { starts(...);
  evidence { ... }; steps(a ~> f, ...) }`.
- **Typed bindings**: `action ~> stepFunction` has one extension per arity, so a step written for
  another action's inputs does not compile. The sample's `TupledFunction` is experimental and was
  not needed.
- **Infix Queries**: `query("retry") find retrySucceeds in retriedThenSucceeded limits four`. A
  product claim on a protocol path needs a `Reads` given, declared once beside the machine.
- **Direct-style checking**: `checked { ... fail(...) ... x.get }` over `scala.util.boundary`, so
  framework checks read as straight-line code.

What changed from the sample:

- **No compile-time search.** `Query.pinned` needs a two-module build, and Queries run in tests, as
  in Go.
- **No `Phased` type class.** Starts and Scenario starts name whole states, as in Go.
- **Evidence is total.** The compiler refuses a fact without a line. The price is two summary lines
  that differ from Go: the product machine names evidence for a fact no step records, and lines come
  in catalog order.
- **The Stainless kernel sits beside the Model.** The Nexus domains and step functions live in
  `temporal/nexuscaller/kernel/` without type classes, so their `Finite` instances are derived in the Model file.

Friction worth knowing:

- **`scala-cli` exits 0 on a `-Werror` error through Bloop,** and leaves class files that let a
  following test run the rejected code. `model/scala/scala.sh` fails on any printed error. Without
  it, the first corpus run recorded two compile-time catches as test-time ones.
- **Name clashes across wildcard imports.** The framework's query outcome became `Verdict`, and the
  realization object became `NexusRealization`.
- **Stainless's front end** crashed with "untyped receiver" on a nested lambda over a type alias.
  Typed helper functions fixed it.
- **Generated protobuf classes nest in outer classes,** because the Testpilot protos do not set
  `java_multiple_files`. `JsonFormat` escapes HTML characters Go's protojson does not, which the
  renderer undoes.

## Agent trial

Three fresh agent sessions per side added a heartbeat timeout to the standalone activity; the runs
and diffs are in [`model/go/trial/`](../go/trial/README.md). Lean could not run: its activity
protocol is already past the elaborator's 256-state bound.

| Side | Wall time | Check runs | Reviewer |
| --- | --- | --- | --- |
| Go | 188, 219, 212 s | 1, 1, 1 | changes requested on all three |
| Scala | 264, 292, 280 s | 1, 1, 1 | accepted with a follow-up on all three |

Both sides passed on the first check every time. The difference is in what the agents did at the
framework's edge: the start request needed four inputs and both frameworks stop at three. Every Go
run folded two deadlines into one struct input; every Scala run added the missing arity in a few
lines beside the Model. Scala's runs took about a minute longer each; the runs do not show where the
minute went.

## Recommendation

On this evidence, Scala is the stronger of the two non-Lean candidates for the model layer. It
matches Go's parity and loop, catches more mistakes at compile time, reads closer to the Lean Models,
and offers proofs when a claim needs every state. Against it: Temporal's server and Testpilot are
Go, so Scala adds a JVM toolchain, a build, and a language the team does not otherwise use, and the
Case producer must go through generated Java classes rather than the server's own types. Stainless
is a research-grade tool whose subset shaped the kernel.

Whether that trade is worth it is a GOV-02 decision on SCP-03, not something this experiment
settles.
