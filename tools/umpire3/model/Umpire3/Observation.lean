import Lean.Data.Json

namespace Umpire3.Observation

inductive Truth where
  | true
  | false
  | unknown
  | conflict
  deriving BEq, DecidableEq, Repr

structure Source where
  identity : String
  clockDomain : String
  sequence : Nat
  reference : String
  causalReferences : List String := []
  entityIdentity : String
  lineage : List String
  payloadDigest : Option String := none
  deriving BEq, DecidableEq, Repr

structure HistoryEvent where
  eventType : String
  eventID : Nat
  workflowID : Option String := none
  runID : Option String := none
  operationID : Option String := none
  ownerEpoch : Option Nat := none
  currentOwnerEpoch : Option Nat := none
  cancellationCommitted : Option Bool := none
  deriving BEq, DecidableEq, Repr

structure MechanismReceipt where
  action : String
  resource : String
  attempt : Nat
  ownerEpoch : Nat
  outcome : String
  deriving BEq, DecidableEq, Repr

structure EvidenceWindow where
  purpose : String
  closed : Bool
  throughSequence : Nat
  deriving BEq, DecidableEq, Repr

inductive FactValue where
  | history (event : HistoryEvent)
  | mechanism (receipt : MechanismReceipt)
  | window (evidenceWindow : EvidenceWindow)
  deriving BEq, DecidableEq, Repr

structure Fact where
  identifier : String
  source : Source
  value : FactValue
  deriving BEq, DecidableEq, Repr

inductive Operation where
  | exists
  | allExist
  | absentWhenClosed
  | allExistAbsentWhenClosed
  deriving BEq, DecidableEq, Repr

inductive OwnerEpochRelation where
  | equal
  | notEqual
  deriving BEq, DecidableEq, Repr

structure Selector where
  factType : String
  kind : String
  ownerEpochRelation : Option OwnerEpochRelation := none
  cancellationCommitted : Option Bool := none
  outcome : Option String := none
  closed : Option Bool := none
  deriving BEq, DecidableEq, Repr

structure Program where
  identifier : String
  observation : String
  operation : Operation
  matchers : List Selector := []
  violations : List Selector := []
  closures : List Selector := []
  deriving BEq, DecidableEq, Repr

structure Evaluation where
  value : Truth
  support : List String := []
  deriving BEq, DecidableEq, Repr

structure Fixture where
  identifier : String
  observation : String
  facts : List Fact
  expected : Evaluation
  deriving BEq, DecidableEq, Repr

def Selector.matchesFact (selector : Selector) (fact : Fact) : Bool :=
  match fact.value with
  | .history event =>
      selector.factType == "history-event" && selector.kind == event.eventType &&
        selector.outcome.isNone &&
        match selector.ownerEpochRelation with
        | none =>
            match selector.cancellationCommitted with
            | none => true
            | some expected => event.cancellationCommitted == some expected
        | some relation =>
            match event.ownerEpoch, event.currentOwnerEpoch with
            | some ownerEpoch, some currentOwnerEpoch =>
                let relationMatches := match relation with
                  | .equal => ownerEpoch == currentOwnerEpoch
                  | .notEqual => ownerEpoch != currentOwnerEpoch
                relationMatches &&
                  match selector.cancellationCommitted with
                  | none => true
                  | some expected => event.cancellationCommitted == some expected
            | _, _ => false
  | .mechanism receipt =>
      selector.factType == "mechanism-receipt" && selector.kind == receipt.action &&
        selector.ownerEpochRelation.isNone && selector.cancellationCommitted.isNone &&
        selector.closed.isNone &&
        match selector.outcome with
        | none => true
        | some expected => receipt.outcome == expected
  | .window evidenceWindow =>
      selector.factType == "evidence-window" && selector.kind == evidenceWindow.purpose &&
        selector.ownerEpochRelation.isNone && selector.cancellationCommitted.isNone &&
        selector.outcome.isNone && selector.closed == some evidenceWindow.closed

private def appendUnique (values : List String) (value : String) : List String :=
  if values.contains value then values else values ++ [value]

def matchingSupport (selectors : List Selector) (facts : List Fact) : List String :=
  (facts.filter fun fact => selectors.any (·.matchesFact fact)).foldl
    (fun result fact => appendUnique result fact.identifier) []

