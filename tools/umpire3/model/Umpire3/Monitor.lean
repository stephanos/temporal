import Lean.Data.Json
import Umpire3.Manifest

namespace Umpire3

structure NormalizedObservation where
  identifier : String
  value : Bool
  deriving DecidableEq, Repr

inductive MonitorExpression where
  | observation (identifier : String) (expected : Bool)
  | all (children : List MonitorExpression)
  | any (children : List MonitorExpression)
  | negated (child : MonitorExpression)
  | implies (premise conclusion : MonitorExpression)

def lookupObservation (identifier : String)
    (observations : List NormalizedObservation) : Option Bool :=
  (observations.find? (·.identifier = identifier)).map (·.value)

mutual
  def MonitorExpression.eval (expression : MonitorExpression)
      (observations : List NormalizedObservation) : Option Bool :=
    match expression with
    | .observation identifier expected =>
        (lookupObservation identifier observations).map (· = expected)
    | .all children => MonitorExpression.evalAll children observations
    | .any children => MonitorExpression.evalAny children observations
    | .negated child => (child.eval observations).map (!·)
    | .implies premise conclusion =>
        match premise.eval observations, conclusion.eval observations with
        | some premiseValue, some conclusionValue => some (!premiseValue || conclusionValue)
        | _, _ => none

  def MonitorExpression.evalAll (children : List MonitorExpression)
      (observations : List NormalizedObservation) : Option Bool :=
    match children with
    | [] => some true
    | child :: rest =>
        match child.eval observations, MonitorExpression.evalAll rest observations with
        | some value, some restValue => some (value && restValue)
        | _, _ => none

  def MonitorExpression.evalAny (children : List MonitorExpression)
      (observations : List NormalizedObservation) : Option Bool :=
    match children with
    | [] => some false
    | child :: rest =>
        match child.eval observations, MonitorExpression.evalAny rest observations with
        | some value, some restValue => some (value || restValue)
        | _, _ => none
end

structure MonitorDeclaration where
  identifier : String
  property : String
  evidence : List String
  coverage : List String
  expression : MonitorExpression

def MonitorDeclaration.Holds (declaration : MonitorDeclaration)
    (observations : List NormalizedObservation) : Prop :=
  declaration.expression.eval observations = some true

partial def MonitorExpression.toJson : MonitorExpression → Lean.Json
  | .observation identifier expected => Lean.Json.mkObj [
      ("operation", "observation"),
      ("observation", identifier),
      ("expected", expected),
    ]
  | .all children => Lean.Json.mkObj [
      ("operation", "all"),
      ("children", Lean.Json.arr (children.map MonitorExpression.toJson).toArray),
    ]
  | .any children => Lean.Json.mkObj [
      ("operation", "any"),
      ("children", Lean.Json.arr (children.map MonitorExpression.toJson).toArray),
    ]
  | .negated child => Lean.Json.mkObj [
      ("operation", "not"),
      ("children", Lean.Json.arr #[child.toJson]),
    ]
  | .implies premise conclusion => Lean.Json.mkObj [
      ("operation", "implies"),
      ("children", Lean.Json.arr #[premise.toJson, conclusion.toJson]),
    ]

private def stringsJson (values : List String) : Lean.Json :=
  Lean.Json.arr (values.map Lean.toJson).toArray

def MonitorDeclaration.toJson (declaration : MonitorDeclaration) : Lean.Json := Lean.Json.mkObj [
  ("identifier", declaration.identifier),
  ("property", declaration.property),
  ("evidence", stringsJson declaration.evidence),
  ("coverage", stringsJson declaration.coverage),
  ("expression", declaration.expression.toJson),
]

def monitorCatalogJson (semanticHash catalogHash : String)
    (declarations : List MonitorDeclaration) : String :=
  (Lean.Json.mkObj [
    ("formatVersion", "umpire3/monitor-programs/v1"),
    ("semanticHash", semanticHash),
    ("catalogHash", catalogHash),
    ("programs", Lean.Json.arr (declarations.map MonitorDeclaration.toJson).toArray),
  ]).compress

end Umpire3
