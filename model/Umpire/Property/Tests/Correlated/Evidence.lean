import Umpire.Property.Tests.Correlated.Fixtures

/-! Partial, repeated and reordered evidence use the same checked correlated transition kernel. -/

namespace Umpire.Property.CorrelatedTests

open Property.Correlated
namespace Evidence

private def plan (target : TestTarget) := Case.Projection.check target {
  id := id "test.projection"
  scopeFields := [id "test.run"]
  operationField := id "test.operation"
  sources := [id "test.source"]
  rules := [
    { kind := id "test.submit", meaning := .submission request },
    { kind := id "test.request", meaning := .confirmed none [(request, result false)] },
    { kind := id "test.both", meaning := .confirmed none [(both, result true)] },
    { kind := id "test.reply", meaning := .confirmed none [(reply, result true)] },
    { kind := id "test.tick", meaning := .confirmed none [(tick, result false)] },
    { kind := id "test.poll", meaning := .irrelevant }]
  limits := {
    events := 1000
    buffered := 1000
    keys := 1000
    support := 10000
    work := 1000000000000
    eventSize := 10000 }
} () state

private def event (ordinal : Nat) (kind : String) (parents : List Nat := [])
    (operation : String := "a") : Case.Projection.Event := {
  identity := { scope, source := id "test.source", ordinal }
  operation
  kind := id ("test." ++ kind)
  runSequences := [ordinal + 1]
  parents := parents.map fun ordinal => { scope, source := id "test.source", ordinal }
}

private def evaluateEvidence (bound : Nat) (events : List Case.Projection.Event)
    (ending : TraceEnding := .«partial») : Option (List PropertyEndpointAnswer) := do
  let target ← targetResult.toOption
  let property ← (property target bound ending).toOption
  let plan ← (plan target).toOption
  let compiled ← (Case.Projection.Correlated.compile plan property limits).toOption
  let initial ← (Case.Projection.Correlated.start plan compiled () scope).toOption
  let run ← (initial.admitMany events).toOption
  pure (run.close.answers.map Prod.snd)

#guard evaluateEvidence 1 [event 1 "request" [0]] == some [.unresolved]
#guard evaluateEvidence 1 [event 1 "request" [0]] .final == some [.unresolved]
#guard evaluateEvidence 1 [event 0 "both", event 2 "request" [1]] == some [.unresolved]
#guard evaluateEvidence 0 [event 0 "request", event 2 "reply" [1]] == some [.violated]
#guard evaluateEvidence 1 [event 0 "request", event 0 "request", event 1 "reply" [0],
  event 1 "reply" [0]] == some [.satisfied]
#guard evaluateEvidence 0 [event 0 "request", event 0 "request", event 1 "reply" [0],
  event 1 "reply" [0]] == some [.violated]
#guard evaluateEvidence 1 [event 0 "request", event 1 "poll", event 2 "submit",
  event 3 "reply" [0]] == some [.satisfied]
#guard evaluateEvidence 1 [event 0 "request", event 1 "tick" [] "b",
  event 2 "tick" [] "b", event 3 "reply" [0]] == some [.satisfied]

private def partialEvents : List Case.Projection.Event := [
  event 2 "reply" [1], event 1 "request" [0], event 0 "poll", event 2 "reply" [1]]

#guard (do
  let target ← targetResult.toOption
  let property ← (property target 1).toOption
  let plan ← (plan target).toOption
  let compiled ← (Case.Projection.Correlated.compile plan property limits).toOption
  let initial ← (Case.Projection.Correlated.start plan compiled () scope).toOption
  let whole ← (initial.admitMany partialEvents).toOption
  pure ((List.range (partialEvents.length + 1)).all fun split =>
    ((initial.admitMany (partialEvents.take split) >>= fun before =>
      before.admitMany (partialEvents.drop split)).toOption.map (fun run => run.close.answers)) ==
      some whole.close.answers)) == some true

#guard (do
  let target ← targetResult.toOption
  let property ← (property target 0).toOption
  let plan ← (plan target).toOption
  let compiled ← (Case.Projection.Correlated.compile plan property limits).toOption
  let initial ← (Case.Projection.Correlated.start plan compiled () scope).toOption
  let violated ← (initial.admit (event 0 "request")).toOption
  pure ((violated.admit (event 0 "both")).isOk,
    violated.answers.map Prod.snd, (violated.close.admit (event 1 "reply")).isOk)) ==
  some (false, [.violated], false)

end Evidence
end Umpire.Property.CorrelatedTests
