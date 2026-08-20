import Lean.Data.Json

namespace Umpire3

structure TypeDeclaration where
  identifier : String
  kind : String
  description : String
  deriving DecidableEq, Repr

structure CapabilityDeclaration where
  identifier : String
  description : String
  deriving DecidableEq, Repr

structure ParameterDeclaration where
  name : String
  type : String
  required : Bool
  deriving DecidableEq, Repr

structure ProjectionDeclaration where
  name : String
  type : String
  deriving DecidableEq, Repr

structure FootprintDeclaration where
  protocol : String
  service : String
  route : String
  deriving DecidableEq, Repr

structure ActionDeclaration where
  identifier : String
  description : String
  parameters : List ParameterDeclaration := []
  dependencies : List String := []
  projections : List ProjectionDeclaration := []
  footprint : List FootprintDeclaration := []
  requiredCapabilities : List String
  deriving DecidableEq, Repr

structure EntityDeclaration where
  identifier : String
  description : String
  deriving DecidableEq, Repr

structure RelationDeclaration where
  identifier : String
  source : String
  target : String
  description : String
  deriving DecidableEq, Repr

structure ObservationDeclaration where
  identifier : String
  description : String
  deriving DecidableEq, Repr

structure EvidenceDeclaration where
  identifier : String
  description : String
  deriving DecidableEq, Repr

structure PropertyDeclaration where
  identifier : String
  description : String
  statementHash : String
  evidence : List String
  deriving DecidableEq, Repr

structure PolicyDeclaration where
  identifier : String
  description : String
  deriving DecidableEq, Repr

structure FaultDeclaration where
  identifier : String
  description : String
  safetyClass : String
  scopeDimensions : List String
  requiredCapabilities : List String
  deriving DecidableEq, Repr

structure ModuleDeclaration where
  identifier : String
  description : String
  deriving DecidableEq, Repr

structure TargetDeclaration where
  identifier : String
  modules : List String
  properties : List String
  deriving DecidableEq, Repr

structure SemanticCatalog where
  version : String
  types : List TypeDeclaration
  capabilities : List CapabilityDeclaration
  actions : List ActionDeclaration
  entities : List EntityDeclaration
  relations : List RelationDeclaration
  observations : List ObservationDeclaration
  evidence : List EvidenceDeclaration
  properties : List PropertyDeclaration
  policies : List PolicyDeclaration
  faults : List FaultDeclaration
  modules : List ModuleDeclaration
  targets : List TargetDeclaration
  deriving DecidableEq, Repr

private def identifiersUnique : List String → Bool
  | [] => true
  | identifier :: rest => !rest.contains identifier && identifiersUnique rest

private def identifiersPresent (identifiers : List String) : Bool :=
  !identifiers.isEmpty && identifiersUnique identifiers && identifiers.all (· != "")

private def identifiersValid (identifiers : List String) : Bool :=
  identifiersUnique identifiers && identifiers.all (· != "")

private def containsIdentifier (identifier : String) (identifiers : List String) : Bool :=
  identifiers.any (· == identifier)

