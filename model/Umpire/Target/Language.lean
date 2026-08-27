import Lean.Data.Json
import Lean.Elab.Term
import Umpire.Core

/-! Target authoring, checked composition, and canonical target projections. -/

namespace Umpire

structure TargetDeclaration
    (LawStatement : DeclarationId → Prop)
    (Setup State Action Outcome Observation : Type) where
  id : DeclarationId
  source : SemanticSource
  declarations : List DeclarationMetadata
  requiredCapabilities : List DeclarationId
  providers : List (CapabilityProvider LawStatement)
  connectors : List (CapabilityConnector LawStatement)
  resolvedSetups : List Setup
  kernel : KernelAvailability Setup State Action Outcome Observation

/-- Optional finite planning is tied propositionally to the exact authoritative target kernel. -/
structure FinitePlanningCapability
    {State Action Outcome Observation : Type}
    (authoritativeStep : State → Action → TransitionResult State Outcome Observation → Prop) where
  actions : List Action
  roleDomainDigest : String
  actionDomainDigest : String
  actionSound : ∀ action, action ∈ actions →
    ∃ state result, authoritativeStep state action result
  actionComplete : ∀ state action result,
    authoritativeStep state action result → action ∈ actions

inductive FinitePlanningAvailability
    {State Action Outcome Observation : Type}
    (authoritativeStep : State → Action → TransitionResult State Outcome Observation → Prop) where
  | unavailable
  | available (capability : FinitePlanningCapability authoritativeStep)

inductive AuthoredPlanningCapability
    {Setup State Action Outcome Observation : Type}
    (availability : KernelAvailability Setup State Action Outcome Observation) where
  | unavailable
  | available
      (kernel : TransitionKernel Setup State Action Outcome Observation)
      (kernelEq : availability = .checked kernel)
      (capability : FinitePlanningCapability kernel.authoritativeStep)

structure CheckedTarget
    (LawStatement : DeclarationId → Prop)
    (Setup State Action Outcome Observation : Type) where
  private mk ::
  id : DeclarationId
  source : SemanticSource
  declarations : List DeclarationMetadata
  requiredCapabilities : List DeclarationId
  providers : List (CapabilityProvider LawStatement)
  connectors : List (CapabilityConnector LawStatement)
  resolvedSetups : List Setup
  kernel : TransitionKernel Setup State Action Outcome Observation
  planning : FinitePlanningAvailability kernel.authoritativeStep := .unavailable
  canonicalMetadata : String
  semanticDigest : String

/-- Closed authoring roles keep compiler locations separate from semantic declaration identity. -/
inductive AuthoringOccurrenceRole where
  | declarationMetadata
  | targetDeclaration
  | providerDefinition
  | providerReference
  | connectorDefinition
  | connectorReference
  | capabilityRequirement
  | lawRequirement
  | lawWitness
  | meaning
  | reconciliation
  | kernel
  deriving BEq, DecidableEq, Ord, Repr

def AuthoringOccurrenceRole.name : AuthoringOccurrenceRole → String
  | .declarationMetadata => "declaration-metadata"
  | .targetDeclaration => "target-declaration"
  | .providerDefinition => "provider-definition"
  | .providerReference => "provider-reference"
  | .connectorDefinition => "connector-definition"
  | .connectorReference => "connector-reference"
  | .capabilityRequirement => "capability-requirement"
  | .lawRequirement => "law-requirement"
  | .lawWitness => "law-witness"
  | .meaning => "meaning"
  | .reconciliation => "reconciliation"
  | .kernel => "kernel"

/-- The owner makes a nested occurrence path unambiguous when identities are reused. -/
inductive AuthoringOccurrenceContext where
  | direct
  | reconciliation (declaration : DeclarationId)
  deriving BEq, DecidableEq, Repr

structure AuthoringOccurrencePath where
  role : AuthoringOccurrenceRole
  owner : DeclarationId
  context : AuthoringOccurrenceContext := .direct
  deriving BEq, DecidableEq, Repr

/-- Nonsemantic occurrence identity derived from a source span and its local ordinal. -/
structure AuthoringOccurrenceId where
  sourcePath : String
  line : Nat
  column : Nat
  endLine : Nat
  endColumn : Nat
  localOrdinal : Nat
  deriving BEq, DecidableEq, Repr

structure AuthoringOccurrence where
  id : AuthoringOccurrenceId
  declarationId : DeclarationId
  path : AuthoringOccurrencePath
  deriving BEq, DecidableEq, Repr

/-- Compiler-only syntax is paired with the pure occurrence row and never enters checked data. -/
structure CapturedAuthoringOccurrence where
  occurrence : AuthoringOccurrence
  reference : Lean.Syntax

