# Umpire DSL opportunities from protocol-modeling languages

Research date: 2026-09-06. This note proposes possible authoring and semantic revisions to the
unimplemented Nexus3 draft and, where useful, to Umpire4 itself. The DSLs are not treated as fixed.
No implementation or accepted specification is changed by this research note. All syntax examples
are conceptual and unimplemented; they illustrate ownership and typing rather than an approved
parser design or spelling.

The comparison is deliberately narrow. Quint and Choreo are useful for typed action composition;
Quint Connect for explicit trace-to-implementation correspondence; Alloy for separate model-finding
and checking commands with visible scopes; Ivy and P for passive, stateful interface monitors; P and
TorXakis for finite scenario composition and input/output ownership; and TLA+ for refinement with
stuttering. None supplies Umpire's full checked bridge from Lean model semantics through correlated
Temporal evidence to a closed Case.

## Ranked opportunities

### 1. Separate controllable inputs from system-produced outputs

The current `Action` type includes both `requestCancel` and `awaitResolution`. The first initiates a
system interaction; the second is observation work that does not cause a product transition. This
distinction is explained in prose and repeated in `Integration.md`, but the Target transition table
still treats both as Actions. That makes a Behavior's action sequence resemble a runtime script and
makes it too easy to count polling as semantic progress.

Borrow the explicit input/output polarity of black-box model-based testing. TorXakis models permitted
inputs and outputs separately and connects those channels to the SUT; P monitors observe events
without producing system side effects. Keep Umpire's `Outcome` as the result of a semantic step, but
distinguish controllable input submission from system output and declare which confirmed boundary
emits each Target transition. Put waiting, polling, and correlation in the Observation/Case
projection rather than in the Target vocabulary.

Conceptual before:

```lean
inductive Action where
  | requestCancel
  | awaitResolution

resolution: cancelRequested + awaitResolution -> oneOf [canceled, succeeded]
```

Conceptual after (unimplemented):

```lean
input submitCancel effect cancelSDKOperationContext
output cancellationRequested
output operationCanceled | completed

transition requestConfirmed: started + output cancellationRequested -> cancelRequested
transition resolution: cancelRequested + output oneOf [operationCanceled, completed] -> terminal

observe cancellationRequested, resolution from correlatedHistory
```

The benefit is semantic ownership: a Behavior may choose when an admitted input is offered but
cannot choose which output the implementation produces. Observation retries become stuttering with
respect to model steps. It also removes the misleading implication that the driver executes an
`awaitResolution` model action to make resolution happen.

Here `submitCancel` authorizes only the Driver effect. It does not establish `cancelRequested` or
consume the confirmed-request semantic step. Only the correlated `cancellationRequested` output
does that. An alternative design may retain one compound request step, but its type must require
both command submission and confirming evidence before the transition is emitted.

The trap is treating outputs as unsolicited raw events. An output must still be an admitted,
role-scoped semantic event reconstructed from correlated evidence; duplicates, unrelated history,
and conflicting evidence cannot advance the model. A command acknowledgement is not automatically
the output it requested. This is a redesign of the Target-facing vocabulary, so it has the largest
migration cost and should be prototyped before cementing the Nexus3 surface.