def SemanticCatalog.wellFormed (catalog : SemanticCatalog) : Bool :=
  catalog.version != "" &&
    identifiersPresent (catalog.types.map (·.identifier)) &&
    identifiersPresent (catalog.capabilities.map (·.identifier)) &&
    identifiersPresent (catalog.actions.map (·.identifier)) &&
    identifiersPresent (catalog.entities.map (·.identifier)) &&
    identifiersPresent (catalog.relations.map (·.identifier)) &&
    identifiersPresent (catalog.observations.map (·.identifier)) &&
    identifiersPresent (catalog.evidence.map (·.identifier)) &&
    identifiersPresent (catalog.properties.map (·.identifier)) &&
    identifiersPresent (catalog.policies.map (·.identifier)) &&
    identifiersPresent (catalog.faults.map (·.identifier)) &&
    identifiersPresent (catalog.modules.map (·.identifier)) &&
    identifiersPresent (catalog.targets.map (·.identifier)) &&
    catalog.actions.all (fun action =>
      identifiersValid (action.parameters.map (·.name)) &&
      action.parameters.all (fun parameter =>
        containsIdentifier parameter.type (catalog.types.map (·.identifier)))) &&
    catalog.actions.all (fun action =>
      identifiersValid action.dependencies && action.dependencies.all (fun dependency =>
        containsIdentifier dependency (catalog.actions.map (·.identifier)))) &&
    catalog.actions.all (fun action =>
      identifiersValid (action.projections.map (·.name)) && action.projections.all (fun projection =>
        containsIdentifier projection.type (catalog.types.map (·.identifier)))) &&
    catalog.actions.all (fun action => action.footprint.all (fun call =>
      call.protocol != "" && call.service != "" && call.route != "")) &&
    catalog.actions.all (fun action => action.requiredCapabilities.all (fun capability =>
      containsIdentifier capability (catalog.capabilities.map (·.identifier)))) &&
    catalog.relations.all (fun relation =>
      containsIdentifier relation.source (catalog.entities.map (·.identifier)) &&
      containsIdentifier relation.target (catalog.entities.map (·.identifier))) &&
    catalog.properties.all (fun property => property.statementHash != "" &&
      property.evidence.all (fun evidence =>
        containsIdentifier evidence (catalog.evidence.map (·.identifier)))) &&
    catalog.faults.all (fun fault => fault.safetyClass != "" &&
      identifiersPresent fault.scopeDimensions && fault.requiredCapabilities.all (fun capability =>
        containsIdentifier capability (catalog.capabilities.map (·.identifier)))) &&
    catalog.targets.all (fun target => target.modules.all (fun module =>
      containsIdentifier module (catalog.modules.map (·.identifier)))) &&
    catalog.targets.all (fun target => target.properties.all (fun property =>
      containsIdentifier property (catalog.properties.map (·.identifier))))

def SemanticCatalog.WellFormed (catalog : SemanticCatalog) : Prop :=
  catalog.wellFormed = true

private def stringsJson (values : List String) : Lean.Json :=
  Lean.Json.arr (values.map Lean.toJson).toArray

private def TypeDeclaration.toJson (declaration : TypeDeclaration) : Lean.Json := Lean.Json.mkObj [
  ("identifier", declaration.identifier),
  ("kind", declaration.kind),
  ("description", declaration.description),
]

private def CapabilityDeclaration.toJson (declaration : CapabilityDeclaration) : Lean.Json := Lean.Json.mkObj [
  ("identifier", declaration.identifier),
  ("description", declaration.description),
]

private def ParameterDeclaration.toJson (declaration : ParameterDeclaration) : Lean.Json := Lean.Json.mkObj [
  ("name", declaration.name),
  ("type", declaration.type),
  ("required", declaration.required),
]

private def ProjectionDeclaration.toJson (declaration : ProjectionDeclaration) : Lean.Json := Lean.Json.mkObj [
  ("name", declaration.name),
  ("type", declaration.type),
]

private def FootprintDeclaration.toJson (declaration : FootprintDeclaration) : Lean.Json := Lean.Json.mkObj [
  ("protocol", declaration.protocol),
  ("service", declaration.service),
  ("route", declaration.route),
]

private def ActionDeclaration.toJson (declaration : ActionDeclaration) : Lean.Json := Lean.Json.mkObj [
  ("identifier", declaration.identifier),
  ("description", declaration.description),
  ("parameters", Lean.Json.arr (declaration.parameters.map ParameterDeclaration.toJson).toArray),
  ("dependencies", stringsJson declaration.dependencies),
  ("projections", Lean.Json.arr (declaration.projections.map ProjectionDeclaration.toJson).toArray),
  ("footprint", Lean.Json.arr (declaration.footprint.map FootprintDeclaration.toJson).toArray),
  ("requiredCapabilities", stringsJson declaration.requiredCapabilities),
]

