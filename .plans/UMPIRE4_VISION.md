# Umpire vision

- Define expected software behavior once and reuse it for design checks, executable tests,
  exploration, and assessment of real executions.
- Give developers a friendly API and beautiful, readable models that expose the feature's promises.
- Keep models small and composable, with explicit relationships between product promises,
  descriptions of implementation behavior, and views at different levels of detail.
- Work across distributed processes without relying on synchronized wall clocks.

This is the product direction, including capabilities beyond the current implementation. The
[shared specification](UMPIRE4_SPEC.md) defines the architectural rules, and
[model semantics](../model/SEMANTICS.md) records supported behavior and current limitations.
[Industry research](UMPIRE4_INSPIRE.md) supplies the rationale for small models, explicit
implementation mappings, controlled faults, independent correctness checks, and qualified results.

## Requirements

### Remote control via Testpilot (#DRIVE)

#### What

Testpilot executes a generated Case containing a Program and a Contract. The Program instructs the
controller and SDK workers; the Contract evaluates declared observations. The executor and Driver
supply generic execution and authorized effects without adding feature-specific behavior or assertions.

#### Why

A generated Case should carry the behavior all the way to execution. Handwritten worker scripts or
assertions for each test would create additional definitions that can drift from the Model.

#### Acceptance Test

Drive a Go SDK worker against a real Temporal server from a generated Case, with no
handwritten workflow, activity, or test code for the example. Show the Model, both IRs, recorded
history, Contract Verdict, and model assessment. State that this demonstrates a programmable worker;
testing an independently handwritten workflow is a separate capability.

### Define regression tests (#REGRESSION)

#### What

Developers retain named Queries whose selected paths become executable Cases checked into version
control. Identical Model definitions, inputs, bounds, strategy, and seed where applicable produce
identical Case bytes. Regressions run independently of exploration budgets.

#### Why

Known behavior needs dependable checks on every change, regardless of what an exploratory campaign
chooses to run. Stable artifacts make changes reviewable and let existing test infrastructure run them.

#### Acceptance Test

Regenerate a regression without a diff and run it without invoking the Model's
authoring toolchain. A behavior change produces a reviewable artifact change with its provenance.

### Replace behavior-focused functional tests (#REPLACE)

#### What

Developers can replace functional suites that exercise modeled product behavior, including relevant
metric and log assertions. Those assertions use declared observations with explicit meaning,
correlation, and observation windows. Specialized unit, race, persistence, schema, authorization,
performance, and handler tests remain complementary. Keep operational instrumentation details outside
the behavior Model unless a concrete Property needs them.

#### Why

Maintaining the same promise in a Model and in handwritten tests duplicates work and invites drift.
Replacement must preserve failure detection, while specialized tests cover concerns outside the Model.

#### Acceptance Test

Replace a complete behavior-focused functional test file, account for every existing
assertion, and demonstrate equivalent failure detection. List unsupported assertions explicitly;
deleting them does not establish replacement.

### Model new features (#DESIGN)

#### What

Developers check a feature's declared safety and progress Properties before implementing it. Each
answer states the assumptions, finite domains, and Limits under which it holds. Exhausted search
budgets, unsupported constructs, unknown behavior, and unsatisfiable Scenarios cannot become success.
Checking a Model establishes claims about that Model; execution evidence tests its correspondence
with the implementation. Product promises remain independent of descriptions of a buggy implementation.

#### Why

Design errors are easier to investigate before they are spread across services and SDKs. Small Models
expose competing decisions, while explicit assumptions and bounds tell developers what was checked.

#### Acceptance Test

Find a counterexample in a deliberately faulty design, correct the design while
keeping the Property, and report the checked scope. A truncated search remains unresolved.

### Assess implementation behavior (#CONFORMANCE)

#### What

Developers can both execute model-selected paths and assess implementation-originated observations
against the Model. The mapping names actions, committed effects, identities, and causal relationships.
Within its declared scope, assessment keeps every execution compatible with partial evidence and
reports inconclusive results when that evidence cannot settle a Property. Reaching an assessment
limit cannot establish success. A satisfied Contract, model conformance, Property satisfaction, and
execution or cleanup success remain separate results.

#### Why

The implementation can take paths the test generator never selected. Assessing observed behavior
reveals discrepancies in code, the Model, or their mapping, and preserves uncertainty when evidence
cannot distinguish a correct execution from an incorrect one.

#### Acceptance Test

Detect an unexpected implementation transition and identify its first divergence.
Missing commit evidence, crossed operation identities, or ambiguous ordering cannot establish a claim
that depends on them. Changing unrelated source timestamps leaves a causally identical assessment
unchanged. Independently faulty implementations and evidence bundles exercise the oracle as well as
faulty Model variants.

### Faults and recovery (#FAULTS)

#### What

Faults are easy-to-author Model actions with explicit runtime controls and evidence of realization.
A requested fault counts only when it occurs on the intended resource at the relevant interruption
point. Missing controls reject before target I/O; an unrealized ordering is reported separately from
a product violation. Faulting Runs use authorized, isolated resources.

