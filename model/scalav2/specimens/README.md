# model/scalav2/specimens

The two authoring specimens fn-107 reviews before the frontend, the IR or the Go checker grows (R1).
Each is a Scala sketch with its product and system boundaries, a finite domain manifest, the
promises it checks, the evidence each promise needs, and trace oracles that later tasks pin:

| File | Specimen | Pinned controls |
| --- | --- | --- |
| [activity.md](activity.md) | Standalone activity admission and its dispatch queue | Stale delivery after pause; duplicate delivery; a queue provider that loses committed work |
| [nexus.md](nexus.md) | Nexus caller close and reset | Permanent rejection after close then reset; acknowledgment by the original run after reset |

## What these files claim

Nothing here is a checked proof. The sketches are hand-reviewed, and no gate builds them.

Code blocks come in two kinds. Each sits under a heading that names its kind, and unsupported
blocks also start with `// UNSUPPORTED`:

- **Supported** blocks use only declarations in `model/scala/umpire` in the working tree on
  2026-09-30. That is base commit `bbe765d70` plus uncommitted edits, which in the cited files are
  only comment, import and package renames, so the line numbers cited hold for both. On
  2026-09-30 they were copied to a scratch directory and put through four steps (see
  [Scratch sanity check](#scratch-sanity-check)):
  1. compiled with `model/scala`;
  2. answered by the existing Scala search;
  3. lifted by `lifter/`;
  4. loaded by `goir/`.

  That shows the vocabulary exists and the oracles are consistent. It is not the Go checking R3
  requires. No gate builds these blocks, so they can drift from the framework.
- **Unsupported** blocks start with `// UNSUPPORTED` and do not compile. They show the smallest
  declarations the specimens appear to need. Each is listed under
  [Proposed extensions](#proposed-extensions), with the task that would add it.

A trace oracle is the expected result a later task must reproduce. It is not a result. Where the
scratch run already produced the witness, the oracle says so and gives the explored-state count.

## The vocabulary the sketches use

Every supported block is written with these existing declarations and no others.

| Declaration | Where | Used for |
| --- | --- | --- |
| `enum … derives Finite`, `final case class … derives Finite` | `model/scala/umpire/Domain.scala:20`, `:41` | States, outcomes, facts, inputs; catalogs in declaration order |
| `Finite.upTo(n)` beside a state's `Finite` | `Domain.scala:33`; lifted at `lifter/Lift.scala:140-144` | Bounded counters (the sketches use enums instead; see the findings) |
| `action(name, party)`, `.on`, `.creates`, `.input[T]`, `.schema`, `.results` | `model/scala/umpire/Action.scala:85`, `:53-60` | Party-performed actions with finite inputs |
| `timer(name)` | `Action.scala:88` | System-owned steps; the only system action kind today |
| `Party`, `Entity`, `Observation` | `Action.scala:4`, `:11`, `:14` | Who acts, what has identity, derived reads |
| `machine[S, O, F](family, name) { forEntity; starts; ends; evidence; steps(a ~> f) }` | `model/scala/umpire/Machine.scala:50-77` | Guarded transitions; one step function per action |
| `Step(outcome, state, facts, because)` in a `List` | `Machine.scala:7`; `SEMANTICS.md` Machines 3 | A list of several steps is an explicit nondeterministic choice; an empty list is a disabled action |
| `refines(product)(map)` | `model/scala/umpire/Refine.scala:13` | Stuttering forward simulation into a product machine |
| `property(name) holds` / `holdsAcross` / `whenAction` | `model/scala/umpire/Claims.scala:42-60`, `:94-100` | Same-step and transition promises |
| `scenario(name).starts(s).actions(…)` / `.free` | `Claims.scala:82-92` | Pinned schedules and free exploration |
| `query(name) verify p in s limits l`, `Limits`, `Reads.through` | `Claims.scala:110-159` | Bounded questions, product claims read through a refinement |
| `check(decls*)` | `model/scala/umpire/Sets.scala:40-65` | Stuck states, duplicate claim names, refinement |

The sketches also reuse existing feature declarations:

- The activity sketch reuses `activity`, `control`, `attemptStart`, `attemptResult`,
  `activityProduct`, `pausedIsNotDispatched` and `Outcome`
  (`model/scala/temporal/standaloneactivity/Model.scala:33-101`, `:182`, `Claims.scala:21-24`).
- The Nexus sketch reuses `caller`, `handler`, `workflow`, `operation` and `complete`, and the
  kernel's `Resolution` (`model/scala/temporal/nexuscaller/Model.scala:26-76`,
  `kernel/Nexus.scala:32-33`).

## Proposed extensions

Every extension below is unsupported today: no frontend, IR or checker has it. A sketch that uses
one says so. Each is sized to the specimens, not to a general language.

| # | Extension | What it declares | Why the specimens need it | Task |
| --- | --- | --- | --- | --- |
| E1 | `monitor` | Passive event monitor with its own finite state, update rules and evaluation points | Monitor state must be part of explored-state identity, and a Run must advance it per compatible execution. Today the only monitor is the search's `(fired, held)` pair (`model/scala/umpire/Search.scala:50`), and claims are a transition predicate or a state predicate | 2, 3, 8 |
| E2 | `channel` | Capacity, order (FIFO or unordered), loss (reliable or lossy) and duplicate delivery, plus the interruption points of each route | Go derives fault alternatives from the declaration. The supported sketches write the redelivery and rejection alternatives by hand | 2, 3, 4, 5 |
| E3 | `provider` / `replaces` | An opaque provider's interface events and all the bounded behavior it allows, and scoped replacement by a detailed provider | Scoped queue refinement; each check that uses the opaque provider names that assumption | 2, 3, 4 |
| E4 | Visible-result projection on `refines` | Which facts and outcomes are product-visible | R3 forbids a stutter that emits a product-visible result. Today's rule ignores the stutter's facts (`Refine.scala:72`, `goir/machine.go:310`, `model/go/umpire/refine.go:83`) | 2, 3, 4 |
| E5 | `assume` and bounded `leadsTo` | Named fairness, retry, retention, reporting and deadline assumptions, and progress claims conditional on them | Progress needs a deadlock witness, a fair non-progress cycle or a violated modeled deadline. Today's search checks safety only | 2, 3, 5 |
| E6 | `hole` | A declared behavioral hole and the results it makes incomplete | Keeps holes apart from admission errors and from disabled actions (see the findings on `SEMANTICS.md`) | 2, 3 |
| E7 | `internal` | A system step that is not a timer | Dispatch, commit and delivery are spelled `timer` today (`Action.scala:88`); SEM-19 wants one word per concept | 2 |
| E8 | `observe` with commitment and correlation | An observation's message, whether it reports invocation or durable commit, and its operation, attempt, delivery, run and causal keys | Crossed or uncommitted evidence must not satisfy a correlated promise. Today an `Observation` is a name, an entity and a field (`Action.scala:14`) | 6, 7, 10 |
| E9 | `scope` | The domain manifest: entity counts, bounds, fault budget, excluded behavior | Receipts carry actual bounds; a query that needs more reports incomplete | 2, 3 |
| E10 | Scenario DAG with `bind`, worker scripts and `requires(control)` | Learned IDs bound once, dependent branches, SDK scripts, required actuators | Lowering into Testpilot Cases; preparation rejects a missing actuator before I/O | 6, 9, 10, 13 |
| E11 | Claims over a state type | One Property declared for every machine over `S` | Each design redeclares its promises: a Property belongs to one machine (`Claims.scala:94-100`). The sketches work around it with a helper `def` | 2 (optional) |
| E12 | Lifting claims | Properties, Scenarios and Queries in the IR | They are Scala lambdas today and are not lifted (`../README.md`, "Not lifted yet") | 2 |

## Trace notation

A trace oracle lists one step per row, in the form `row → outcome state [facts]`:

- **Row, state and fact keys** are the framework's keys (`SEMANTICS.md`, Keys). A row key is the
  source state key followed by the action class key.
- **Observations** are split in three:
  - **public:** a frontend RPC response or history event Testpilot can read today;
  - **internal:** a server commit or hook that needs task 10;
  - **none:** nothing records it.
- **Evaluation point:** the step after which a promise's verdict is read.

Search answers use the spellings of `Verdict` (`Claims.scala:162-167`). Runtime assessments are
`satisfied`, `violated` or `inconclusive`, and runtime trace conformance is reported separately
from them. A refinement failure is a definition error, raised before any search. It is never a
counterexample, and never a pass.

## Monitor obligations common to both specimens

- A monitor is passive. It reads steps and never disables one, so a monitor cannot remove a
  counterexample.
- Its state is part of the explored state. Two histories that carry different obligations stay
  distinct even when the machine state is equal. Search does this today only for `(fired, held)`
  (`Search.scala:50`, `:104-114`).
- Each monitor declares its evaluation point, and a verdict is read there and nowhere else.
- At runtime a monitor advances once for every execution still compatible with the evidence and its
  causal order. If they agree, that is the verdict. If they disagree, the verdict is `inconclusive`.
- Missing evidence never becomes a Boolean operand.
- Today's supported encodings approximate this:
  - a transition predicate (`holdsAcross`) for "never from A to B";
  - a state predicate (`holds` with no `when`) for invariants;
  - a counter in the machine state (`Active` in the activity sketch) where a monitor would keep one.

  The counter is model state, so the refinement map must ignore it, and it is not passive in
  principle. E1 moves it out.

## Testpilot reuse and evolution

Testpilot is the execution boundary for every executable scenario. These parts carry over as they
are:

| Reused | Where | Specimen use |
| --- | --- | --- |
| Case, Program, Contract; Case format 1.0 | `proto/internal/temporal/server/api/testpilot/v1/case.proto:8-21` | One lowered bounded plan and deterministic Contract per scenario |
| `Prepare(case, profile)`, no Driver I/O | `common/testing/testpilot/prepare.go:27-51` | Admission of lowered Cases |
| Preflight: identity, `Driver.Validate`, Monitor creation, before `Open` | `common/testing/testpilot/prepare.go:56-78` | Rejecting a missing actuator before target I/O |
| `Driver` and `Session`, including `InvokeRPC` and `InjectFault` | `common/testing/testpilot/driver.go:98-102`, `common/testing/testpilot/contract/driver.go:83-99` | Public controls and the realized fault |
| Whole public WorkflowService catalog, any unary method | `common/testing/testpilot/temporal/catalog.go:15-33`, `common/testing/testpilot/internal/ir/catalog.go:247-264` | StartActivityExecution, PauseActivityExecution, DescribeActivityExecution and ResetWorkflowExecution are admissible. No Case uses them yet |
| `ReadEvidence` polling; history and read evidence sources | `proto/internal/temporal/server/api/testpilot/v1/program.proto:28-61` | Status reads, history events |
| CorrelatedEvidence: scope, operation key, parents, dense ordinals | `proto/internal/temporal/server/api/testpilot/v1/correlated.proto:213-232`, `common/testing/testpilot/internal/verification/correlated.go:125-226` | Correlating operation, attempt and run evidence |
| `WORKER_STOP` / `WORKER_RESUME` faults and one `FAULT_INJECTED` event each | `proto/internal/temporal/server/api/testpilot/v1/instruction.proto:111-118`, `common/testing/testpilot/internal/execution/scheduler.go:810-814` | The one realized fault that R8 requires, if a worker fault suffices |
| Handle slots, `NexusHandlerReply`, `NexusOperationCompletion` | `proto/internal/temporal/server/api/testpilot/v1/instruction.proto:140-147` | Nexus sync and async completion (`../../scala/temporal/nexuscaller/Realization.scala:173-205`) |
| Offline `Evaluate` through the same `Observe` | `common/testing/testpilot/prepared_case.go:33-42`, `common/testing/testpilot/internal/verification/evaluator.go:503-541` | Offline replay of a live assessment |
| Canary build-tag seams and `runWith` | `tools/canary/cmd/umpire-canary/seams_harness.go`, `tools/canary/controller/run.go:176-196` | Starting point for the test-only Case/Profile binding |

These primitives are missing. Each one blocks a named experiment:

| Missing | Evidence | Needed by | Task |
| --- | --- | --- | --- |
| Activity SDK activation | `ActivityActivation` exists (`proto/internal/temporal/server/api/testpilot/v1/program.proto:131-135`), but no opcode is admitted in an activity entrypoint (`common/testing/testpilot/internal/execution/dataflow.go:145-151`), the worker Driver skips them (`common/testing/testpilot/temporal/worker/driver.go:265-284`), and the SDK registers no activity (`common/testing/testpilot/temporal/worker/sdk.go:29`, `:42`) | Scala-declared Go SDK activity script; functional and canary completion | 13 (6 for the script vocabulary) |
| Durable-commit observation | Run Events record RPC outcomes. Persisted state is visible only by reading it back | Attempt-admission commit correlated with activity, attempt and causal pause (R8) | 10, with 6 for the observation declaration |
| Hold-delivery control | `FaultKind` holds only worker lifecycle values, "nothing here reaches the server" (`proto/internal/temporal/server/api/testpilot/v1/instruction.proto:111-118`). The server Session refuses faults. The candidate hook is `DispatchTaskHook` (`chasm/lib/activity/tasks.go:16-26`, `:69-77`), which runs after `Validate` (`tasks.go:46-54`) | Holding the stale dispatch past the pause | 10 |
| Authored monitor / prepared assessment | `MonitorFactory` is internal (`common/testing/testpilot/internal/execution/contracts.go:20-29`). The facade "does not accept replacement monitors" (`common/testing/testpilot/internal/verification/README.md:6`) | Trace conformance and property assessment beside the Contract Verdict, live and offline | 12 (seam), 7 (adapter) |
| Server-side faults (commit failure, acknowledgment loss) | Same `FaultKind` limits | Only if the realized fault is not a worker fault | 10 |
| Workflow activity command realization | `schedule_activity_task_command_attributes` is unrealized (`common/testing/testpilot/internal/execution/typed.go:35-37`) | Only if the selected parity tests need a workflow-scheduled activity | 9 |

## Authoring baseline

These are the R1 authoring measurements. They were taken on 2026-09-30 on Darwin 25.6.0 arm64 with
scala-cli 1.17.1, a warm Bloop server, on the same working tree. Logs are under
`.flow/tmp/fn-107/measure/` and `.flow/tmp/fn-107/sketch/`, which are not committed.

**Feature-specific source size.** Lines are total lines / non-blank non-comment lines.

| Source | Lines |
| --- | --- |
| Existing activity Model and Claims (`../../scala/temporal/standaloneactivity/`) | 600 / 357 |
| Existing Nexus caller Model, Claims, kernel and action dispatch (`../../scala/temporal/nexuscaller/`) | 697 / 375 |
| Existing Nexus realization (`Realization.scala`) | 291 / 202 |
| Existing Go activity model and claims (`../../go/standaloneactivity/`) | 869 / 586 |
| Existing Go Nexus model and claims (`../../go/nexuscaller/`) | 835 / 504 |
| Activity sketch, supported block ([activity.md](activity.md)) | 173 / 119 |
| Nexus sketch, supported block ([nexus.md](nexus.md)) | 209 / 144 |
| Framework, for scale (`../../scala/umpire/`) | 2,267 / 1,584 |

**Edit-to-diagnostic.** Each figure is one run, so treat them as indicative.

| Step | Seconds | Note |
| --- | --- | --- |
| Compile `model/scala`, no change | 0.56-0.90 | Two runs |
| Compile a scratch copy, cold / warm | 5.78 / 0.68 | |
| Edit (drop one `evidence` case) → compile diagnostic | 1.38 | Diagnostic at `Model.scala:371`, "match may not be exhaustive". **Exit status 0**; see finding F1 |
| Behavioral edit → green compile | 1.55 | Paused activity made dispatchable |
| Package `model/scala` | 0.95 | |
| Lift the Nexus roots | 3.34 | Output identical to `../ir/nexus-caller.json` |
| Edit → lift refusal (the `unsupported` fixture: package, then lift) | 2.51 | Refused at `Unsupported.scala:18` |
| Lift `activityProduct` | 3.66 | Lifts |
| Lift `activityProtocol` | 3.63 | Refused at `Model.scala:263`: varargs `ProtocolFact*` |
| Sketch: compile and answer all 35 queries in the Scala framework | 2.25-5.88 | |
| Sketch: package, then lift one root | 1.47 + 2.1-2.6 | Per root |
| Sketch: goir loads all five lifted machines | 0.47 | |
| `go test ./model/scalav2/goir/...` from the checked-in IR | 1.76 | No JVM |
| `make umpire-check-scala` (host baseline) | 45.27 | `.flow/tmp/fn-107/baseline-metadata.json` |

A new specimen machine therefore reaches a Go diagnostic in about 5.5 s: compile 1.4, package 1.5,
lift 2.2 and load 0.5. The JVM start of each separate scala-cli invocation accounts for most of it.

## Scratch sanity check

The supported blocks were copied verbatim into `.flow/tmp/fn-107/sketch/` and checked with
`.flow/tmp/fn-107/sketch/check.sh`, which does four things:

1. compiles them with `model/scala/project.scala`, `umpire` and `temporal`;
2. answers every Query in them with the existing Scala search and runs `check` on each machine;
3. re-packages them under `package temporal.specimens.*`, because the lifter reads only TASTy under
   `temporal/`, then lifts each design machine;
4. builds the lifted IR with `goir.Build`.

The current review-corrected blocks were rechecked in `check-r2-final.log`; all 35 Queries were
answered, and focused controls reject a third delivery and both positive acknowledgments without
custody. `Retained.pending` includes an outstanding transferable delivery obligation. The timing
table above records the initial baseline runs.

It found the following. The per-trace details are in the specimen files.

- All five design machines lift and load. Four build successfully; `staleAdmission` reaches its
  expected refinement build error. For the four successful builds, goir's reachable-state counts
  equal the Scala free
  search's explored counts (activity 10; Nexus 61, 76 and 71).
- `staleAdmission`'s declared refinement fails at row `paused-queued-none-attemptStart` in both the
  Scala and the Go rule. The corrected design refines `activityProduct` with 27 stutters, none of
  which records a product fact.
- Scala `check` and goir report the same stuck state for both faulty Nexus designs,
  `resetOpen-false-done-succeeded-none-none-none`, and none for the corrected design.
- Getting the sketches to lift took two rewrites. Each is a finding below.

## Findings for later tasks

| # | Finding | Evidence | Owner |
| --- | --- | --- | --- |
| F1 | Under `-Werror`, a warning promoted to an error (a non-exhaustive `evidence` match) prints "Error compiling project", yet `scala-cli compile` and `scala-cli package` exit 0 and `package` writes the jar. A type error exits 1. `run.sh` relies on exit status (`set -e`), so the "missing evidence line is a compile error" guarantee (`Machine.scala:71-72`) does not stop the gate at compile | Reproduced in a scratch copy with scala-cli 1.17.1, direct and through `mise exec` | Host: `run.sh` is in no task's Touches |
| F2 | `activityProtocol` does not lift. The first refusal is varargs `recorded: ProtocolFact*` (`Model.scala:263`). `because = …` as a named argument (`Model.scala:291`, `:307`) is also refused, as the sketch showed. `(a + 1).min(…)` (`Model.scala:261`) is outside the lifted operators (`Lift.scala:162-164`); that last one is inferred, because the lift stopped earlier | Lift log `.flow/tmp/fn-107/measure/lift-activity-protocol.out` | 4 (rewrite) or 2 (extend) |
| F3 | The refinement rule accepts a stutter that records a product-visible fact. `activityProtocol` has 240 such results out of 552 stutters: `start` recording statusScheduled (192), pause of a held attempt recording statusPaused (24), and unpause of a pause request recording statusStarted (24). R3's visible-output rule would reject the existing activity refinement unless the product projection changes | `.flow/tmp/fn-107/sketch/stutters.log`; `Refine.scala:72` | 3, 4 |
| F4 | `SEMANTICS.md` calls a value that no `match` case matches "an error, a hole in the Model". The spec separates admission errors, declared holes and reachable undeclared holes (which make results incomplete) | `../SEMANTICS.md`, Expressions | 2 |
| F5 | One `Finite.upTo` range covers every `Int` field of a record (`Lift.scala:140-144`, `:81-84`), so the sketches use small enums for counters | | 2 |
| F6 | The lifter reads only TASTy under `temporal/` (`Lift.scala:471`). A root that names nothing writes an empty Model with exit 0, because `lifter.model` is set even when no root matched (`Lift.scala:456-463`, `:485`) | Scratch lift before re-packaging | 2 |
| F7 | A pattern variable bound inside a local `val`'s right-hand side is refused: only names owned by a `def` become IR variables (`Lift.scala:281`). The Nexus sketch moves that match into a helper function | Scratch lift, `CloseReset.scala:118` | 2 |
| F8 | A Query through `Reads.through` runs the refinement check first, so a design whose refinement fails reports a model error rather than a counterexample to the product Property. The negative controls are therefore stated as system Properties as well | Activity oracle A5 | 3 (receipts keep the two apart) |
| F9 | The Nexus operation entity is keyed by `scheduledEvent` (`nexuscaller/Model.scala:36`), which belongs to one run's history. The close/reset design needs an identity that spans the original run and its successor | [nexus.md](nexus.md) | 5, 6 |
