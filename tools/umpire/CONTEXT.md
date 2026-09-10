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
A set of safety and bounded-liveness properties evaluated against a Program and its Run.
_Avoid_: Rulebook, checks, oracle specification

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
from the Run unless separately projected.
_Avoid_: Variable, evidence

**Observation**:
A declared typed field on a Run Event that a Contract is allowed to inspect.
_Avoid_: Raw payload, Slot, log entry

## Testpilot verification

**Evaluator**:
The private Testpilot component that applies a Contract to a Program and its Run, either incrementally or after the Run closes.
_Avoid_: Verifier, Oracle, Referee

**Verdict**:
The Evaluator's conclusion that a Contract is satisfied, violated, or inconclusive, with references to the supporting Run Events.
_Avoid_: Test result, ruling

**Horizon**:
The single bound a bounded-liveness rule declares, in one unit: the Run Events the rule evaluated since its last transition, or elapsed milliseconds on the host that produced the Run.
_Avoid_: Deadline, timeout, TTL

**Fault**:
A deliberate outage a Program asks a Driver to realize, declared as an instruction and recorded as its own Run Event once realized.
_Avoid_: Chaos, failure injection, error

**Profile**:
The authorization snapshot naming the roles, methods, capabilities and resource bindings one Case is permitted to use, frozen before a Driver is built.
_Avoid_: Config, environment, policy file

**Capability**:
One entry of the closed set of instruction kinds a Profile authorizes; a Case whose Program uses one the Profile does not name rejects at Prepare.
_Avoid_: Permission, feature flag, scope
