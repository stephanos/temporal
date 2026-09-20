# Umpire

Umpire authors and produces versioned, bounded Cases. Testpilot executes those Cases and determines whether their recorded outcomes satisfy declared properties.

## Definition

**Producer**:
A source that creates an Umpire Case. A compiler is a Producer that translates another representation, such as a Lean model.
_Avoid_: Planner, author

**Case**:
A coherent pairing of one Program and one Contract.
_Avoid_: Scenario, experiment, plan

**Program**:
A bounded acyclic graph of instructions describing interactions and declared captures.
_Avoid_: Plan, Playbook, script

**Contract**:
A set of safety and bounded-liveness Rules evaluated against a Program and its Run.
_Avoid_: Rulebook, checks, oracle specification

**Rule**:
One state machine inside a Contract, with an initial state, finite transitions, terminal satisfied or violated states, and, when it is a bounded-liveness Rule, one Deadline.
_Avoid_: Property (that is the model-side term), assertion, check

**Correlated**:
A Rule tracked separately per operation and correlated by an explicit key, so one operation's obligations never discharge another's.
_Avoid_: Scoped, per-instance, keyed, clause (a Correlated Rule is not the Property clause it is lowered from)

**Expression**:
The one expression language of a Case: a literal, a Reference, a field path read, a presence test, a comparison, or the negation, conjunction or disjunction of Expressions. Instruction inputs and guards, Contract transition predicates, correlated rule conditions and evidence-lift guards are all Expressions, and where one appears decides which References it may use.
_Avoid_: Program expression, Contract expression, correlated predicate, correlation operand

**Reference**:
The leaf of an Expression that names a value instead of spelling it: a Slot, an instruction outcome, the Run, an environment binding, an Observation, the evaluated Run Event, a capture, an evidence field, a correlated capture or step, or the value an evidence lift projects. Preparation rejects a Reference the Expression's context does not admit, located at its path.
_Avoid_: Ref, variable, operand (that is whatever Expression a path, comparison or negation applies to)

**Case-local name**:
The short name a Case's Program and Contract use for a Definition ID: the shortest dotted suffix no other Definition ID in the Case shares, such as `operation-identity`. Provenance maps each local name that differs from its Definition ID back to it; a model value is likewise written as its declared spelling, and a parameterized value's encoding is recorded in provenance as a fingerprint.
_Avoid_: Alias, short ID, Definition ID (that is the global name a local name stands for)

## Umpire authoring

**Admitted Query**:
A Query that has passed every check it needs before it can be searched against one checked Model: its Property, its Scenario, its Known Gaps, the Query itself, and the search view over that Model. It is the model-side authoring sense of a Query, not a Case or a Run, and it makes no claim about what a search will find.
_Avoid_: Checked Query (that is only the Query stage), planned Query, kernel

**Evidence structure**:
The ordering facts, closures, and per-link support of one offline Evidence bundle or accepted trace, analyzed once and judged for one audience, the raw bundle or the accepted trace, which each report the first fault in their own precedence. It belongs to Umpire's offline Evidence, not to the Testpilot Observation a Contract inspects, and its faults say whether evidence can be ordered and closed, never whether a Property holds.
_Avoid_: Findings, structural analysis, Observation structure

**Derived rule**:
One monitor Rule a Case carries whose read paths, comparison and literal are derived from a checked field Property, together with the certificate that every field it reads is one the Property compares and every literal is one the Case's realization assigns. It is a single Rule of a Contract, not the Contract itself, and it lives on the model side of the seam: Testpilot evaluates it like any other Rule and never sees the Property it came from.
_Avoid_: Authored rule, Contract (that is the set of Rules), monitor Property