structure AuthoringDiagnostic where
  error : DeclarationError
  path : AuthoringOccurrencePath
  original : Option AuthoringOccurrenceId
  offending : AuthoringOccurrenceId
  deriving BEq, DecidableEq, Repr

/-- Ordinary target input keeps semantic declarations explicit without exposing checked fields. -/
structure TargetDefinition
    (Setup State Action Outcome Observation : Type) where
  id : DeclarationId
  source : SemanticSource
  declarations : List DeclarationMetadata
  requiredCapabilities : List DeclarationId
  resolvedSetups : List Setup
  kernel : KernelAvailability Setup State Action Outcome Observation

private structure TargetCompositionPayload (LawStatement : DeclarationId → Prop) where
  providers : List (CapabilityProvider LawStatement)
  connectors : List (CapabilityConnector LawStatement)

/-- Explicit provider and connector choices whose collection is owned by Target. -/
structure TargetComposition (LawStatement : DeclarationId → Prop) where
  private mk ::
  private payload : TargetCompositionPayload LawStatement

namespace TargetComposition

def empty : TargetComposition LawStatement := ⟨{ providers := [], connectors := [] }⟩

def provide
    (composition : TargetComposition LawStatement)
    (provider : CapabilityProvider LawStatement) : TargetComposition LawStatement :=
  ⟨{ composition.payload with providers := composition.payload.providers ++ [provider] }⟩

def connect
    (composition : TargetComposition LawStatement)
    (connector : CapabilityConnector LawStatement) : TargetComposition LawStatement :=
  ⟨{ composition.payload with connectors := composition.payload.connectors ++ [connector] }⟩

end TargetComposition

structure AuthoredTarget
    (LawStatement : DeclarationId → Prop)
    (Setup State Action Outcome Observation : Type) where
  private mk ::
  private declaration : TargetDeclaration LawStatement Setup State Action Outcome Observation
  private occurrences : List AuthoringOccurrence
  private planning : AuthoredPlanningCapability declaration.kernel

namespace AuthoredTarget

/-- Assemble the ordinary authored value while Target owns provider and connector collection. -/
def make
    (definition : TargetDefinition Setup State Action Outcome Observation)
    (composition : TargetComposition LawStatement := .empty)
    (planning : AuthoredPlanningCapability definition.kernel := .unavailable)
    (occurrences : List AuthoringOccurrence := []) :
    AuthoredTarget LawStatement Setup State Action Outcome Observation :=
  let declaration : TargetDeclaration LawStatement Setup State Action Outcome Observation := {
    id := definition.id
    source := definition.source
    declarations := definition.declarations
    requiredCapabilities := definition.requiredCapabilities
    providers := composition.payload.providers
    connectors := composition.payload.connectors
    resolvedSetups := definition.resolvedSetups
    kernel := definition.kernel
  }
  ⟨declaration, occurrences, planning⟩

/-- Replace only elaboration locations; checked semantic inputs remain unchanged. -/
def withOccurrences
    (authored : AuthoredTarget LawStatement Setup State Action Outcome Observation)
    (occurrences : List AuthoringOccurrence) :
    AuthoredTarget LawStatement Setup State Action Outcome Observation :=
  ⟨authored.declaration, occurrences, authored.planning⟩

/-- Preserve a checked semantic target while making exhaustive planning explicitly unavailable. -/
def withoutPlanning
    (authored : AuthoredTarget LawStatement Setup State Action Outcome Observation) :
    AuthoredTarget LawStatement Setup State Action Outcome Observation :=
  ⟨authored.declaration, authored.occurrences, .unavailable⟩

end AuthoredTarget

private structure TargetValidationError where
  error : DeclarationError
  path : AuthoringOccurrencePath
  occurrenceIdentity : DeclarationId
  source : SemanticSource

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def withoutClosingBrace (value : String) : String :=
  (value.dropEnd 1).toString

private def idLe (left right : DeclarationId) : Bool :=
  decide (left.value ≤ right.value)

private def sourceLe (left right : SemanticSource) : Bool :=
  decide (left.path < right.path) ||
    (left.path == right.path && decide (left.line < right.line)) ||
    (left.path == right.path && left.line == right.line && decide (left.column ≤ right.column))

private def declarationLe (left right : DeclarationMetadata) : Bool :=
  decide (left.id.value < right.id.value) ||
    (left.id == right.id && decide (left.kind.name < right.kind.name)) ||
    (left.id == right.id && left.kind == right.kind && sourceLe left.source right.source)

