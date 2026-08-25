import Lean.Data.Json
import Std

namespace Umpire

/-! The common, pure semantic substrate shared by the Umpire authoring languages. -/

structure DeclarationId where
  value : String
  deriving BEq, DecidableEq, Hashable, Ord, Repr

namespace DeclarationId

def of (value : String) : DeclarationId := ⟨value⟩

private def isIdentifierCharacter (character : Char) : Bool :=
  character.isAlphanum || character == '-' || character == '_'

private def isNamespaceSegment (segment : String) : Bool :=
  segment != "" && segment.toList.all isIdentifierCharacter

def isNamespaced (id : DeclarationId) : Bool :=
  let segments := id.value.splitOn "."
  segments.length > 1 && segments.all isNamespaceSegment

end DeclarationId

inductive DeclarationKind where
  | state
  | action
  | outcome
  | observation
  | relation
  | capability
  | provider
  | law
  | connector
  | target
  | kernel
  deriving BEq, DecidableEq, Ord, Repr

def DeclarationKind.name : DeclarationKind → String
  | .state => "state"
  | .action => "action"
  | .outcome => "outcome"
  | .observation => "observation"
  | .relation => "relation"
  | .capability => "capability"
  | .provider => "provider"
  | .law => "law"
  | .connector => "connector"
  | .target => "target"
  | .kernel => "kernel"

structure SemanticSource where
  path : String
  line : Nat := 0
  column : Nat := 0
  provenance : String := "authored"
  deriving BEq, DecidableEq, Repr

structure DeclarationMetadata where
  id : DeclarationId
  kind : DeclarationKind
  source : SemanticSource
  version : Nat := 1
  contractDigest : String
  documentation : String := ""
  deriving BEq, DecidableEq, Repr

inductive BoundUnit where
  | semanticTransitions
  | selectedActions
  | observationPositions
  | logicalTime
  deriving BEq, DecidableEq, Ord, Repr

def BoundUnit.name : BoundUnit → String
  | .semanticTransitions => "semantic-transitions"
  | .selectedActions => "selected-actions"
  | .observationPositions => "observation-positions"
  | .logicalTime => "logical-time"

structure TypedBound where
  value : Nat
  unit : BoundUnit
  deriving BEq, DecidableEq, Ord, Repr

structure SemanticValue where
  identity : DeclarationId
  value : String
  deriving BEq, DecidableEq, Ord, Repr

structure SemanticTraceStep (State Action Outcome Observation : Type) where
  selectedAction : Action
  modelOutcome : Outcome
  resultingState : State
  observations : List Observation
  deriving BEq, DecidableEq, Repr

/-- Pure model data only. Execution evidence and qualification are deliberately absent. -/
structure SemanticTrace (State Action Outcome Observation : Type) where
  initialState : State
  steps : List (SemanticTraceStep State Action Outcome Observation)
  deriving BEq, DecidableEq, Repr

structure TransitionResult (State Outcome Observation : Type) where
  modelOutcome : Outcome
  resultingState : State
  observations : List Observation
  deriving BEq, DecidableEq, Repr

structure KernelMetadata where
  id : DeclarationId
  version : Nat := 1
  contractDigest : String
  source : SemanticSource
  deriving BEq, DecidableEq, Repr

/--
The target-owned finite transition kernel. The proof fields make every emitted initial state and
step sound, and make each authoritative relation complete with respect to the finite enumerators.
-/
structure TransitionKernel (Setup State Action Outcome Observation : Type) where
  metadata : KernelMetadata
  initialStates : Setup → List State
  authoritativeInitial : Setup → State → Prop
  initialSound : ∀ setup state, state ∈ initialStates setup → authoritativeInitial setup state
  initialComplete : ∀ setup state, authoritativeInitial setup state → state ∈ initialStates setup
  steps : State → Action → List (TransitionResult State Outcome Observation)
  authoritativeStep :
    State → Action → TransitionResult State Outcome Observation → Prop
  stepSound : ∀ state action result,
    result ∈ steps state action → authoritativeStep state action result
  stepComplete : ∀ state action result,
    authoritativeStep state action result → result ∈ steps state action

