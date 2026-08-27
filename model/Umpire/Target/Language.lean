import Lean.Data.Json
import Lean.Elab.Term
import Umpire.Core

/-! Target authoring, checked composition, and canonical target projections. -/

namespace Umpire

structure TargetDeclaration
    (LawStatement : LawDefinition → Prop)
    (Setup State Action Outcome Observation : Type) where
  id : DefinitionId
  source : SourceLocation
  definitions : List DefinitionMetadata
  requiredCapabilities : List DefinitionId
  providers : List (CapabilityProvider LawStatement)
  connectors : List (CapabilityConnector LawStatement)
  resolvedSetups : List Setup
  kernel : KernelAvailability Setup State Action Outcome Observation

/-- Optional finite planning is tied propositionally to the exact authoritative target kernel. -/
structure FinitePlanningCapability
    {State Action Outcome Observation : Type}
    (authoritativeStep : State → Action → TransitionResult State Outcome Observation → Prop) where
  actions : List Action
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

structure TargetInitialStateRow where
  setup : String
  state : String
  deriving BEq, DecidableEq, Ord, Repr

structure TargetTransitionRow where
  state : String
  action : String
  modelOutcome : String
  resultingState : String
  observations : List String
  deriving BEq, DecidableEq, Ord, Repr

/-- Canonical executable behavior evaluated over the complete finite Target Behavior Domain. -/
structure TargetBehaviorDescription where
  setups : List String
  states : List String
  actions : List String
  outcomes : List String
  observations : List String
  initialStates : List TargetInitialStateRow
  transitions : List TargetTransitionRow
  deriving BEq, DecidableEq, Repr

structure CheckedTarget
    (LawStatement : LawDefinition → Prop)
    (Setup State Action Outcome Observation : Type) where
  private mk ::
  id : DefinitionId
  source : SourceLocation
  definitions : List DefinitionMetadata
  requiredCapabilities : List DefinitionId
  providers : List (CapabilityProvider LawStatement)
  connectors : List (CapabilityConnector LawStatement)
  resolvedSetups : List Setup
  kernel : TransitionKernel Setup State Action Outcome Observation
  behaviorDescription : TargetBehaviorDescription
  planning : FinitePlanningAvailability kernel.authoritativeStep := .unavailable
  canonicalMetadata : String
  behaviorFingerprint : BehaviorFingerprint

/-- Closed authoring roles keep compiler locations separate from Model Definition IDs. -/
inductive AuthoringOccurrenceRole where
  | definitionMetadata
  | targetDefinition
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
  | .definitionMetadata => "definition-metadata"
  | .targetDefinition => "target-definition"
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
  | reconciliation (definitionId : DefinitionId)
  deriving BEq, DecidableEq, Repr

structure AuthoringOccurrencePath where
  role : AuthoringOccurrenceRole
  owner : DefinitionId
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
  definitionId : DefinitionId
  path : AuthoringOccurrencePath
  deriving BEq, DecidableEq, Repr

/-- Compiler-only syntax is paired with the pure occurrence row and never enters checked data. -/
structure CapturedAuthoringOccurrence where
  occurrence : AuthoringOccurrence
  reference : Lean.Syntax

structure AuthoringDiagnostic where
  error : DefinitionError
  path : AuthoringOccurrencePath
  original : Option AuthoringOccurrenceId
  offending : AuthoringOccurrenceId
  deriving BEq, DecidableEq, Repr

/-- Ordinary target input keeps semantic definitions explicit without exposing checked fields. -/
structure TargetDefinition
    (Setup State Action Outcome Observation : Type) where
  id : DefinitionId
  source : SourceLocation
  definitions : List DefinitionMetadata
  requiredCapabilities : List DefinitionId
  resolvedSetups : List Setup
  kernel : KernelAvailability Setup State Action Outcome Observation

private structure TargetCompositionPayload (LawStatement : LawDefinition → Prop) where
  providers : List (CapabilityProvider LawStatement)
  connectors : List (CapabilityConnector LawStatement)