**Entity**:
Something with identity that a machine keeps state for, declared with the entities it refers to and the key recorded data names an instance by. State belongs to the machines that track it, not to the entity.
_Avoid_: Role (that is the runtime's symbolic resource), object, resource

**Party**:
Who performs an action, declared by naming it on the action. `system` is the implementation under test, performs no declared action and owns the timers; a set binds every other party.
_Avoid_: Actor, client, environment (that was a party name, and a binding, and collided with both)

**Action**:
What a party does to an entity, with typed inputs over finite domains and an optional protobuf schema. Each member of an input domain is a class; a fault is an action of the party that causes it, and a timer is not an action but `system` behavior a machine owns.
_Avoid_: Interface (rejected as a method-set contract duplicating Action), operation (that is the entity), command, RPC (those are what a realization binds an action to)

**Machine**:
One entity's state and what each action does to it, written as a step function over a structure of finite fields and enumerated into the checked table. It names its starts, ends, timers, setup parameters and the observation that confirms each Fact.
_Avoid_: Model (that is the checked behavior as a whole), statemachine, state machine, mechanism machine (MOD-02 gives mechanisms to `Temporal.System`)

**Refinement**:
A machine that refines another through a state map, checked as a stuttering forward simulation over the two tables, so a Property declared on the abstract machine is read on the refining one's paths.
_Avoid_: Link, Implementation Link (SEM-08 reserves that for the Feature-to-System connection), abstraction (that is the map's direction, not the relation)

**Set**:
Queries grouped by purpose -- functional, canary or exploratory -- with every party but `system` bound `driven` (the Case performs its actions) or `observed` (the world does, and the verifier reads which class occurred).
_Avoid_: Suite, test binding, environment binding (the earlier names of `driven` and `observed`), campaign

**Realization**:
The platform-owned binding of a Model to a runtime: each action class to an instruction, each observation to where it is recorded, each timer to a duration, each switch to its values. The Producer assembles a Case's Program and Contract from a Query's witness and it.
_Avoid_: Template, Program template, adapter, driver (that is the runtime side)

**Abstraction Claim**:
An `examples:` line on an action: the claim that every realized value of a class behaves alike, with the example a functional Case runs and records in its provenance. An exploration tries the other members.
_Avoid_: Representative (the earlier word for the example), equivalence class (a class is a member of a domain), coverage

## Testpilot execution

**Driver**:
The boundary that binds a Program's symbolic roles to a target environment and performs its primitive interactions.
_Avoid_: Host, harness

**Executor**:
The private Testpilot component that interprets a Program through a Driver and produces a Run.
_Avoid_: Runner, engine, player

**Run**:
The authoritative record of one attempted Program execution, including its ordered Run Events and terminal disposition.
_Avoid_: Trace, log, result

**Run Event**:
An immutable fact appended to a Run about an execution attempt, outcome, or lifecycle transition.
_Avoid_: Calling the entire event an Observation, callback

**Slot**:
Immutable, single-assignment typed operational data passed between Program instructions and omitted
from the Run unless separately recorded as an Observation. A Slot is also the only place an opaque
effect handle lives.
_Avoid_: Variable, evidence, opaque capability (a handle a Slot holds is an opaque handle)

**Observation**:
A declared typed field on a Run Event that a Contract is allowed to inspect.
_Avoid_: Raw payload, Slot, log entry

**Response read**:
A declared read of one field path out of a successful RPC response into Slots, Observations or correlated evidence, once or once per element. It is neither Umpire's Projection, which reads Run values into model Steps, nor the correlated projection a `Contract.correlated` capability admits steps through.
_Avoid_: Response projection, projection, extraction

**Outage plan**:
The worker Driver's internal admission answer for the Faults one Program declares: the task queue each named role resolves to, and whether the Run needs worker groups no other Run shares. It is neither the Fault instruction the Program declares nor the Run Event recorded once a Fault is realized, and it never leaves the Driver.
_Avoid_: Plan, fault plan, outage schedule

## Testpilot verification

**Evaluator**:
The private Testpilot component that applies a Contract to a Program and its Run, either incrementally or after the Run closes.
_Avoid_: Verifier, Oracle, Referee

**Verdict**:
The Evaluator's conclusion that a Contract is satisfied, violated, or inconclusive, with references to the supporting Run Events.
_Avoid_: Test result, ruling

**Deadline**:
The single bound a bounded-liveness Rule declares, in one unit: the Run Events the Rule evaluated since its last transition, or elapsed milliseconds on the host that produced the Run.
_Avoid_: Horizon, timeout, TTL

**Fault**:
A deliberate outage a Program asks a Driver to realize, declared as an instruction and recorded as its own Run Event once realized.
_Avoid_: Chaos, failure injection, error

**Profile**:
The authorization snapshot naming the roles, methods, Opcodes, resource bindings and resource ceilings one Case is permitted to use, frozen before a Driver is built. A Case declares only the bounds that carry its behavior; the Profile's ceilings bound everything else.
_Avoid_: Config, environment, policy file

**Opcode**:
One entry of the closed set of instruction kinds a Profile authorizes; a Case whose Program uses one the Profile does not name rejects at Prepare. Distinct from the spec's **Capability**, the named behavior one model component requires and another supplies, and from the Correlated Contract capability a Case's `Contract.correlated` carries; always say which one.
_Avoid_: Capability, permission, feature flag, scope
