# Umpire

Umpire describes portable, bounded interactions with a target system and determines whether their recorded outcomes satisfy declared properties.

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

## Execution

**Host**:
The boundary that binds a Program's symbolic roles to a target environment and performs its primitive interactions.
_Avoid_: Adapter, driver, harness

**Executor**:
The Umpire component that interprets a Program through a Host and produces a Run.
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

## Verification

**Evaluator**:
The Umpire component that applies a Contract to a Program and its Run, either incrementally or after the Run closes.
_Avoid_: Verifier, Oracle, Referee

**Verdict**:
The Evaluator's conclusion that a Contract is satisfied, violated, or inconclusive, with references to the supporting Run Events.
_Avoid_: Test result, ruling
