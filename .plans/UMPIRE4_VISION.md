# Umpire vision

- Describe expected software behavior once, then use it to check designs, generate tests,
  explore possible failures, and check what the running system did.
- Give developers a friendly API and beautiful, readable models that make a feature's promises clear.
- Keep models small and easy to combine. Show how product promises relate to actual behavior
  and how detailed models relate to simpler ones.
- State a behavioral promise that many Temporal features share once, as a reusable protocol, and
  let each feature adopt it instead of restating it.
- Work across distributed processes whose clocks may disagree.

This vision includes goals not yet implemented. The [shared specification](UMPIRE4_SPEC.md)
defines the architectural rules; [model semantics](../model/SEMANTICS.md) describes what works today
and its limits. [Industry research](UMPIRE4_INSPIRE.md) explains the lessons behind these choices.

A **Model** describes allowed behavior. A **Property** states a promise that behavior must meet.
A **Query** asks a question about the Model. A **Case** is a generated test, and a **Run** records
one execution of that test.


## High-level Architecture

Umpire splits into components that either **know** something about behavior or only **carry** what
others declared. All knowledge of what Temporal does lives in the Models; every other component is
mechanical, and most of them know nothing about Temporal.

```
  Models ──► Lifter ──► Umpire IR ──┬──► Checker ─────► answers and witnesses
 (Scala)                            ├──► Test generator ──► Case ──► Testpilot ──► Run
                                    └──► Judge ◄─────────────────────────────────────┘
```

### Components

#### Models

The Scala description of Temporal's behavior: a Temporal-agnostic DSL framework, a Temporal kit of
shared capabilities, laws and realization vocabulary, and one folder per feature.

- Knows behavior: Yes, it is the only component that does.
- Knows Temporal: Yes, except the DSL framework.

#### Lifter

Translates the compiled Models into the Umpire IR, and refuses what it cannot translate.

- Knows behavior: No.
- Knows Temporal: Only the kit's vocabulary, so the IR can carry it.

#### Umpire IR

The versioned artifact that connects the Scala Models to the Go tooling.

- Knows behavior: It carries it.
- Knows Temporal: Only as declared names and payload types.

#### Checker

Interprets the IR, answers Queries by bounded exhaustive exploration, and finds witnesses.

- Knows behavior: No.
- Knows Temporal: No.

#### Test generator

Turns a Query's witness and its realization into an executable Testpilot Case.

- Knows behavior: No.
- Knows Temporal: No, only through the realization.

#### Testpilot

Runs a Case against a real server and records the Run. A Temporal Driver performs the Temporal side
effects and reports exactly what happened.

- Knows behavior: No.
- Knows Temporal: Only the Driver.

#### Judge

Decides whether a recorded Run satisfies its Case's Contract and is explained by the Model.

- Knows behavior: No.
- Knows Temporal: No.

### Rules

- **The Models are the only smart component.** What a feature promises, what counts as evidence, how
  long things may take and what an outcome means are declared in the Models and nowhere else.
- **Everything else is mechanical.** Other components translate, evaluate or execute what was
  declared. A rule of their own is generic, written down once and tested; anything else is a defect
  to move into the Models.
- **Temporal stays at the edges.** Only the Models and Testpilot's Temporal Driver know Temporal.
  The DSL framework, IR, checker, test generator, Testpilot runtime and judge stay
  Temporal-agnostic.
- **One meaning, one source.** The IR has one interpretation that every Go component shares, and
  every fact has one declaration. Other checkers may give a second opinion but never decide.
- **Declared, deterministic, fail closed.** Nothing is defaulted silently; what is not declared is
  refused. The same inputs give the same IR, answers and Cases, and a verdict depends only on the
  Run and its declarations.
- **Artifacts are the interfaces.** Components meet through versioned artifacts (IR, Case, Run) and
  depend inward: features on the kit, the kit on the framework, the Go tools on the IR.

## Requirements

### Remote control via Testpilot (#DRIVE)

#### What

Testpilot runs a generated Case containing a Program and a Contract. The Program tells the controller
and SDK workers what to do; the Contract checks the required observations. The executor and Driver
carry out allowed instructions without adding their own feature-specific behavior or assertions.

#### Why

The test's instructions and checks should come from the generated Case. Separate handwritten worker
scripts or assertions would duplicate behavior and could get out of sync with the Model.

#### Acceptance Test

