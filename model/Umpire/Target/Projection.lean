import Umpire.Target.Data

/-!
Canonical Target behavior and serialization have one pure implementation below admission.
TargetProjection contains shared implementation helpers; syntax and checked-value assembly are
owned by Frontend and Semantics respectively.
-/

namespace Umpire

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

def TargetProjection.canonicalStrings (values : List String) : List String :=
  values.mergeSort |>.eraseDups

open TargetProjection

private def initialRowLe (left right : TargetInitialStateRow) : Bool :=
  decide (left.setup < right.setup) ||
    (left.setup == right.setup && decide (left.state ≤ right.state))

private def transitionRowLe (left right : TargetTransitionRow) : Bool :=
  compare left right != .gt

def TargetProjection.invalidBehaviorDomainEncoding?
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

def Machine.describeBehavior
    (kernel : Machine Setup State Action Outcome Observation)
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
        modelOutcome := domain.encodeOutcome result.outcome
        resultingState := domain.encodeState result.state
        observations := result.facts.map domain.encodeObservation
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
def Machine.behaviorDescription?
    (kernel : Machine Setup State Action Outcome Observation) :
    Option TargetBehaviorDescription :=
  match kernel.behaviorDomain with
  | .complete domain => some (kernel.describeBehavior domain)
  | .missing => none
  | .incomplete _ => none

private def withoutClosingBrace (value : String) : String :=
  (value.dropEnd 1).toString

def TargetProjection.idLe (left right : DefinitionId) : Bool :=
  decide (left.value ≤ right.value)

private def sourceLe (left right : SourceLocation) : Bool :=
  decide (left.path < right.path) ||
    (left.path == right.path && decide (left.line < right.line)) ||
    (left.path == right.path && left.line == right.line && decide (left.column ≤ right.column))

def TargetProjection.definitionLe (left right : DefinitionMetadata) : Bool :=
  decide (left.id.value < right.id.value) ||
    (left.id == right.id && decide (left.kind.name < right.kind.name)) ||
    (left.id == right.id && left.kind == right.kind && sourceLe left.source right.source)

def TargetProjection.providerLe (left right : CapabilityProvider LawStatement) : Bool :=
  decide (left.id.value ≤ right.id.value)

def TargetProjection.connectorLe (left right : CapabilityConnector LawStatement) : Bool :=
  decide (left.id.value ≤ right.id.value)

def TargetProjection.meaningLe (left right : MeaningProvision) : Bool :=
  decide (left.definitionId.value < right.definitionId.value) ||
    (left.definitionId == right.definitionId &&
      decide (left.canonicalBehavior ≤ right.canonicalBehavior))

def TargetProjection.reconciliationLe (left right : Reconciliation) : Bool :=
  decide (left.definitionId.value < right.definitionId.value) ||
    (left.definitionId == right.definitionId &&
      decide (left.canonicalBehavior ≤ right.canonicalBehavior))

def TargetProjection.lawLe (left right : LawDefinition) : Bool :=
  decide (left.id.value < right.id.value) ||
    (left.id == right.id && decide (left.body ≤ right.body))

def TargetProjection.canonicalIds (ids : List DefinitionId) : List DefinitionId :=
  ids.mergeSort idLe |>.eraseDups

def TargetProjection.sourcePath (source : SourceLocation) : String :=
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

private def kernelSemanticJson (metadata : MachineMetadata) : String :=
  "{\"id\":" ++ quote metadata.id.value ++
    ",\"version\":" ++ toString metadata.version ++ "}"

def canonicalMachineMetadataJson (metadata : MachineMetadata) : String :=
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
    ",\"transitions\":" ++ array (description.transitions.map transitionRowJson) ++
    (if description.terminalConditions.isEmpty then "" else
      ",\"terminalConditions/v1\":" ++
        array (description.terminalConditions.map fun states => array (states.map quote))) ++ "}"

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

def TargetProjection.targetSemanticJson
    (id : DefinitionId)
    (definitions : List DefinitionMetadata)
    (requiredCapabilities : List DefinitionId)
    (providers : List (CapabilityProvider LawStatement))
    (connectors : List (CapabilityConnector LawStatement))
    (kernel : MachineMetadata)
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

def TargetProjection.targetMetadataJson
    (target : TargetDeclaration LawStatement Setup State Action Outcome Observation)
    (kernel : MachineMetadata)
    (behavior : TargetBehaviorDescription) : String :=
  "{\"semantic\":" ++ targetSemanticJson target.id target.definitions
      target.requiredCapabilities target.providers target.connectors kernel behavior ++
    ",\"source\":" ++ sourceJson target.source ++
    ",\"definitionMetadata\":" ++
      array (target.definitions.mergeSort definitionLe |>.map canonicalDefinitionMetadataJson) ++
    ",\"machineMetadata\":" ++ canonicalMachineMetadataJson kernel ++ "}"

end Umpire