def conflictSupport (facts : List Fact) : List String :=
  facts.foldl (fun result fact =>
    if facts.any fun other =>
        (other.identifier == fact.identifier && !(other == fact)) ||
          other.source.entityIdentity != fact.source.entityIdentity
    then appendUnique result fact.identifier
    else result) []

def Program.evaluate (program : Program) (facts : List Fact) : Evaluation :=
  let conflicts := conflictSupport facts
  if !conflicts.isEmpty then
    { value := .conflict, support := conflicts }
  else
    match program.operation with
    | .exists =>
        let support := matchingSupport program.matchers facts
        if support.isEmpty then { value := .unknown } else { value := .true, support }
    | .allExist =>
        let complete := program.matchers.all fun selector => facts.any selector.matchesFact
        let support := matchingSupport program.matchers facts
        if complete then { value := .true, support } else { value := .unknown }
    | .absentWhenClosed =>
        let violations := matchingSupport program.violations facts
        if !violations.isEmpty then
          { value := .false, support := violations }
        else
          let closures := matchingSupport program.closures facts
          if closures.isEmpty then { value := .unknown } else { value := .true, support := closures }
    | .allExistAbsentWhenClosed =>
        let violations := matchingSupport program.violations facts
        if !violations.isEmpty then
          { value := .false, support := violations }
        else
          let complete := program.matchers.all fun selector => facts.any selector.matchesFact
          let closures := matchingSupport program.closures facts
          if !complete || closures.isEmpty then { value := .unknown }
          else { value := .true, support := matchingSupport program.matchers facts ++ closures }

theorem evaluate_exists_true_iff (program : Program) (facts : List Fact)
    (operation : program.operation = .exists) (consistent : conflictSupport facts = []) :
    (program.evaluate facts).value = .true ↔
      (matchingSupport program.matchers facts).isEmpty = false := by
  by_cases matched : matchingSupport program.matchers facts = []
  · simp [Program.evaluate, operation, consistent, matched]
  · simp [Program.evaluate, operation, consistent, matched]

theorem evaluate_absence_true_iff (program : Program) (facts : List Fact)
    (operation : program.operation = .absentWhenClosed) (consistent : conflictSupport facts = []) :
    (program.evaluate facts).value = .true ↔
      matchingSupport program.violations facts = [] ∧
        (matchingSupport program.closures facts).isEmpty = false := by
  by_cases violations : matchingSupport program.violations facts = []
  · by_cases closures : matchingSupport program.closures facts = []
    · simp [Program.evaluate, operation, consistent, violations, closures]
    · simp [Program.evaluate, operation, consistent, violations, closures]
  · simp [Program.evaluate, operation, consistent, violations]

theorem evaluate_absence_false_iff (program : Program) (facts : List Fact)
    (operation : program.operation = .absentWhenClosed) (consistent : conflictSupport facts = []) :
    (program.evaluate facts).value = .false ↔
      (matchingSupport program.violations facts).isEmpty = false := by
  by_cases violations : matchingSupport program.violations facts = []
  · by_cases closures : matchingSupport program.closures facts = []
    · simp [Program.evaluate, operation, consistent, violations, closures]
    · simp [Program.evaluate, operation, consistent, violations, closures]
  · simp [Program.evaluate, operation, consistent, violations]

private def stringsJson (values : List String) : Lean.Json :=
  Lean.Json.arr (values.map Lean.toJson).toArray

private def optionStringField (name : String) : Option String → List (String × Lean.Json)
  | none => []
  | some value => [(name, value)]

private def optionNatField (name : String) : Option Nat → List (String × Lean.Json)
  | none => []
  | some value => [(name, value)]

private def optionBoolField (name : String) : Option Bool → List (String × Lean.Json)
  | none => []
  | some value => [(name, value)]

def Source.toJson (source : Source) : Lean.Json := Lean.Json.mkObj <|
  ([
    ("identity", source.identity),
    ("clockDomain", source.clockDomain),
    ("sequence", source.sequence),
    ("reference", source.reference),
  ] : List (String × Lean.Json)) ++
  (if source.causalReferences.isEmpty then [] else
    [("causalReferences", stringsJson source.causalReferences)]) ++
  ([
    ("entityIdentity", source.entityIdentity),
    ("lineage", stringsJson source.lineage),
  ] : List (String × Lean.Json)) ++ optionStringField "payloadDigest" source.payloadDigest

