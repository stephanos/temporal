/-!
Reachability over a finite list of nodes whose edges are a caller predicate. Callers keep their own
node and key types: correlated events keyed by identity, Evidence records keyed by record identity,
or rule orderings keyed by the rule they precede. Both inputs are finite, so the walk is total.
-/
namespace Shared.Reachability

/--
`reaches nodes key edge before after` holds when `after` is `before`, or is the key of a node the
walk discovers from `before`: a node is discovered when `edge current node` holds for an already
discovered key `current`. The walk is breadth-first over node positions and visits each position at
most once, so a node whose key is `before` is never rediscovered and one pass per node suffices.
-/
def reaches {Node Key : Type} [DecidableEq Key] (nodes : List Node) (key : Node → Key)
    (edge : Key → Node → Bool) (before after : Key) : Bool := Id.run do
  if decide (before = after) then return true
  let nodes := nodes.toArray
  let mut visited := nodes.map fun node => decide (key node = before)
  let mut frontier := [before]
  for _ in [:nodes.size] do
    let current :: rest := frontier | return false
    frontier := rest
    for index in [:nodes.size] do
      if !(visited[index]?).getD false then
        if let some candidate := nodes[index]? then
          if edge current candidate then
            if decide (key candidate = after) then return true
            visited := visited.set! index true
            frontier := frontier ++ [key candidate]
  return false

end Shared.Reachability
