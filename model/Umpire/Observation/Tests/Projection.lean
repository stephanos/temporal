import Umpire.Observation.Projection
import Umpire.Target.Tests.Validation

/-! Transactional evidence projection, independent of product-specific history. -/

namespace Umpire.Observation.ProjectionTests

open Projection

private def target := TargetTests.checkedTestTarget
private def id := DefinitionId.of
private def scope : List (DefinitionId × String) := [(id "test.run", "run-1")]
private def declaration : Declaration Bool Bool Bool Bool := {
  id := id "test.projection"
  scopeFields := [id "test.run"]
  operationField := id "test.operation"
  sources := [id "test.source", id "test.other"]
  rules := [
    { kind := id "test.submit", meaning := .submission true },
    { kind := id "test.confirm", meaning := .confirmed (some true)
        [(true, TargetTests.transition false true)] },
    { kind := id "test.finish", meaning := .confirmed none
        [(false, TargetTests.transition true false)] },
    { kind := id "test.irrelevant", meaning := .irrelevant }]
  limits := {
    events := 100
    buffered := 100
    keys := 100
    support := 10000
    work := 1000000000000
    eventSize := 10000 }
}

private def plan := check target declaration () false
private def event (ordinal : Nat) (kind : String) (parents : List Nat := []) : Event := {
  identity := { scope, source := id "test.source", ordinal }
  operation := "operation-1"
  kind := id kind
  runSequences := [100 + ordinal]
  parents := parents.map fun ordinal => { scope, source := id "test.source", ordinal }
}

#guard plan.isOk
#guard (do
  let checked ← plan
  let run ← checked.start scope
  let (run, progress) ← run.admit (event 0 "test.submit")
  pure (progress.isStutter && run.steps.isEmpty && run.accepted.length == 1)).toOption == some true

private def error? (result : Except Error α) : Option Error :=
  match result with
  | .error error => some error
  | .ok _ => none

private def feed {target : CheckedTarget TargetTests.TestLawStatement Unit Bool Bool Bool Bool}
    {checked : Checked target} (run : Run checked) (events : List Event) :
    Except Error (Run checked) := events.foldlM (fun run event => do
      let (next, _) ← run.admit event
      pure next) run

private def confirmed : List Event := [event 0 "test.submit", event 1 "test.confirm" [0]]

#guard (do
  let checked ← plan
  let initial ← checked.start scope
  let run ← feed initial confirmed
  let (duplicate, progress) ← run.admit (event 1 "test.confirm" [0])
  let (irrelevant, ignored) ← duplicate.admit (event 2 "test.irrelevant")
  pure (progress.isStutter && ignored.isStutter && duplicate.work > run.work &&
    irrelevant.steps.length == 1 && irrelevant.accepted.length == 3 &&
    (irrelevant.steps.head?.map Projection.Step.support).getD [] == [(event 0 "").identity, (event 1 "").identity])).toOption == some true

#guard (do
  let checked ← plan
  let initial ← checked.start scope
  let (buffered, progress) ← initial.admit (event 2 "test.finish" [1])
  let (buffered, _) ← buffered.admit (event 1 "test.confirm" [0])
  let (released, emissions) ← buffered.admit (event 0 "test.submit")
  let pending := match progress with | .pending _ => true | _ => false
  pure (pending && emissions.emissions.length == 2 && released.pending.isEmpty &&
      released.steps.map (fun (step : Projection.Step target) => step.result.state) == [true, false] &&
      released.steps.map (fun (step : Projection.Step target) => step.directSupport.map Identity.ordinal) == [[1, 0], [2, 1]] &&
      released.steps.map (fun (step : Projection.Step target) => step.support.map Identity.ordinal) == [[0, 1], [0, 1, 2]] &&
      released.steps.map (fun (step : Projection.Step target) => step.directRunSequences) ==
        [[100, 101], [101, 102]] &&
      released.steps.map (fun (step : Projection.Step target) => step.runSequences) ==
        [[100, 101], [100, 101, 102]] &&
      released.steps.all (fun (step : Projection.Step target) => step.scope == scope && step.operation == "operation-1"))).toOption == some true

