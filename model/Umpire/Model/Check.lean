import Umpire.Model.Canonical

/-!
Checked Target semantics and the single pure admission authority. Private checked and authored
constructors stay beside their validators and proof-carrying replacement operation. Pure occurrence
selection preserves diagnostic precedence without depending on captured syntax or elaboration.
The model default stores native_decide syntax for caller elaboration; admission itself
retains its existing proof obligations and does not evaluate that default.
-/

namespace Umpire

open Canonical

structure CheckedModel
    (LawStatement : Law → Prop)
    (Setup State Action Outcome Observation : Type) where
  private mk ::
  id : DefinitionId
  source : SourceLocation
  definitions : List DefinitionMetadata
  requiredCapabilities : List DefinitionId
  providers : List (Provider LawStatement)
  connectors : List (Connector LawStatement)
  resolvedSetups : List Setup
  /-- One eligible-state set per constituent; every set must match. Empty metadata never closes. -/
  terminalConditions : List (List State) := []
  machine : Machine Setup State Action Outcome Observation
  behaviorTable : BehaviorTable
  isTerminal : State → Bool := fun _ => false
  planning : FinitePlanningAvailability machine.authoritativeStep := .unavailable
  canonicalMetadata : String
  behaviorFingerprint : BehaviorFingerprint

private structure ProvidersPayload (LawStatement : Law → Prop) where
  providers : List (Provider LawStatement)
  connectors : List (Connector LawStatement)

/-- Explicit provider and connector choices whose collection is owned by Target. -/
structure Providers (LawStatement : Law → Prop) where
  private mk ::
  private payload : ProvidersPayload LawStatement

namespace Providers

def empty : Providers LawStatement := ⟨{ providers := [], connectors := [] }⟩

def provide
    (composition : Providers LawStatement)
    (provider : Provider LawStatement) : Providers LawStatement :=
  ⟨{ composition.payload with providers := composition.payload.providers ++ [provider] }⟩

def connect
    (composition : Providers LawStatement)
    (connector : Connector LawStatement) : Providers LawStatement :=
  ⟨{ composition.payload with connectors := composition.payload.connectors ++ [connector] }⟩

end Providers

structure DraftModel
    (LawStatement : Law → Prop)
    (Setup State Action Outcome Observation : Type) where
  private mk ::
  private declaration : ModelSpec LawStatement Setup State Action Outcome Observation
  private occurrences : List SourceRef
  private planning : AuthoredPlanningCapability declaration.machine

namespace DraftModel

/-- Assemble the ordinary draft while the model owns provider and connector collection. -/
def make
    (spec : ModelSpec LawStatement Setup State Action Outcome Observation)
    (providers : Providers LawStatement := .empty)
    (planning : AuthoredPlanningCapability spec.machine := .unavailable)
    (occurrences : List SourceRef := []) :
    DraftModel LawStatement Setup State Action Outcome Observation :=
  let declaration : ModelSpec LawStatement Setup State Action Outcome Observation := {
    spec with
    providers := spec.providers ++ providers.payload.providers
    connectors := spec.connectors ++ providers.payload.connectors
  }
  ⟨declaration, occurrences, planning⟩

/-- Replace only elaboration locations; checked semantic inputs remain unchanged. -/
def withOccurrences
    (authored : DraftModel LawStatement Setup State Action Outcome Observation)
    (occurrences : List SourceRef) :
    DraftModel LawStatement Setup State Action Outcome Observation :=
  ⟨authored.declaration, occurrences, authored.planning⟩

/-- Preserve a checked semantic target while making exhaustive planning explicitly unavailable. -/
def withoutPlanning
    (authored : DraftModel LawStatement Setup State Action Outcome Observation) :
    DraftModel LawStatement Setup State Action Outcome Observation :=
  ⟨authored.declaration, authored.occurrences, .unavailable⟩

end DraftModel

