import Umpire.Operation.Action
import Umpire.Target.FiniteMachine

/-!
Parameterized Actions use the existing finite-table and checked Target owners. The authored domain
supplies the entire action catalog; transition rows still supply every state-dependent alternative.
The domain's versioned scope is retained as semantic metadata, including representative-only claims.
-/
namespace Umpire.Operation

variable {owner : RpcOwner} {Request Response Failure Setup State Outcome Fact : Type}
  {template : ActionTemplate owner Request Response Failure} {limits : Value.Limits}
  {domain : ParameterDomain template limits}

/-- Exact authored parameter enumeration, using checked instance keys rather than display labels. -/
def ParameterDomain.catalog (domain : ParameterDomain template limits) :
    FiniteCatalog (ActionInstance template limits) :=
  domain.actions.map fun action => ⟨action, action.canonical⟩

/-- Catalog membership is exactly authored instance membership, in the same order. -/
theorem ParameterDomain.catalog_exact (domain : ParameterDomain template limits) :
    domain.catalog.values = domain.actions := by
  simp [catalog, FiniteCatalog.values, Function.comp_def]

/-- A validated finite table whose action domain is exactly the separately authored parameters. -/
structure ParameterTable (domain : ParameterDomain template limits)
    (Setup State Outcome Fact : Type) where
  private mk ::
  validated : ValidatedFiniteTable Setup State (ActionInstance template limits) Outcome Fact
  exactActions : validated.table.actions.values = domain.actions

/-- Check exact domain agreement before using the established table closure/executability checker. -/
def ParameterDomain.validateTable [DecidableEq Setup] [DecidableEq State]
    [DecidableEq Outcome] [DecidableEq Fact]
    (domain : ParameterDomain template limits)
    (table : FiniteTable Setup State (ActionInstance template limits) Outcome Fact) :
    Except FiniteTableError (ParameterTable domain Setup State Outcome Fact) := do
  if table.actions = domain.catalog then
    let validated ← table.validate
    if exactActions : validated.table.actions.values = domain.actions then
      pure ⟨validated, exactActions⟩
    else throw (.outOfDomain .action)
  else throw (.outOfDomain .action)

/-- Derive the authoritative machine without filtering or selecting any row's result alternatives. -/
def ParameterTable.machine (table : ParameterTable domain Setup State Outcome Fact)
    (metadata : MachineMetadata) [DecidableEq Setup] [DecidableEq State]
    [DecidableEq Outcome] [DecidableEq Fact] :
    FiniteMachine Setup State (ActionInstance template limits) Outcome Fact :=
  table.validated.machine metadata

/-- Finite kernel membership proves exactly the authored parameter sample, not full schema coverage. -/
theorem ParameterTable.actionDomain_iff (table : ParameterTable domain Setup State Outcome Fact)
    (metadata : MachineMetadata) [DecidableEq Setup] [DecidableEq State]
    [DecidableEq Outcome] [DecidableEq Fact]
    (action : ActionInstance template limits) :
    (table.machine metadata).kernel.actionDomain action ↔ action ∈ domain.actions := by
  change action ∈ table.validated.table.actions.values ↔ action ∈ domain.actions
  rw [table.exactActions]

/-- Every authoritative step is exactly an authored row alternative at the prior state and arguments. -/
theorem ParameterTable.step_iff (table : ParameterTable domain Setup State Outcome Fact)
    (metadata : MachineMetadata) [DecidableEq Setup] [DecidableEq State]
    [DecidableEq Outcome] [DecidableEq Fact]
    (state : State) (action : ActionInstance template limits)
    (result : Step State Outcome Fact) :
    (table.machine metadata).kernel.authoritativeStep state action result ↔
      ∃ row ∈ table.validated.table.transitions,
        row.source = state ∧ row.action = action ∧ result ∈ row.results := by
  change result ∈ table.validated.table.transitions.flatMap
    (fun row => if row.source = state ∧ row.action = action then row.results else []) ↔ _
  simp only [List.mem_flatMap]
  constructor
  · rintro ⟨row, member, emitted⟩
    by_cases selected : row.source = state ∧ row.action = action
    · exact ⟨row, member, selected.1, selected.2, by simpa [selected] using emitted⟩
    · simp [selected] at emitted
  · rintro ⟨row, member, source, arguments, emitted⟩
    exact ⟨row, member, by simpa [source, arguments] using emitted⟩

/-- Admit through normal ModelValue Target semantics and automatically bind domain meaning to the
template's Action metadata. Supplied duplicate Action metadata rejects through normal Target checks. -/
def ParameterDomain.checkTarget [DecidableEq Setup] [DecidableEq State]
    [DecidableEq Outcome] [DecidableEq Fact]
    (domain : ParameterDomain template limits)
    (table : FiniteTable Setup State (ActionInstance template limits) Outcome Fact)
    (identity : FiniteModelIdentity Setup State (ActionInstance template limits) Outcome Fact)
    (definition : FiniteTargetDefinition)
    (composition : TargetComposition LawStatement := .empty) :
    Except FiniteTargetAdmissionError
      (CheckedTarget LawStatement (List RoleBinding) ModelValue ModelValue ModelValue ModelValue) := do
  let _ ← domain.validateTable table |>.mapError FiniteTargetAdmissionError.invalidTable
  table.checkModelTarget { identity with actionId := fun _ => template.identity }
    { definition with definitions := {
        id := template.identity, kind := .action, source := definition.source,
        canonicalBehavior := domain.canonical } :: definition.definitions }
    composition

end Umpire.Operation