/-- Explicit provider and connector choices whose collection is owned by Target. -/
structure TargetComposition (LawStatement : LawDefinition → Prop) where
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
    (LawStatement : LawDefinition → Prop)
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
    definitions := definition.definitions
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
  error : DefinitionError
  path : AuthoringOccurrencePath
  occurrenceDefinitionId : DefinitionId
  source : SourceLocation

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def canonicalStrings (values : List String) : List String :=
  values.mergeSort |>.eraseDups

private def initialRowLe (left right : TargetInitialStateRow) : Bool :=
  decide (left.setup < right.setup) ||
    (left.setup == right.setup && decide (left.state ≤ right.state))

private def transitionRowLe (left right : TargetTransitionRow) : Bool :=
  compare left right != .gt

private def invalidBehaviorDomainEncoding?
    (domain : TargetBehaviorDomain setupDomain stateDomain actionDomain outcomeDomain
      observationDomain initialStates steps) : Option String :=
  if domain.setups.length != (canonicalStrings (domain.setups.map domain.encodeSetup)).length then
    some "setup-encoding"
  else if domain.states.length != (canonicalStrings (domain.states.map domain.encodeState)).length then
    some "state-encoding"
  else if domain.actions.length != (canonicalStrings (domain.actions.map domain.encodeAction)).length then
    some "action-encoding"
  else if domain.outcomes.length != (canonicalStrings (domain.outcomes.map domain.encodeOutcome)).length then
    some "outcome-encoding"
  else if domain.observations.length !=
      (canonicalStrings (domain.observations.map domain.encodeObservation)).length then
    some "observation-encoding"
  else
    none

def TransitionKernel.describeBehavior
    (kernel : TransitionKernel Setup State Action Outcome Observation)
    (domain : TargetBehaviorDomain kernel.setupDomain kernel.stateDomain kernel.actionDomain
      kernel.outcomeDomain kernel.observationDomain kernel.initialStates kernel.steps) :
    TargetBehaviorDescription :=
  let initialStates := domain.setups.flatMap fun setup =>
    (kernel.initialStates setup).map fun state => {
      setup := domain.encodeSetup setup
      state := domain.encodeState state
    }
  let transitions := domain.states.flatMap fun state =>
    domain.actions.flatMap fun action =>
      (kernel.steps state action).map fun result => {
        state := domain.encodeState state
        action := domain.encodeAction action
        modelOutcome := domain.encodeOutcome result.modelOutcome
        resultingState := domain.encodeState result.resultingState
        observations := result.observations.map domain.encodeObservation
      }
  {
    setups := canonicalStrings (domain.setups.map domain.encodeSetup)
    states := canonicalStrings (domain.states.map domain.encodeState)
    actions := canonicalStrings (domain.actions.map domain.encodeAction)
    outcomes := canonicalStrings (domain.outcomes.map domain.encodeOutcome)
    observations := canonicalStrings (domain.observations.map domain.encodeObservation)
    initialStates := initialStates.eraseDups |>.mergeSort initialRowLe
    transitions := transitions.eraseDups |>.mergeSort transitionRowLe
  }

/-- Project the canonical behavior sealed by a complete finite kernel domain. -/
def TransitionKernel.behaviorDescription?
    (kernel : TransitionKernel Setup State Action Outcome Observation) :
    Option TargetBehaviorDescription :=
  match kernel.behaviorDomain with
  | .complete domain => some (kernel.describeBehavior domain)
  | .missing => none
  | .incomplete _ => none

private def withoutClosingBrace (value : String) : String :=
  (value.dropEnd 1).toString

private def idLe (left right : DefinitionId) : Bool :=
  decide (left.value ≤ right.value)

private def sourceLe (left right : SourceLocation) : Bool :=
  decide (left.path < right.path) ||
    (left.path == right.path && decide (left.line < right.line)) ||
    (left.path == right.path && left.line == right.line && decide (left.column ≤ right.column))

private def definitionLe (left right : DefinitionMetadata) : Bool :=
  decide (left.id.value < right.id.value) ||
    (left.id == right.id && decide (left.kind.name < right.kind.name)) ||
    (left.id == right.id && left.kind == right.kind && sourceLe left.source right.source)