#guard (do
  let checked ← plan
  let initial ← checked.start scope
  let run ← feed initial (confirmed ++ [event 2 "test.finish" [1]])
  let (buffered, _) ← run.admit (event 4 "test.confirm" [0, 2])
  let (buffered, _) ← buffered.admit (event 5 "test.confirm" [0, 4])
  let failed := buffered.admit (event 3 "test.irrelevant")
  pure (error? failed == some (Error.invalidTransition (event 5 "").identity) &&
    buffered.accepted.length == 5 && buffered.pending.map Identity.ordinal == [4, 5] &&
    buffered.steps.map (fun (step : Projection.Step target) => step.result.state) == [true, false] &&
    buffered.state "operation-1" == false)).toOption == some true

#guard (do
  let checked ← plan
  let run ← checked.start scope
  let (run, _) ← run.admit (event 0 "test.irrelevant" [1])
  pure (error? (run.admit (event 1 "test.irrelevant" [0])) == some Error.causalCycle &&
    error? run.close == some (Error.incomplete [(event 0 "").identity]))).toOption == some true

#guard (do
  let checked ← plan
  let initial ← checked.start scope
  let run ← feed initial confirmed
  pure ([
    error? (run.admit { event 1 "test.confirm" [0] with operation := "other" }),
    error? (run.admit { event 2 "test.finish" [1] with operation := "other" }),
    error? (run.admit (event 2 "test.unknown")),
    error? (run.admit (event 2 "test.confirm" [0])),
    error? (run.admit (event 2 "test.confirm")),
    error? (run.admit { event 2 "test.irrelevant" with identity :=
      { scope := [(id "test.run", "wrong")], source := id "test.source", ordinal := 2 } })] == [
    some (Error.identityConflict (event 1 "").identity),
    some (Error.wrongOperation (event 2 "").identity (event 1 "").identity),
    some (Error.unsupportedEvidence (id "test.unknown")),
    some (Error.invalidTransition (event 2 "").identity),
    some (Error.missingSubmission (event 2 "").identity),
    some Error.wrongScope])).toOption == some true

private def bounded (limits : Limits) (events : List Event) : Option Error := error? do
  let checked ← check target { declaration with limits } () false
  let run ← checked.start scope
  let _ ← feed run events
  pure ()

#guard [
  bounded { declaration.limits with events := 0 } [event 0 "test.submit"],
  bounded { declaration.limits with buffered := 0 } [event 1 "test.confirm" [0]],
  bounded { declaration.limits with keys := 0 } [event 0 "test.submit"],
  bounded { declaration.limits with support := 0 } [event 0 "test.submit"],
  bounded { declaration.limits with work := 0 } [event 0 "test.submit"],
  bounded { declaration.limits with eventSize := 0 } [event 0 "test.submit"]] == [
  some Error.eventsExhausted, some Error.bufferExhausted, some Error.keysExhausted,
  some Error.supportExhausted, some Error.workExhausted, some Error.eventSizeExhausted]

private def fieldsDeclaration : Declaration Bool Bool Bool Bool := {
  declaration with rules := [{ kind := id "test.fields", meaning := .irrelevant, fields := [
    (⟨id "test.retained", .natural⟩, .retain),
    (⟨id "test.redacted", .text⟩, .redact),
    (⟨id "test.rejected", .text⟩, .reject)] }] }

private def fieldsEvent : Event := { event 0 "test.fields" with fields := [
  ⟨id "test.retained", some (.natural 42)⟩, ⟨id "test.redacted", none⟩] }

#guard (do
  let checked ← check target fieldsDeclaration () false
  let run ← checked.start scope
  let (accepted, _) ← run.admit fieldsEvent
  pure (accepted.accepted == [fieldsEvent] && [
    error? (run.admit { fieldsEvent with fields := [⟨id "test.unknown", none⟩] }),
    error? (run.admit { fieldsEvent with fields := [⟨id "test.retained", some (.text "wrong")⟩] }),
    error? (run.admit { fieldsEvent with fields := fieldsEvent.fields ++
      [({ id := id "test.rejected", value := some (.text "raw") } : Field)] }),
    error? (run.admit { fieldsEvent with fields := [⟨id "test.redacted", some (.text "raw")⟩] })] == [
    some (Error.unauthorizedField (id "test.unknown")),
    some (Error.unauthorizedField (id "test.retained")),
    some (Error.unauthorizedField (id "test.rejected")),
    some (Error.unauthorizedField (id "test.redacted"))])).toOption == some true

