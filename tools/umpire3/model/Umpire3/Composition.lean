import Lean.Data.Json
import Umpire3.Registration

namespace Umpire3

structure ContractGuarantee where
  identifier : String
  statementHash : String
  theoremName : String
  statement : String
  axioms : List String
  trustBadge : String
  deriving DecidableEq, Repr

def ContractGuarantee.ofGuarantee (guarantee : Guarantee) : ContractGuarantee where
  identifier := guarantee.identifier
  statementHash := "derived"
  theoremName := guarantee.resolved.name
  statement := guarantee.resolved.statement
  axioms := guarantee.resolved.axioms
  trustBadge := if guarantee.resolved.axioms.isEmpty then "kernel" else "kernel-with-declared-axioms"

structure ContractRequirement where
  providerModule : String
  guarantee : String
  statementHash : String
  theoremName : String
  statement : String
  axioms : List String
  trustBadge : String
  deriving DecidableEq, Repr

def ContractRequirement.ofRequirement (providerModule : String) (provider : Guarantee)
    (_ : Requirement provider) : ContractRequirement where
  providerModule
  guarantee := provider.identifier
  statementHash := "derived"
  theoremName := provider.resolved.name
  statement := provider.resolved.statement
  axioms := provider.resolved.axioms
  trustBadge := if provider.resolved.axioms.isEmpty then "kernel" else "kernel-with-declared-axioms"

structure ModelObligation where
  identifier : String
  kind : String
  status : String
  detail : String
  deriving DecidableEq, Repr

structure ModuleContract where
  identifier : String
  rank : Nat
  owns : List String := []
  provides : List ContractGuarantee := []
  requires : List ContractRequirement := []
  interferenceActions : List String := []
  obligations : List ModelObligation := []
  deriving DecidableEq, Repr

structure ProjectionOmission where
  identifier : String
  reason : String
  maxCount : Nat
  deriving DecidableEq, Repr

structure TargetProjection where
  identifier : String
  modules : List String
  properties : List String
  retainedActions : List String
  omissions : List ProjectionOmission := []
  deriving DecidableEq, Repr

structure Composition where
  proof : ResolvedTheorem
  modules : List ModuleContract
  targets : List TargetProjection
  deriving DecidableEq, Repr

private def uniqueNonempty (values : List String) : Bool :=
  values.all (· != "") && values.eraseDups.length = values.length

private def findModule? (modules : List ModuleContract) (identifier : String) : Option ModuleContract :=
  modules.find? (·.identifier = identifier)

private def requirementSatisfied (modules : List ModuleContract)
    (consumer : ModuleContract) (requirement : ContractRequirement) : Bool :=
  match findModule? modules requirement.providerModule with
  | none => false
  | some provider =>
      provider.rank < consumer.rank && provider.provides.any (fun guarantee =>
        guarantee.identifier = requirement.guarantee &&
          guarantee.theoremName = requirement.theoremName &&
          guarantee.statement = requirement.statement &&
          guarantee.axioms = requirement.axioms)

private def targetMetadataValid (modules : List ModuleContract) (target : TargetProjection) : Bool :=
  target.identifier != "" && uniqueNonempty target.modules &&
    uniqueNonempty target.properties && uniqueNonempty target.retainedActions &&
    target.modules.all (fun identifier => (findModule? modules identifier).isSome) &&
    target.modules.all (fun identifier =>
      match findModule? modules identifier with
      | none => false
      | some module => module.interferenceActions.all target.retainedActions.contains) &&
    target.omissions.all (fun omission =>
      omission.identifier != "" && omission.reason != "" && omission.maxCount > 0)