private def providerLe (left right : CapabilityProvider LawStatement) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def connectorLe (left right : CapabilityConnector LawStatement) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def meaningLe (left right : MeaningProvision) : Bool :=
  decide (left.definitionId.value < right.definitionId.value) ||
    (left.definitionId == right.definitionId &&
      decide (left.canonicalBehavior ≤ right.canonicalBehavior))

private def reconciliationLe (left right : Reconciliation) : Bool :=
  decide (left.definitionId.value < right.definitionId.value) ||
    (left.definitionId == right.definitionId &&
      decide (left.canonicalBehavior ≤ right.canonicalBehavior))

private def lawLe (left right : LawDefinition) : Bool :=
  decide (left.id.value < right.id.value) ||
    (left.id == right.id && decide (left.body ≤ right.body))

private def canonicalIds (ids : List DefinitionId) : List DefinitionId :=
  ids.mergeSort idLe |>.eraseDups

private def sourcePath (source : SourceLocation) : String :=
  if source.path == "" then "<unknown>" else source.path

private def sourceJson (source : SourceLocation) : String :=
  "{\"path\":" ++ quote source.path ++
    ",\"line\":" ++ toString source.line ++
    ",\"column\":" ++ toString source.column ++
    ",\"provenance\":" ++ quote source.provenance ++ "}"

private def lawJson (law : LawDefinition) : String :=
  "{\"id\":" ++ quote law.id.value ++
    ",\"body\":" ++ quote law.body ++ "}"

private def meaningJson (meaning : MeaningProvision) : String :=
  "{\"id\":" ++ quote meaning.definitionId.value ++
    ",\"kind\":" ++ quote meaning.kind.name ++
    ",\"canonicalBehavior\":" ++ quote meaning.canonicalBehavior ++ "}"

private def definitionSemanticJson (declaration : DefinitionMetadata) : String :=
  "{\"id\":" ++ quote declaration.id.value ++
    ",\"kind\":" ++ quote declaration.kind.name ++
    ",\"version\":" ++ toString declaration.version ++
    ",\"canonicalBehavior\":" ++ quote declaration.canonicalBehavior ++ "}"

def canonicalDefinitionMetadataJson (declaration : DefinitionMetadata) : String :=
  withoutClosingBrace (definitionSemanticJson declaration) ++
    ",\"source\":" ++ sourceJson declaration.source ++
    ",\"documentation\":" ++ quote declaration.documentation ++ "}"

private def providerSemanticJson (provider : CapabilityProvider LawStatement) : String :=
  let laws := provider.contract.requiredLaws.mergeSort lawLe
  "{\"id\":" ++ quote provider.id.value ++
    ",\"capabilityId\":" ++ quote provider.contract.id.value ++
    ",\"capabilityVersion\":" ++ toString provider.contract.version ++
    ",\"canonicalBehavior\":" ++ quote provider.contract.canonicalBehavior ++
    ",\"meanings\":" ++ array (provider.meanings.mergeSort meaningLe |>.map meaningJson) ++
    ",\"laws\":" ++ array (laws.map lawJson) ++ "}"

def canonicalCapabilityProviderJson (provider : CapabilityProvider LawStatement) : String :=
  withoutClosingBrace (providerSemanticJson provider) ++
    ",\"source\":" ++ sourceJson provider.source ++ "}"

private def reconciliationJson (reconciliation : Reconciliation) : String :=
  "{\"id\":" ++ quote reconciliation.definitionId.value ++
    ",\"kind\":" ++ quote reconciliation.kind.name ++
    ",\"providers\":" ++ array (canonicalIds reconciliation.providers |>.map (quote ∘ DefinitionId.value)) ++
    ",\"canonicalBehavior\":" ++ quote reconciliation.canonicalBehavior ++ "}"

private def connectorSemanticJson (connector : CapabilityConnector LawStatement) : String :=
  let laws := connector.requiredLaws.mergeSort lawLe
  "{\"id\":" ++ quote connector.id.value ++
    ",\"version\":" ++ toString connector.version ++
    ",\"canonicalBehavior\":" ++ quote connector.canonicalBehavior ++
    ",\"reconciliations\":" ++
      array (connector.reconciliations.mergeSort reconciliationLe |>.map reconciliationJson) ++
    ",\"laws\":" ++ array (laws.map lawJson) ++ "}"

