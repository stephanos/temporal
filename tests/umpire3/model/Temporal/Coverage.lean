import Lean.Data.Json
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

def json (semanticHash catalogHash : String) : String :=
  (Lean.Json.mkObj [
    ("formatVersion", "umpire3/coverage-denominator/v1"),
    ("semanticHash", semanticHash),
    ("catalogHash", catalogHash),
    ("targets", Lean.Json.arr #[Lean.Json.mkObj [
      ("identifier", "feature-nexus"),
      ("property", "nexus-operation.closure"),
      ("edges", Lean.Json.arr (edges.map Edge.toJson).toArray)
    ]])
  ]).compress

end Umpire3.Temporal.Coverage
