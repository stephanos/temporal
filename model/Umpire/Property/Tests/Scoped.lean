import Umpire.Property.Tests.Scoped.Fixtures
import Umpire.Property.Elab
import Umpire.Property.Scoped.Reference

/-! Scoped admission, independent deadlines, per-operation ticks, and resource failure boundaries. -/

namespace Umpire.Property.ScopedTests

open Property.Scoped

private def answer (bound : Nat) (steps : List Transition)
    (endpoint : PropertyScopedEndpoint := .«partial») : Option PropertyEndpointAnswer :=
  (evaluate bound endpoint steps).toOption.bind fun answers => answers.head?.map Prod.snd

#guard ([0, 1, 3] : List Nat).all fun bound =>
  answer bound [step "a" both] == some .satisfied
#guard ([1, 3, 10] : List Nat).all fun bound =>
  answer bound ([step "a" request] ++ List.replicate (bound - 1) (step "a" tick) ++
    [step "a" reply]) == some .satisfied
#guard ([0, 1, 3] : List Nat).all fun bound =>
  answer bound ([step "a" request] ++ List.replicate bound (step "a" tick)) == some .violated &&
  answer bound ([step "a" request] ++ List.replicate bound (step "a" tick) ++
    [step "a" reply]) == some .violated
#guard answer 1 [step "a" request] == some .unresolved
#guard answer 1 [step "a" request] .final == some .violated
#guard answer 2 [step "a" request, step "a" request, step "a" reply] == some .satisfied
#guard answer 1 [step "a" request, step "b" tick, step "b" tick, step "b" tick,
  step "a" reply] == some .satisfied
#guard answer 1 [step "a" request, step "b" reply] == some .unresolved

#guard (do
  let target ← targetResult.toOption
  let property ← (property target 1 .final).toOption
  let compiled ← (compile target property [id "test.run"] (id "test.operation") limits).toOption
  let initial ← (compiled.start () state scope).toOption
  let run ← (initial.consume (step "a" request)).toOption
  let resolved ← (run.consume (step "a" reply)).toOption
  pure (run.answers.map Prod.snd, run.close.answers.map Prod.snd,
    resolved.close.answers.map Prod.snd, initial.answers true |>.map Prod.snd)) ==
  some ([.unresolved], [.violated], [.satisfied], [.unresolved])

private def error? (value : Except Error α) : Option Error :=
  match value with | .ok _ => none | .error error => some error

#guard error? (evaluate 1 .«partial» [{ step "a" request with priorState := quiet }]) ==
  some (.invalidTransition "a")
#guard error? (evaluate 1 .«partial» [{ step "a" request with result := result true }]) ==
  some (.invalidTransition "a")
#guard error? (evaluate 1 .«partial» [{ step "a" request with scope := [(id "test.run", "other")] }]) ==
  some .wrongScope
#guard error? (evaluate 1 .«partial» [{ step "a" request with operationField := id "test.other" }]) ==
  some .wrongScope
#guard error? (evaluate 1 .«partial» [step "a" request] { limits with transitions := 0 }) ==
  some .transitionsExhausted
#guard error? (evaluate 1 .«partial» [step "a" request] { limits with obligations := 0 }) ==
  some .obligationsExhausted
#guard error? (evaluate 1 .«partial» [step "a" request] { limits with work := 1 }) ==
  some .workExhausted
#guard (evaluate 1 .«partial» [step "a" request] { limits with work := 49 }).isOk
#guard ([10, 100] : List Nat).all fun count =>
  (evaluate count .«partial» (List.replicate count (step "a" request))
    { limits with obligations := count }).isOk &&
  error? (evaluate (count + 1) .«partial» (List.replicate (count + 1) (step "a" request))
    { limits with obligations := count }) == some .obligationsExhausted

private def admissionError (changed : PropertyScopedClause) : Option PropertyError := do
  let target ← targetResult.toOption
  match Property.check (context target) ({ declaration 1 with scopedClauses := [changed] }) with
  | .ok _ => none
  | .error error => some error

#guard ([
  { clause 1 with trigger := .any [] },
  { clause 1 with trigger := .atom { field := .priorState, reference := id "test.state" } },
  { clause 1 with response := .atom { field := .selectedAction, reference := id "test.trigger" } },
  { clause 1 with response := .atom { field := .outcome, reference := id "test.missing" } },
  { clause 1 with key := id "test.run" },
  { clause 1 with scope := [] },
  { clause 1 with bound := 18446744073709551616 }
] : List PropertyScopedClause).all fun changed =>
  (admissionError changed).any fun error => error.definitionId == changed.id &&
    error.sourcePath == source.path

#guard (do
  let target ← targetResult.toOption
  let first := clause 1
  let second := { clause 2 with id := id "test.scoped.second" }
  let a ← (Property.check (context target)
    ({ declaration 1 with scopedClauses := [first, second] })).toOption
  let b ← (Property.check (context target)
    ({ declaration 1 with scopedClauses := [second, first] })).toOption
  pure (a.behaviorFingerprint == b.behaviorFingerprint && a.canonicalMetadata == b.canonicalMetadata)) == some true

#guard (do
  let target ← targetResult.toOption
  let property ← (property target 1).toOption
  pure (error? (compile target property [id "test.other"] (id "test.operation") limits),
    (checkPropertyEvaluationInput property { initialState := state, steps := [] }).isOk)) ==
  some (some (.unsupported (id "test.scoped.response") "producer scope/key mismatch"), false)

private def streams : Nat → List (List Coordinate)
  | 0 => [[]]
  | count + 1 => (streams count).flatMap fun tail =>
      [⟨false, false⟩, ⟨false, true⟩, ⟨true, false⟩, ⟨true, true⟩].map (· :: tail)

private def independent (bound : Nat) (points : List Coordinate) : Bool :=
  (points.zipIdx).all fun (point, first) =>
    !point.trigger || (points.zipIdx).any fun (candidate, second) =>
      candidate.response && first ≤ second && second ≤ first + bound

#guard (List.range 5).all fun bound => (streams 5).all fun points =>
  ((consumeMany bound [] points).all fun obligation => decide (obligation = .satisfied)) ==
    independent bound points

#guard (do
  let target ← targetResult.toOption
  pure ([PropertyPredicate.resultingStateIs state, PropertyPredicate.factIs fact].all fun responsePredicate =>
    let result := do
      let property ← (Property.check (context target) ({
        declaration 0 with scopedClauses := [{ clause 0 with response := responsePredicate }] })).toOption
      let compiled ← (compile target property [id "test.run"] (id "test.operation") limits).toOption
      let initial ← (compiled.start () state scope).toOption
      let run ← (initial.consume (step "a" both)).toOption
      pure (run.answers.map Prod.snd)
    result == some [.satisfied])) == some true

/-- error: Unknown constant `Umpire.PropertyScopedEndpoint.terminalModel` -/
#guard_msgs in
#check PropertyScopedEndpoint.terminalModel

/-- info: 'Umpire.Property.Scoped.Execution.closed_property' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Property.Scoped.Execution.closed_property

/-- info: 'Umpire.Property.Scoped.checked_eventuallyWithin_agrees' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Property.Scoped.checked_eventuallyWithin_agrees
/-- info: 'Umpire.Property.Scoped.Run.consume' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Property.Scoped.Run.consume
/-- info: 'Umpire.Property.Scoped.Run.consumeMany_append' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Property.Scoped.Run.consumeMany_append
/-- info: 'Umpire.Case.Projection.Scoped.Run.admitMany_append' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Case.Projection.Scoped.Run.admitMany_append

end Umpire.Property.ScopedTests
