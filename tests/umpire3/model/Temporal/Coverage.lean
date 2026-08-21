import Lean.Data.Json
import Temporal.Catalog
import Temporal.Composition
import Temporal.Monitors
import Temporal.Product.NexusLifecycle

namespace Umpire3.Temporal.Coverage

open Umpire3.Temporal.Product.NexusLifecycle

private def Edge.toJson (edge : Edge) : Lean.Json := Lean.Json.mkObj [
  ("identifier", edge.identifier),
  ("fromState", edge.fromState),
  ("action", edge.action),
  ("toState", edge.toState),
  ("requiresFault", edge.requiresFault),
  ("standaloneOnly", edge.standaloneOnly)
]

structure Point where
  dimension : String
  identifier : String
  source : String

private def Point.toJson (point : Point) : Lean.Json := Lean.Json.mkObj [
  ("dimension", point.dimension),
  ("identifier", point.identifier),
  ("source", point.source),
]

private def targetProjection (identifier : String) : Option TargetProjection :=
  (Umpire3.Temporal.Composition.composition.targets).find? (·.identifier = identifier)

private def targetModules (target : TargetProjection) : List ModuleContract :=
  target.modules.filterMap fun identifier =>
    (Umpire3.Temporal.Composition.composition.modules).find? (·.identifier = identifier)

private def targetCapabilities (target : TargetProjection) : List String :=
  target.retainedActions.flatMap fun identifier =>
    match (Umpire3.Temporal.catalog.actions).find? (·.identifier = identifier) with
    | none => []
    | some action => action.requiredCapabilities

private def transitionPoints (target : TargetProjection) : List Point :=
  target.retainedActions.map fun action => {
    dimension := "transition"
    identifier := target.identifier ++ "/" ++ action
    source := "composition:" ++ target.identifier
  }

private def relationPoints (target : TargetProjection) : List Point :=
  (targetModules target).flatMap fun module =>
    (module.owns.filter (·.startsWith "relation:")).map fun relation => {
      dimension := "relation"
      identifier := target.identifier ++ "/" ++ relation
      source := module.identifier
    }

private def propertyPoints (target : TargetProjection) (property : String) : List Point :=
  match (Umpire3.Temporal.catalog.properties).find? (·.identifier = property) with
  | none => []
  | some declaration => [{
      dimension := "property"
      identifier := target.identifier ++ "/" ++ property
      source := declaration.theoremName
    }]

private def faultPoints (target : TargetProjection) : List Point :=
  let capabilities := targetCapabilities target
  ((Umpire3.Temporal.catalog.faults).filter fun fault =>
    fault.requiredCapabilities.all capabilities.contains).map fun fault => {
      dimension := "fault"
      identifier := target.identifier ++ "/" ++ fault.identifier
      source := "catalog:" ++ fault.identifier
    }

private def observationPoints (target : TargetProjection) (property : String) : List Point :=
  match Monitors.declarations.find? (·.property = property) with
  | none => []
  | some monitor => monitor.coverage.map fun observation => {
      dimension := "observation"
      identifier := target.identifier ++ "/" ++ observation
      source := monitor.identifier
    }

private def refinementPoints (target : TargetProjection) : List Point :=
  (targetModules target).flatMap fun module =>
    (module.obligations.filter (·.kind = "refinement")).map fun obligation => {
      dimension := "refinement"
      identifier := target.identifier ++ "/" ++ obligation.identifier
      source := module.identifier
    }

private def points (target : TargetProjection) (property : String) : List Point :=
  transitionPoints target ++ relationPoints target ++ propertyPoints target property ++
    faultPoints target ++ observationPoints target property ++ refinementPoints target

private def targetPropertyToJson (target : TargetDeclaration) (property : String) : Lean.Json :=
  match targetProjection target.identifier with
  | none => Lean.Json.mkObj [
      ("identifier", target.identifier),
      ("property", property),
      ("status", "coverage-undefined"),
      ("reason", "The catalog target has no checked composition projection."),
      ("points", Lean.Json.arr #[]),
      ("edges", Lean.Json.arr #[]),
    ]
  | some projection =>
      let targetEdges := if target.identifier = "feature-nexus" &&
          property = "nexus-operation.closure" then edges else []
      Lean.Json.mkObj [
        ("identifier", target.identifier),
        ("property", property),
        ("status", "coverage-defined"),
        ("points", Lean.Json.arr ((points projection property).map Point.toJson).toArray),
        ("edges", Lean.Json.arr (targetEdges.map Edge.toJson).toArray),
      ]

private def targetsJson : Array Lean.Json :=
  ((Umpire3.Temporal.catalog.targets).flatMap fun target =>
    target.properties.map (targetPropertyToJson target)).toArray

def json (semanticHash catalogHash : String) : String :=
  (Lean.Json.mkObj [
    ("formatVersion", "umpire3/coverage-denominator/v3"),
    ("semanticHash", semanticHash),
    ("catalogHash", catalogHash),
    ("targets", Lean.Json.arr targetsJson)
  ]).compress

end Umpire3.Temporal.Coverage