private def providerLe (left right : CapabilityProvider LawStatement) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def connectorLe (left right : CapabilityConnector LawStatement) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def meaningLe (left right : MeaningProvision) : Bool :=
  decide (left.declaration.value < right.declaration.value) ||
    (left.declaration == right.declaration && decide (left.semanticDigest ≤ right.semanticDigest))

private def reconciliationLe (left right : Reconciliation) : Bool :=
  decide (left.declaration.value < right.declaration.value) ||
    (left.declaration == right.declaration && decide (left.semanticDigest ≤ right.semanticDigest))

private def lawLe (left right : LawRequirement) : Bool :=
  decide (left.id.value < right.id.value) ||
    (left.id == right.id && decide (left.semanticDigest ≤ right.semanticDigest))

private def canonicalIds (ids : List DeclarationId) : List DeclarationId :=
  ids.mergeSort idLe |>.eraseDups

private def sourcePath (source : SemanticSource) : String :=
  if source.path == "" then "<unknown>" else source.path

private def sourceJson (source : SemanticSource) : String :=
  "{\"path\":" ++ quote source.path ++
    ",\"line\":" ++ toString source.line ++
    ",\"column\":" ++ toString source.column ++
    ",\"provenance\":" ++ quote source.provenance ++ "}"

private def lawJson (law : LawRequirement) : String :=
  "{\"id\":" ++ quote law.id.value ++
    ",\"semanticDigest\":" ++ quote law.semanticDigest ++ "}"

private def meaningJson (meaning : MeaningProvision) : String :=
  "{\"id\":" ++ quote meaning.declaration.value ++
    ",\"kind\":" ++ quote meaning.kind.name ++
    ",\"semanticDigest\":" ++ quote meaning.semanticDigest ++ "}"

private def declarationSemanticJson (declaration : DeclarationMetadata) : String :=
  "{\"id\":" ++ quote declaration.id.value ++
    ",\"kind\":" ++ quote declaration.kind.name ++
    ",\"version\":" ++ toString declaration.version ++
    ",\"contractDigest\":" ++ quote declaration.contractDigest ++ "}"

def canonicalDeclarationMetadataJson (declaration : DeclarationMetadata) : String :=
  withoutClosingBrace (declarationSemanticJson declaration) ++
    ",\"source\":" ++ sourceJson declaration.source ++
    ",\"documentation\":" ++ quote declaration.documentation ++ "}"

private def providerSemanticJson (provider : CapabilityProvider LawStatement) : String :=
  let laws := provider.contract.requiredLaws.mergeSort lawLe
  "{\"id\":" ++ quote provider.id.value ++
    ",\"capabilityId\":" ++ quote provider.contract.id.value ++
    ",\"capabilityVersion\":" ++ toString provider.contract.version ++
    ",\"capabilityDigest\":" ++ quote provider.contract.semanticDigest ++
    ",\"meanings\":" ++ array (provider.meanings.mergeSort meaningLe |>.map meaningJson) ++
    ",\"laws\":" ++ array (laws.map lawJson) ++ "}"

def canonicalCapabilityProviderJson (provider : CapabilityProvider LawStatement) : String :=
  withoutClosingBrace (providerSemanticJson provider) ++
    ",\"source\":" ++ sourceJson provider.source ++ "}"

private def reconciliationJson (reconciliation : Reconciliation) : String :=
  "{\"id\":" ++ quote reconciliation.declaration.value ++
    ",\"kind\":" ++ quote reconciliation.kind.name ++
    ",\"providers\":" ++ array (canonicalIds reconciliation.providers |>.map (quote ∘ DeclarationId.value)) ++
    ",\"semanticDigest\":" ++ quote reconciliation.semanticDigest ++ "}"

private def connectorSemanticJson (connector : CapabilityConnector LawStatement) : String :=
  let laws := connector.requiredLaws.mergeSort lawLe
  "{\"id\":" ++ quote connector.id.value ++
    ",\"version\":" ++ toString connector.version ++
    ",\"semanticDigest\":" ++ quote connector.semanticDigest ++
    ",\"reconciliations\":" ++
      array (connector.reconciliations.mergeSort reconciliationLe |>.map reconciliationJson) ++
    ",\"laws\":" ++ array (laws.map lawJson) ++ "}"

def canonicalCapabilityConnectorJson (connector : CapabilityConnector LawStatement) : String :=
  withoutClosingBrace (connectorSemanticJson connector) ++
    ",\"source\":" ++ sourceJson connector.source ++ "}"