private structure LocatedDefinitionError where
  error : DefinitionError
  path : SourceRefPath
  occurrenceDefinitionId : DefinitionId
  source : SourceLocation

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
    (path : SourceRefPath)
    (occurrenceDefinitionId : DefinitionId)
    (offendingValue : String)
    (relatedDefinitionIds : List DefinitionId := []) : LocatedDefinitionError := {
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
    (path : SourceRefPath) : Except LocatedDefinitionError Unit :=
  if id.value == "" then
    .error (validationError .emptyDefinitionId owner source path id "<empty>" [id])
  else if !id.isNamespaced then
    .error (validationError .invalidDefinitionId owner source path id id.value [id])
  else
    .ok ()

private def requireUniqueIds
    (owner : DefinitionId)
    (source : SourceLocation)
    (path : SourceRefPath)
    (ids : List DefinitionId) : Except LocatedDefinitionError Unit :=
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
    (path : SourceRefPath) : Except LocatedDefinitionError Unit := do
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
    (role : SourceRefRole)
    (owner : DefinitionId) : SourceRefPath :=
  { role, owner }

private def reconciliationOccurrencePath
    (role : SourceRefRole)
    (connector reconciliation : DefinitionId) : SourceRefPath :=
  { role, owner := connector, context := .reconciliation reconciliation }

private def validateDefinitions
    (target : ModelSpec LawStatement Setup State Action Outcome Observation) :
    Except LocatedDefinitionError (List DefinitionMetadata) := do
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

private def validateLawProofs
    (definitions : List DefinitionMetadata)
    (owner : DefinitionId)
    (source : SourceLocation)
    (requirements : List Law)
    (witnesses : List (LawProof LawStatement)) : Except LocatedDefinitionError Unit := do
  requireUniqueIds owner source (occurrencePath .lawRequirement owner)
    (requirements.map Law.id)
  requireUniqueIds owner source (occurrencePath .lawProof owner)
    (witnesses.map (fun witness => witness.definition.id))
  for requirement in requirements.mergeSort lawLe do
    requireDefinition definitions owner source requirement.id .law
      (occurrencePath .lawRequirement owner)
    match definitions.find? (fun declaration => declaration.id == requirement.id) with
    | some declaration =>
        if declaration.behaviorVersion != requirement.body then
          throw (validationError .lawContractMismatch owner source
            (occurrencePath .lawRequirement owner) requirement.id
            (requirement.id.value ++ ": expected " ++ declaration.behaviorVersion ++
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
      (occurrencePath .lawProof owner)
    match requirements.find? (fun requirement => requirement == witness.definition) with
    | none =>
        throw (validationError .unexpectedLaw owner source (occurrencePath .lawProof owner)
          witness.definition.id witness.definition.id.value
          [witness.definition.id])
    | some _ => pure ()

private def validateProvider
    (definitions : List DefinitionMetadata)
    (targetId : DefinitionId)
    (provider : Provider LawStatement) : Except LocatedDefinitionError Unit := do
  requireDefinition definitions provider.id provider.source provider.id .provider
    (occurrencePath .providerDefinition targetId)
  requireDefinition definitions provider.id provider.source provider.contract.id .capability
    (occurrencePath .capabilityRequirement provider.id)
  validateLawProofs definitions provider.id provider.source
    provider.contract.requiredLaws provider.lawProofs
  requireUniqueIds provider.id provider.source (occurrencePath .meaning provider.id)
    (provider.meanings.map Meaning.definitionId)
  for meaning in provider.meanings.mergeSort meaningLe do
    requireDefinition definitions provider.id provider.source meaning.definitionId meaning.kind
      (occurrencePath .meaning provider.id)

private def validateConnector
    (definitions : List DefinitionMetadata)
    (activeProviders : List DefinitionId)
    (targetId : DefinitionId)
    (connector : Connector LawStatement) : Except LocatedDefinitionError Unit := do
  requireDefinition definitions connector.id connector.source connector.id .connector
    (occurrencePath .connectorDefinition targetId)
  validateLawProofs definitions connector.id connector.source
    connector.requiredLaws connector.lawProofs
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
  meaning : Meaning
  source : SourceLocation

private def distinctStrings (items : List String) : List String :=
  items.mergeSort |>.eraseDups

private def connectorMatches
    (connector : Connector LawStatement)
    (definitionId : DefinitionId)
    (providers : List DefinitionId) : Bool :=
  connector.reconciliations.any fun reconciliation =>
    reconciliation.definitionId == definitionId &&
      canonicalIds reconciliation.providers == canonicalIds providers

private def validateConflicts
    (providers : List (Provider LawStatement))
    (connectors : List (Connector LawStatement)) : Except LocatedDefinitionError Unit := do
  let owners := providers.flatMap fun provider =>
    provider.meanings.map fun meaning => { provider := provider.id, meaning, source := provider.source }
  let definitions := canonicalIds (owners.map fun owner => owner.meaning.definitionId)
  for declaration in definitions do
    let matching := owners.filter (fun owner => owner.meaning.definitionId == declaration)
    let behaviors := distinctStrings (matching.map fun owner => owner.meaning.behaviorVersion)
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
            (connector.id :: rest.map Connector.id))

private def validateCapabilities
    (target : ModelSpec LawStatement Setup State Action Outcome Observation)
    (definitions : List DefinitionMetadata)
    (providers : List (Provider LawStatement))
    (connectors : List (Connector LawStatement)) : Except LocatedDefinitionError Unit := do
  requireUniqueIds target.id target.source (occurrencePath .providerDefinition target.id)
    (providers.map Provider.id)
  requireUniqueIds target.id target.source (occurrencePath .connectorDefinition target.id)
    (connectors.map Connector.id)
  for provider in providers do
    validateProvider definitions target.id provider
  for connector in connectors do
    validateConnector definitions (providers.map Provider.id) target.id connector
  requireUniqueIds target.id target.source (occurrencePath .capabilityRequirement target.id)
    target.requiredCapabilities
  for capability in canonicalIds target.requiredCapabilities do
    requireDefinition definitions target.id target.source capability .capability
      (occurrencePath .capabilityRequirement target.id)
    if !(providers.any fun provider => provider.contract.id == capability) then
      throw (validationError .missingProvider target.id target.source
        (occurrencePath .capabilityRequirement target.id) capability capability.value [capability])
  validateConflicts providers connectors

private def composeModel
    (target : ModelSpec LawStatement Setup State Action Outcome Observation) :
    Except LocatedDefinitionError
      (CheckedModel LawStatement Setup State Action Outcome Observation) := do
  let definitions ← validateDefinitions target
  requireDefinition definitions target.id target.source target.id .target
    (occurrencePath .modelSpec target.id)
  let providers := target.providers.mergeSort providerLe
  let connectors := target.connectors.mergeSort connectorLe
  validateCapabilities target definitions providers connectors
  let kernel ← match target.machine with
    | .checked kernel => pure kernel
    | .incomplete metadata missingProofs =>
        requireDefinition definitions target.id target.source metadata.id .machine
          (occurrencePath .machine target.id)
        throw (validationError .incompleteMachine target.id metadata.source
          (occurrencePath .machine target.id) metadata.id metadata.id.value missingProofs)
  requireDefinition definitions target.id target.source kernel.metadata.id .machine
    (occurrencePath .machine target.id)
  let vocabulary ← match kernel.vocabulary with
    | .missing =>
        throw (validationError .missingVocabulary target.id kernel.metadata.source
          (occurrencePath .machine target.id) kernel.metadata.id kernel.metadata.id.value
          [kernel.metadata.id])
    | .incomplete missingCoverage =>
        throw (validationError .incompleteVocabulary target.id kernel.metadata.source
          (occurrencePath .machine target.id) kernel.metadata.id kernel.metadata.id.value
          missingCoverage)
    | .complete domain => pure domain
  match invalidBehaviorDomainEncoding? vocabulary with
  | some encoding =>
      throw (validationError .incompleteVocabulary target.id kernel.metadata.source
        (occurrencePath .machine target.id) kernel.metadata.id encoding [kernel.metadata.id])
  | none => pure ()
  let terminalConditions := target.terminalConditions.map fun states =>
    canonicalStrings (states.map vocabulary.encodeState)
  for states in terminalConditions do
    if !(states.all fun state => (kernel.describeBehavior vocabulary).states.contains state) then
      throw (validationError .incompleteVocabulary target.id kernel.metadata.source
        (occurrencePath .machine target.id) kernel.metadata.id "terminal-state" [kernel.metadata.id])
  let behavior := { kernel.describeBehavior vocabulary with
    terminalConditions := terminalConditions.mergeSort |>.eraseDups }
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
    terminalConditions := target.terminalConditions
    isTerminal := fun state => !terminalConditions.isEmpty &&
      terminalConditions.all (fun states => states.contains (vocabulary.encodeState state))
    machine := kernel
    behaviorTable := behavior
    canonicalMetadata := targetMetadataJson target kernel.metadata behavior
    behaviorFingerprint := behaviorFingerprintOf semantic
  }

private def occurrenceIdLe (left right : SourceSpan) : Bool :=
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

private def occurrenceLe (left right : SourceRef) : Bool :=
  occurrenceIdLe left.id right.id

private def fallbackOccurrenceId (source : SourceLocation) : SourceSpan := {
  sourcePath := sourcePath source
  line := source.line
  column := source.column
  endLine := source.line
  endColumn := source.column
  localOrdinal := 0
}

private def locatedError
    (occurrences : List SourceRef)
    (detailed : LocatedDefinitionError) : LocatedError :=
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
def checkModel
    (authored : DraftModel LawStatement Setup State Action Outcome Observation) :
    Except LocatedError
      (CheckedModel LawStatement Setup State Action Outcome Observation) :=
  match composeModel authored.declaration with
  | .ok checked =>
      match authored.planning with
      | .unavailable => .ok checked
      | .available machine _ capability =>
          .ok { checked with machine, planning := .available capability }
  | .error detailed => .error (locatedError authored.occurrences detailed)

/-- Produce a checked authored target directly while keeping extraction and proof-relation
re-ascription inside the Target boundary. Invalid definitions should use `checkModel` or
`elabModel` when their typed diagnostic is needed. -/
def model
    (authored : DraftModel LawStatement Setup State Action Outcome Observation)
    (valid : (checkModel authored).toOption.isSome = true := by native_decide) :
    CheckedModel LawStatement Setup State Action Outcome Observation :=
  let checked := (checkModel authored).toOption.get valid
  match authored.planning with
  | .unavailable => checked
  | .available machine _ capability => {
      checked with
      machine
      planning := .available capability
    }

/-- Rebind implementation enumerators while proving the checked semantic kernel is unchanged. -/
def CheckedModel.withEquivalentMachine
    (target : CheckedModel LawStatement Setup State Action Outcome Observation)
    (machine : Machine Setup State Action Outcome Observation)
    (_metadata : machine.metadata = target.machine.metadata)
    (_domains : machine.setupDomain = target.machine.setupDomain ∧
      machine.stateDomain = target.machine.stateDomain ∧
      machine.actionDomain = target.machine.actionDomain ∧
      machine.outcomeDomain = target.machine.outcomeDomain ∧
      machine.observationDomain = target.machine.observationDomain)
    (_initial : machine.authoritativeInitial = target.machine.authoritativeInitial)
    (_step : machine.authoritativeStep = target.machine.authoritativeStep)
    (_behavior : machine.behaviorTable? = some { target.behaviorTable with terminalConditions := [] })
    (planning : FinitePlanningAvailability machine.authoritativeStep := .unavailable) :
    CheckedModel LawStatement Setup State Action Outcome Observation := {
  target with
  machine
  planning
}

def canonicalCheckedModelJson
    (target : CheckedModel LawStatement Setup State Action Outcome Observation) : String :=
  target.canonicalMetadata

end Umpire