def canonicalCapabilityConnectorJson (connector : CapabilityConnector LawStatement) : String :=
  withoutClosingBrace (connectorSemanticJson connector) ++
    ",\"source\":" ++ sourceJson connector.source ++ "}"

private def kernelSemanticJson (metadata : KernelMetadata) : String :=
  "{\"id\":" ++ quote metadata.id.value ++
    ",\"version\":" ++ toString metadata.version ++ "}"

def canonicalKernelMetadataJson (metadata : KernelMetadata) : String :=
  withoutClosingBrace (kernelSemanticJson metadata) ++
    ",\"source\":" ++ sourceJson metadata.source ++ "}"

private def initialStateRowJson (row : TargetInitialStateRow) : String :=
  "{\"setup\":" ++ quote row.setup ++ ",\"state\":" ++ quote row.state ++ "}"

private def transitionRowJson (row : TargetTransitionRow) : String :=
  "{\"state\":" ++ quote row.state ++
    ",\"action\":" ++ quote row.action ++
    ",\"modelOutcome\":" ++ quote row.modelOutcome ++
    ",\"resultingState\":" ++ quote row.resultingState ++
    ",\"observations\":" ++ array (row.observations.map quote) ++ "}"

private def targetBehaviorDescriptionJson (description : TargetBehaviorDescription) : String :=
  "{\"domains\":{\"setups\":" ++ array (description.setups.map quote) ++
    ",\"states\":" ++ array (description.states.map quote) ++
    ",\"actions\":" ++ array (description.actions.map quote) ++
    ",\"outcomes\":" ++ array (description.outcomes.map quote) ++
    ",\"observations\":" ++ array (description.observations.map quote) ++ "}" ++
    ",\"initialStates\":" ++ array (description.initialStates.map initialStateRowJson) ++
    ",\"transitions\":" ++ array (description.transitions.map transitionRowJson) ++ "}"

def canonicalDefinitionErrorJson (error : DefinitionError) : String :=
  "{\"kind\":" ++ quote error.kind.name ++
    ",\"definitionId\":" ++ quote error.definitionId.value ++
    ",\"sourcePath\":" ++ quote error.sourcePath ++
    ",\"offendingValue\":" ++ quote error.offendingValue ++
    ",\"relatedDefinitionIds\":" ++
      array (canonicalIds error.relatedDefinitionIds |>.map (quote ∘ DefinitionId.value)) ++ "}"

private def authoringOccurrenceIdJson (id : AuthoringOccurrenceId) : String :=
  "{\"sourcePath\":" ++ quote id.sourcePath ++
    ",\"line\":" ++ toString id.line ++
    ",\"column\":" ++ toString id.column ++
    ",\"endLine\":" ++ toString id.endLine ++
    ",\"endColumn\":" ++ toString id.endColumn ++
    ",\"localOrdinal\":" ++ toString id.localOrdinal ++ "}"

private def authoringOccurrenceContextJson : AuthoringOccurrenceContext → String
  | .direct => quote "direct"
  | .reconciliation definitionId =>
      "{\"reconciliation\":" ++ quote definitionId.value ++ "}"

private def authoringOccurrencePathJson (path : AuthoringOccurrencePath) : String :=
  "{\"role\":" ++ quote path.role.name ++
    ",\"owner\":" ++ quote path.owner.value ++
    ",\"context\":" ++ authoringOccurrenceContextJson path.context ++ "}"

def canonicalAuthoringDiagnosticJson (diagnostic : AuthoringDiagnostic) : String :=
  "{\"error\":" ++ canonicalDefinitionErrorJson diagnostic.error ++
    ",\"original\":" ++
      (diagnostic.original.map authoringOccurrenceIdJson |>.getD "null") ++
    ",\"offending\":" ++ authoringOccurrenceIdJson diagnostic.offending ++
    ",\"path\":" ++ authoringOccurrencePathJson diagnostic.path ++ "}"

