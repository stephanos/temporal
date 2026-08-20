import Lean.Data.Json

namespace Umpire3

def formatVersion : String := "umpire3/v2"

def leanVersion : String := "4.33.0"

structure SemanticProofDependency where
  identifier : String
  statementHash : String

structure SemanticProofManifest where
  identifier : String
  theoremName : String
  statementHash : String
  assumptions : List SemanticProofDependency := []

private def SemanticProofDependency.toJson (dependency : SemanticProofDependency) : Lean.Json :=
  Lean.Json.mkObj [
    ("identifier", dependency.identifier),
    ("statementHash", dependency.statementHash),
  ]

def SemanticProofManifest.toJson (manifest : SemanticProofManifest)
    (semanticHash : String) : Lean.Json := Lean.Json.mkObj [
  ("formatVersion", formatVersion),
  ("identifier", manifest.identifier),
  ("theorem", manifest.theoremName),
  ("statementHash", manifest.statementHash),
  ("semanticHash", semanticHash),
  ("leanVersion", leanVersion),
  ("assumptions", Lean.Json.arr (manifest.assumptions.map SemanticProofDependency.toJson).toArray),
]

def SemanticProofManifest.json (manifest : SemanticProofManifest)
    (semanticHash : String) : String :=
  (manifest.toJson semanticHash).compress

end Umpire3