private def kernelSemanticJson (metadata : KernelMetadata) : String :=
  "{\"id\":" ++ quote metadata.id.value ++
    ",\"version\":" ++ toString metadata.version ++
    ",\"contractDigest\":" ++ quote metadata.contractDigest ++ "}"

def canonicalKernelMetadataJson (metadata : KernelMetadata) : String :=
  withoutClosingBrace (kernelSemanticJson metadata) ++
    ",\"source\":" ++ sourceJson metadata.source ++ "}"

def canonicalDeclarationErrorJson (error : DeclarationError) : String :=
  "{\"kind\":" ++ quote error.kind.name ++
    ",\"declarationId\":" ++ quote error.declarationId.value ++
    ",\"sourcePath\":" ++ quote error.sourcePath ++
    ",\"offendingValue\":" ++ quote error.offendingValue ++
    ",\"relatedIdentities\":" ++
      array (canonicalIds error.relatedIdentities |>.map (quote ∘ DeclarationId.value)) ++ "}"

private def authoringOccurrenceIdJson (id : AuthoringOccurrenceId) : String :=
  "{\"sourcePath\":" ++ quote id.sourcePath ++
    ",\"line\":" ++ toString id.line ++
    ",\"column\":" ++ toString id.column ++
    ",\"endLine\":" ++ toString id.endLine ++
    ",\"endColumn\":" ++ toString id.endColumn ++
    ",\"localOrdinal\":" ++ toString id.localOrdinal ++ "}"

private def authoringOccurrenceContextJson : AuthoringOccurrenceContext → String
  | .direct => quote "direct"
  | .reconciliation declaration =>
      "{\"reconciliation\":" ++ quote declaration.value ++ "}"

private def authoringOccurrencePathJson (path : AuthoringOccurrencePath) : String :=
  "{\"role\":" ++ quote path.role.name ++
    ",\"owner\":" ++ quote path.owner.value ++
    ",\"context\":" ++ authoringOccurrenceContextJson path.context ++ "}"

def canonicalAuthoringDiagnosticJson (diagnostic : AuthoringDiagnostic) : String :=
  "{\"error\":" ++ canonicalDeclarationErrorJson diagnostic.error ++
    ",\"original\":" ++
      (diagnostic.original.map authoringOccurrenceIdJson |>.getD "null") ++
    ",\"offending\":" ++ authoringOccurrenceIdJson diagnostic.offending ++
    ",\"path\":" ++ authoringOccurrencePathJson diagnostic.path ++ "}"

private def targetSemanticJson
    (id : DeclarationId)
    (declarations : List DeclarationMetadata)
    (requiredCapabilities : List DeclarationId)
    (providers : List (CapabilityProvider LawStatement))
    (connectors : List (CapabilityConnector LawStatement))
    (kernel : KernelMetadata) : String :=
  "{\"id\":" ++ quote id.value ++
    ",\"declarations\":" ++
      array (declarations.mergeSort declarationLe |>.map declarationSemanticJson) ++
    ",\"requiredCapabilities\":" ++
      array (canonicalIds requiredCapabilities |>.map (quote ∘ DeclarationId.value)) ++
    ",\"providers\":" ++ array (providers.mergeSort providerLe |>.map providerSemanticJson) ++
    ",\"connectors\":" ++ array (connectors.mergeSort connectorLe |>.map connectorSemanticJson) ++
    ",\"kernel\":" ++ kernelSemanticJson kernel ++ "}"

private def targetMetadataJson
    (target : TargetDeclaration LawStatement Setup State Action Outcome Observation)
    (kernel : KernelMetadata) : String :=
  "{\"semantic\":" ++ targetSemanticJson target.id target.declarations
      target.requiredCapabilities target.providers target.connectors kernel ++
    ",\"source\":" ++ sourceJson target.source ++
    ",\"declarationMetadata\":" ++
      array (target.declarations.mergeSort declarationLe |>.map canonicalDeclarationMetadataJson) ++
    ",\"kernelMetadata\":" ++ canonicalKernelMetadataJson kernel ++ "}"

private def declarationError
    (kind : DeclarationErrorKind)
    (declarationId : DeclarationId)
    (source : SemanticSource)
    (offendingValue : String)
    (relatedIdentities : List DeclarationId := []) : DeclarationError := {
  kind
  declarationId := if declarationId.value == "" then
    DeclarationId.of "umpire.declaration.anonymous"
  else
    declarationId
  sourcePath := sourcePath source
  offendingValue
  relatedIdentities := canonicalIds relatedIdentities
}