def Composition.metadataValid (composition : Composition) : Bool :=
  !composition.modules.isEmpty && !composition.targets.isEmpty &&
    uniqueNonempty (composition.modules.map (·.identifier)) &&
    uniqueNonempty (composition.modules.flatMap (·.owns)) &&
    uniqueNonempty (composition.modules.flatMap (fun module => module.provides.map (·.identifier))) &&
    composition.modules.all (fun module =>
      uniqueNonempty module.owns || module.owns.isEmpty) &&
    composition.modules.all (fun module =>
      module.requires.all (requirementSatisfied composition.modules module)) &&
    composition.modules.all (fun module => module.obligations.all (fun obligation =>
      obligation.identifier != "" && obligation.kind != "" &&
        (obligation.status = "metadata-present" || obligation.status = "metadata-missing") &&
        obligation.detail != "")) &&
    uniqueNonempty (composition.targets.map (·.identifier)) &&
    composition.targets.all (targetMetadataValid composition.modules)

def Composition.MetadataValid (composition : Composition) : Prop :=
  composition.metadataValid = true

private def stringsJson (values : List String) : Lean.Json :=
  Lean.Json.arr (values.map Lean.toJson).toArray

private def ContractGuarantee.toJson (guarantee : ContractGuarantee) : Lean.Json := Lean.Json.mkObj [
  ("identifier", guarantee.identifier),
  ("statementHash", guarantee.statementHash),
  ("theorem", guarantee.theoremName),
  ("statement", guarantee.statement),
  ("axioms", stringsJson guarantee.axioms),
  ("trustBadge", guarantee.trustBadge),
]

private def ContractRequirement.toJson (requirement : ContractRequirement) : Lean.Json := Lean.Json.mkObj [
  ("providerModule", requirement.providerModule),
  ("guarantee", requirement.guarantee),
  ("statementHash", requirement.statementHash),
  ("theorem", requirement.theoremName),
  ("statement", requirement.statement),
  ("axioms", stringsJson requirement.axioms),
  ("trustBadge", requirement.trustBadge),
]

private def ModelObligation.toJson (obligation : ModelObligation) : Lean.Json := Lean.Json.mkObj [
  ("identifier", obligation.identifier),
  ("kind", obligation.kind),
  ("status", obligation.status),
  ("detail", obligation.detail),
]

private def ModuleContract.toJson (module : ModuleContract) : Lean.Json := Lean.Json.mkObj [
  ("identifier", module.identifier),
  ("rank", module.rank),
  ("owns", stringsJson module.owns),
  ("provides", Lean.Json.arr (module.provides.map ContractGuarantee.toJson).toArray),
  ("requires", Lean.Json.arr (module.requires.map ContractRequirement.toJson).toArray),
  ("interferenceActions", stringsJson module.interferenceActions),
  ("obligations", Lean.Json.arr (module.obligations.map ModelObligation.toJson).toArray),
]

private def ProjectionOmission.toJson (omission : ProjectionOmission) : Lean.Json := Lean.Json.mkObj [
  ("identifier", omission.identifier),
  ("reason", omission.reason),
  ("maxCount", omission.maxCount),
]

private def TargetProjection.toJson (target : TargetProjection) : Lean.Json := Lean.Json.mkObj [
  ("identifier", target.identifier),
  ("modules", stringsJson target.modules),
  ("properties", stringsJson target.properties),
  ("retainedActions", stringsJson target.retainedActions),
  ("omissions", Lean.Json.arr (target.omissions.map ProjectionOmission.toJson).toArray),
]

def compositionJson (semanticHash dependencyHash catalogHash : String)
    (composition : Composition) : String :=
  (Lean.Json.mkObj [
    ("formatVersion", "umpire3/composition/v4"),
    ("resultClass", "composition-proved"),
    ("trustBadge", if composition.proof.axioms.isEmpty && composition.modules.all (fun module =>
      module.provides.all (·.axioms.isEmpty) && module.requires.all (·.axioms.isEmpty)) then
        "kernel" else "kernel-with-declared-axioms"),
    ("semanticHash", semanticHash),
    ("sourceDigest", semanticHash),
    ("dependencyDigest", dependencyHash),
    ("artifactDigest", "derived"),
    ("catalogHash", catalogHash),
    ("proof", composition.proof.declaration.toJson),
    ("modules", Lean.Json.arr (composition.modules.map ModuleContract.toJson).toArray),
    ("targets", Lean.Json.arr (composition.targets.map TargetProjection.toJson).toArray),
  ]).compress

end Umpire3
