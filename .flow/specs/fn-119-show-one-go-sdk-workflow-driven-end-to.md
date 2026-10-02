# Show one Go SDK workflow driven end to end from the IRs

## Goal & Context
<!-- scope: business -->

The pipeline's promise is that a feature is described once, in a Scala Model, and everything that runs follows from it: the checks, the test Cases, the workers that play each party, and the verdict. Nothing shows that promise on the thing most Temporal developers know best, an ordinary workflow running on the Go SDK.

The existing Models cover a Nexus operation as its caller sees it and a standalone activity. In both, the Go SDK worker is a supporting role. A newcomer who asks "can this test my workflow?" has no example to read and run.

This spec adds that example. One small workflow feature is modeled in Scala, and a Go SDK worker executes the workflow against a real server with no hand-written Go for the feature: no workflow function, no activity function, no test function and no assertion. Every piece is derived from the Umpire IR and the Testpilot IR. A walkthrough shows each artifact on the way, so a reader can see what was written (the Model) and what was generated (everything else).

It serves two readers: the developer evaluating whether to model a feature, and the project itself, since an example with no escape hatch shows where the pipeline still needs manual code.

## Architecture & Data Models
<!-- scope: technical -->

**What exists.** `fn-107-scala-umpire-prototype-for-standalone` task 22 generates lowered Case files and runs them with one generic live runner, so no Go test is written per scenario. Testpilot's Temporal Driver already runs a workflow through the Go SDK without a workflow function per Case: a dynamic workflow interprets the Case's script and issues each command through the SDK. So the mechanism is in place.

**What is missing.** The Driver realizes one workflow command type today, scheduling a Nexus operation. An ordinary workflow needs more. The example is chosen to need the fewest new primitives that still read as a real workflow.

**The example.** A workflow that runs one activity and completes with its result, with the activity failing once and succeeding on retry in a second path, and timing out in a third. It is the workflow a Go SDK tutorial starts with.

| Layer | What the example contributes | Hand-written |
| --- | --- | --- |
| Model | domains, the machine, Properties, Scenarios, Queries, the realization | yes, in Scala: this is the only authored part |
| Umpire IR | lifted from the Model | no |
| Testpilot Cases | lowered from the IR, one per Query | no |
| Workflow execution | the Driver's dynamic workflow issues the Case's commands through the Go SDK | no |
| Activity execution | the Driver's activity activation answers each attempt as the path says | no |
| Test | the generic live runner discovers and runs the Cases | no |
| Verdict and assessment | Testpilot evaluates the Contract, conformance explains the Run by the Model | no |

**New Driver primitives.** Scheduling an activity from a workflow, awaiting its result and completing the workflow with it. Each is a general primitive of the Driver, declared in the Profile like the existing one, and usable by any later Model. None is written for this example alone.

## API Contracts
<!-- scope: technical -->

**One command runs it.** A developer runs the example with one documented command against the in-process cluster, and sees per Query: the Case that ran, the Run's Verdict and the model assessment.

**The walkthrough.** A page beside the example shows, for one Query, in order: the lines of the Model that declare it, the excerpt of the Umpire IR they lift to, the lowered Case, the workflow history of the Run, and the Verdict. It names the command that produces each. It is linked from the model's README as the place to start.

## Edge Cases & Constraints
<!-- scope: technical -->

