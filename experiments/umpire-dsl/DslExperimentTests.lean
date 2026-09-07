import DslExperiment.Model
import DslExperiment.Property
import DslExperiment.Projection
import DslExperiment.Query

/-! Checked examples catch request-as-resolution, lost nondeterminism, and terminal restart. -/

namespace DslExperiment.Tests

example : advance .started .requested = some .requested := by decide
example : advance .requested .canceled = some .canceled := by decide
example : advance .requested .completed = some .succeeded := by decide
example : advance .requested .tick = some .requested := by decide
example : advance .started .canceled = none := by decide
example : advance .succeeded .requested = none := by decide
example : advance .canceled .tick = none := by decide

private def resolves : ResponseClause :=
  ⟨.event .requested, .anyOf [.canceled, .completed], 1, .operationTransitions⟩

example : evaluate resolves .runtimePrefix [⟨0, .requested⟩] = .inconclusive := by decide
example : evaluate resolves .closedModel [⟨0, .requested⟩] = .violated := by decide
example : evaluate resolves .closedModel [⟨0, .requested⟩, ⟨0, .canceled⟩] = .satisfied := by decide
example : evaluate resolves .closedModel [⟨0, .requested⟩, ⟨0, .completed⟩] = .satisfied := by decide
example : evaluate resolves .runtimePrefix [⟨0, .requested⟩, ⟨0, .tick⟩] = .violated := by decide
example : evaluate resolves .runtimePrefix [⟨0, .requested⟩, ⟨1, .tick⟩] = .inconclusive := by decide
example : evaluate resolves .runtimePrefix
    [⟨0, .requested⟩, ⟨1, .requested⟩, ⟨1, .completed⟩] = .inconclusive := by decide
example : evaluate resolves .closedModel
    [⟨0, .requested⟩, ⟨1, .requested⟩, ⟨1, .completed⟩, ⟨0, .canceled⟩] = .satisfied := by decide

private def submit : Evidence := ⟨7, 10, 0, .submitCancel, none⟩
private def request : Evidence := ⟨7, 11, 0, .observed .requested, some 10⟩
private def complete : Evidence := ⟨7, 12, 0, .observed .completed, some 11⟩

example : (evaluateEvidence resolves 7 [submit]).projection.world = initial := by decide
example : (evaluateEvidence resolves 7 [submit, request]).finish = .inconclusive := by decide
example : (evaluateEvidence resolves 7 [submit, request, complete]).finish = .satisfied := by decide
example : (evaluateEvidence resolves 7 [submit, request, request, complete]).projection.emitted =
    [⟨⟨0, .requested⟩, [10, 11]⟩, ⟨⟨0, .completed⟩, [10, 11, 12]⟩] := by decide
example : (evaluateEvidence resolves 7 [request, complete, submit]).projection.emitted =
    [⟨⟨0, .requested⟩, [10, 11]⟩, ⟨⟨0, .completed⟩, [10, 11, 12]⟩] := by decide
example : (evaluateEvidence resolves 7 [request]).finish = .inconclusive := by decide
example : (evaluateEvidence resolves 7
    [submit, request, { request with kind := .observed .canceled }]).projection.error =
    some .conflictingIdentity := by decide

example : (query resolves (.exact [⟨0, .canceled⟩]) .verify .closedFinitePrefixes 2 100).answer =
    .unsatisfiable := by decide
example : (query resolves (.exact []) .verify .closedFinitePrefixes 2 100).answer = .unexercised := by decide
example : (query resolves (.resolved 0) .verify .closedFinitePrefixes 2 100).answer = .verified := by decide
example : (query resolves (.resolved 0) .witness .closedFinitePrefixes 2 100).answer = .witness := by decide
example : (query resolves .any .verify .closedFinitePrefixes 2 100).answer = .counterexample := by decide
example : (query resolves (.resolved 0) .verify .closedFinitePrefixes 2 0).answer = .unknown := by decide
example : (query resolves (.resolved 0) .verify .closedFinitePrefixes 2 0).satisfiable = none := by decide
example : (query resolves (.exact [⟨0, .requested⟩]) .verify .runtimePrefixes 2 100).answer =
    .unknown := by decide