private def targetSemanticJson
    (id : DefinitionId)
    (definitions : List DefinitionMetadata)
    (requiredCapabilities : List DefinitionId)
    (providers : List (CapabilityProvider LawStatement))
    (connectors : List (CapabilityConnector LawStatement))
    (kernel : KernelMetadata)
    (behavior : TargetBehaviorDescription) : String :=
  "{\"id\":" ++ quote id.value ++
    ",\"declarations\":" ++
      array (definitions.mergeSort definitionLe |>.map definitionSemanticJson) ++
    ",\"requiredCapabilities\":" ++
      array (canonicalIds requiredCapabilities |>.map (quote ∘ DefinitionId.value)) ++
    ",\"providers\":" ++ array (providers.mergeSort providerLe |>.map providerSemanticJson) ++
    ",\"connectors\":" ++ array (connectors.mergeSort connectorLe |>.map connectorSemanticJson) ++
    ",\"kernel\":" ++ kernelSemanticJson kernel ++
    ",\"behavior\":" ++ targetBehaviorDescriptionJson behavior ++ "}"

private def targetMetadataJson
    (target : TargetDeclaration LawStatement Setup State Action Outcome Observation)
    (kernel : KernelMetadata)
    (behavior : TargetBehaviorDescription) : String :=
  "{\"semantic\":" ++ targetSemanticJson target.id target.definitions
      target.requiredCapabilities target.providers target.connectors kernel behavior ++
    ",\"source\":" ++ sourceJson target.source ++
    ",\"definitionMetadata\":" ++
      array (target.definitions.mergeSort definitionLe |>.map canonicalDefinitionMetadataJson) ++
    ",\"kernelMetadata\":" ++ canonicalKernelMetadataJson kernel ++ "}"

private def definitionError
    (kind : DefinitionErrorKind)
    (definitionId : DefinitionId)
    (source : SourceLocation)
    (offendingValue : String)
    (relatedDefinitionIds : List DefinitionId := []) : DefinitionError := {
  kind
  definitionId := if definitionId.value == "" then
    DefinitionId.of "umpire.definition.anonymous"
  else
    definitionId
  sourcePath := sourcePath source
  offendingValue
  relatedDefinitionIds := canonicalIds relatedDefinitionIds
}

private def validationError
    (kind : DefinitionErrorKind)
    (definitionId : DefinitionId)
    (source : SourceLocation)
    (path : AuthoringOccurrencePath)
    (occurrenceDefinitionId : DefinitionId)
    (offendingValue : String)
    (relatedDefinitionIds : List DefinitionId := []) : TargetValidationError := {
  error := definitionError kind definitionId source offendingValue relatedDefinitionIds
  path
  occurrenceDefinitionId
  source
}

private def firstDuplicateId : List DefinitionId → Option DefinitionId
  | first :: second :: rest =>
      if first == second then some first else firstDuplicateId (second :: rest)
  | _ => none

private def firstDuplicateDefinition : List DefinitionMetadata → Option DefinitionMetadata
  | first :: second :: rest =>
      if first.id == second.id then some first else firstDuplicateDefinition (second :: rest)
  | _ => none

private def requireDefinitionId
    (owner : DefinitionId)
    (source : SourceLocation)
    (id : DefinitionId)
    (path : AuthoringOccurrencePath) : Except TargetValidationError Unit :=
  if id.value == "" then
    .error (validationError .emptyDefinitionId owner source path id "<empty>" [id])
  else if !id.isNamespaced then
    .error (validationError .invalidDefinitionId owner source path id id.value [id])
  else
    .ok ()

private def requireUniqueIds
    (owner : DefinitionId)
    (source : SourceLocation)
    (path : AuthoringOccurrencePath)
    (ids : List DefinitionId) : Except TargetValidationError Unit :=
  match firstDuplicateId (ids.mergeSort idLe) with
  | some duplicate =>
      .error (validationError .duplicateDefinitionId owner source path duplicate
        duplicate.value [duplicate])
  | none => .ok ()

