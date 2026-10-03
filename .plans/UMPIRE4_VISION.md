# Umpire vision

- define a single model for software behavior
- very developer-friendly API to define models and tests
- model should be beautiful
- works in distributed processes where clock skew can happen

---

## Use cases

### Remote control via TestPilot

To instruct the Go test execution, the TestPilot IR is used to define the test plan.
The test driver/executor is dumb; it merely follows the instructions in the IR.
For Go Workers, they need to be instructed on how to respond to inputs with the TestPilot IR (similiar to how kitchensink works).

Acceptance test: A Go SDK worker is driven by the TestPilot IR.

### Define set of regression tests

Developers can define a clear set of traces/paths through the model that are converted into executable tests.
Those tests are defined in an IR that can be checked into version control.
Re-generating the IR from the model should always produce the same set of tests.

### Replacement of functional tests

Developers want to use the model to replace all existing functional tests.
This means anything the functional tests can drive in behavior and assert must be supported.
For example, not to forget metrics and log assertions. 

Acceptance test: show that a complete functional test suite/file can be replaced by the model.

### Modeling new features

Developers want to use the model to verify if a new feature they want to add is safe and correct.

### Faults

Faults are an integral part of the model; they are first class citizens.
Faults should be easy to define and use.
They need tight integration with the runtime implementation to actually observe them.

Acceptance test: Regression and exploration tests that use faults.

### Exploration

Developers can explore the model using various strategies.
For instance, they can define certain constraints such as "use Nexus". or "always restart the worker" (a fault).
So basically the developer defines certain parameters/constraints and the model explores the state space automatically.
This is helpful when (a) simply hunting for bugs or (b) understanding the system's behavior under certain conditions or (c) reproducing a bug.
This will also used in nightly CI jobs to catch regressions.

Acceptance test: Show that the model can explore many different scenarios and find bugs. 

### Guided Exploration

An advanced way to run explorations is to use coverage-guided exploration; meaning that the model is guided by coverage data to explore the system.
Discovering new behavior is a reward that the model should aim to maximize.

### Orthogonal and re-use

The model needs to be multi-dimensional.
Features are composed in breadth and depth: a feature relies on underlying functionality (e.g. CHASM, task delivery) and can work with other features (e.g. Nexus and Update). The model needs to reflect that.
Features have lifecycles: there are rewrites (new versions), migrations, and deprecations.
Since features rely on other features, they must integrate with them.

### Level of Detail

Since the entire system of Temporal is complex, the model would become too large.
To counter that, individual models should allow different levels of detail.
That means, a model, for example, for task matching should allow different levels of detail: coarse (e.g. matching by task type) and fine (e.g. matching by task attributes).
During trace/path exploration, the developer can select the level of detail they want to see; and receive feedback on the complexity of the selected model subset.

### White box vs Black box

Developers can use the same model and regression tests for verifying locally, on CI, CICD (cloud deployment) and canary (production).
When a developer selects a model path that requires white box access in an environment that only provides gRPC access, they receive an error immediately, ideally at compile time.

Acceptance test: Show an example of a set of tests for functional tests (white box) and canary (black box).
Acceptance test: Show an example how particular tests are not executable with canary since they require white box.

### Simulated Worker vs Real SDK Worker

When a new feature is developed, there is no Worker impl available yet, so the simulated worker is used instead.
It needs to be clear to the developer that the simulated worker is not a real Worker and should not be used in production.
It needs to be easy to migrate to the real SDK Worker when it becomes available.

Acceptance test: Show an example of a set of tests that can be run with the simulated worker and are migrated to the real SDK Worker when it becomes available.

### Ownership by Feature

Following Conway's Law, the model should reflect the reality of the system being modeled: multiple teams own separte features.
And even within a feature - like "Workflow" - there are sub-team ownership rules to apply.
This means the model needs to be very composable/extensible. Maybe one team owns the basics and another builds a separate feature on top.
And when finding a bug, this needs to be retracable to the owning team.

### Known bug

See temporal/model/specs/KNOWN_BUG.md