private def validationError
    (kind : DeclarationErrorKind)
    (declarationId : DeclarationId)
    (source : SemanticSource)
    (path : AuthoringOccurrencePath)
    (occurrenceIdentity : DeclarationId)
    (offendingValue : String)
    (relatedIdentities : List DeclarationId := []) : TargetValidationError := {
  error := declarationError kind declarationId source offendingValue relatedIdentities
  path
  occurrenceIdentity
  source
}

private def firstDuplicateId : List DeclarationId → Option DeclarationId
  | first :: second :: rest =>
      if first == second then some first else firstDuplicateId (second :: rest)
  | _ => none

private def firstDuplicateDeclaration : List DeclarationMetadata → Option DeclarationMetadata
  | first :: second :: rest =>
      if first.id == second.id then some first else firstDuplicateDeclaration (second :: rest)
  | _ => none

private def requireIdentity
    (owner : DeclarationId)
    (source : SemanticSource)
    (id : DeclarationId)
    (path : AuthoringOccurrencePath) : Except TargetValidationError Unit :=
  if id.value == "" then
    .error (validationError .emptyIdentity owner source path id "<empty>" [id])
  else if !id.isNamespaced then
    .error (validationError .invalidIdentity owner source path id id.value [id])
  else
    .ok ()

private def requireUniqueIds
    (owner : DeclarationId)
    (source : SemanticSource)
    (path : AuthoringOccurrencePath)
    (ids : List DeclarationId) : Except TargetValidationError Unit :=
  match firstDuplicateId (ids.mergeSort idLe) with
  | some duplicate =>
      .error (validationError .duplicateIdentity owner source path duplicate
        duplicate.value [duplicate])
  | none => .ok ()

private def requireDeclaration
    (declarations : List DeclarationMetadata)
    (owner : DeclarationId)
    (source : SemanticSource)
    (id : DeclarationId)
    (expectedKind : DeclarationKind)
    (path : AuthoringOccurrencePath) : Except TargetValidationError Unit := do
  requireIdentity owner source id path
  match declarations.find? (fun declaration => declaration.id == id) with
  | none => throw (validationError .unknownIdentity owner source path id id.value [id])
  | some declaration =>
      if declaration.kind == expectedKind then
        pure ()
      else
        throw (validationError .wrongKind owner source path id
          (id.value ++ ": expected " ++ expectedKind.name ++ ", found " ++ declaration.kind.name)
          [id])

private def occurrencePath
    (role : AuthoringOccurrenceRole)
    (owner : DeclarationId) : AuthoringOccurrencePath :=
  { role, owner }

private def reconciliationOccurrencePath
    (role : AuthoringOccurrenceRole)
    (connector reconciliation : DeclarationId) : AuthoringOccurrencePath :=
  { role, owner := connector, context := .reconciliation reconciliation }

private def validateDeclarations
    (target : TargetDeclaration LawStatement Setup State Action Outcome Observation) :
    Except TargetValidationError (List DeclarationMetadata) := do
  let declarations := target.declarations.mergeSort declarationLe
  for declaration in declarations do
    requireIdentity declaration.id declaration.source declaration.id
      (occurrencePath .declarationMetadata declaration.id)
  match firstDuplicateDeclaration declarations with
  | some duplicate =>
      throw (validationError .duplicateIdentity duplicate.id duplicate.source
        (occurrencePath .declarationMetadata duplicate.id) duplicate.id duplicate.id.value
        [duplicate.id])
  | none => pure declarations

private def validateLawWitnesses
    (declarations : List DeclarationMetadata)
    (owner : DeclarationId)
    (source : SemanticSource)
    (requirements : List LawRequirement)
    (witnesses : List (LawWitness LawStatement)) : Except TargetValidationError Unit := do
  requireUniqueIds owner source (occurrencePath .lawRequirement owner)
    (requirements.map LawRequirement.id)
  requireUniqueIds owner source (occurrencePath .lawWitness owner)
    (witnesses.map (fun witness => witness.requirement.id))
  for requirement in requirements.mergeSort lawLe do
    requireDeclaration declarations owner source requirement.id .law
      (occurrencePath .lawRequirement owner)
    match declarations.find? (fun declaration => declaration.id == requirement.id) with
    | some declaration =>
        if declaration.contractDigest != requirement.semanticDigest then
          throw (validationError .lawContractMismatch owner source
            (occurrencePath .lawRequirement owner) requirement.id
            (requirement.id.value ++ ": expected " ++ declaration.contractDigest ++
              ", found " ++ requirement.semanticDigest)
            [requirement.id])
    | none => pure ()
    match witnesses.find? (fun witness => witness.requirement == requirement) with
    | none =>
        throw (validationError .missingLaw owner source (occurrencePath .lawRequirement owner)
          requirement.id requirement.id.value [requirement.id])
    | some _ => pure ()
  for witness in witnesses do
    requireDeclaration declarations owner source witness.requirement.id .law
      (occurrencePath .lawWitness owner)
    match requirements.find? (fun requirement => requirement == witness.requirement) with
    | none =>
        throw (validationError .unexpectedLaw owner source (occurrencePath .lawWitness owner)
          witness.requirement.id witness.requirement.id.value
          [witness.requirement.id])
    | some _ => pure ()