private def requireDefinition
    (definitions : List DefinitionMetadata)
    (owner : DefinitionId)
    (source : SourceLocation)
    (id : DefinitionId)
    (expectedKind : DefinitionKind)
    (path : AuthoringOccurrencePath) : Except TargetValidationError Unit := do
  requireDefinitionId owner source id path
  match definitions.find? (fun declaration => declaration.id == id) with
  | none => throw (validationError .unknownDefinitionId owner source path id id.value [id])
  | some declaration =>
      if declaration.kind == expectedKind then
        pure ()
      else
        throw (validationError .wrongKind owner source path id
          (id.value ++ ": expected " ++ expectedKind.name ++ ", found " ++ declaration.kind.name)
          [id])

private def occurrencePath
    (role : AuthoringOccurrenceRole)
    (owner : DefinitionId) : AuthoringOccurrencePath :=
  { role, owner }

private def reconciliationOccurrencePath
    (role : AuthoringOccurrenceRole)
    (connector reconciliation : DefinitionId) : AuthoringOccurrencePath :=
  { role, owner := connector, context := .reconciliation reconciliation }

private def validateDefinitions
    (target : TargetDeclaration LawStatement Setup State Action Outcome Observation) :
    Except TargetValidationError (List DefinitionMetadata) := do
  let definitions := target.definitions.mergeSort definitionLe
  for declaration in definitions do
    requireDefinitionId declaration.id declaration.source declaration.id
      (occurrencePath .definitionMetadata declaration.id)
  match firstDuplicateDefinition definitions with
  | some duplicate =>
      throw (validationError .duplicateDefinitionId duplicate.id duplicate.source
        (occurrencePath .definitionMetadata duplicate.id) duplicate.id duplicate.id.value
        [duplicate.id])
  | none => pure definitions

private def validateLawWitnesses
    (definitions : List DefinitionMetadata)
    (owner : DefinitionId)
    (source : SourceLocation)
    (requirements : List LawDefinition)
    (witnesses : List (LawWitness LawStatement)) : Except TargetValidationError Unit := do
  requireUniqueIds owner source (occurrencePath .lawRequirement owner)
    (requirements.map LawDefinition.id)
  requireUniqueIds owner source (occurrencePath .lawWitness owner)
    (witnesses.map (fun witness => witness.definition.id))
  for requirement in requirements.mergeSort lawLe do
    requireDefinition definitions owner source requirement.id .law
      (occurrencePath .lawRequirement owner)
    match definitions.find? (fun declaration => declaration.id == requirement.id) with
    | some declaration =>
        if declaration.canonicalBehavior != requirement.body then
          throw (validationError .lawContractMismatch owner source
            (occurrencePath .lawRequirement owner) requirement.id
            (requirement.id.value ++ ": expected " ++ declaration.canonicalBehavior ++
              ", found " ++ requirement.body)
            [requirement.id])
    | none => pure ()
    match witnesses.find? (fun witness => witness.definition == requirement) with
    | none =>
        throw (validationError .missingLaw owner source (occurrencePath .lawRequirement owner)
          requirement.id requirement.id.value [requirement.id])
    | some _ => pure ()
  for witness in witnesses do
    requireDefinition definitions owner source witness.definition.id .law
      (occurrencePath .lawWitness owner)
    match requirements.find? (fun requirement => requirement == witness.definition) with
    | none =>
        throw (validationError .unexpectedLaw owner source (occurrencePath .lawWitness owner)
          witness.definition.id witness.definition.id.value
          [witness.definition.id])
    | some _ => pure ()

private def validateProvider
    (definitions : List DefinitionMetadata)
    (targetId : DefinitionId)
    (provider : CapabilityProvider LawStatement) : Except TargetValidationError Unit := do
  requireDefinition definitions provider.id provider.source provider.id .provider
    (occurrencePath .providerDefinition targetId)
  requireDefinition definitions provider.id provider.source provider.contract.id .capability
    (occurrencePath .capabilityRequirement provider.id)
  validateLawWitnesses definitions provider.id provider.source
    provider.contract.requiredLaws provider.lawWitnesses
  requireUniqueIds provider.id provider.source (occurrencePath .meaning provider.id)
    (provider.meanings.map MeaningProvision.definitionId)
  for meaning in provider.meanings.mergeSort meaningLe do
    requireDefinition definitions provider.id provider.source meaning.definitionId meaning.kind
      (occurrencePath .meaning provider.id)

