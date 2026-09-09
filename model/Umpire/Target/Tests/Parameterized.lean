import Umpire.Target.Parameterized
import Umpire.Value.Field

/-! Parameter domains, checked Atom denotation, Target-owned alternatives, and bounded planning. -/

namespace Umpire.TargetTests.Parameterized
open Umpire Operation Value

private def schema : Schema := ⟨"M", [{
  name := "M", protoSyntax := "proto3", descriptor := "m", fileContext := "", references := [],
  valueShape := some (.message [
    ⟨1, "payload", .bytes, .singular, .optional, none⟩]) }]⟩
private def owner : RpcOwner where
  Witness _ _ := Unit
  schema _ := ⟨"example.Call", schema, schema, [], false, false⟩
private def limits : Limits := ⟨8, 10000, 1024, 100⟩
private def bounds : RuntimeBounds := ⟨8, 1024, 100⟩
private def binding := checkRpc owner (Request := Unit) (Response := Unit) ()
  (owner.schema (Request := Unit) (Response := Unit) ())
def actionId : DefinitionId := .of "example.action.call"
private def template := binding >>= fun checked =>
  ActionTemplate.check (rpc Empty checked) actionId
private def raw (byte : UInt8) := message "M" [(1, literal (.bytes [byte]))]
def checkDomain (resources : Limits) (samples : List Raw) (coverage : ParameterCoverage)
    (scope : RuntimeScope) : Except String
      ((t : ActionTemplate owner Unit Unit Empty) × ParameterDomain t resources) := do
  let t ← template |>.mapError (fun _ => "template")
  let d ← ParameterDomain.check t resources samples coverage scope |>.mapError (fun _ => "domain")
  pure ⟨t, d⟩

def domain (scope : RuntimeScope) := checkDomain limits [raw 0, raw 1] .sampled scope

variable {t : ActionTemplate owner Unit Unit Empty}

#guard (domain .samplesOnly).toOption.map (·.2.actions.length) == some 2
#guard ((domain .samplesOnly).toOption.bind fun ⟨_, d⟩ =>
  (d.admitRuntime limits (raw 2)).toOption.map (·.arguments.value)).isNone
#guard ((domain (.schema bounds)).toOption.bind fun ⟨_, d⟩ =>
  (d.admitRuntime limits (raw 2)).toOption.map (·.arguments.value)) == some (raw 2)
#guard ((domain (.schema bounds)).toOption.bind fun ⟨_, d⟩ =>
  (d.admitRuntime { limits with bytes := 1 } (raw 2)).toOption.map (·.arguments.value)).isNone
#guard (do
  checkDomain limits [raw 0, raw 1] (.abstracted "all-bytes") .samplesOnly
    |>.mapError (fun _ => "unsupported")).toOption.isNone
#guard (do
  let t ← template.toOption
  pure (match ParameterDomain.check t limits [raw 0, raw 1] (.abstracted "all-bytes") .samplesOnly with
    | .error (.unsupportedAbstraction "all-bytes") => true
    | _ => false)) == some true
#guard (domain .samplesOnly).toOption.map (fun ⟨_, d⟩ => d.actions.map (·.arguments.value)) ==
  some [raw 0, raw 1]