set_option maxRecDepth 4096 in
example : (query resolves .any .verify .terminalWorlds 4 1000).answer = .verified := by decide
example : (explore 2 11).complete = true := by decide
example : (explore 2 10).complete = false := by decide

example : (Scenario.ordered ⟨0, .requested⟩ ⟨0, .completed⟩).accepts
    [⟨0, .requested⟩, ⟨1, .tick⟩, ⟨0, .completed⟩] = true := by decide
example : (Scenario.either (.exact [⟨0, .canceled⟩]) (.exact [⟨0, .completed⟩])).accepts
    [⟨0, .completed⟩] = true := by decide

example : (evaluateEvidence resolves 7
    [⟨7, 20, 0, .observed .requested, some 20⟩]).projection.error =
    some .causalConflict := by decide
example : (evaluateEvidence resolves 7
    [⟨7, 20, 0, .observed .requested, some 21⟩,
      ⟨7, 21, 0, .observed .tick, some 20⟩]).projection.error =
    some .causalConflict := by decide

private def invalidChild : Evidence := ⟨7, 13, 0, .observed .tick, some 12⟩

example : (evaluateEvidence resolves 7 [request, complete, invalidChild, submit]).projection.emitted =
    [] := by decide
example : (evaluateEvidence resolves 7 [request, complete, invalidChild, submit]).finish =
    .inconclusive := by decide

example : evaluateAdmitted resolves .closedModel [⟨0, .completed⟩] = none := by decide
example : evaluateAdmitted resolves .closedModel [⟨0, .tick⟩] = none := by decide
example : evaluateAdmitted resolves .closedModel [⟨0, .requested⟩, ⟨0, .completed⟩] =
    some .satisfied := by decide

private def countedTick : Evidence := ⟨7, 12, 0, .observed .tick, some 11⟩

example : (evaluateEvidence resolves 7 [submit, request, countedTick,
    ⟨7, 13, 0, .gap, none⟩]).finish = .violated := by decide
example : (evaluateEvidence resolves 7 [submit, request, countedTick,
    { countedTick with kind := .observed .completed }]).projection.error =
    some .conflictingIdentity := by decide
example : (evaluateEvidence resolves 7 [submit, request, countedTick,
    { countedTick with kind := .observed .completed }]).finish = .violated := by decide

example : evaluate resolves .closedModel
    [⟨0, .requested⟩, ⟨0, .tick⟩, ⟨0, .completed⟩] = .violated := by decide
example : evaluate { resolves with bound := 0, response := .event .requested }
    .closedModel [⟨0, .requested⟩] = .satisfied := by decide
example : evaluate { resolves with bound := 2 } .closedModel
    [⟨0, .requested⟩, ⟨0, .requested⟩, ⟨0, .completed⟩] = .satisfied := by decide

example : (evaluateEvidence resolves 7
    [submit, request, { complete with operation := 1 }]).finish = .inconclusive := by decide
example : (evaluateEvidence resolves 7
    [submit, request, ⟨7, 40, 1, .unrelated, none⟩]).monitor.triggers = 1 := by decide

/--
error: Application type mismatch: The argument
  EvidenceKind.submitCancel
has type
  EvidenceKind
but is expected to have type
  StepPredicate
in the application
  ResponseClause.mk EvidenceKind.submitCancel
-/
#guard_msgs (error) in
example : ResponseClause :=
  whenever EvidenceKind.submitCancel eventually (.event .completed) within 1 operationTransitions

/--
error: Type mismatch
  100
has type
  Nat
but is expected to have type
  Clock
-/
#guard_msgs (error) in
example : ResponseClause := { resolves with clock := (100 : Nat) }

end DslExperiment.Tests