private def validateConnector
    (definitions : List DefinitionMetadata)
    (activeProviders : List DefinitionId)
    (targetId : DefinitionId)
    (connector : CapabilityConnector LawStatement) : Except TargetValidationError Unit := do
  requireDefinition definitions connector.id connector.source connector.id .connector
    (occurrencePath .connectorDefinition targetId)
  validateLawWitnesses definitions connector.id connector.source
    connector.requiredLaws connector.lawWitnesses
  requireUniqueIds connector.id connector.source (occurrencePath .reconciliation connector.id)
    (connector.reconciliations.map Reconciliation.definitionId)
  for reconciliation in connector.reconciliations.mergeSort reconciliationLe do
    requireDefinition definitions connector.id connector.source
      reconciliation.definitionId reconciliation.kind (occurrencePath .reconciliation connector.id)
    let providerPath := reconciliationOccurrencePath .providerReference connector.id
      reconciliation.definitionId
    requireUniqueIds connector.id connector.source providerPath reconciliation.providers
    for provider in reconciliation.providers.mergeSort idLe do
      requireDefinition definitions connector.id connector.source provider .provider providerPath
      if !activeProviders.contains provider then
        throw (validationError .missingProvider connector.id connector.source
          providerPath provider provider.value [provider])

private structure MeaningOwner where
  provider : DefinitionId
  meaning : MeaningProvision
  source : SourceLocation

private def distinctStrings (items : List String) : List String :=
  items.mergeSort |>.eraseDups

private def connectorMatches
    (connector : CapabilityConnector LawStatement)
    (definitionId : DefinitionId)
    (providers : List DefinitionId) : Bool :=
  connector.reconciliations.any fun reconciliation =>
    reconciliation.definitionId == definitionId &&
      canonicalIds reconciliation.providers == canonicalIds providers

private def validateConflicts
    (providers : List (CapabilityProvider LawStatement))
    (connectors : List (CapabilityConnector LawStatement)) : Except TargetValidationError Unit := do
  let owners := providers.flatMap fun provider =>
    provider.meanings.map fun meaning => { provider := provider.id, meaning, source := provider.source }
  let definitions := canonicalIds (owners.map fun owner => owner.meaning.definitionId)
  for declaration in definitions do
    let matching := owners.filter (fun owner => owner.meaning.definitionId == declaration)
    let behaviors := distinctStrings (matching.map fun owner => owner.meaning.canonicalBehavior)
    if behaviors.length > 1 then
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
    (definitions : List DefinitionMetadata)
    (providers : List (CapabilityProvider LawStatement))
    (connectors : List (CapabilityConnector LawStatement)) : Except TargetValidationError Unit := do
  requireUniqueIds target.id target.source (occurrencePath .providerDefinition target.id)
    (providers.map CapabilityProvider.id)
  requireUniqueIds target.id target.source (occurrencePath .connectorDefinition target.id)
    (connectors.map CapabilityConnector.id)
  for provider in providers do
    validateProvider definitions target.id provider
  for connector in connectors do
    validateConnector definitions (providers.map CapabilityProvider.id) target.id connector
  requireUniqueIds target.id target.source (occurrencePath .capabilityRequirement target.id)
    target.requiredCapabilities
  for capability in canonicalIds target.requiredCapabilities do
    requireDefinition definitions target.id target.source capability .capability
      (occurrencePath .capabilityRequirement target.id)
    if !(providers.any fun provider => provider.contract.id == capability) then
      throw (validationError .missingProvider target.id target.source
        (occurrencePath .capabilityRequirement target.id) capability capability.value [capability])
  validateConflicts providers connectors