private def validateProvider
    (declarations : List DeclarationMetadata)
    (targetId : DeclarationId)
    (provider : CapabilityProvider LawStatement) : Except TargetValidationError Unit := do
  requireDeclaration declarations provider.id provider.source provider.id .provider
    (occurrencePath .providerDefinition targetId)
  requireDeclaration declarations provider.id provider.source provider.contract.id .capability
    (occurrencePath .capabilityRequirement provider.id)
  validateLawWitnesses declarations provider.id provider.source
    provider.contract.requiredLaws provider.lawWitnesses
  requireUniqueIds provider.id provider.source (occurrencePath .meaning provider.id)
    (provider.meanings.map MeaningProvision.declaration)
  for meaning in provider.meanings.mergeSort meaningLe do
    requireDeclaration declarations provider.id provider.source meaning.declaration meaning.kind
      (occurrencePath .meaning provider.id)

private def validateConnector
    (declarations : List DeclarationMetadata)
    (activeProviders : List DeclarationId)
    (targetId : DeclarationId)
    (connector : CapabilityConnector LawStatement) : Except TargetValidationError Unit := do
  requireDeclaration declarations connector.id connector.source connector.id .connector
    (occurrencePath .connectorDefinition targetId)
  validateLawWitnesses declarations connector.id connector.source
    connector.requiredLaws connector.lawWitnesses
  requireUniqueIds connector.id connector.source (occurrencePath .reconciliation connector.id)
    (connector.reconciliations.map Reconciliation.declaration)
  for reconciliation in connector.reconciliations.mergeSort reconciliationLe do
    requireDeclaration declarations connector.id connector.source
      reconciliation.declaration reconciliation.kind (occurrencePath .reconciliation connector.id)
    let providerPath := reconciliationOccurrencePath .providerReference connector.id
      reconciliation.declaration
    requireUniqueIds connector.id connector.source providerPath reconciliation.providers
    for provider in reconciliation.providers.mergeSort idLe do
      requireDeclaration declarations connector.id connector.source provider .provider providerPath
      if !activeProviders.contains provider then
        throw (validationError .missingProvider connector.id connector.source
          providerPath provider provider.value [provider])

private structure MeaningOwner where
  provider : DeclarationId
  meaning : MeaningProvision
  source : SemanticSource

private def distinctStrings (items : List String) : List String :=
  items.mergeSort |>.eraseDups

private def connectorMatches
    (connector : CapabilityConnector LawStatement)
    (declaration : DeclarationId)
    (providers : List DeclarationId) : Bool :=
  connector.reconciliations.any fun reconciliation =>
    reconciliation.declaration == declaration &&
      canonicalIds reconciliation.providers == canonicalIds providers

private def validateConflicts
    (providers : List (CapabilityProvider LawStatement))
    (connectors : List (CapabilityConnector LawStatement)) : Except TargetValidationError Unit := do
  let owners := providers.flatMap fun provider =>
    provider.meanings.map fun meaning => { provider := provider.id, meaning, source := provider.source }
  let declarations := canonicalIds (owners.map fun owner => owner.meaning.declaration)
  for declaration in declarations do
    let matching := owners.filter (fun owner => owner.meaning.declaration == declaration)
    let digests := distinctStrings (matching.map fun owner => owner.meaning.semanticDigest)
    if digests.length > 1 then
      let providerIds := canonicalIds (matching.map MeaningOwner.provider)
      let reconcilers := connectors.filter fun connector =>
        connectorMatches connector declaration providerIds
      match reconcilers.mergeSort connectorLe with
      | [] =>
          match matching with
          | first :: _ =>
              throw (validationError .conflictingProviders declaration first.source
                (occurrencePath .meaning first.provider) declaration declaration.value providerIds)
          | [] => pure ()
      | [_] => pure ()
      | connector :: rest =>
          throw (validationError .ambiguousConnector declaration connector.source
            (occurrencePath .reconciliation connector.id) declaration declaration.value
            (connector.id :: rest.map CapabilityConnector.id))

