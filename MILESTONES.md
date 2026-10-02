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

The Scala front end (`model/scalav2`) replaces the Lean model as the place features are authored.
Scala definitions lift to one finite semantic IR, and generic Go consumers check it, lower it to
Testpilot Cases, and run those Cases as functional tests and canary checks. The Lean toolchain is
removed from this checkout, and the Lean-only specs are closed as won't-do. See [SCALA.md](.plans/SCALA.md) and
[UMPIRE4_SPEC.md](.plans/UMPIRE4_SPEC.md).

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