private def composeTargetDetailed
    (target : TargetDeclaration LawStatement Setup State Action Outcome Observation) :
    Except TargetValidationError
      (CheckedTarget LawStatement Setup State Action Outcome Observation) := do
  let definitions ← validateDefinitions target
  requireDefinition definitions target.id target.source target.id .target
    (occurrencePath .targetDefinition target.id)
  let providers := target.providers.mergeSort providerLe
  let connectors := target.connectors.mergeSort connectorLe
  validateCapabilities target definitions providers connectors
  let kernel ← match target.kernel with
    | .checked kernel => pure kernel
    | .incomplete metadata missingProofs =>
        requireDefinition definitions target.id target.source metadata.id .kernel
          (occurrencePath .kernel target.id)
        throw (validationError .incompleteKernel target.id metadata.source
          (occurrencePath .kernel target.id) metadata.id metadata.id.value missingProofs)
  requireDefinition definitions target.id target.source kernel.metadata.id .kernel
    (occurrencePath .kernel target.id)
  let behaviorDomain ← match kernel.behaviorDomain with
    | .missing =>
        throw (validationError .missingBehaviorDomain target.id kernel.metadata.source
          (occurrencePath .kernel target.id) kernel.metadata.id kernel.metadata.id.value
          [kernel.metadata.id])
    | .incomplete missingCoverage =>
        throw (validationError .incompleteBehaviorDomain target.id kernel.metadata.source
          (occurrencePath .kernel target.id) kernel.metadata.id kernel.metadata.id.value
          missingCoverage)
    | .complete domain => pure domain
  match invalidBehaviorDomainEncoding? behaviorDomain with
  | some encoding =>
      throw (validationError .incompleteBehaviorDomain target.id kernel.metadata.source
        (occurrencePath .kernel target.id) kernel.metadata.id encoding [kernel.metadata.id])
  | none => pure ()
  let behavior := kernel.describeBehavior behaviorDomain
  let semantic := targetSemanticJson target.id definitions target.requiredCapabilities
    providers connectors kernel.metadata behavior
  pure {
    id := target.id
    source := target.source
    definitions
    requiredCapabilities := canonicalIds target.requiredCapabilities
    providers
    connectors
    resolvedSetups := target.resolvedSetups
    kernel
    behaviorDescription := behavior
    canonicalMetadata := targetMetadataJson target kernel.metadata behavior
    behaviorFingerprint := behaviorFingerprintOf semantic
  }

/-- Check and canonicalize one target composition without relying on declaration or instance order. -/
def composeTarget
    (target : TargetDeclaration LawStatement Setup State Action Outcome Observation) :
    Except DefinitionError (CheckedTarget LawStatement Setup State Action Outcome Observation) :=
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

private def fallbackOccurrenceId (source : SourceLocation) : AuthoringOccurrenceId := {
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
    occurrence.definitionId == detailed.occurrenceDefinitionId && occurrence.path == detailed.path)
    |>.mergeSort occurrenceLe
  let fallback := fallbackOccurrenceId detailed.source
  if detailed.error.kind == .duplicateDefinitionId then
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

/-- Ordinary Target authoring returns one checked Target or one located typed diagnostic. -/
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
re-ascription inside the Target boundary. Invalid definitions should use `checkTarget` or
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
    (_domains : kernel.setupDomain = target.kernel.setupDomain ∧
      kernel.stateDomain = target.kernel.stateDomain ∧
      kernel.actionDomain = target.kernel.actionDomain ∧
      kernel.outcomeDomain = target.kernel.outcomeDomain ∧
      kernel.observationDomain = target.kernel.observationDomain)
    (_initial : kernel.authoritativeInitial = target.kernel.authoritativeInitial)
    (_step : kernel.authoritativeStep = target.kernel.authoritativeStep)
    (_behavior : kernel.behaviorDescription? = some target.behaviorDescription)
    (planning : FinitePlanningAvailability kernel.authoritativeStep := .unavailable) :
    CheckedTarget LawStatement Setup State Action Outcome Observation := {
  target with
  kernel
  planning
}

/-- Capture one syntax occurrence as a nonsemantic source-span/ordinal token. -/
def captureAuthoringOccurrence
    (reference : Lean.Syntax)
    (definitionId : DefinitionId)
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
      definitionId
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
