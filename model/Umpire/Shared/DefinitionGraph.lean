import Umpire.Core

/-!
Deterministic structural analysis for graphs whose nodes are Umpire Definition IDs.

`DefinitionGraph.analyze` keeps canonical structure, validation findings, topological order, and
cycle evidence separate. Authoring languages consume those stages in their own order and retain
their domain-specific diagnostics and cycle-witness policies.
-/

namespace Umpire.DefinitionGraph

/-- A directed edge between Definition IDs. -/
structure Edge where
  before : DefinitionId
  after : DefinitionId
  deriving BEq, DecidableEq, Repr

/-- Findings that depend only on the supplied node declarations. -/
structure NodeFindings where
  duplicate : Option DefinitionId
  deriving BEq, DecidableEq, Repr

/-- One canonical edge whose source or destination is absent from the graph's nodes. -/
structure UnknownEndpoints where
  edge : Edge
  beforeKnown : Bool
  afterKnown : Bool
  deriving BEq, DecidableEq, Repr

/-- Structural facts for one canonical edge, without a language-specific error policy. -/
structure EdgeEvidence where
  edge : Edge
  beforeKnown : Bool
  afterKnown : Bool
  isSelf : Bool
  hasReverse : Bool
  deriving BEq, DecidableEq, Repr

/-- Independently consumable duplicate, self-edge, and unknown-endpoint findings. -/
structure EdgeFindings where
  duplicate : Option Edge
  self : Option Edge
  unknownEndpoints : List UnknownEndpoints
  perEdge : List EdgeEvidence
  deriving BEq, DecidableEq, Repr

/-- Deterministic evidence from both supported historical cycle-selection traversals. -/
structure CycleEvidence where
  cyclicNodes : List DefinitionId
  canonicalWitness : DefinitionId
  residualPredecessorWalk : List DefinitionId
  residualPredecessorWitness : DefinitionId
  deriving BEq, DecidableEq, Repr

/-- Total staged analysis of one Definition-ID graph. -/
structure Analysis where
  canonicalNodes : List DefinitionId
  canonicalEdges : List Edge
  nodeFindings : NodeFindings
  edgeFindings : EdgeFindings
  topologicalOrder : Option (List DefinitionId)
  cycleEvidence : Option CycleEvidence
  deriving BEq, DecidableEq, Repr

private def definitionIdLe (left right : DefinitionId) : Bool :=
  decide (left.value ≤ right.value)

private def edgeLe (left right : Edge) : Bool :=
  decide (left.before.value < right.before.value) ||
    (left.before == right.before && decide (left.after.value ≤ right.after.value))

private def firstDuplicateEdge : List Edge → Option Edge
  | first :: second :: rest =>
      if first == second then some first else firstDuplicateEdge (second :: rest)
  | _ => none

private structure Graph where
  indegree : Std.HashMap DefinitionId Nat
  outgoing : Std.HashMap DefinitionId (List DefinitionId)
  incoming : Std.HashMap DefinitionId (List DefinitionId)

private def buildGraph (nodes : List DefinitionId) (edges : List Edge) : Graph :=
  edges.foldl (init := {
    indegree := nodes.foldl (init := {}) fun degrees node => degrees.insert node 0
    outgoing := {}
    incoming := {}
  }) fun graph edge => {
    indegree := graph.indegree.modify edge.after (fun count => count + 1)
    outgoing := graph.outgoing.insert edge.before (edge.after :: graph.outgoing.getD edge.before [])
    incoming := graph.incoming.insert edge.after (edge.before :: graph.incoming.getD edge.after [])
  }

private def visitTopologically
    (outgoing : Std.HashMap DefinitionId (List DefinitionId))
    (indegree : Std.HashMap DefinitionId Nat)
    (pending visited : List DefinitionId) :
    Nat → List DefinitionId × Std.HashMap DefinitionId Nat
  | 0 => (visited.reverse, indegree)
  | fuel + 1 =>
      match pending with
      | [] => (visited.reverse, indegree)
      | current :: rest =>
          let (indegree, pending) := (outgoing.getD current []).foldl (init := (indegree, rest))
            fun (degrees, pending) next =>
              let remaining := degrees.getD next 0 - 1
              let degrees := degrees.insert next remaining
              let pending := if remaining == 0 then next :: pending else pending
              (degrees, pending)
          visitTopologically outgoing indegree pending (current :: visited) fuel