private def validateCapabilities
    (target : TargetDeclaration LawStatement Setup State Action Outcome Observation)
    (declarations : List DeclarationMetadata)
    (providers : List (CapabilityProvider LawStatement))
    (connectors : List (CapabilityConnector LawStatement)) : Except TargetValidationError Unit := do
  requireUniqueIds target.id target.source (occurrencePath .providerDefinition target.id)
    (providers.map CapabilityProvider.id)
  requireUniqueIds target.id target.source (occurrencePath .connectorDefinition target.id)
    (connectors.map CapabilityConnector.id)
  for provider in providers do
    validateProvider declarations target.id provider
  for connector in connectors do
    validateConnector declarations (providers.map CapabilityProvider.id) target.id connector
  requireUniqueIds target.id target.source (occurrencePath .capabilityRequirement target.id)
    target.requiredCapabilities
  for capability in canonicalIds target.requiredCapabilities do
    requireDeclaration declarations target.id target.source capability .capability
      (occurrencePath .capabilityRequirement target.id)
    if !(providers.any fun provider => provider.contract.id == capability) then
      throw (validationError .missingProvider target.id target.source
        (occurrencePath .capabilityRequirement target.id) capability capability.value [capability])
  validateConflicts providers connectors

private def composeTargetDetailed
    (target : TargetDeclaration LawStatement Setup State Action Outcome Observation) :
    Except TargetValidationError
      (CheckedTarget LawStatement Setup State Action Outcome Observation) := do
  let declarations ← validateDeclarations target
  requireDeclaration declarations target.id target.source target.id .target
    (occurrencePath .targetDeclaration target.id)
  let providers := target.providers.mergeSort providerLe
  let connectors := target.connectors.mergeSort connectorLe
  validateCapabilities target declarations providers connectors
  let kernel ← match target.kernel with
    | .checked kernel => pure kernel
    | .incomplete metadata missingProofs =>
        requireDeclaration declarations target.id target.source metadata.id .kernel
          (occurrencePath .kernel target.id)
        throw (validationError .incompleteKernel target.id metadata.source
          (occurrencePath .kernel target.id) metadata.id metadata.id.value missingProofs)
  requireDeclaration declarations target.id target.source kernel.metadata.id .kernel
    (occurrencePath .kernel target.id)
  let semantic := targetSemanticJson target.id declarations target.requiredCapabilities
    providers connectors kernel.metadata
  pure {
    id := target.id
    source := target.source
    declarations
    requiredCapabilities := canonicalIds target.requiredCapabilities
    providers
    connectors
    resolvedSetups := target.resolvedSetups
    kernel
    canonicalMetadata := targetMetadataJson target kernel.metadata
    semanticDigest := semanticDigestOf semantic
  }

/-- Check and canonicalize one target composition without relying on declaration or instance order. -/
def composeTarget
    (target : TargetDeclaration LawStatement Setup State Action Outcome Observation) :
    Except DeclarationError (CheckedTarget LawStatement Setup State Action Outcome Observation) :=
  match composeTargetDetailed target with
  | .ok checked => .ok checked
  | .error detailed => .error detailed.error

private def occurrenceIdLe (left right : AuthoringOccurrenceId) : Bool :=
  decide (left.sourcePath < right.sourcePath) ||
    (left.sourcePath == right.sourcePath && decide (left.line < right.line)) ||
    (left.sourcePath == right.sourcePath && left.line == right.line &&
      decide (left.column < right.column)) ||
    (left.sourcePath == right.sourcePath && left.line == right.line &&
      left.column == right.column && decide (left.endLine < right.endLine)) ||
    (left.sourcePath == right.sourcePath && left.line == right.line &&
      left.column == right.column && left.endLine == right.endLine &&
      decide (left.endColumn < right.endColumn)) ||
    (left.sourcePath == right.sourcePath && left.line == right.line &&
      left.column == right.column && left.endLine == right.endLine &&
      left.endColumn == right.endColumn && decide (left.localOrdinal ≤ right.localOrdinal))

private def occurrenceLe (left right : AuthoringOccurrence) : Bool :=
  occurrenceIdLe left.id right.id

private def fallbackOccurrenceId (source : SemanticSource) : AuthoringOccurrenceId := {
  sourcePath := sourcePath source
  line := source.line
  column := source.column
  endLine := source.line
  endColumn := source.column
  localOrdinal := 0
}