- **No manual code means no manual code.** The repository contains no Go file whose subject is this example: no workflow, activity, test, fixture builder or assertion. A check enforces it by searching Go sources for the example's names. If a step cannot be derived, the gap is recorded as a finding with the missing primitive, and the example does not paper over it.
- **A failing path must fail.** An example that only passes shows little. One Query is expected to find a violation against a deliberately faulty variant of the Model, so the walkthrough also shows what a counterexample and a violated Verdict look like.
- **It is a real run.** The workflow runs on a Go SDK worker against a real server, and the history shown in the walkthrough is from that run. Nothing is simulated.
- **It uses the finished DSL.** The example is written after the Scala cleanup, with typed Temporal API references and declared API behavior, so that it shows the authoring experience the project intends. It contains no proto name as a string and no literal wait.
- **It stays small.** The Model fits on a few screens. A feature that needs more belongs in a real Model.
- **It is kept working.** The example runs in the same gates as the other Models, so it cannot rot into documentation that no longer runs.
- **Honest scope.** The workflow is the Driver's interpreter executing commands the Case carries. The example shows that a workflow's behavior can be driven and checked without writing one. It does not show testing a workflow function a user wrote, and the walkthrough says so.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A Model of the example workflow exists beside the other Models, with a product machine, at least three Queries (completion, retry then completion, timeout) and a realization, and it passes the model gate. Errors: a construct it needs that the DSL lacks is a finding against the DSL, and the example does not work around it in Go.
- **R2:** The Testpilot Temporal Driver realizes the workflow commands the example needs through the Go SDK, as general primitives a Profile authorizes. Errors: a Case carrying a command the Profile does not list is rejected at preparation with the command named, as today; a command the Driver cannot realize yet is listed with its type.
- **R3:** Every Query of the example lowers to a Case, and the generic live runner runs each against the in-process cluster with a satisfied Verdict and a conforming assessment, live and replayed. Errors: a Query that does not lower is listed with the located reason and blocks the spec's close.
- **R4:** No Go source file in the repository names the example, its workflow type, its activity type or any of its Queries, and a check in the gates fails on one. Errors: generated files and the walkthrough are exempt and are listed by path pattern in the check.
- **R5:** One Query runs against a deliberately faulty variant and yields a counterexample at the model level and a violated Verdict at the run level, each shown in the walkthrough. Errors: a faulty variant that passes means the example proves nothing and blocks the close.
- **R6:** A walkthrough follows one Query from its Scala declaration through the Umpire IR, the lowered Case and the Run's workflow history to the Verdict, naming the command that produces each artifact. The model's README links to it. Errors: a fresh reader with no context (a subagent given only the walkthrough and the commands) reproduces the Run and states what was authored and what was generated; whatever it cannot do is fixed in the page.
- **R7:** One documented command runs the whole example. The done summary states its wall-clock time, the lines of Scala authored, and the lines of Go added for general Driver primitives (no error surface beyond the command's exit status).
- **R8:** The example runs in the model gate and in the live test job, so a change that breaks it fails a gate (no error surface).
- **R9:** The done summary lists every place the example needed something the pipeline did not have, with what was done about each: built as a general primitive, or left as a named gap.

## Boundaries
<!-- scope: business -->

- No testing of a hand-written workflow function. That is a different capability and a later question.
- No second example. One workflow, kept small.
- No SDK other than Go.
- No Driver primitive beyond what this example needs.
- No change to how Testpilot evaluates a Contract.
- No canary use of the example.

## Decision Context
<!-- scope: both — conditionally substructured -->

The owner asked on 2026-10-01 for an example of a Go SDK workflow driven fully end to end by the IRs with no manual code, to show how well Testpilot and the rest work.

**Partly covered already.** fn-107 task 22 removes per-scenario Go tests, and the Driver already interprets a workflow from a Case. What no spec covered is a workflow-centred example, the Driver primitives an ordinary workflow needs, and a walkthrough.

**An activity workflow.** It is the workflow every Go SDK user has written, it needs only three new primitives, and its retry and timeout paths show the Model earning its keep. A timer-only workflow would need fewer primitives and would show less. A workflow with signals and child workflows would show more and would turn the spec into Driver work.

**The check for zero Go.** Without an enforced check, a helper slips in and the example keeps claiming what it no longer shows.

**After the cleanup.** The example is what newcomers copy. Written before the DSL settles, it would teach the old forms and have to be rewritten.

## Parked unknowns

- Whether the lowering can place a workflow's activity attempts with the existing realization declarations or needs a new one. R1 and R3 answer it.
- Whether the example's Model lives with the Temporal Models or in an `examples` folder of its own. The module map of fn-115 decides.