private def loadEvents (count : Nat) : List Event := (List.range count).map fun ordinal =>
  { event ordinal "test.submit" with operation := "operation-" ++ toString ordinal }

#guard (do
  let checked ← plan
  let first ← checked.start scope
  let second ← checked.start [(id "test.run", "run-2")]
  let small ← feed first (loadEvents 10)
  let large ← feed first (loadEvents 100)
  pure (small.accepted.length == 10 && large.accepted.length == 100 &&
    first.accepted.isEmpty && second.accepted.isEmpty && first.work == 0 && second.work == 0 &&
    error? (second.admit (event 0 "test.submit")) == some Error.wrongScope &&
    bounded { declaration.limits with keys := 10 } (loadEvents 100) == some Error.keysExhausted &&
    bounded { declaration.limits with work := small.work } (loadEvents 100) == some Error.workExhausted)).toOption == some true

#guard error? (check target { declaration with version := 2 } () false) == some Error.unsupportedVersion
#guard error? (check target declaration () true) == some Error.invalidInitialState
#guard (do
  let checked ← plan
  let reordered ← check target { declaration with
    rules := declaration.rules.reverse
    sources := declaration.sources.reverse } () false
  let changed ← check target { declaration with limits := { declaration.limits with work := 1 } }
    () false
  pure (checked.behaviorFingerprint == reordered.behaviorFingerprint &&
    checked.behaviorFingerprint != changed.behaviorFingerprint)).toOption == some true

#guard (do
  let checked ← plan
  let initial ← checked.start scope
  let run ← feed initial confirmed
  let crossSource := { event 0 "test.finish" with identity :=
    { scope, source := id "test.other", ordinal := 0 } }
  pure (!(run.admit crossSource).isOk)).toOption == some true

#guard (do
  let checked ← plan
  let initial ← checked.start scope
  let (first, _) ← initial.admit (event 0 "test.submit")
  let (duplicate, _) ← first.admit (event 0 "test.submit")
  pure (decide (duplicate.work > first.work))).toOption == some true

private def terminalTarget := checkedTarget (AuthoredTarget.make
  { TargetTests.targetDefinitionOf TargetTests.testTarget with terminalConditions := [[true]] }
  (TargetTests.targetCompositionOf TargetTests.testTarget))

#guard (do
  let checked ← check terminalTarget declaration () false
  let initial ← checked.start scope
  let run ← feed initial confirmed
  let closed ← run.close
  let twice ← closed.close
  pure (error? initial.close == some (Error.nonterminal []) && closed.isClosed && twice.isClosed &&
    error? (closed.admit (event 2 "test.irrelevant")) == some Error.closed &&
    closed.steps.length == run.steps.length && closed.work > run.work)).toOption == some true

#guard (do
  let checked ← plan
  let initial ← checked.start scope
  let run ← feed initial confirmed
  let other := { event 0 "test.submit" with identity :=
    { scope, source := id "test.other", ordinal := 0 }, runSequences := [200] }
  let (separate, _) ← initial.admit other
  let (separate, _) ← separate.admit (event 0 "test.submit")
  pure (separate.accepted.length == 2 &&
    error? run.close == some (Error.nonterminal ["operation-1"]) &&
    error? (initial.admit { event 0 "test.submit" with runSequences := [] }) ==
      some Error.invalidEvidenceSupport &&
    error? (run.admit { event 1 "test.confirm" [0] with runSequences := [999] }) ==
      some (Error.identityConflict (event 1 "").identity))).toOption == some true

#guard (do
  let checked ← plan
  let initial ← checked.start scope
  let child := { event 0 "test.confirm" [0] with identity :=
    { scope, source := id "test.other", ordinal := 0 }, runSequences := [200] }
  let (buffered, _) ← initial.admit child
  let (released, emitted) ← buffered.admit (event 0 "test.submit")
  pure (emitted.emissions.length == 1 && released.pending.isEmpty &&
    released.steps.map (fun (step : Projection.Step target) => step.runSequences) == [[100, 200]])).toOption == some true