/-- Missing proof obligations are representable only before target composition. -/
inductive KernelAvailability (Setup State Action Outcome Observation : Type) where
  | checked (kernel : TransitionKernel Setup State Action Outcome Observation)
  | incomplete (metadata : KernelMetadata) (missingProofs : List DeclarationId)

structure LawRequirement where
  id : DeclarationId
  semanticDigest : String
  deriving BEq, DecidableEq, Ord, Repr

/-- A law witness retains portable identity while proving the target's authoritative proposition. -/
structure LawWitness (LawStatement : DeclarationId → Prop) where
  requirement : LawRequirement
  proof : LawStatement requirement.id

structure CapabilityContract where
  id : DeclarationId
  version : Nat := 1
  semanticDigest : String
  requiredLaws : List LawRequirement
  deriving BEq, DecidableEq, Repr

structure MeaningProvision where
  declaration : DeclarationId
  kind : DeclarationKind
  semanticDigest : String
  deriving BEq, DecidableEq, Repr

structure CapabilityProvider (LawStatement : DeclarationId → Prop) where
  id : DeclarationId
  source : SemanticSource
  contract : CapabilityContract
  meanings : List MeaningProvision
  lawWitnesses : List (LawWitness LawStatement)

structure Reconciliation where
  declaration : DeclarationId
  kind : DeclarationKind
  providers : List DeclarationId
  semanticDigest : String
  deriving BEq, DecidableEq, Repr

structure CapabilityConnector (LawStatement : DeclarationId → Prop) where
  id : DeclarationId
  source : SemanticSource
  version : Nat := 1
  semanticDigest : String
  reconciliations : List Reconciliation
  requiredLaws : List LawRequirement
  lawWitnesses : List (LawWitness LawStatement)

inductive DeclarationErrorKind where
  | emptyIdentity
  | invalidIdentity
  | duplicateIdentity
  | unknownIdentity
  | wrongKind
  | missingLaw
  | unexpectedLaw
  | lawContractMismatch
  | missingProvider
  | conflictingProviders
  | ambiguousConnector
  | incompleteKernel
  deriving BEq, DecidableEq, Ord, Repr

def DeclarationErrorKind.name : DeclarationErrorKind → String
  | .emptyIdentity => "empty-identity"
  | .invalidIdentity => "invalid-identity"
  | .duplicateIdentity => "duplicate-identity"
  | .unknownIdentity => "unknown-identity"
  | .wrongKind => "wrong-kind"
  | .missingLaw => "missing-law"
  | .unexpectedLaw => "unexpected-law"
  | .lawContractMismatch => "law-contract-mismatch"
  | .missingProvider => "missing-provider"
  | .conflictingProviders => "conflicting-providers"
  | .ambiguousConnector => "ambiguous-connector"
  | .incompleteKernel => "incomplete-kernel"

structure DeclarationError where
  kind : DeclarationErrorKind
  declarationId : DeclarationId
  sourcePath : String
  offendingValue : String
  relatedIdentities : List DeclarationId
  deriving BEq, DecidableEq, Repr

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

structure CheckedTarget
    (LawStatement : DeclarationId → Prop)
    (Setup State Action Outcome Observation : Type) where
  id : DeclarationId
  source : SemanticSource
  declarations : List DeclarationMetadata
  requiredCapabilities : List DeclarationId
  providers : List (CapabilityProvider LawStatement)
  connectors : List (CapabilityConnector LawStatement)
  resolvedSetups : List Setup
  kernel : TransitionKernel Setup State Action Outcome Observation
  canonicalMetadata : String
  semanticDigest : String

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

def semanticDigestOf (canonicalSemanticValue : String) : String :=
  "umpire-semantic/v1:" ++ canonicalSemanticValue

