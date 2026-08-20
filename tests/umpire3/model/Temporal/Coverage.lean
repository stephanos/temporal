import Lean.Data.Json
import Temporal.Catalog
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

private def targetPropertyToJson (target : TargetDeclaration) (property : String) : Lean.Json :=
  if target.identifier = "feature-nexus" && property = "nexus-operation.closure" then
    Lean.Json.mkObj [
      ("identifier", target.identifier),
      ("property", property),
      ("status", "coverage-defined"),
      ("edges", Lean.Json.arr (edges.map Edge.toJson).toArray),
    ]
  else
    Lean.Json.mkObj [
      ("identifier", target.identifier),
      ("property", property),
      ("status", "coverage-undefined"),
      ("reason", "No model-derived coverage denominator has been generated for this target and property."),
      ("edges", Lean.Json.arr #[]),
    ]

private def targetsJson : Array Lean.Json :=
  (Temporal.catalog.targets.flatMap fun target =>
    target.properties.map (targetPropertyToJson target)).toArray

def json (semanticHash catalogHash : String) : String :=
  (Lean.Json.mkObj [
    ("formatVersion", "umpire3/coverage-denominator/v2"),
    ("semanticHash", semanticHash),
    ("catalogHash", catalogHash),
    ("targets", Lean.Json.arr targetsJson)
  ]).compress

end Umpire3.Temporal.Coverage