private def EntityDeclaration.toJson (declaration : EntityDeclaration) : Lean.Json := Lean.Json.mkObj [
  ("identifier", declaration.identifier),
  ("description", declaration.description),
]

private def RelationDeclaration.toJson (declaration : RelationDeclaration) : Lean.Json := Lean.Json.mkObj [
  ("identifier", declaration.identifier),
  ("source", declaration.source),
  ("target", declaration.target),
  ("description", declaration.description),
]

private def ObservationDeclaration.toJson (declaration : ObservationDeclaration) : Lean.Json := Lean.Json.mkObj [
  ("identifier", declaration.identifier),
  ("description", declaration.description),
]

private def EvidenceDeclaration.toJson (declaration : EvidenceDeclaration) : Lean.Json := Lean.Json.mkObj [
  ("identifier", declaration.identifier),
  ("description", declaration.description),
]

private def PropertyDeclaration.toJson (declaration : PropertyDeclaration) : Lean.Json := Lean.Json.mkObj [
  ("identifier", declaration.identifier),
  ("description", declaration.description),
  ("statementHash", declaration.statementHash),
  ("evidence", stringsJson declaration.evidence),
]

private def PolicyDeclaration.toJson (declaration : PolicyDeclaration) : Lean.Json := Lean.Json.mkObj [
  ("identifier", declaration.identifier),
  ("description", declaration.description),
]

private def FaultDeclaration.toJson (declaration : FaultDeclaration) : Lean.Json := Lean.Json.mkObj [
  ("identifier", declaration.identifier),
  ("description", declaration.description),
  ("safetyClass", declaration.safetyClass),
  ("scopeDimensions", stringsJson declaration.scopeDimensions),
  ("requiredCapabilities", stringsJson declaration.requiredCapabilities),
]

private def ModuleDeclaration.toJson (declaration : ModuleDeclaration) : Lean.Json := Lean.Json.mkObj [
  ("identifier", declaration.identifier),
  ("description", declaration.description),
]

private def TargetDeclaration.toJson (declaration : TargetDeclaration) : Lean.Json := Lean.Json.mkObj [
  ("identifier", declaration.identifier),
  ("modules", stringsJson declaration.modules),
  ("properties", stringsJson declaration.properties),
]

def SemanticCatalog.toJson (catalog : SemanticCatalog) (semanticHash : String) : Lean.Json := Lean.Json.mkObj [
  ("formatVersion", "umpire3/catalog/v1"),
  ("catalogVersion", catalog.version),
  ("semanticHash", semanticHash),
  ("types", Lean.Json.arr (catalog.types.map TypeDeclaration.toJson).toArray),
  ("capabilities", Lean.Json.arr (catalog.capabilities.map CapabilityDeclaration.toJson).toArray),
  ("actions", Lean.Json.arr (catalog.actions.map ActionDeclaration.toJson).toArray),
  ("entities", Lean.Json.arr (catalog.entities.map EntityDeclaration.toJson).toArray),
  ("relations", Lean.Json.arr (catalog.relations.map RelationDeclaration.toJson).toArray),
  ("observations", Lean.Json.arr (catalog.observations.map ObservationDeclaration.toJson).toArray),
  ("evidence", Lean.Json.arr (catalog.evidence.map EvidenceDeclaration.toJson).toArray),
  ("properties", Lean.Json.arr (catalog.properties.map PropertyDeclaration.toJson).toArray),
  ("policies", Lean.Json.arr (catalog.policies.map PolicyDeclaration.toJson).toArray),
  ("faults", Lean.Json.arr (catalog.faults.map FaultDeclaration.toJson).toArray),
  ("modules", Lean.Json.arr (catalog.modules.map ModuleDeclaration.toJson).toArray),
  ("targets", Lean.Json.arr (catalog.targets.map TargetDeclaration.toJson).toArray),
]

end Umpire3