def source : SourceLocation := ⟨"parameterized.lean", 1, 1, "authored"⟩
def value (name key : String) := ModelValue.named (.of name) key
private def table (d : ParameterDomain t limits) :
    FiniteTable Unit Nat (ActionInstance t limits) Nat Nat := {
  setups := [⟨(), "setup"⟩]
  states := [⟨0, "idle"⟩, ⟨1, "done"⟩]
  actions := d.catalog
  outcomes := [⟨0, "accepted"⟩, ⟨1, "rejected"⟩]
  facts := []
  initial := [⟨(), [0]⟩]
  transitions := d.actions.zipIdx |>.flatMap fun (action, i) => [
    ⟨"idle-" ++ toString i, 0, action, [⟨0, 1, []⟩, ⟨1, 0, []⟩]⟩,
    ⟨"done-" ++ toString i, 1, action, [⟨1, 1, []⟩]⟩]
}
private def identity (t : ActionTemplate owner Unit Unit Empty) : FiniteModelIdentity Unit Nat (ActionInstance t limits) Nat Nat := {
  setupBindings := fun _ => []
  stateId := fun _ => .of "example.state.phase"
  actionId := fun _ => actionId
  outcomeId := fun _ => .of "example.outcome.call"
  factId := fun _ => .of "example.fact.call"
}
private def definition : FiniteTargetDefinition := {
  id := .of "example.target.call", source,
  metadata := { id := .of "example.kernel.call", source }
  requiredCapabilities := []
  definitions := [
    ⟨.of "example.target.call", .target, source, 1, "target", ""⟩,
    ⟨.of "example.kernel.call", .kernel, source, 1, "kernel", ""⟩,
    ⟨.of "example.state.phase", .state, source, 1, "phase", ""⟩,
    ⟨.of "example.outcome.call", .outcome, source, 1, "outcome", ""⟩]
}
def target (scope : RuntimeScope) (samples : List Raw := [raw 0, raw 1]) :
    Except String (CheckedTarget (fun _ => True)
      (List RoleBinding) ModelValue ModelValue ModelValue ModelValue) := do
  let ⟨t, d⟩ ← checkDomain limits samples .sampled scope
  d.checkTarget (table d) (identity t) definition |>.mapError (fun _ => "target")

#guard (target .samplesOnly).toOption.isSome
#guard (do
  let ⟨_, d⟩ ← (domain .samplesOnly).toOption
  let t ← (target .samplesOnly).toOption
  pure (d.actions.map fun a =>
    ((t.kernel.steps (value "example.state.phase" "idle") a.modelValue).map (·.modelOutcome.value),
     (t.kernel.steps (value "example.state.phase" "done") a.modelValue).map (·.modelOutcome.value)))) ==
  some [(["accepted", "rejected"], ["rejected"]), (["accepted", "rejected"], ["rejected"])]
#guard (do
  let ⟨_, d⟩ ← (domain .samplesOnly).toOption
  pure (d.actions.all fun a =>
    (d.decode a.modelValue).map (·.arguments.value) == some a.arguments.value)) == some true
#guard (do
  let ⟨_, d⟩ ← (domain (.schema bounds)).toOption
  let a ← (d.admitRuntime limits (raw 2)).toOption
  let t ← (target (.schema bounds)).toOption
  pure (d.decode a.modelValue |>.isNone,
    t.kernel.steps (value "example.state.phase" "idle") a.modelValue |>.isEmpty,
    d.actions.length)) == some (true, true, 2)
#guard (target .samplesOnly).toOption.map (·.behaviorFingerprint) !=
  (target .samplesOnly [raw 0, raw 2]).toOption.map (·.behaviorFingerprint)
#guard (target .samplesOnly).toOption.map (·.behaviorFingerprint) !=
  (target (.schema bounds)).toOption.map (·.behaviorFingerprint)
#guard (target .samplesOnly).toOption.map (·.behaviorFingerprint) ==
  (target .samplesOnly [raw 1, raw 0]).toOption.map (·.behaviorFingerprint)
#guard (do
  let ⟨_, d⟩ ← (domain .samplesOnly).toOption
  let receipt ← (Lean.Json.parse d.canonical).toOption
  (receipt.getObjValAs? String "formatVersion").toOption) == some "umpire-parameter-domain/v1"
#guard (do
  let ⟨_, d⟩ ← (domain .samplesOnly).toOption
  pure ((d.validateTable { table d with actions := d.catalog.take 1 }).toOption.isNone,
    (d.validateTable { table d with transitions := [] }).toOption.isNone)) == some (true, true)


#guard (do
  let ⟨t, d⟩ ← (domain .samplesOnly).toOption
  let first ← d.actions.head?
  let decoded ← d.decode first.modelValue
  let field ← (Field.reference owner t.declaration.reference .request "M" 1 source).toOption
  let cursor ← ((Field.root decoded.arguments).field field source).toOption
  let typed ← (cursor.refine .bytes .singular .optional source).toOption
  let present ← (typed.establish source).toOption
  (present.scalar source).toOption) == some (.bytes [0])
#guard (do
  let ⟨_, d⟩ ← (domain .samplesOnly).toOption
  let first ← d.actions.head?
  pure ((d.decode { first.modelValue with definitionId := DefinitionId.of "example.wrong.call" }).isNone,
    (d.decode { first.modelValue with value := "parameterized-v2-" ++ first.canonical }).isNone,
    (d.decode (ModelValue.named actionId "payload=0")).isNone)) == some (true, true, true)
