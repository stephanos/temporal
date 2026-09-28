import Lean.Data.Name

/-!
Pure `feature-entity-uniqueness` checking for the Temporal Lean model.

One entity or action name is declared once across the production modules under `Temporal.Feature`:
two declarations of one name are two meanings for one word, and a composition that synchronizes on
the name cannot say which it means. The rule reads declarations, which the import-graph rules cannot
see, so the driver collects them from an imported environment and hands them here as plain data.
-/

namespace ModelLint.Entity

/-- The declaration kinds the rule compares. An entity and an action may share a name; two of one
kind may not. -/
inductive Kind where
  | entity
  | action
  deriving Repr, BEq, DecidableEq

def Kind.label : Kind → String
  | .entity => "entity"
  | .action => "action"

/-- One declaration and the module that declares it. -/
structure Declaration where
  kind : Kind
  name : String
  module : Lean.Name
  deriving Repr, BEq

/-- A duplicate the model carries on purpose until the spec that removes it lands: the declaring
modules it may appear in, and that spec. A declaration of the name in any other module fails. -/
structure Allowance where
  kind : Kind
  name : String
  modules : Array Lean.Name
  removedBy : String
  deriving Repr, BEq

/-- One declaration whose name another in-scope module already declares. -/
structure Violation where
  kind : Kind
  name : String
  /-- The first module, in name order, that declares the name, other than `duplicate`. -/
  first : Lean.Name
  duplicate : Lean.Name
  deriving Repr, BEq

private def followUp : String :=
  "the fn-92 follow-up: a use-case entity key, the operation entity's move, and the fixture and \
  canary re-pin"

/-- The duplicates that exist once the worker entity module lands. Start and Outage correlate a
workflow by different keys, and the caller and Outage modules keep their own worker faults, until
the grammar lets a use case choose its key; that change removes every entry here. -/
def allowlist : Array Allowance := #[
  { kind := .entity, name := "workflow", removedBy := followUp, modules := #[
      `Temporal.Feature.Nexus.Caller.Model,
      `Temporal.Feature.Workflow.Outage.Model,
      `Temporal.Feature.Workflow.Start.Model] },
  { kind := .action, name := "startWorkflow", removedBy := followUp, modules := #[
      `Temporal.Feature.Workflow.Outage.Model,
      `Temporal.Feature.Workflow.Start.Model] },
  { kind := .action, name := "workerStop", removedBy := followUp, modules := #[
      `Temporal.Feature.Nexus.Caller.Model,
      `Temporal.Feature.Worker.Model,
      `Temporal.Feature.Workflow.Outage.Model] },
  { kind := .action, name := "workerResume", removedBy := followUp, modules := #[
      `Temporal.Feature.Worker.Model,
      `Temporal.Feature.Workflow.Outage.Model] }
]

private def excludedComponents : Array String := #["Tests", "Success"]

/-- Whether a module is a production feature module: under `Temporal.Feature`, and neither a test
module nor a specimen. Tests and the success specimen declare their own vocabulary on purpose. -/
def inScope (module : Lean.Name) : Bool :=
  let components := module.components.map (·.toString)
  (`Temporal.Feature).isPrefixOf module && module != `Temporal.Feature &&
    !components.any (excludedComponents.contains ·) &&
    !(components.getLast?.getD "").endsWith "Tests"

private def nameLt (left right : Lean.Name) : Bool :=
  left.toString < right.toString

/-- Every in-scope declaration whose name another in-scope module of the same kind declares. The
first declaring module in name order holds the name; with an allowance, the allowance's modules hold
it instead, so a module the allowance does not name fails even when it sorts first. Deterministic:
violations follow the declaring modules' name order within each name, and names in order of first
declaration. -/
def check (allowlist : Array Allowance) (declarations : Array Declaration) :
    Array Violation := Id.run do
  let inScopeDeclarations := declarations.filter (inScope ·.module)
  let mut seen : Array (Kind × String) := #[]
  let mut violations := #[]
  for declaration in inScopeDeclarations do
    let group := (declaration.kind, declaration.name)
    if seen.contains group then continue
    seen := seen.push group
    let modules := ((inScopeDeclarations.filter fun other =>
        other.kind == declaration.kind && other.name == declaration.name).map (·.module)).toList
      |>.eraseDups |>.toArray.qsort nameLt
    if modules.size < 2 then continue
    let allowed := (allowlist.find? fun allowance =>
        allowance.kind == declaration.kind && allowance.name == declaration.name).map (·.modules)
    for module in modules do
      let holds := match allowed with
        | some holders => holders.contains module
        | none => module == modules[0]!
      if holds then continue
      let some first := modules.find? (· != module) | continue
      violations := violations.push {
        kind := declaration.kind, name := declaration.name, first, duplicate := module }
  violations

/-- Render one deterministic diagnostic naming both declaring modules. -/
def Violation.render (violation : Violation) : String :=
  s!"[model-entity/feature-entity-uniqueness] duplicate {violation.kind.label} \
    {violation.name}: {violation.first} and {violation.duplicate}"

end ModelLint.Entity