private def followResidualPredecessors
    (incoming : Std.HashMap DefinitionId (List DefinitionId))
    (indegree : Std.HashMap DefinitionId Nat)
    (current : DefinitionId)
    (visited : List DefinitionId) : Nat → Option (List DefinitionId × DefinitionId)
  | 0 => none
  | fuel + 1 =>
      if visited.contains current then
        some ((current :: visited).reverse, current)
      else
        let predecessors := (incoming.getD current []).filter
          (fun predecessor => decide (indegree.getD predecessor 0 > 0))
        match predecessors.mergeSort definitionIdLe with
        | predecessor :: _ =>
            followResidualPredecessors incoming indegree predecessor (current :: visited) fuel
        | [] => none

private partial def pathExists
    (edges : List Edge)
    (current target : DefinitionId)
    (visited : List DefinitionId := []) : Bool :=
  if current == target then true
  else if visited.contains current then false
  else
    (edges.filter fun edge => edge.before == current).any fun edge =>
      pathExists edges edge.after target (current :: visited)

/-- Analyze canonical structure and every graph-validation stage without choosing domain errors. -/
def analyze (nodes : List DefinitionId) (edges : List Edge) : Analysis :=
  let canonicalNodes := DefinitionId.canonicalSet nodes
  let sortedEdges := edges.mergeSort edgeLe
  let canonicalEdges := sortedEdges.eraseDups
  let perEdge := canonicalEdges.map fun edge =>
    let beforeKnown := canonicalNodes.contains edge.before
    let afterKnown := canonicalNodes.contains edge.after
    {
      edge
      beforeKnown
      afterKnown
      isSelf := edge.before == edge.after
      hasReverse := canonicalEdges.any fun reverse =>
        reverse.before == edge.after && reverse.after == edge.before
    }
  let unknownEndpoints := perEdge.filterMap fun evidence =>
    if evidence.beforeKnown && evidence.afterKnown then none else some {
      edge := evidence.edge
      beforeKnown := evidence.beforeKnown
      afterKnown := evidence.afterKnown
    }
  let validEdges := canonicalEdges.filter fun edge =>
    canonicalNodes.contains edge.before && canonicalNodes.contains edge.after && edge.before != edge.after
  let graph := buildGraph canonicalNodes validEdges
  let pending := canonicalNodes.filter (fun node => graph.indegree.getD node 0 == 0)
  let (visited, residualIndegree) :=
    visitTopologically graph.outgoing graph.indegree pending [] canonicalNodes.length
  let residualNodes := canonicalNodes.filter
    (fun node => decide (residualIndegree.getD node 0 > 0))
  let cyclicNodes := canonicalNodes.filter fun node =>
    (validEdges.filter fun edge => edge.before == node).any fun edge =>
      pathExists validEdges edge.after node [node]
  let cycleEvidence :=
    match residualNodes, cyclicNodes with
    | start :: _, canonicalWitness :: _ =>
        match followResidualPredecessors graph.incoming residualIndegree start []
            (canonicalNodes.length + 1) with
        | some (residualPredecessorWalk, residualPredecessorWitness) => some {
            cyclicNodes
            canonicalWitness
            residualPredecessorWalk
            residualPredecessorWitness
          }
        | none => none
    | _, _ => none
  {
    canonicalNodes
    canonicalEdges
    nodeFindings := { duplicate := DefinitionId.firstDuplicate nodes }
    edgeFindings := {
      duplicate := firstDuplicateEdge sortedEdges
      self := canonicalEdges.find? fun edge => edge.before == edge.after
      unknownEndpoints
      perEdge
    }
    topologicalOrder := if visited.length == canonicalNodes.length then some visited else none
    cycleEvidence
  }

end Umpire.DefinitionGraph