Drive a Go SDK worker against a real Temporal server from a generated Case, with no handwritten
workflow, activity, or test code for the example. Show the Model source, compiled Model, generated
Case, recorded history, and results of both Contract and Model checks. This demonstrates a programmable
worker; testing an independently handwritten workflow is a separate capability.

### Define regression tests (#REGRESSION)

#### What

Developers save named Queries and check their generated Cases into version control. The same Model
definitions, inputs, bounds, strategy, and random seed, where used, produce exactly the same Case
files. Regression tests run regardless of the time or resources allocated to exploration.

#### Why

Known behavior needs dependable checks on every change, regardless of which tests exploration selects.
Stable test files make changes easy to review and work with existing test infrastructure.

#### Acceptance Test

Regenerate a regression without a diff and run it without the tools used to author the Model.
A behavior change produces a reviewable test change that identifies the Model definitions it came from.

### Replace behavior-focused functional tests (#REPLACE)

#### What

Developers can replace functional tests for behavior covered by the Model, including relevant metric
and log checks. Each observation defines what it means, which operation it belongs to, and when it
must be collected. Specialized unit, race, persistence, schema, authorization, performance, and handler
tests still have a role. Keep monitoring details outside the behavior Model unless a Property needs them.

#### Why

Maintaining the same promise in a Model and in handwritten tests duplicates work and can lead to
disagreement. Replacement must still catch the same failures; specialized tests cover other concerns.

#### Acceptance Test

Replace a complete behavior-focused functional test file. Account for every existing assertion and
show that the replacement catches the same failures. List any unsupported assertions; deleting them
does not count as replacing them.

### Model new features (#DESIGN)

#### What

Before implementing a feature, developers check what it must never do and what it must eventually do.
Each result states its assumptions, finite sets of inputs and states, and search limits.
Reaching a search limit, encountering unsupported or unknown behavior, or finding no path that
meets a Scenario cannot count as success. Checking a Model does not prove that the implementation
matches it; that needs execution evidence. Product promises stay separate from descriptions of
how a buggy implementation behaves.

#### Why

Design errors are easier to investigate before they spread across services and SDKs. Small Models
make design choices easier to compare. Stated assumptions and limits tell developers what was checked.

#### Acceptance Test

Find a path that breaks a Property in a deliberately faulty design. Fix the design without weakening
the Property and report what was checked. An unfinished search remains unresolved.

### Assess implementation behavior (#CONFORMANCE)

#### What

Developers can run tests selected by the Model and check observed behavior, including paths the
generator did not choose. A mapping connects observations to Model actions, committed changes,
operation identities, and cause-and-effect relationships. Within the stated scope, the check keeps
all possible executions that fit the evidence. If that evidence cannot settle a Property, the result
is inconclusive. Reaching a checking limit cannot count as success. Report separately whether the
Contract passed, the behavior matched the Model, the Properties held, execution succeeded, and cleanup
succeeded.

#### Why

The running system can behave in ways the test generator never selected. Checking what happened can
reveal mistakes in the code, the Model, or the mapping between them. Missing evidence must not turn
a possible failure into a pass.

#### Acceptance Test

Detect unexpected behavior and show the first step that differs from the Model. Missing evidence of
a commit, mixed-up operation identities, or unclear event ordering cannot prove a claim that depends
on them. Changing unrelated source timestamps without changing causal order leaves the result unchanged. Test the
checker with independently faulty code and evidence records, as well as faulty Models.

### Faults and recovery (#FAULTS)

#### What

Faults are easy-to-write Model actions linked to controls in the running system. A requested fault
counts only when evidence confirms it happened to the intended resource at the intended point.
Reject tests with missing controls before contacting the target. Report a fault that missed its
intended timing separately from a product bug. Fault tests use authorized, isolated resources.

Recovery Properties state which workers, routes, and storage services must work, which may stay
broken, and how soon progress is expected. Keep checking safety even when a condition needed for
progress is missing or cannot be observed.

#### Why

Many distributed failures depend on a precisely timed interruption or lost acknowledgment. Confirming
the fault and its timing makes the test meaningful. Stating what recovery needs helps distinguish
a stuck implementation from an environment where progress is impossible.

#### Acceptance Test

Hold an activity task, commit a pause, then release the old task. Confirm the actual order and the
server's decision to accept or reject it. After the faults, restore the resources needed for progress
while leaving an unrelated worker unavailable. Distinguish a stuck implementation from missing
resources needed for recovery.

### Exploration (#EXPLORE)

#### What