private def authoringDiagnostic
    (occurrences : List AuthoringOccurrence)
    (detailed : TargetValidationError) : AuthoringDiagnostic :=
  let matching := occurrences.filter (fun occurrence =>
    occurrence.declarationId == detailed.occurrenceIdentity && occurrence.path == detailed.path)
    |>.mergeSort occurrenceLe
  let fallback := fallbackOccurrenceId detailed.source
  if detailed.error.kind == .duplicateIdentity then
    match matching with
    | original :: offending :: _ => {
        error := detailed.error
        path := detailed.path
        original := some original.id
        offending := offending.id
      }
    | [offending] => {
        error := detailed.error
        path := detailed.path
        original := none
        offending := offending.id
      }
    | [] => {
        error := detailed.error
        path := detailed.path
        original := none
        offending := fallback
      }
  else
    match matching with
    | offending :: _ => {
        error := detailed.error
        path := detailed.path
        original := none
        offending := offending.id
      }
    | [] => {
        error := detailed.error
        path := detailed.path
        original := none
        offending := fallback
      }

/-- Ordinary Target authoring returns one checked semantic value or one located typed diagnostic. -/
def checkTarget
    (authored : AuthoredTarget LawStatement Setup State Action Outcome Observation) :
    Except AuthoringDiagnostic
      (CheckedTarget LawStatement Setup State Action Outcome Observation) :=
  match composeTargetDetailed authored.declaration with
  | .ok checked =>
      match authored.planning with
      | .unavailable => .ok checked
      | .available kernel _ capability =>
          .ok { checked with kernel, planning := .available capability }
  | .error detailed => .error (authoringDiagnostic authored.occurrences detailed)

/-- Produce a checked authored target directly while keeping extraction and proof-relation
re-ascription inside the Target boundary. Invalid declarations should use `checkTarget` or
`elaborateTarget` when their typed diagnostic is needed. -/
def checkedTarget
    (authored : AuthoredTarget LawStatement Setup State Action Outcome Observation)
    (valid : (checkTarget authored).toOption.isSome = true := by native_decide) :
    CheckedTarget LawStatement Setup State Action Outcome Observation :=
  let checked := (checkTarget authored).toOption.get valid
  match authored.planning with
  | .unavailable => checked
  | .available kernel _ capability => {
      checked with
      kernel
      planning := .available capability
    }

/-- Rebind implementation enumerators while proving the checked semantic kernel is unchanged. -/
def CheckedTarget.withEquivalentKernel
    (target : CheckedTarget LawStatement Setup State Action Outcome Observation)
    (kernel : TransitionKernel Setup State Action Outcome Observation)
    (_metadata : kernel.metadata = target.kernel.metadata)
    (_initial : kernel.authoritativeInitial = target.kernel.authoritativeInitial)
    (_step : kernel.authoritativeStep = target.kernel.authoritativeStep)
    (planning : FinitePlanningAvailability kernel.authoritativeStep := .unavailable) :
    CheckedTarget LawStatement Setup State Action Outcome Observation := {
  target with
  kernel
  planning
}

/-- Capture one syntax occurrence as a nonsemantic source-span/ordinal token. -/
def captureAuthoringOccurrence
    (reference : Lean.Syntax)
    (declarationId : DeclarationId)
    (path : AuthoringOccurrencePath)
    (localOrdinal : Nat) : Lean.Elab.Term.TermElabM CapturedAuthoringOccurrence := do
  let fileMap ← Lean.getFileMap
  let sourcePath ← Lean.getFileName
  let startOffset := reference.getPos?.getD 0
  let endOffset := reference.getTailPos?.getD startOffset
  let startPosition := fileMap.toPosition startOffset
  let endPosition := fileMap.toPosition endOffset
  pure {
    occurrence := {
      id := {
        sourcePath
        line := startPosition.line
        column := startPosition.column
        endLine := endPosition.line
        endColumn := endPosition.column
        localOrdinal
      }
      declarationId
      path
    }
    reference
  }

/-- Run the ordinary adapter once and emit its typed failure at the selected captured occurrence. -/
def elaborateTarget
    (authored : AuthoredTarget LawStatement Setup State Action Outcome Observation)
    (captured : List CapturedAuthoringOccurrence) :
    Lean.Elab.Term.TermElabM
      (CheckedTarget LawStatement Setup State Action Outcome Observation) := do
  let authored := authored.withOccurrences (captured.map CapturedAuthoringOccurrence.occurrence)
  match checkTarget authored with
  | .ok checked => pure checked
  | .error diagnostic =>
      let message := s!"target authoring failed: {canonicalAuthoringDiagnosticJson diagnostic}"
      match captured.find? (fun item => item.occurrence.id == diagnostic.offending) with
      | some item => Lean.throwErrorAt item.reference message
      | none => Lean.throwError message

def canonicalCheckedTargetJson
    (target : CheckedTarget LawStatement Setup State Action Outcome Observation) : String :=
  target.canonicalMetadata

end Umpire