Recovery Properties state which workers, routes, and persistence services must be usable, which
failures may remain permanent, and the bound within which progress is expected. Safety remains
checked when a progress premise fails or cannot be observed.

#### Why

Many distributed failures depend on a precise interruption or lost acknowledgment. Observing the
fault and its timing makes the test meaningful; explicit recovery premises distinguish a stuck
implementation from an environment in which progress is impossible.

#### Acceptance Test

Hold an activity dispatch, commit pause, and release the stale delivery. Confirm the
actual ordering and authoritative admission result. After a fault phase, restore the resources needed
for progress while leaving an unrelated worker unavailable, and distinguish a stuck implementation
from missing environmental prerequisites.

### Exploration (#EXPLORE)

#### What

Developers declare scenarios, variations, faults, and bounds, such as exercising Nexus with a worker
restart. Broad, inexpensive Model Search selects witnesses for targeted implementation execution.
Runtime campaigns report which intended paths actually occurred and remain sampled even when they
attempt every declared target. These mechanisms support design exploration and nightly bug finding.

The current executable exploration enumerates finite edits to pinned Queries. General constraints
over executions and adaptive strategies are later capabilities that need their own supported scope.

#### Why

Handwritten scenarios cover only the combinations their authors anticipate. Searching the Model
cheaply lets developers examine more combinations and spend expensive deployment time on selected
witnesses with a clear purpose.

#### Acceptance Test

Under a fixed budget, find a seeded defect and report Model coverage, executable
candidates, realized targets, lowering refusals, and inconclusive Runs separately.

### Guided exploration (#GUIDANCE)

#### What

Model-owned coverage goals guide selection toward useful new conditions. Reports distinguish Model
states and edges, realized runtime targets, input-class members, fault/control conditions,
observation coverage, and Property activation. An attempted Case or an obligation never activated
does not count as a successfully exercised runtime target.

Challenge representative inputs and generator assumptions with independently constructed workloads,
including concurrent operations, duplicate deliveries, and unusual identity relationships.

#### Why

A campaign can repeatedly exercise familiar behavior while missing the same race or input relation.
Coverage feedback directs the next attempt toward untested conditions, and independent workloads
challenge blind spots shared by the Model's examples and generator.

#### Acceptance Test

Compare guided selection with a baseline under the same budget. Demonstrate more
realized conditions or confirmed defects, and retain a defect missed by the original input generator.

### Replay, reduction, and promotion (#REPLAY)

#### What

Developers can take a discovered violation through offline evaluation of its recorded Run, fresh
implementation reruns, Model-admitted reduction, and reviewed promotion into a permanent regression.
Artifacts retain the Model and Case identities, Profile, implementation/runtime versions, and any
controlled schedule or seed needed for reproduction. Deterministic generation and evaluation do not
promise an identical internal schedule in a fresh distributed execution.

#### Why

A discovered failure becomes useful when a developer can explain it, reproduce it, and retain a
focused regression. Keeping the execution basis prevents a changed environment or Model from
silently changing what a replay establishes.

#### Acceptance Test

Reevaluate a recorded violation, reproduce its failure key in two fresh Runs, reduce
it without losing the failure, and produce a reviewable regression. Report failures to reproduce
explicitly and reject incompatible stale artifacts.

### Feature composition and reuse (#COMPOSE)

#### What

Features compose through dependencies such as CHASM and task delivery and through interactions such
as Nexus with Update. Each component declares the interface behavior it supplies, what other
components must preserve, and the progress it depends on. Local results are reusable only where
those obligations are met; remaining assumptions stay visible.

Executable composition is a separate milestone from checking composed Models. Current Composition
Queries are verification-only, and current lowering covers one operation, with limits on evidence
from multiple activities. General cross-feature execution must close those gaps explicitly.

#### Why

Features share infrastructure and affect each other's state and progress. Explicit component
obligations allow reuse while exposing interference that isolated feature checks would miss.

#### Acceptance Test

Reuse a provider in two feature Models, reject an incompatible provider, and execute
a regression with interacting features or concurrent operations. One operation's completion cannot
discharge another's obligation.

### Select the level of detail (#ZOOM)

#### What

Developers select detail per module and question, keeping unrelated components coarse. A selection
that changes Search records its exact variants, interfaces, projections, assumptions, and bounds.
The relationship between variants must justify the claims reused; a state mapping alone does not
preserve every witness or progress bound. Presentation-only simplification leaves the checked Model
unchanged. Tools report the selected scope's complexity and the limits of that estimate.

#### Why

Expanding every subsystem makes a Model difficult to understand and explore. Choosing detail around
the question preserves relevant interruption points without multiplying unrelated state spaces.

#### Acceptance Test

Replace an opaque queue with a detailed provider for an activity-admission check.
Expose a meaningful interruption without expanding unrelated features, preserve the applicable
product Property, and reject a provider that loses committed work.

### White-box and black-box environments (#PORTABILITY)

#### What

Developers reuse Model definitions, Properties, and compatible Cases locally, in CI, against deployed
clusters, and in production canaries. Each environment admits the subset its permissions, controls,
observations, isolation, and impact limits support. Missing declared capabilities reject before
target I/O, and at compile time where the relevant environment is already known. Incomplete runtime
evidence remains inconclusive even after successful admission.