def HistoryEvent.toJson (event : HistoryEvent) : Lean.Json := Lean.Json.mkObj <|
  ([
    ("eventType", event.eventType),
    ("eventID", event.eventID),
  ] : List (String × Lean.Json)) ++ optionStringField "workflowID" event.workflowID ++
    optionStringField "runID" event.runID ++
    optionStringField "operationID" event.operationID ++
    optionNatField "ownerEpoch" event.ownerEpoch ++
    optionNatField "currentOwnerEpoch" event.currentOwnerEpoch ++
    optionBoolField "cancellationCommitted" event.cancellationCommitted

def MechanismReceipt.toJson (receipt : MechanismReceipt) : Lean.Json := Lean.Json.mkObj [
  ("action", receipt.action),
  ("resource", receipt.resource),
  ("attempt", receipt.attempt),
  ("ownerEpoch", receipt.ownerEpoch),
  ("outcome", receipt.outcome),
]

def EvidenceWindow.toJson (window : EvidenceWindow) : Lean.Json := Lean.Json.mkObj [
  ("purpose", window.purpose),
  ("closed", window.closed),
  ("throughSequence", window.throughSequence),
]

def Fact.toJson (fact : Fact) : Lean.Json := Lean.Json.mkObj <| ([
  ("identifier", fact.identifier),
  ("source", fact.source.toJson),
] : List (String × Lean.Json)) ++ match fact.value with
  | .history event => [("history", event.toJson)]
  | .mechanism receipt => [("mechanism", receipt.toJson)]
  | .window evidenceWindow => [("window", evidenceWindow.toJson)]

def OwnerEpochRelation.toJsonString : OwnerEpochRelation → String
  | .equal => "equal"
  | .notEqual => "not-equal"

def Selector.toJson (selector : Selector) : Lean.Json := Lean.Json.mkObj <| ([
  ("factType", selector.factType),
  ("kind", selector.kind),
] : List (String × Lean.Json)) ++ (match selector.ownerEpochRelation with
  | none => []
  | some relation => [("ownerEpochRelation", relation.toJsonString)]) ++
  optionBoolField "cancellationCommitted" selector.cancellationCommitted ++
  optionStringField "outcome" selector.outcome ++
  optionBoolField "closed" selector.closed

def Operation.toJsonString : Operation → String
  | .exists => "exists"
  | .allExist => "all-exist"
  | .absentWhenClosed => "absent-when-closed"
  | .allExistAbsentWhenClosed => "all-exist-absent-when-closed"

def Program.toJson (program : Program) : Lean.Json := Lean.Json.mkObj <| ([
  ("identifier", program.identifier),
  ("observation", program.observation),
  ("operation", program.operation.toJsonString),
] : List (String × Lean.Json)) ++ (if program.matchers.isEmpty then [] else
  [("matches", Lean.Json.arr (program.matchers.map Selector.toJson).toArray)]) ++
  (if program.violations.isEmpty then [] else
    [("violations", Lean.Json.arr (program.violations.map Selector.toJson).toArray)]) ++
  (if program.closures.isEmpty then [] else
    [("closures", Lean.Json.arr (program.closures.map Selector.toJson).toArray)])

def Truth.toJsonString : Truth → String
  | .true => "true"
  | .false => "false"
  | .unknown => "unknown"
  | .conflict => "conflict"

def Evaluation.toJson (evaluation : Evaluation) : Lean.Json := Lean.Json.mkObj <| ([
  ("value", evaluation.value.toJsonString),
] : List (String × Lean.Json)) ++ if evaluation.support.isEmpty then [] else [("support", stringsJson evaluation.support)]

def Fixture.toJson (fixture : Fixture) : Lean.Json := Lean.Json.mkObj [
  ("identifier", fixture.identifier),
  ("observation", fixture.observation),
  ("facts", Lean.Json.arr (fixture.facts.map Fact.toJson).toArray),
  ("expected", fixture.expected.toJson),
]

def catalogJson (semanticHash catalogHash : String) (programs : List Program)
    (fixtures : List Fixture) : String :=
  (Lean.Json.mkObj [
    ("formatVersion", "umpire3/observation-programs/v1"),
    ("semanticHash", semanticHash),
    ("catalogHash", catalogHash),
    ("programs", Lean.Json.arr (programs.map Program.toJson).toArray),
    ("fixtures", Lean.Json.arr (fixtures.map Fixture.toJson).toArray),
  ]).compress

end Umpire3.Observation
