import DslExperiment

/-! Executable differential tests and stable receipts. `--timing` adds an explicitly variable
measurement; ordinary output is deterministic and suitable for a repeated-run comparison. -/

open DslExperiment

private def resolves : ResponseClause :=
  whenever (.event .requested) eventually (.anyOf [.canceled, .completed]) within 1 operationTransitions

private def words : Nat → List (List Step)
  | 0 => [[]]
  | n + 1 => [] :: (words n).flatMap fun trace => alphabet.map fun step => step :: trace

private def requireCheck (condition : Bool) (description : String) : IO Unit :=
  unless condition do throw (IO.userError description)

private def checkProperties : IO Nat := do
  let mut comparisons := 0
  let predicates : List StepPredicate :=
    [.event .requested, .event .tick, .anyOf [.canceled, .completed], .anyOf []]
  for trace in words 4 do
    for trigger in predicates do
      for response in predicates do
        for bound in List.range 3 do
          let clause : ResponseClause := ⟨trigger, response, bound, .operationTransitions⟩
          for closure in [Closure.closedModel, .runtimePrefix] do
            requireCheck (evaluate clause closure trace == reference clause closure trace)
              s!"monitor/reference mismatch: {repr (clause, closure, trace)}"
            comparisons := comparisons + 1
  return comparisons

private def encode (trace : List Step) : List Evidence := Id.run do
  let mut nextId := 1
  let mut parents : Nat × Nat := (0, 0)
  let mut events := []
  for step in trace do
    let mut parent := if step.operation == 0 then parents.1 else parents.2
    if step.event == .requested then
      events := events ++ [⟨7, nextId, step.operation, .submitCancel, none⟩]
      parent := nextId
      nextId := nextId + 1
    events := events ++ [⟨7, nextId, step.operation, .observed step.event, some parent⟩]
    parents := if step.operation == 0 then (nextId, parents.2) else (parents.1, nextId)
    nextId := nextId + 1
  return events

private def operationSteps (operation : Fin 2) (steps : List Step) : List Step :=
  steps.filter fun step => step.operation == operation

private def checkEvidence : IO Nat := do
  let search := explore 5 1000
  requireCheck search.complete "evidence corpus unexpectedly incomplete"
  let mut comparisons := 0
  for trace in search.traces do
    let events := encode trace
    let noise : Evidence := ⟨999, 1, 0, .gap, none⟩
    let variants := [events, events.reverse, events.flatMap (fun event => [event, event]),
      noise :: events ++ [⟨7, 999, 1, .unrelated, none⟩]]
    for variant in variants do
      let observed := evaluateEvidence resolves 7 variant
      let emitted := observed.projection.emitted.map Emission.step
      requireCheck observed.projection.error.isNone s!"projection rejected valid trace: {repr variant}"
      requireCheck observed.projection.pending.isEmpty "causally supported evidence remained pending"
      for operation in [0, 1] do
        requireCheck (operationSteps operation emitted == operationSteps operation trace)
          "projection changed per-operation causal order"
      requireCheck ((replay emitted) == replay trace) "projection changed final state"
      requireCheck (observed.finish == evaluate resolves .runtimePrefix trace)
        "online/offline property mismatch"
      requireCheck (observed.projection.emitted.all fun emission => !emission.support.isEmpty)
        "emission lacks evidence support"
      comparisons := comparisons + 1
  let overCapacity := (List.range 129).map fun id =>
    ({ run := 7, id, operation := 0, kind := .unrelated } : Evidence)
  let bounded := evaluateEvidence resolves 7 overCapacity
  requireCheck (bounded.projection.error == some .capacity && bounded.finish == .inconclusive)
    "evidence work limit established satisfaction"
  return comparisons

private def printQueries : IO Unit := do
  let cases : List (String × Scenario × Question × Nat) := [
    ("both-outcomes", .resolved 0, .verify, 100),
    ("witness", .resolved 0, .witness, 100),
    ("impossible", .exact [⟨0, .canceled⟩], .verify, 100),
    ("absent-trigger", .exact [], .verify, 100),
    ("closed-pending", .exact [⟨0, .requested⟩], .verify, 100),
    ("work-limit", .resolved 0, .verify, 0)
  ]
  for (name, scenario, question, budget) in cases do
    let report := query resolves scenario question .closedFinitePrefixes 2 budget
    IO.println s!"{name}: {repr report}"
  IO.println s!"runtime-pending: {repr (query resolves (.exact [⟨0, .requested⟩])
    .verify .runtimePrefixes 2 100)}"
  IO.println s!"terminal-worlds: {repr (query resolves .any .verify .terminalWorlds 4 1000)}"

def main (args : List String) : IO Unit := do
  let start ← IO.monoNanosNow
  let propertyComparisons ← checkProperties
  let evidenceComparisons ← checkEvidence
  IO.println s!"property_reference_comparisons={propertyComparisons}"
  IO.println s!"evidence_variants={evidenceComparisons}"
  IO.println s!"finite_prefixes_depth_5={(explore 5 1000).traces.length}"
  printQueries
  if args.contains "--timing" then IO.println s!"checks_ns={(← IO.monoNanosNow) - start}"
