# A comparative Scala implementation of the Umpire model layer

Plan, 2026-09-29. It adds a third implementation of the Umpire model layer, in Scala 3, beside the
Lean one in `model/lean/` and the Go one in `model/go/` ([UMPIRE_GO](UMPIRE_GO.md)). It ports the
same Models, checks them against the same Lean evidence, and adds one thing neither of the others
has: step functions that Stainless proves properties about for every state, not only the states a
table enumerates. It changes no rule and approves no design; like the Go experiment, its result is
input to a GOV-02 decision on SCP-03.

The starting point is the uncompiled Scala sample in `cmp/scala/` (`Umpire.scala`,
`NexusCaller.scala`, `StandaloneActivity.scala`, `Pins.scala`, and its README). The experiment
compiles that design, keeps what survives contact with the compiler, and records what did not.

## 1. The question

What does an Umpire Model look like in Scala 3, and what does Scala buy over Go and Lean? In
particular:

- Does the Scala sample's authoring surface (enums, case classes, `match` step functions,
  context-function declaration blocks, `infix` claims, `Refines` witnesses) compile and read as the
  sample claims?
- Can it reproduce the Lean answers the Go port reproduces: tables, IDs, refinement rows, query
  outcomes and witnesses, exploration targets, canonical strings, fingerprints, and the seven
  `nexusCallerTests-*-case.json` fixtures byte for byte?
- Does Stainless add a check neither Go nor Lean's bounded tables give, at a loop cost an author
  accepts?

## 2. Scope

In scope, mirroring `model/go/`:

- An `umpire` library: finite domains, actions and classes, machines and tables, reachability,
  stuck states, refinement, composition, Properties, Scenarios, Limits, Queries, bounded search,
  exploration targets, sets, Definition IDs, canonical JSON, Behavior Fingerprints, and predicate
  lowering.
- Three Models: the worker, the Nexus caller (from
  `model/lean/Temporal/Feature/Nexus/Caller/Model.lean` and `model/go/nexuscaller`), and the
  standalone activity (from `model/go/standaloneactivity` and `cmp/scala/StandaloneActivity.scala`).
- Pins translated from the Go pins, which translate the Lean pins.
- Parity tests against the Lean dumps under `model/go/parity/testdata/lean/`, read in place so the
  two experiments share one oracle.
- The Case producer, emitting Testpilot protobuf messages through generated Java classes and
  rendering them to the fixture bytes.
- The four generated views (Markdown table, Mermaid diagram, summary, diff).
- Stainless proofs over the Nexus protocol's step functions (section 5).
- The error corpus cases from `model/go/corpus/cases.json`, with a `scala` edit per case.