Developers choose scenarios, variations, faults, and limits, such as testing Nexus with a worker
restart. Searching the Model finds example paths to run against the implementation. Test runs report
which intended paths actually happened. They sample real behavior rather than exhaustively checking
it, even if they attempt every selected target. Use this both to explore designs and to find bugs
in nightly runs.

Today, executable exploration tries a finite set of changes to Queries with fixed paths. More general
conditions on allowed executions, and strategies that adapt to results, are later goals. Each must
state what it supports and where its limits lie.

#### Why

Handwritten scenarios cover only the combinations their authors anticipate. Model searches are cheap
enough to explore more combinations, then spend expensive deployment time on useful examples.

#### Acceptance Test

Find a deliberately introduced bug within a fixed budget. Report separately what the Model search
covered, which Cases could run, which targets actually occurred, which paths could not become
executable Cases, and which Runs were inconclusive.

### Guided exploration (#GUIDANCE)

#### What

Coverage goals in the Model guide test selection toward untested conditions. Report Model states and
transitions, targets reached during execution, inputs tried within each declared class, fault and
control conditions, observations collected, and which Properties were triggered. Attempting a Case
does not mean its target occurred. A requirement whose triggering conditions never occurred was not
tested.

Also use independently written workloads to challenge the generator's assumptions. Include concurrent
operations, duplicate deliveries, and unusual relationships between operation or task identities.

#### Why

Tests can repeat familiar behavior while missing the same race or combination of inputs. Coverage
feedback helps choose what to try next. Independent workloads can reveal blind spots shared by the
Model's examples and the generator.

#### Acceptance Test

Compare guided selection with a baseline under the same budget. Reach more conditions in actual
execution or find more confirmed bugs. Keep a regression for a bug the original input generator missed.

### Replay, reduction, and promotion (#REPLAY)

#### What

Developers can recheck a saved failing Run offline, rerun the test against the implementation, shorten
it while keeping it valid under the Model, and review it for use as a permanent regression. Saved
records include the Model and Case identities, execution Profile, code and runtime versions, and any
controlled schedule or random seed needed to reproduce the failure. Repeatable test generation and
evaluation do not guarantee that a fresh distributed run follows the same internal schedule.

#### Why

A failure becomes useful when a developer can explain it, reproduce it, and keep a focused regression.
Saving the versions, settings, and controls prevents a changed environment or Model from silently
changing what a replay proves.

#### Acceptance Test

Recheck a recorded violation and reproduce the same failure, identified by its failure key, in two
fresh Runs. Shorten the test without losing that failure and produce a reviewable regression. Report
when reproduction fails, and reject saved records that are no longer compatible.

### Feature composition and reuse (#COMPOSE)

#### What

Features combine through shared dependencies, such as CHASM and task delivery, and through interactions,
such as Nexus with Update. Each component states what it promises, what other components must leave
intact, and what progress it needs from them. Reuse a component's results only when those requirements
are met, and keep any remaining assumptions visible.

Checking combined Models and running combined features are separate milestones. Today, Composition
Queries check Models but cannot generate executable Cases. Case generation supports one operation,
with limits on evidence from multiple activities. Cross-feature execution must address those gaps.

#### Why

Features share infrastructure and affect each other's state and progress. Clear component requirements
allow reuse and reveal conflicts that testing each feature alone would miss.

#### Acceptance Test

Reuse a shared dependency model (a provider) in two feature Models, and reject an incompatible provider.
Run a regression with interacting features or concurrent operations. One operation's completion cannot
satisfy another operation's requirement.

### Reusable behavioral protocols (#PROTOCOLS)

#### What

Many Temporal entities offer the same operations: close, terminate, pause and unpause, cancel, describe,
deadlines and retry. Each operation carries a family of promises, not just a request shape. After a
terminate, for example, the entity is closed and stays closed, later changes are rejected, in-flight
work is dropped, and describe and history report it. Umpire states such promises once, as a protocol,
and each feature adopts it.

An entity declares the capabilities it has, such as `Closable`, `Terminable`, `Pausable` or
`Dispatchable`, and binds each capability's parameters: its status type, its closed states, the action
that dispatches work, the error a rejected change returns. Each capability brings its laws, written
with readable named patterns ("once closed, keeps its status", "never dispatched while paused").
Interaction laws apply on their own when an entity has two capabilities together: an entity that is
both `Pausable` and `Dispatchable` promises that nothing is dispatched while paused, without the author
listing that law. Each law says what it does not promise. An entity that differs from a law opts out
or overrides it with a recorded reason. Laws reach the same checks, Queries and generated Cases as
hand-written Properties, and the protocol's realization helpers live in the shared Temporal kit.