Sources: [TorXakis getting started](https://torxakis.org/userdocs/stable/getting-started.html),
[P monitor semantics](https://p-org.github.io/P/manual/monitors/), and
[Quint Connect's Driver/State bridge](https://docs.rs/quint-connect/latest/quint_connect/).

### 2. Expose existing Behavior constraints, then evaluate a finite scenario algebra

`actions exactly [...]` is excellent for a pinned Regression but too specific as the default
scenario notation. It makes a scenario brittle when an irrelevant or concurrent semantic step is
added, and it can hide the difference between “this causal pattern must occur” and “these are the
only steps.” P composes a finite checking scenario from modules and attached monitors; Quint runs
compose actions with sequencing and expectations. These support readable scenario composition, but
Umpire should preserve its declarative trace constraints rather than adopt an imperative run model.

The first part is a surface opportunity. The current `BehaviorDeclaration` already represents
allowed and forbidden Actions, required occurrences, occurrence bounds, ordering, sequences,
adjacencies, `actionsExactly`, and `traceExactly`. A frontend can expose those capabilities through
small combinators and lower directly to the existing checked declaration.

Conceptual before:

```lean
behavior cancellationRace on lifecycle
  actions exactly [start: awaitStart, request: requestCancel, resolution: awaitResolution]
```

Conceptual after (unimplemented):

```lean
behavior cancellationRace on lifecycle
  require start: acknowledged
  then submit: input submitCancel
  then request: output cancellationRequested
  then resolution: output oneOf [operationCanceled, completed]
  forbid input submitCancel after resolution
  allow interleaving unrelated
```

Exact replay remains explicit:

```lean
behavior cancellationRaceRegression on lifecycle
  trace exactly [start, submit, request, resolution]
```

The benefit is that exploration scenarios describe causal intent while regressions retain exact
identity. Ordering, adjacency, occurrence bounds, and exactness can expose current functionality.
True union (`either`), bounded repetition, and general interleaving are a separate algebra extension:
they are not necessarily representable by one existing conjunctive `BehaviorDeclaration`. They
would require a checked normal form such as a finite union of Behavior alternatives, plus planner,
identity, satisfiability, and artifact support.

The semantic traps are substantial. `either` must be union, not priority; `then` should mean a
partial-order edge unless named `adjacent`; `interleave` must say which roles may interleave; and
two different surface expressions with the same checked normal form should receive the same
semantic fingerprint. Avoid general regex or process algebra until emptiness, finiteness,
canonicalization, and diagnostic source mapping are defined. Shipping the existing constraint
surface does not require committing to that larger algebra.

Sources: [Quint runs and modes](https://quint.sh/docs/lang),
[P test cases](https://p-org.github.io/P/manual/testcases/), and
[P module composition](https://p-org.github.io/P/manual/modulesystem/).

### 3. Reify language modes and effect capabilities in types

Umpire4 requires separate Property, Behavior, Query, and Observation languages, but their internal
declarations still expose many IDs and encoded values that can be combined incorrectly before the
checker rejects them. Quint assigns every expression a mode and deliberately makes Action and
Temporal modes incomparable. Choreo goes further at the library level: a `Transition` returns local
post-state plus a typed set of effects, and a separate `EffectProcessor` applies custom effects to
the global context.

Use that idea to make the public Lean frontend elaborate into mode-indexed expressions and closed
effect capabilities. This strengthens the existing semantic separation; it need not change the
portable checked representation.

Conceptual before:

```lean
when action requestCancel
require terminalResponse: eventually fact terminal within 1 operation_transition
```

Conceptual after (unimplemented):

```lean
property cancellationResolves on lifecycle :=
  onStep (output cancellationRequested) fun trigger =>
    respond (same operation trigger) (output terminal)
      (within 1 operationTransition)
```

The elaborator should know that `output cancellationRequested` is a step predicate, `same operation
trigger` is a scope key, `output terminal` is a response predicate, and `within` consumes a
compatible logical clock. It should reject a runtime Observation in a Property, a system-output
effect inside a Behavior selector, or milliseconds paired with operation-transition coordinates at
the source expression rather than after a stringly declaration is assembled.

Typed effects can similarly keep Case instructions closed: a model input may lower only through a
declared capability; evidence projection may emit model outputs but cannot dispatch commands; a
Monitor may update private monitor state and return a verdict transition but cannot produce driver
effects.

The trap is creating an elaborate type system that merely mirrors every implementation record.
Keep a small set of modes and capability constructors, elaborate to current canonical data, and
preserve explicit checked IDs at artifact boundaries. Choreo is a design example, not a suitable
dependency: its message-soup and global-context machinery solve a different problem.

Sources: [Quint's mode system](https://quint.sh/docs/lang) and
[Choreo's typed transitions and effects](https://github.com/quint-co/choreo/blob/main/choreo.qnt).

### 4. Make response scope and logical clock first-class clause parameters

`for operation` plus `within 1 operation_transition` states the right intent, but the connection
between trigger identity, response identity, and the counter is distributed across prose. Model the
clause as an obligation template that creates a keyed instance on each trigger:

```text
ResponseClause ScopeKey Clock Trigger Response Bound
```

Conceptual syntax (unimplemented):

```lean
on each request: output cancellationRequested
track by request.operation
expect output terminal where output.operation == request.operation
within 1 tick of request.operation.transitions
```

Each trigger captures an immutable key and starting coordinate. Only matching semantic transitions
advance its clock; matching responses discharge it; trace closure with a live obligation is
inconclusive or violated according to the declared finite-trace semantics. This representation
naturally handles several concurrent operations and repeated requests without a global counter.

The benefit is one exact meaning that can drive finite model evaluation and monitor lowering. It
also makes the current unsupported status honest: a Case producer rejects until its evidence
projection can supply the declared key and clock.

The trap is silently equating “same abstract state” with a stuttering step. A labeled self-loop may
be a counted operation transition. Conversely, duplicate reads and other operations' events are
zero ticks. The clock is defined by admitted labeled semantic steps, not state inequality, wall
time, instruction count, or raw Run Event count. Multiple live obligations also require an explicit
policy: independent discharge is the safe default; coalescing by key changes the Property.

Sources: [TLA+ stuttering](https://lamport.azurewebsites.net/tla/rhtml/stuttering-step.html) and
[Ivy monitors as interface specifications](https://microsoft.github.io/ivy/examples/specification.html).

### 5. Give Query a three-part validity contract: satisfiable, exercised, and answered

Alloy's `run` searches for a satisfying instance while `check` searches for a counterexample to an
assertion, and every command carries a scope. Alloy also warns that a model or predicate with no
instances is inconsistent. Umpire already specifies that an unsatisfiable Behavior is an error,
but a conditional Property can still verify vacuously when its trigger never occurs in a nonempty
Behavior.

Make trigger coverage an explicit query obligation, separate from the Property result:

```lean
query cancellationSafety on lifecycle
  require satisfiable cancellationRace
  require exercises cancellationIsARequest.requestState atLeast 1
  verify cancellationIsARequest in cancellationRace
  limits shortTrace
```

For an unconditional Property, `exercises` is unnecessary. For a witness query, distinguish four
outcomes: witness found, exhaustive absence within semantic Limits, search limit reached, and
scenario unsatisfiable. For universal verification, distinguish verified, counterexample found,
limit reached, unsatisfiable, and nonvacuity failure. These are result statuses, not booleans.

The benefit is that a green conditional verification says both that the relevant behavior occurred
and that every admitted occurrence satisfied the clause. It also turns common modeling mistakes
into focused diagnostics.

The trap is making nonvacuity implicit for every Property. Some invariants intentionally cover
Behaviors where the guarded event is absent. Require it in the Query, or offer a named query policy
whose expansion is recorded in the fingerprint. Search budgets remain separate from trace scopes;
failure to sample a trigger is never proof of its absence.

Source: [Alloy language specification, instances and `run`/`check`](https://alloytools.org/spec.html).

### 6. Compile Properties once to passive monitor IR and reuse it online and offline

P `spec` machines are stateful observers that cannot send, receive, create machines, or otherwise
affect the system. Ivy specifications synchronize monitor actions with calls and returns across an
interface. These are useful precedents for Umpire's existing rule that a Monitor cannot dispatch
work or mutate a Run.

Define a small deterministic monitor IR as the checked lowering target for the supported Property
fragment. The same IR should evaluate a Model Trace, consume a closed Run offline, and create one
fresh Run-local monitor during execution. The evidence projection occurs before the monitor: it
emits typed semantic events with keys and coordinates; the monitor sees no Temporal API records.

Conceptual pipeline (unimplemented):

```text
Property declaration
  -> checked scoped-obligation monitor IR
  -> model-trace evaluator
  -> online Run-local evaluator
  -> offline closed-Run evaluator
```

The benefit is semantic reuse at the highest-risk seam. A bounded response clause, Boolean
combination, and verdict closure rule are implemented once and tested independently. Case
compilation assembles already checked monitor data rather than re-expressing the Property.

The trap is authoring monitors directly. That would create another Property language and violate
AUT-07. Runtime monitoring also does not establish exhaustive model verification, and an offline
Run cannot recover missing or ambiguous evidence. Monitor compilation needs a checked theorem or
executable correspondence test against canonical Property evaluation for every supported form;
unsupported forms reject rather than lower approximately.

Sources: [P passive spec machines](https://p-org.github.io/P/manual/monitors/) and
[Ivy specifications and assume/guarantee ownership](https://microsoft.github.io/ivy/examples/specification.html).

### 7. Make evidence projection a checked refinement with explicit stutter, emit, and reject results

Quint Connect records the selected model action and nondeterministic picks, drives the corresponding
implementation operation, projects implementation state, and compares it with model state. That is
a useful concrete bridge, but direct state equality is insufficient for Temporal: one semantic step
may require several Run Events, irrelevant events must stutter, and missing or conflicting evidence
must fail closed. TLA+ refinement explains why lower-level steps may leave abstract state unchanged,
but Umpire additionally needs labeled-step and logical-clock preservation.

Give each implementation link a total checked result over the relevant evidence window:

```lean
inductive ProjectionResult where
  | stutter (support : EvidenceSupport)
  | emit (event : SemanticEvent) (support : Nonempty EvidenceSupport)
  | reject (diagnostic : ProjectionDiagnostic)
```

Conceptual declaration (unimplemented):

```lean
refinement nexusHistory to lifecycle
  correlate by [namespace, workflowRun, scheduledEvent]
  duplicate historyEventId => stutter
  NexusOperationCancelRequested => emit operation cancellationRequested
  NexusOperationCanceled        => emit operation operationCanceled
  NexusOperationCompleted       => emit operation completed
  otherwise relevant            => reject unsupportedEvidence
```

Every emitted semantic event should retain its exact supporting Run Event sequence and projection
version. A successful runtime replay then establishes conformance of that observed prefix under the
named projection; it does not prove that the implementation has all model behaviors or that a model
witness will materialize. A model checker counterexample still requires Exact Replay before it can
support a Regression claim.

The benefit is reviewable evidence ownership, deterministic deduplication, and precise divergence
reports. It also makes refinement coverage inspectable: every model output needed by a selected
Case must have a supported projection, and every effectful input must have an authorized Driver
capability.

The traps are hiding semantic events based only on unchanged projected state, accepting an emitted
event without nonempty causal support, and claiming full refinement from finite tests. TLA+
stuttering is a semantic guide rather than a library dependency; Quint Connect's current Rust
driver is likewise an implementation reference, not a direct fit for Go/Testpilot or Lean artifacts.

Sources: [Quint Connect state extraction and comparison](https://docs.rs/quint-connect/latest/quint_connect/trait.State.html),
[Quint Connect reproducible trace execution](https://docs.rs/quint-connect/latest/quint_connect/), and
[TLA+ stuttering and refinement background](https://lamport.azurewebsites.net/tla/advanced.html?unhideBut=hide-stuttering&unhideDiv=stuttering).

## Dependency recommendation

These borrowed ideas do not themselves require another external runtime: opportunities 2 through 6
can be explored as Lean frontends and checked canonical Umpire data, while opportunities 1 and 7
need semantic prototypes before choosing syntax. Whether adopting a checker or modeling framework
pays for itself is a separate architecture and maintenance decision. The projects remain useful
executable design references:

| Source | Borrow now | Actual dependency? |
| --- | --- | --- |
| Quint | Expression modes, finite run readability, explicit nondeterminism | No; importing another behavioral authority would violate the single Lean authoring path |
| Choreo | Local transition plus typed effect shape | No; its message-soup runtime and Quint generics are outside Umpire's boundary |
| Quint Connect | Named step dispatch, projected state/evidence divergence, reproducible seeds | No; its Rust/Quint trace bridge does not provide Temporal correlation or Case admission |
| Alloy | Separate witness/check commands, explicit scope, satisfiability diagnostics | No; borrow query semantics and UX |
| Ivy | Interface polarity, stateful monitors, assume/guarantee ownership | No; borrow the monitor and capability model |
| P | Passive spec machines and compositional finite test scenarios | No; borrow observer restrictions and scenario composition |
| TorXakis | Black-box input/output distinction and adapter boundary | No; its separate language/toolchain would duplicate model authority |
| TLA+ | Stuttering refinement and explicit fairness distinctions | No; formalize the corresponding relation in Lean |

Veil's reuse potential, including deeper or core adoption, is evaluated separately. It does not
already implement the seven authoring and evidence opportunities above, so choosing it would not
remove the need to resolve them.

## Suggested order

First define semantic event polarity and the evidence projection result, because they determine what
the Target, Behavior, and runtime monitor consume. Next define the scoped-obligation monitor IR and
prove or test its correspondence with canonical Property evaluation. Then add mode-indexed frontend
expressions and the Query nonvacuity contract. Finally expose existing Behavior constraints through
small canonical combinators, and evaluate union/repetition/interleaving as a distinct extension.
This order avoids polishing `await*` and `actions exactly` syntax that the deeper ownership model
may replace.
