# Umpire4 milestones

The current state of Umpire work: what is being built, what is left, and what is not being done.
Flow (`.flow/`, `flowctl`) is the record for specs and tasks; this page is the overview across them.

As of 2026-10-01.

## Keeping this page current

- This page describes the present. Rewrite a status in place; do not append dated entries.
- Remove a spec or task from this page when it is done. Flow and git keep the history.
- Close a cancelled or abandoned spec in Flow (tasks blocked, a "Closed: won't do" note in the
  spec) and remove it from this page.
- Update the "As of" date with every edit.

## Direction

The Scala model (`model/scalav2`) is the model. Lean is the past, and nothing has to look like it.
Scala declares, a lifter reads the declarations into the Umpire IR, and generic Go consumers check
that IR, lower it to Testpilot Cases, and run those Cases as functional tests and canary checks. The
Umpire IR and the Testpilot IR are what connect the parts. The Lean toolchain is removed from this
checkout, and the Lean-only specs are closed as won't-do. See [SCALA.md](.plans/SCALA.md) and
[UMPIRE4_SPEC.md](.plans/UMPIRE4_SPEC.md).

The planned work below makes the repository match that direction: the layout first, then the Scala
layer, then the Models.

## Active

### fn-107: Scala Umpire prototype for standalone activities and Nexus

19 of 22 tasks done. The remaining tasks run in this order:

| Task | Title | Status |
| --- | --- | --- |
| fn-107.10 | Add activity race controls and durable observations through Testpilot | In progress |
| fn-107.22 | Generate lowered Case files and run them with one generic live runner | Todo, after .10 |
| fn-107.11 | Close the exploration, regression, and trace-inspection loop | Todo, after .10 and .22; closes the spec |

Where the prototype stands:

- Six of the nine standalone-activity Queries lower to Cases that Testpilot admits: `completion`,
  `nonRetryableFailure`, `retry`, `pauseResume`, `terminate`, `scheduleToStartTimeout`.
- Three Queries do not lower and have no owning task: `cancel` and `cancelRequest` (the attempt
  record follows later evidence), and `startToCloseTimeout` (an attempt that gives no answer).
- Hold-delivery and durable-commit observation are declared `unsupported` in the Scala activity
  realization until fn-107.10 supplies them.
- The live tests of lowered Cases are hand-written Go tests per scenario
  (`tests/testpilot_scala_{activity,nexus}_test.go`). fn-107.22 replaces them with generated Case
  files and one generic runner.
- `pauseResume` uses 3.37M of the default 4M per-event Contract work budget. A Case with six
  pieces of evidence of one operation will exceed the default ceilings.

## Planned

None of the specs below has tasks yet. Each starts after the spec it waits for
closes, and the dependencies are recorded in Flow.

| Order | Spec | In one line | Waits for |
| --- | --- | --- | --- |
| 1 | fn-115 | Restructure: `model/` is the Scala model, `tools/umpire/` its Go tooling, the rest archived | fn-107 |
| 2 | fn-113 | Shrink the Scala framework to a declaration DSL, port the lifter to ScalaPB | fn-115 |
| 3 | fn-112 | Rewrite the standalone activity Model as the DSL showcase | fn-113 |
| 4 | fn-114 | Roll the showcase's constructs out to every other Model | fn-112 |
| any time after 1 | fn-116 | Spike: should the IR's expressions be CEL | fn-115 |

### fn-115: Make the Scala model the model and archive the Lean-era work

Changes the layout to say what is true, after fn-107 closes.

- `model/` becomes the Scala model (today `model/scalav2`) and holds no Go code.
- `tools/umpire/` becomes the Go code that loads, checks, lowers and exports the IR (today
  `model/scalav2/goir`, `backends` and the parts of `model/go` they import).
- `model/lean`, `leanv2`, `go`, `quint`, `scala` move to `model0/`, and today's `tools/umpire` moves
  to `tools/umpire0/`. Both leave the Go build, and no live code imports them.
- A module map for the model, the Umpire tooling and Testpilot, in Go and Scala, is approved by the
  owner before any package is split, merged or renamed. Import rules are enforced by a test.
- The parity tests against the hand-written Go models and the Lean dumps become checked-in goldens
  of what the reader derives. The later specs use those goldens as their baseline.
- `run.sh`, `gen.sh` and `scala.sh` are replaced by one Scala program, and the lifter's fixture
  checks become tests of the lifter.
- The IR's proto package is renamed from `modelir` to `umpire`, and `goir` and `scalav2` stop being
  names.
- `model/README.md` describes the whole system for a reader new to it: the layers, the two IRs,
  one worked example from a Scala declaration to a Verdict, and a Mermaid diagram.

Open questions its first task answers: how much of `tools/canary` and of the old `evaluation`,
`recordedrun`, `replay`, `publish` and `binding` packages is live, and how the Lean-rendered Case
fixtures and the canary's pinned Case are regenerated from the Scala model.

### fn-113: Clean up the Scala model layer around the IR

Scala declares, the lifter reads, Go evaluates. Four parts:

- **A. Dead code.** Delete about 390 lines of the framework that have no caller: `Canonical.scala`,
  `Lower.scala` and the `Alterer` plumbing.
- **B. ScalaPB.** Replace the protobuf-java builders in the lifter with ScalaPB case classes. Which
  ScalaPB release fits Scala 3.9.0 and the pinned protoc is unverified and is the first step.
- **C. One evaluator.** Retire the native Scala evaluator (tables, search, refinement, composition),
  after an audit covers each of the 32 munit tests on the IR in Go. `UmpireSet` and `Coverage`,
  which have no IR form, go with it.
- **D. Lean and Stainless residue.** Fold the Nexus kernel into ordinary Scala and drop
  Lean-mirroring code and citations.

It also stops the lifter from writing compiler-synthesized names such as `_$1` into the IR.
Target: the framework at most 1,300 lines (2,873 today), an estimate from current file sizes.

### fn-112: Make the standalone activity Scala Model a DSL showcase

Rewrites the standalone activity Model to read as the best Scala the DSL allows, without changing
its behavior: machine derivation (`rebind`, `extend`, `unmonitored`) in place of copied machines,
compositions keyed by fields in place of strings, names taken from `val`s, and a shared Temporal kit
for what the activity and Nexus realizations both use. Each new construct is built once, in the
lifter, which is why it follows fn-113. It no longer carries three rules that fn-113
reverses: agreement with the native evaluator, the ban on libraries, and the freeze on Lean
comments.

### fn-114: State every Scala Model declaration once

Split out of fn-113 so the specs run in a line. Rolls fn-112's constructs out to the Nexus caller, its close policy, the worker and the lifter
fixtures, and removes the string-named forms from the framework. Realizations refer to their own
ids by value, identity evidence lines go, and the contents of each IR file are declared in Scala
with one lifter run writing all of them.

### fn-116: Decide whether the Umpire IR's expressions become CEL

A spike with a report and no change on the main branch. It translates the expressions of two Models
to CEL, evaluates them with `cel-go` and `cel-java`, and measures the result against the IR's own
expression tree. The saving would be the Go expression evaluator (671 lines) and its semantics
document. The cost is that CEL has no pattern matching, user functions or record update, which the
lifter would have to compile away.
