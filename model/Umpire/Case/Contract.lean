import Umpire.Case.Run

/-!
The closed deterministic Contract monitor IR.

Each rule is a finite state machine evaluated in declaration order over immutable Run Events.
Bounded-liveness horizons close only at recorded timeout or Run-closure coordinates.
-/

namespace Umpire.Case

/-- The version-one monitor classes. -/
inductive ContractRuleKind where
  | safety
  | boundedLiveness
  deriving BEq, DecidableEq, Repr

/-- Whether a Contract state is nonterminal, satisfied, or violated. -/
inductive ContractTerminalState where
  | nonterminal
  | satisfied
  | violated
  deriving BEq, DecidableEq, Repr

/-- One named Contract state and its terminal meaning. -/
structure ContractState where
  stateId : String
  terminal : ContractTerminalState := .nonterminal
  deriving BEq, DecidableEq, Repr

/-- Which Run Event coordinate supports a taken transition. -/
inductive ContractSupport where
  | none
  | matchingEvent
  deriving BEq, DecidableEq, Repr

/-- One declaration-ordered transition from a source state. -/
structure ContractTransition where
  transitionId : String
  sourceState : String
  targetState : String
  eventKinds : List RunEventKind
  predicate : ValueExpression
  support : ContractSupport := .none
  deriving BEq, Repr

/-- A bounded-liveness deadline and the violation state entered when it closes. -/
structure ContractHorizon where
  elapsedMilliseconds : Nat
  violationStateId : String
  deriving BEq, DecidableEq, Repr

/-- One finite deterministic safety or bounded-liveness monitor machine. -/
structure ContractRule where
  ruleId : String
  kind : ContractRuleKind
  initialState : String
  states : List ContractState
  transitions : List ContractTransition
  horizon : Option ContractHorizon := none
  deriving BEq, Repr

/-- Static ceilings for Contract validation and event-by-event evaluation. -/
structure ContractLimits where
  maxRules : Nat
  maxStates : Nat
  maxTransitions : Nat
  maxExpressionDepth : Nat
  maxWorkPerEvent : Nat
  maxTotalWork : Nat
  deriving BEq, DecidableEq, Repr

/-- Closed deterministic monitor machines over declared Run Observations. -/
structure Contract where
  contractId : String
  rules : List ContractRule
  limits : ContractLimits
  deriving BEq, Repr

end Umpire.Case