Out of scope: any change to `model/lean/`, `model/go/`, Testpilot, `tools/umpire/`, the Makefile or
the checked-in fixtures; the topology layer; the agent trial (it runs once for all three sides under
the Go plan's T11).

## 3. Toolchain and layout

- Scala 3.7 LTS-track through `scala-cli` 1.17 and the JDK pinned in `mise.toml` (temurin-27), munit for tests.
- protobuf-java 4.x and protobuf-java-util for `JsonFormat`; `protoc` from mise generates the Java
  classes for the Testpilot proto closure into a gitignored jar.
- Stainless 0.10.2, the `stainless-dotty-standalone` release, downloaded to
  `UMPIRE_SCALA_TOOLS` (default `/tmp/umpire-scala-tools`) by `tools.sh`. It bundles Z3 and cvc5.

```
model/scala/
  README.md             how to run; what each script checks
  run.sh                generate protos if stale, compile with -Werror, test, prove, views
  tools.sh              fetch Stainless
  gen-proto.sh          protoc -> Java -> gen/testpilot-proto.jar
  project.scala         scala-cli directives: Scala version, dependencies, options
  umpire/               the framework (package umpire), built alone first
  umpire/prelude/       the runtime half of the kernel prelude
  umpire/caseproducer/ umpire/views/
  temporal/             the Temporal Models (package temporal.*)
  temporal/nexuscaller/kernel/  step functions shared with Stainless (section 5)
  temporal/views/       which views are rendered
  temporal/test/        munit pins, parity, Case bytes, kernel agreement, views
  proofs/umpire/        the Stainless half of the prelude
  proofs/temporal/      the lemmas
  goldens/views/        rendered views
```

## 4. Porting rules

The Go plan's section 5.2 table holds, unchanged: member order, row order, state and class keys,
Definition IDs, fingerprints, target canonical content, search order, limits, exploration targets,
ProtoJSON key order and fixture bytes. The Go port is the executable reference for each; where Go
and Lean differ (the product-state count of `terminalHoldsEverywhere`, `pausedIsNotDispatched`),
Scala follows Go and the report says so.

Scala-specific rules:

- Domains are `enum`s and `case class`es with `derives Finite`. Declaration order is catalog
  order, and the last field varies fastest, as in Lean.
- A class key spells an enum case by its name and a parametrised case as `name-field`, e.g.
  `handlerError-true`, matching the Go `Keyed` spellings.
- Step functions are total `match`es compiled under `-Werror`, so a missing case fails the build.
- A Query's Property and Scenario share a state type, or the Query takes an explicit `Refines`
  value: the typed pairing Go gets from `VerifyRefined`.
- Declaration errors (a duplicate name, an example of the wrong type, a step outside the domain)
  are collected, never thrown from object initialisation, and reported by `Check`, as in Go.

## 5. Stainless

Stainless verifies a subset of Scala: enums, case classes, `Int`/`BigInt`/`Boolean`, `match`,
`require` and `ensuring`, and its own `List`. It rejects `scala.List`, and its library is compiled
with a nightly Scala 3.10, so it cannot be linked into the normal build.

The kernel design keeps one source for both:

- `temporal/nexuscaller/kernel/` holds the Nexus protocol's domains, state, and step functions in the subset. Each step
  function returns `Steps[ProtocolState]`, a type the kernel imports from `kernel.prelude`.
- `umpire/prelude/Prelude.scala` defines `Steps` as `scala.List` for the normal build.
- `proofs/umpire/Prelude.scala` defines `Steps` as `stainless.collection.List` for Stainless.
- The Nexus Model in `temporal/nexuscaller/` wraps the kernel step functions into the framework's `Step`
  rows; the table, the pins and the Case bytes run on the same code Stainless proves.

Lemmas, each stated for every state rather than the 192 a table enumerates:

1. Every protocol step keeps the attempt count within its bound.
2. No step leaves a terminal phase except the `notFound` completion, which keeps the state.
3. Refinement: every protocol step maps under `productOf` to a product stutter or a product step.
4. The backoff timer records nothing: the step is silent, so a canary set cannot name it.

The report compares these with the Lean `decide +kernel` refinement theorem, which covers the
enumerated table only, and with the Go test that checks the same rows.

## 6. Tasks

| Task | Deliverable | Done when |
| --- | --- | --- |
| **S0 Scaffold** | `project.scala`, `tools.sh`, `gen-proto.sh`, `run.sh`, README | `run.sh` compiles an empty library and runs Stainless on the kernel |
| **S1 Framework core** | `Finite`, actions and classes, `Machine`, `Table`, `restrict` | A toy machine's table matches the Go toy test |
| **S2 Refinement and composition** | `Refines`, refinement rows, `Compose` | Toy refinement and composition tests pass |
| **S3 Claims and search** | Property, Scenario, Limits, Query, BFS, coverage targets, sets, `Check` | Toy queries find the Go witnesses |
| **S4 Canonical** | canonical JSON, fingerprints, lowering | Toy canonical strings match Go |
| **S5 Worker and Nexus caller** | Models, claims, pins, kernel | All translated Nexus pins pass |
| **S6 Parity** | tests against `model/go/parity/testdata/lean` | Tables, IDs, refinement, queries, targets, canonical strings and fingerprints equal |
| **S7 Standalone activity** | Model, claims, pins | Pins pass; product parity with Lean |
| **S8 Case producer** | proto jar, producer, Case byte test | Seven fixtures byte-identical |
| **S9 Views** | four views, goldens | Goldens equal the Go goldens' content |
| **S10 Stainless** | kernel lemmas | All verification conditions valid in `run.sh` |
| **S11 Corpus** | `scala` edits, harness support | Twelve cases recorded |
| **S12 Report** | `model/scala/RESULTS.md` | Lines, loop times, parity, corpus, Stainless findings, compared with Go and Lean |

## 7. Risks

- **Byte parity** depends on `JsonFormat` agreeing with Go `protojson` on key order and escaping.
  The renderer re-indents textually and undoes Gson's HTML escapes; the Case test names the first
  differing path.
- **Stainless subset drift.** A kernel construct Stainless rejects forces the kernel to stay small.
  The Models keep everything outside the step functions in ordinary Scala.
- **Macro-time search** (`Query.pinned` in the sample) is not attempted: it needs a two-module
  build that `scala-cli` does not give. Queries run in tests, as in Go.
- **Drift into a rewrite.** Same bound as the Go plan: no production change.