A law becomes part of a protocol only when at least two entities adopt it. What the entity does
differently stays in the feature's own Model.

#### Why

Temporal features differ less than their Models do. Restating the same promises in each feature lets
them drift apart, hides which promises are universal and which are feature-specific choices, and makes
every new feature start from nothing. A shared protocol gives a new entity its checks and regression
tests for free, makes deviations explicit, and lets reviewers read one definition of what "terminate"
or "pause" means across the server.

#### Acceptance Test

Declare `Terminable` and `Pausable` for standalone activity and for a second entity. Both get the
shared laws and their generated Cases without restating them, and the interaction law between pausing
and dispatch applies to both without being listed. An entity whose rejection error differs overrides
that law with a recorded reason. A Model that breaks a law is rejected with the law's name and the
entity's binding.

### Select the level of detail (#ZOOM)

#### What

Developers choose how much detail each module needs for the question being checked. Unrelated modules
stay simple. Start with a product/interface Model that knows the public RPCs and user-relevant
commitments. Add checked refinements where a question needs distributed coordination, persistence,
recovery or further detail. There is no fixed number of levels and no requirement that modules have
equal depth. A separate public-protocol Model is optional when it adds behavior beyond the product
Model's request shapes and runtime bindings. The [refinement design](MODEL_REFINEMENT.md) records
this direction and the contracts still to settle.

If a choice changes the search, record the exact model variants, interfaces, mappings
between levels of detail, assumptions, and limits. Show why each reused claim still holds: matching
states alone does not preserve every example path or progress bound. Simplifying only the display
leaves the checked Model unchanged. Tools estimate the chosen Model's complexity and explain the
estimate's limits.

#### Why

Adding detail everywhere makes a Model hard to understand and search. Focusing on the question keeps
important interruption points visible without adding unrelated combinations of states.

#### Acceptance Test

Replace a simple queue model with a detailed provider to check whether an activity can start. Expose
an important interruption without adding detail to unrelated features. Preserve the relevant product
Property and reject a provider that loses committed work.
Refine that provider again to expose a recovery cut while keeping unrelated modules unchanged.
Show which required behavior and progress claims survive each mapping; state matching alone cannot
count as preserving them.

### White-box and black-box environments (#PORTABILITY)

#### What

Developers reuse Models, Properties, and compatible Cases locally, in CI, against deployed clusters,
and in limited production checks (canaries). White-box environments expose internal controls and
evidence; black-box environments expose only public interfaces. Each environment accepts only tests
its permissions, controls, observations, isolation, and impact limits support. Reject tests with
missing capabilities before contacting the target, and at compile time when the environment is known.
A test that is allowed to run may still produce too little evidence for a conclusion.

Model detail and environment access are independent choices. A detailed Model can support design
checking without an environment capable of observing its internals. Runtime bindings state which
actions and evidence an environment supports; they do not introduce another copy of the product
behavior. Distinguish an unknown commitment from a commitment known not to have occurred.

#### Why

A feature's promise should stay the same from local development to production. Environments provide
different controls and evidence. Checks before execution and clear limits on results make those
differences visible without changing the promise.

#### Acceptance Test

Run one unchanged Case locally and against a black-box deployment, stating which resources it uses
in each environment. Reject a white-box-only Case before contacting that deployment. Show how less
evidence limits a claim without changing the promise or assuming different machines' clocks agree.

### Simulated workers and real SDK workers (#WORKERS)

#### What

Before SDK support exists, a simulated worker can stand in for it. Results identify that substitution
and its limits. Simulated workers are not used in production. Moving to a real SDK worker preserves
the product Properties and makes any changes to controls and evidence clear.

#### Why

Server feature development can start before SDK support. A simulated worker lets design and server
checks proceed. Moving to a real worker then shows which checks cover the SDK itself.

#### Acceptance Test

Run a set of tests with a simulated worker, move them to a real SDK worker, and show which claims
now have evidence from the real SDK.

### Explore real implementation code under controlled dependencies (#SIMULATION)

#### What

A separate experiment called "Umpire" runs real Temporal component code with controlled queues,
clocks, or storage. It varies execution schedules independently of the Model's chosen paths and checks
the observed behavior against the same Properties. The real code makes the decisions, rather than a
simulated worker or Driver returning expected answers. Start with one point where the system accepts
or retains work. Expand only if the value justifies maintaining models of more dependencies.