#guard (checkDomain limits [raw 0, raw 0] .sampled .samplesOnly).toOption.isNone
#guard (checkDomain limits [raw 0, raw 1] .fixed .samplesOnly).toOption.isNone
#guard (checkDomain limits [raw 0] .fixed .samplesOnly).toOption.isSome
#guard (checkDomain limits
  [message "M" [(1, literal (.text "wrong-type"))]] .sampled .samplesOnly).toOption.isNone
#guard (checkDomain limits
  [message "WrongSchema" []] .sampled .samplesOnly).toOption.isNone
#guard (do
  let ⟨_, d⟩ ← (domain (.schema bounds)).toOption
  pure ((d.admitRuntime { limits with work := 1 } (raw 2)).toOption.isNone,
    (d.admitRuntime { limits with work := 100000 } (raw 2)).toOption.map (·.modelValue) ==
      (d.admitRuntime limits (raw 2)).toOption.map (·.modelValue))) == some (true, true)
#guard (do
  let ⟨_, d⟩ ← (domain (.schema bounds)).toOption
  pure (match d.admitRuntime { limits with work := 0 } (raw 2) with
    | .error (.runtimeResource _) => true
    | _ => false)) == some true
#guard (do
  let ⟨_, d⟩ ← (checkDomain limits [raw 0] .sampled
    (.schema { bounds with bytes := 1 })).toOption
  pure (match d.admitRuntime limits (raw 2) with
    | .error .outOfScope => true | _ => false)) == some true
#guard (checkDomain limits (List.range 20 |>.map fun n => raw n.toUInt8)
  .sampled .samplesOnly).toOption.map (·.2.actions.length) == some 20

private def schemaIdentity (s : RpcSchema) : Option String := do
  let selected : RpcOwner := { Witness := fun _ _ => Unit, schema := fun _ => s }
  let binding ← (checkRpc selected (Request := Unit) (Response := Unit) () s).toOption
  let t ← (ActionTemplate.check (rpc Empty binding) (DefinitionId.of "example.action.call")).toOption
  let argument ← (Value.check selected t.declaration.reference .request limits (raw 0)).toOption
  pure (ActionInstance.mk argument).canonical

private def selectedSchema := owner.schema (Request := Unit) (Response := Unit) ()
#guard (schemaIdentity selectedSchema).isSome
#guard schemaIdentity selectedSchema != schemaIdentity { selectedSchema with fullName := "example.Other" }
-- Version 2 identity names the selected operation instead of re-encoding the descriptor closure
-- that selection carries, so a payload key no longer separates two catalogs that publish the same
-- method under different descriptors, shared inputs or value shapes. Nothing weaker is admitted:
-- `checkRpc` rejects any candidate that is not exactly its own owner's selection, and a domain's
-- samples share one template, so the constant identity cannot collide two distinct requests.
#guard schemaIdentity selectedSchema == schemaIdentity { selectedSchema with
  response := { schema with nodes := schema.nodes.map fun n => { n with descriptor := "new-descriptor" } } }
#guard schemaIdentity selectedSchema == schemaIdentity { selectedSchema with
  schemaInputs := schema.nodes }
#guard schemaIdentity selectedSchema == schemaIdentity { selectedSchema with
  response := { schema with nodes := schema.nodes.map fun n => { n with
    valueShape := some (.message [⟨1, "payload", .bytes, .singular, .required, none⟩]) } } }

/-- error: Application type mismatch -/
#guard_msgs (error, substring := true) in
example (response : Checked owner t.declaration.reference .response limits) :
    ActionInstance t limits := ⟨response⟩

private def wrongOwner : RpcOwner where
  Witness _ _ := Bool
  schema _ := ⟨"example.Call", schema, schema, [], false, false⟩

/-- error: Application type mismatch -/
#guard_msgs (error, substring := true) in
example (request : Checked wrongOwner (Request := Unit) (Response := Unit) false .request limits) :
    ActionInstance t limits := ⟨request⟩

/-- error: Application type mismatch -/
#guard_msgs (error, substring := true) in
example (other : ActionTemplate owner Bool Nat Empty)
    (request : Checked owner t.declaration.reference .request limits) :
    ActionInstance other limits := ⟨request⟩

end Umpire.TargetTests.Parameterized