def canonicalTypedBoundJson (bound : TypedBound) : String :=
  "{\"value\":" ++ toString bound.value ++ ",\"unit\":" ++ quote bound.unit.name ++ "}"

private def targetMetadataJson
    (target : TargetDeclaration LawStatement Setup State Action Outcome Observation)
    (kernel : KernelMetadata) : String :=
  "{\"semantic\":" ++ targetSemanticJson target.id target.declarations
      target.requiredCapabilities target.providers target.connectors kernel ++
    ",\"source\":" ++ sourceJson target.source ++
    ",\"declarationMetadata\":" ++
      array (target.declarations.mergeSort declarationLe |>.map canonicalDeclarationMetadataJson) ++
    ",\"kernelMetadata\":" ++ canonicalKernelMetadataJson kernel ++ "}"

private def error
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
    (id : DeclarationId) : Except DeclarationError Unit :=
  if id.value == "" then
    .error (error .emptyIdentity owner source "<empty>" [id])
  else if !id.isNamespaced then
    .error (error .invalidIdentity owner source id.value [id])
  else
    .ok ()

private def requireUniqueIds
    (owner : DeclarationId)
    (source : SemanticSource)
    (ids : List DeclarationId) : Except DeclarationError Unit :=
  match firstDuplicateId (ids.mergeSort idLe) with
  | some duplicate => .error (error .duplicateIdentity owner source duplicate.value [duplicate])
  | none => .ok ()

private def requireDeclaration
    (declarations : List DeclarationMetadata)
    (owner : DeclarationId)
    (source : SemanticSource)
    (id : DeclarationId)
    (expectedKind : DeclarationKind) : Except DeclarationError Unit := do
  requireIdentity owner source id
  match declarations.find? (fun declaration => declaration.id == id) with
  | none => throw (error .unknownIdentity owner source id.value [id])
  | some declaration =>
      if declaration.kind == expectedKind then
        pure ()
      else
        throw (error .wrongKind owner source
          (id.value ++ ": expected " ++ expectedKind.name ++ ", found " ++ declaration.kind.name)
          [id])

private def validateDeclarations
    (target : TargetDeclaration LawStatement Setup State Action Outcome Observation) :
    Except DeclarationError (List DeclarationMetadata) := do
  let declarations := target.declarations.mergeSort declarationLe
  for declaration in declarations do
    requireIdentity declaration.id declaration.source declaration.id
  match firstDuplicateDeclaration declarations with
  | some duplicate =>
      throw (error .duplicateIdentity duplicate.id duplicate.source duplicate.id.value [duplicate.id])
  | none => pure declarations

private def validateLawWitnesses
    (declarations : List DeclarationMetadata)
    (owner : DeclarationId)
    (source : SemanticSource)
    (requirements : List LawRequirement)
    (witnesses : List (LawWitness LawStatement)) : Except DeclarationError Unit := do
  requireUniqueIds owner source (requirements.map LawRequirement.id)
  requireUniqueIds owner source (witnesses.map (fun witness => witness.requirement.id))
  for requirement in requirements.mergeSort lawLe do
    requireDeclaration declarations owner source requirement.id .law
    match declarations.find? (fun declaration => declaration.id == requirement.id) with
    | some declaration =>
        if declaration.contractDigest != requirement.semanticDigest then
          throw (error .lawContractMismatch owner source
            (requirement.id.value ++ ": expected " ++ declaration.contractDigest ++
              ", found " ++ requirement.semanticDigest)
            [requirement.id])
    | none => pure ()
    match witnesses.find? (fun witness => witness.requirement == requirement) with
    | none => throw (error .missingLaw owner source requirement.id.value [requirement.id])
    | some _ => pure ()
  for witness in witnesses do
    requireDeclaration declarations owner source witness.requirement.id .law
    match requirements.find? (fun requirement => requirement == witness.requirement) with
    | none =>
        throw (error .unexpectedLaw owner source witness.requirement.id.value
          [witness.requirement.id])
    | some _ => pure ()