#### Why

Models leave out implementation details that can contain bugs. Running real code with controlled
dependencies can expose those bugs and repeat the schedules that caused them. The same Properties
still define correct behavior.

#### Acceptance Test

Find and reproduce a code bug that searching the Model alone did not expose. Save the source revision
and controlled schedule or random seed. State which dependencies and scheduling choices remain
outside the experiment.

### Feature ownership and lifecycle (#OWNERSHIP)

#### What

Teams and subteams own Model definitions and the mappings that connect them to code. Findings identify
the relevant definitions and owners, including failures that span teams. Ownership changes must not
lose saved regressions' identities or records of where they came from.

Rewrites, migrations, and deprecations state which versions each promise covers. Compatibility checks
cover behavior during the transition, including work already in progress where relevant.

#### Why

Models need to change with the teams and code they describe. Clear ownership tells people who can
act on a finding. Version-specific promises keep rewrites and migrations from losing guarantees.

#### Acceptance Test

Identify the owners of a cross-feature failure. Keep regressions through an implementation rewrite
and test a version transition without weakening the existing promise or treating new behavior as
an already acknowledged bug.

### Known bugs (#KNOWNBUGS)

#### What

Developers can acknowledge a violation without fixing it immediately. While a Known Bug is active,
matching failures produce warnings instead of failing the check, but only for the stated Model or
implementation scope. Keep the violation and evidence visible, and preserve the correct Property,
regression, safety stops, and cleanup. A developer reviews fix evidence and explicitly marks the bug
fixed; if it returns, the regression fails. Unrelated failures and missing evidence are handled as
usual. Outdated or uncertain matches cannot hide errors. Incomplete or inconclusive results prove
neither that the bug returned nor that it was fixed. A clean run can prompt review, but cannot
automatically mark the bug fixed.

#### Why

Teams need to keep testing while a fix is pending, without hiding new failures or weakening the
promise. The Model and implementation can be fixed at different times; fixing one does not prove
the other is fixed.

### Developer experience (#AUTHORING)

#### What

A Temporal developer can write, review, and change a Model without knowing how the checker or executor
works internally. Errors and failing examples point to the relevant declaration, explain what was
expected and observed, and show the first difference. The API should make Models easy to read and
results easy to explain.

Technically comfortable product owners review the actual executable product/interface Model.
It uses public RPCs directly and gives meaningful request shapes readable names, including required
options and allowed values. Shorthands reuse the underlying declarations rather than duplicating
lifecycles or Properties. Request constraints describe what the caller asks for; responses and
evidence establish what happened. A call need not be one atomic transition.

Include distinctions that change what users can rely on, such as an update received but not yet
durable. State what survives which failures and for how long. Keep the storage mechanism and other
internal explanations in refinements, and keep runtime bindings outside the main product narrative.

#### Why

The Model stays useful only if feature developers can maintain it as part of ordinary work. Clear
definitions and failure reports let people outside the Umpire team review Models and investigate bugs.

#### Acceptance Test

A developer outside the Umpire team adds a variation, explains a deliberately introduced failure, and
saves a regression without changing framework internals. Measure time from edit to result, review
effort, and the extra work to map a second feature's observations to its Model.
A product owner explains an update's receipt, durability and completion promises from the executable
Model. A named request shape generates the declared options and recognizes matching recorded
requests from one definition. Contradictory options are rejected, and requesting an acceptance stage
cannot itself prove that acceptance occurred.

## Delivery and acceptance

### What

Demonstrate one small Temporal promise end to end, reusing the prototype and reporting what is still
missing. Start with activity admission: deciding whether an activity may start. Then add a second
feature and show that a replacement dependency model preserves the required promises. Running combined
features, supporting more general exploration conditions, adapting test selection to results, and
simulating real code are separate later milestones. Each needs its own acceptance evidence; generating
a Case or demonstrating a worker does not prove all of them.

### Why

A complete example shows whether writing a Model, checking it, running tests, collecting evidence,
and saving regressions work together. A second feature tests reuse before expanding to broader
exploration and combined features.

### Acceptance Test

A developer states the Property, finds a faulty design, triggers the relevant failure against real
code, fixes the implementation without weakening the promise, and saves a regression. The second
feature follows the same process and shows how much extra Model-writing and mapping work it needs.