#guard (do
  let checked ← check target { declaration with rules := declaration.rules ++ [({
    kind := id "test.bundle"
    meaning := .confirmed (some true)
      [(true, TargetTests.transition false true), (false, TargetTests.transition true false)] } :
      Rule Bool Bool Bool Bool)] }
    () false
  let initial ← checked.start scope
  let (submitted, _) ← initial.admit (event 0 "test.submit")
  let (released, emissions) ← submitted.admit (event 1 "test.bundle" [0])
  pure (emissions.emissions.length == 2 && released.state "operation-1" == false &&
    released.steps.map (fun (step : Projection.Step target) => step.runSequences) == [[100, 101], [100, 101]])).toOption == some true

#guard (do
  let checked ← check target { declaration with limits := { declaration.limits with support := 1 } }
    () false
  let initial ← checked.start scope
  let (buffered, _) ← initial.admit (event 1 "test.confirm" [0])
  pure (error? (buffered.admit (event 0 "test.submit")) == some Error.supportExhausted &&
    buffered.steps.isEmpty && buffered.pending == [(event 1 "").identity])).toOption == some true

#guard (do
  let checked ← plan
  let initial ← checked.start scope
  let (run, _) ← initial.admit (event 0 "test.submit")
  let limited ← check target { declaration with limits := { declaration.limits with work := run.work } }
    () false
  let fresh ← limited.start scope
  let (last, _) ← fresh.admit (event 0 "test.submit")
  pure (last.work == run.work &&
    error? (last.admit (event 0 "test.submit")) == some Error.workExhausted)).toOption == some true

#guard (do
  let checked ← plan
  let initial ← checked.start scope
  pure ([
    error? (initial.admit { event 0 "test.submit" with identity :=
      { scope, source := id "test.unknown", ordinal := 0 } }),
    error? (initial.admit { event 0 "test.submit" with runSequences := [0] }),
    error? (initial.admit { event 0 "test.submit" with runSequences := [1, 1] })] == [
    some Error.unknownSource, some Error.invalidEvidenceSupport, some Error.invalidEvidenceSupport])).toOption == some true

#guard error? (check target { declaration with sources := [] } () false) == some Error.invalidDeclaration
#guard error? (check target { declaration with rules := declaration.rules ++ declaration.rules }
  () false) == some Error.invalidDeclaration
#guard error? (check target { fieldsDeclaration with rules := [{
  kind := id "test.fields"
  fields := [(⟨id "test.retained", .natural⟩, .hash (some (id "test.digest")))]
  meaning := .irrelevant }] } () false) == some Error.unsupportedDisposition

#guard ([
  { declaration with id := id "malformed" },
  { declaration with scopeFields := [id "malformed"] },
  { declaration with operationField := id "malformed" },
  { declaration with sources := [id "malformed"] },
  { declaration with rules := [{ kind := id "malformed", meaning := .irrelevant }] },
  { declaration with rules := [{
    kind := id "test.fields"
    meaning := .irrelevant
    fields := [(⟨id "malformed", .text⟩, .retain)] }] }] : List (Declaration Bool Bool Bool Bool)).all (fun input => error? (check target input () false) == some Error.invalidDeclaration)

private def orderingBridge : Event := { event 0 "test.irrelevant" [1] with
  identity := { scope, source := id "test.other", ordinal := 0 }
  runSequences := [200] }

private def orderingFinish : Event := { event 1 "test.finish" with
  identity := { scope, source := id "test.other", ordinal := 1 }
  runSequences := [201] }

#guard (do
  let checked ← plan
  let initial ← checked.start scope
  let direct ← feed initial (confirmed ++ [orderingBridge, orderingFinish])
  let buffered ← feed initial [orderingFinish, orderingBridge, event 1 "test.confirm" [0]]
  let (released, progress) ← buffered.admit (event 0 "test.submit")
  pure (progress.emissions.length == 2 && [direct, released].all (fun run =>
    run.pending.isEmpty && run.state "operation-1" == false &&
    run.steps.map (fun (step : Projection.Step target) => step.result.state) == [true, false] &&
    run.steps.map (fun (step : Projection.Step target) => step.directSupport) ==
      [[(event 1 "").identity, (event 0 "").identity], [orderingFinish.identity]] &&
    run.steps.map (fun (step : Projection.Step target) => step.support) ==
      [[(event 0 "").identity, (event 1 "").identity], [orderingFinish.identity]] &&
    run.steps.map (fun (step : Projection.Step target) => step.runSequences) == [[100, 101], [201]]))).toOption == some true

end Umpire.Observation.ProjectionTests
