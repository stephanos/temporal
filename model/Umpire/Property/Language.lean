import Umpire.Target

/-! Implementation behind the `Umpire.Property` public facade. -/

namespace Umpire

/-! Portable, pure properties over capability-limited Model Traces. -/

inductive PropertyTraceField where
  | state
  | priorState
  | resultingState
  | selectedAction
  | modelOutcome
  | observation
  | relation
  deriving BEq, DecidableEq, Ord, Repr

def PropertyTraceField.name : PropertyTraceField → String
  | .state => "state"
  | .priorState => "prior-state"
  | .resultingState => "resulting-state"
  | .selectedAction => "selected-action"
  | .modelOutcome => "model-outcome"
  | .observation => "observation"
  | .relation => "relation"

def PropertyTraceField.definitionKind : PropertyTraceField → DefinitionKind
  | .state | .priorState | .resultingState => .state
  | .selectedAction => .action
  | .modelOutcome => .outcome
  | .observation => .observation
  | .relation => .relation

inductive ValueConstraint where
  | present
  | equals (value : String)
  | notEquals (value : String)
  | naturalAtMost (value : Nat)
  | naturalAtLeast (value : Nat)
  deriving BEq, DecidableEq, Ord, Repr

structure PropertyPattern where
  field : PropertyTraceField
  reference : DefinitionId
  constraint : ValueConstraint := .present
  deriving BEq, DecidableEq, Ord, Repr

/-- Match one trace value by Definition ID and exact payload. -/
def PropertyPattern.exact
    (field : PropertyTraceField)
    (reference : DefinitionId)
    (value : String) : PropertyPattern := {
  field
  reference
  constraint := .equals value
}

structure PropertyLimitProfile where
  id : DefinitionId
  source : SourceLocation
  limit : Limit
  deriving BEq, DecidableEq, Repr

inductive PropertyLimit where
  | exact (limit : Limit)
  | named (profile : DefinitionId) (expectedUnit : LimitUnit)
  deriving BEq, DecidableEq, Repr

inductive PropertyClause where
  | stateInvariant (id : DefinitionId) (state : PropertyPattern)
  | transitionContract (id : DefinitionId) (precondition postcondition : PropertyPattern)
  | identityRelation (id : DefinitionId) (relation : PropertyPattern)
  | inputOutput (id : DefinitionId) (input output : PropertyPattern)
  | ordered
      (id : DefinitionId)
      (before after : PropertyPattern)
      (unit : LimitUnit := .semanticTransitions)
  | eventuallyWithin
      (id : DefinitionId)
      (trigger response : PropertyPattern)
      (limit : PropertyLimit)
  | quiescentWithin
      (id : DefinitionId)
      (trigger forbidden : PropertyPattern)
      (limit : PropertyLimit)
  deriving BEq, DecidableEq, Repr

def PropertyClause.id : PropertyClause → DefinitionId
  | .stateInvariant id _
  | .transitionContract id _ _
  | .identityRelation id _
  | .inputOutput id _ _
  | .ordered id _ _ _
  | .eventuallyWithin id _ _ _
  | .quiescentWithin id _ _ _ => id

structure PropertyDeclaration where
  id : DefinitionId
  source : SourceLocation
  version : Nat := 1
  requires : List DefinitionId
  clauses : List PropertyClause
  logicalTimeSource : Option DefinitionId := none
  documentation : String := ""
  deriving BEq, DecidableEq, Repr

/-- An opaque expert declaration is recognizable for rejection, but its callback never enters the
portable declaration, checked property, planner input, or artifact types. -/
inductive PropertyAuthoring where
  | portable (declaration : PropertyDeclaration)
  | opaque (id : DefinitionId) (source : SourceLocation)
  deriving BEq, DecidableEq, Repr

end Umpire