#### Why

A feature's promise should remain consistent from local development to production. Environments
provide different controls and evidence, so admission and result qualification must make those
differences visible without redefining the promise.

#### Acceptance Test

Run one unchanged Case locally and against a black-box deployment with explicit
environment bindings. Reject a white-box-only Case before I/O in that deployment. Show how weaker
evidence limits a claim, without changing the promised behavior or trusting cross-host timestamps.

### Simulated workers and real SDK workers (#WORKERS)

#### What

Before SDK support exists, a simulated worker can stand in for the missing participant. Results name
that substitution and its limits. These workers are not used in production, and migration to a real
SDK worker preserves the product Properties while making any changed controls and evidence explicit.

#### Why

Server feature development can precede SDK support. A simulated participant lets design and server
checks proceed, while an explicit migration shows when the real SDK's behavior has been exercised.

#### Acceptance Test

Run a set of tests with a simulated worker, migrate it to a real SDK worker, and
show which claims now have evidence from the real SDK.

### Explore real implementation code under controlled dependencies (#SIMULATION)

#### What

A separate experiment called "Umpire" runs actual Temporal component code with controlled queue,
clock, or storage dependencies. Its schedules vary independently of model-selected witnesses, and
its observations are assessed against the same Properties. This supplies evidence beyond a simulated worker or a
Driver returning expected answers. Begin with one admission or retention boundary; broader simulation
depends on demonstrated value and the cost of maintaining the dependency models.

#### Why

Abstract Models omit implementation details that can contain bugs. Running real component code
under controlled dependencies can expose those defects and make their schedules repeatable, while
the shared Properties supply the correctness checks.

#### Acceptance Test

Discover and repeat a code defect that abstract search alone did not expose. Retain
the source revision and controlled schedule or seed, and state which dependencies and scheduling
choices remain outside the experiment.

### Feature ownership and lifecycle (#OWNERSHIP)

#### What

Teams and subteams own Model definitions and their implementation mappings. Findings identify the
relevant definitions and owners, including failures at a boundary shared by teams. Ownership changes
must not erase the identities and provenance of retained regressions.

Rewrites, migrations, and deprecations preserve explicit version scopes. Compatibility checks cover
the behavior promised during a transition, including in-flight work where applicable.

#### Why

Models need to evolve with the teams and implementations they describe. Clear ownership makes
findings actionable, and version scopes keep a rewrite or migration from losing an existing promise.

#### Acceptance Test

Attribute a cross-feature failure to the relevant owners. Retain regressions across
an implementation rewrite and demonstrate a version transition without silently weakening the
existing promise or applying an old acknowledgment to new behavior.

### Known bugs (#KNOWNBUGS)

#### What

Developers can acknowledge a discovered violation without fixing it immediately. While a Known Bug
is active, matching occurrences within its declared Model or implementation scope produce warnings
instead of failing the check. The violation and its evidence remain visible, and the correct Property,
regression, safety stops, and cleanup behavior stay intact. After reviewing fix evidence, a developer
explicitly marks the bug fixed; the retained regression makes recurrence an error. Unrelated failures
and missing evidence retain their ordinary treatment. Stale or ambiguous matches cannot suppress
errors; incomplete or inconclusive results establish neither recurrence nor a fix. A clean run can
prompt review but cannot automatically mark a bug fixed.

#### Why

Teams need to continue checking correct behavior while a fix is pending, without hiding new failures
or weakening the promise. The Model and implementation can be fixed at different times, so evidence
of a fix in one cannot establish that the other is fixed.

### Developer experience (#AUTHORING)

#### What

An ordinary Temporal developer can author, review, and change a Model without understanding checker
or executor internals. Errors and counterexamples identify the declaration, expected behavior,
observed evidence, and first divergence. Readability and the ability to explain a result are part of
the API's quality.

#### Why

The Model stays useful only if feature developers can maintain it as part of ordinary work. Readable
declarations and understandable failures make review and diagnosis possible beyond the Umpire team.

#### Acceptance Test

A developer outside the Umpire team adds a variation, explains a seeded
counterexample, and retains a regression without changing framework internals. Measure edit-to-answer
time, review effort, and the additional mapping work required for a second feature.

## Delivery and acceptance

### What

Use one small Temporal promise as the reference demonstration, reusing existing prototype support
and reporting its remaining gaps. Activity admission supplies a concrete starting boundary. Then
demonstrate reuse with a second feature and a checked provider replacement. Executable
cross-feature composition, general exploration constraints, adaptive guidance, and implementation
simulation are distinct follow-on milestones. Each needs its own acceptance evidence; successful
Case generation or a worker demonstration does not establish all of them.

### Why

A complete example reveals whether authoring, checking, execution, evidence, and regression retention
work together. A second feature then tests whether those mechanisms are reusable before the scope
expands to broader exploration and composition.

### Acceptance Test

A developer states the Property, finds a faulty design, realizes the relevant failure against real
code, fixes the implementation without weakening the promise, and retains a regression. The second
feature reuses the same pipeline and makes the additional authoring and mapping effort visible.