private def validateProvider
    (declarations : List DeclarationMetadata)
    (provider : CapabilityProvider LawStatement) : Except DeclarationError Unit := do
  requireDeclaration declarations provider.id provider.source provider.id .provider
  requireDeclaration declarations provider.id provider.source provider.contract.id .capability
  validateLawWitnesses declarations provider.id provider.source
    provider.contract.requiredLaws provider.lawWitnesses
  requireUniqueIds provider.id provider.source (provider.meanings.map MeaningProvision.declaration)
  for meaning in provider.meanings.mergeSort meaningLe do
    requireDeclaration declarations provider.id provider.source meaning.declaration meaning.kind

private def validateConnector
    (declarations : List DeclarationMetadata)
    (activeProviders : List DeclarationId)
    (connector : CapabilityConnector LawStatement) : Except DeclarationError Unit := do
  requireDeclaration declarations connector.id connector.source connector.id .connector
  validateLawWitnesses declarations connector.id connector.source
    connector.requiredLaws connector.lawWitnesses
  requireUniqueIds connector.id connector.source
    (connector.reconciliations.map Reconciliation.declaration)
  for reconciliation in connector.reconciliations.mergeSort reconciliationLe do
    requireDeclaration declarations connector.id connector.source
      reconciliation.declaration reconciliation.kind
    requireUniqueIds connector.id connector.source reconciliation.providers
    for provider in reconciliation.providers.mergeSort idLe do
      requireDeclaration declarations connector.id connector.source provider .provider
      if !activeProviders.contains provider then
        throw (error .missingProvider connector.id connector.source provider.value [provider])

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
    (connectors : List (CapabilityConnector LawStatement)) : Except DeclarationError Unit := do
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
              throw (error .conflictingProviders declaration first.source declaration.value providerIds)
          | [] => pure ()
      | [_] => pure ()
      | connector :: rest =>
          throw (error .ambiguousConnector declaration connector.source declaration.value
            (connector.id :: rest.map CapabilityConnector.id))

private def validateCapabilities
    (target : TargetDeclaration LawStatement Setup State Action Outcome Observation)
    (declarations : List DeclarationMetadata)
    (providers : List (CapabilityProvider LawStatement))
    (connectors : List (CapabilityConnector LawStatement)) : Except DeclarationError Unit := do
  requireUniqueIds target.id target.source (providers.map CapabilityProvider.id)
  requireUniqueIds target.id target.source (connectors.map CapabilityConnector.id)
  for provider in providers do
    validateProvider declarations provider
  for connector in connectors do
    validateConnector declarations (providers.map CapabilityProvider.id) connector
  requireUniqueIds target.id target.source target.requiredCapabilities
  for capability in canonicalIds target.requiredCapabilities do
    requireDeclaration declarations target.id target.source capability .capability
    if !(providers.any fun provider => provider.contract.id == capability) then
      throw (error .missingProvider target.id target.source capability.value [capability])
  validateConflicts providers connectors

/-- Check and canonicalize one target composition without relying on declaration or instance order. -/
def composeTarget
    (target : TargetDeclaration LawStatement Setup State Action Outcome Observation) :
    Except DeclarationError (CheckedTarget LawStatement Setup State Action Outcome Observation) := do
  let declarations ← validateDeclarations target
  requireDeclaration declarations target.id target.source target.id .target
  let providers := target.providers.mergeSort providerLe
  let connectors := target.connectors.mergeSort connectorLe
  validateCapabilities target declarations providers connectors
  let kernel ← match target.kernel with
    | .checked kernel => pure kernel
    | .incomplete metadata missingProofs =>
        requireDeclaration declarations target.id target.source metadata.id .kernel
        throw (error .incompleteKernel target.id metadata.source metadata.id.value missingProofs)
  requireDeclaration declarations target.id target.source kernel.metadata.id .kernel
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

def canonicalCheckedTargetJson
    (target : CheckedTarget LawStatement Setup State Action Outcome Observation) : String :=
  target.canonicalMetadata

end Umpire
