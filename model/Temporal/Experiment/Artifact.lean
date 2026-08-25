import Lean.Data.Json
import Temporal.Experiment.Query

namespace Temporal.Experiment

/-! Portable, environment-independent products of pure model planning. -/

def SelectionReason.name : SelectionReason → String
  | .satisfyingWitness => "satisfying-witness"
  | .violatingCounterexample => "violating-counterexample"
  | .behaviorSelection => "behavior-selection"

structure ObservationCheckpoint where
  transition : Nat
  observations : List SemanticValue
  deriving BEq, DecidableEq, Repr

structure PortableProperty where
  identity : DeclarationId
  semanticDigest : String
  requirements : List DeclarationId
  deriving BEq, DecidableEq, Repr

structure ArtifactProvenance where
  sourceIdentities : List DeclarationId
  sources : List SemanticSource
  deriving BEq, DecidableEq, Repr

/--
The runtime-facing request remains explicit model data. Requested actions and model-owned outcomes
are deliberately separate, and no field claims that runtime execution or evidence occurred.
-/
structure DrivePlan where
  formatVersion : String
  semanticIdentity : String
  queryIdentity : DeclarationId
  querySemanticDigest : String
  behaviorIdentity : DeclarationId
  behaviorSemanticDigest : String
  targetIdentity : DeclarationId
  targetSemanticDigest : String
  kernelIdentity : DeclarationId
  kernelSemanticDigest : String
  bindings : List RoleBinding
  symbolicRoles : List ResourceRole
  semanticPreconditions : List SetupConstraint
  initialState : SemanticValue
  requestedActions : List SemanticValue
  modelOutcomes : List SemanticValue
  resultingStates : List SemanticValue
  linearExtension : List DeclarationId
  selectedChoices : List SemanticValue
  selectedVariants : List SemanticValue
  requestedFaults : List SemanticValue
  capabilityRequirements : List DeclarationId
  expandedBounds : QueryBounds
  checkpoints : List ObservationCheckpoint
  selectionReason : SelectionReason
  explored : ExploredCounts
  omissions : List String
  provenance : ArtifactProvenance
  deriving BEq, DecidableEq, Repr

/-- The portable envelope consumed by later execution, checking, replay, and generation work. -/
structure ExperimentSpec where
  formatVersion : String
  semanticIdentity : String
  querySemanticDigest : String
  plan : DrivePlan
  properties : List PortableProperty
  observationRequirements : List DeclarationId
  provenance : ArtifactProvenance
  deriving BEq, DecidableEq, Repr

def canonicalPlannerOmissions : List String := [
  "artifact-migrations",
  "artifact-reading",
  "evidence-qualification",
  "execution-evidence",
  "promotion",
  "runtime-scheduler-order",
  "runtime-storage-order",
  "runtime-transport-order"
]

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def idLe (left right : DeclarationId) : Bool :=
  decide (left.value ≤ right.value)

private def valueLe (left right : SemanticValue) : Bool :=
  decide (left.identity.value < right.identity.value) ||
    (left.identity == right.identity && decide (left.value ≤ right.value))

private def bindingLe (left right : RoleBinding) : Bool :=
  decide (left.role.value < right.role.value) ||
    (left.role == right.role && valueLe left.value right.value)

private def sourceLe (left right : SemanticSource) : Bool :=
  decide (left.path < right.path) ||
    (left.path == right.path && decide (left.line < right.line)) ||
    (left.path == right.path && left.line == right.line && decide (left.column < right.column)) ||
    (left.path == right.path && left.line == right.line && left.column == right.column &&
      decide (left.provenance ≤ right.provenance))

private def propertyLe (left right : PortableProperty) : Bool :=
  decide (left.identity.value ≤ right.identity.value)

private def canonicalIds (ids : List DeclarationId) : List DeclarationId :=
  ids.mergeSort idLe |>.eraseDups

private def canonicalValues (values : List SemanticValue) : List SemanticValue :=
  values.mergeSort valueLe |>.eraseDups

private def valueJson (value : SemanticValue) : String :=
  "{\"identity\":" ++ quote value.identity.value ++
    ",\"value\":" ++ quote value.value ++ "}"

private def roleJson (role : ResourceRole) : String :=
  "{\"identity\":" ++ quote role.id.value ++
    ",\"valueKind\":" ++ quote role.valueKind.name ++ "}"

private def bindingJson (binding : RoleBinding) : String :=
  "{\"role\":" ++ quote binding.role.value ++
    ",\"value\":" ++ valueJson binding.value ++ "}"

private def operandJson : SetupOperand → String
  | .role identity =>
      "{\"kind\":\"role\",\"identity\":" ++ quote identity.value ++ "}"
  | .value value =>
      "{\"kind\":\"value\",\"value\":" ++ valueJson value ++ "}"

private def preconditionJson (constraint : SetupConstraint) : String :=
  "{\"identity\":" ++ quote constraint.id.value ++
    ",\"relation\":" ++ quote constraint.relation.name ++
    ",\"left\":" ++ operandJson constraint.left ++
    ",\"right\":" ++ operandJson constraint.right ++ "}"

private def typedBoundJson (bound : TypedBound) : String :=
  "{\"value\":" ++ toString bound.value ++
    ",\"unit\":" ++ quote bound.unit.name ++ "}"

private def boundsJson (bounds : QueryBounds) : String :=
  "{\"behavior\":{\"transitions\":" ++ typedBoundJson bounds.behavior.transitions ++
    ",\"selectedActions\":" ++ typedBoundJson bounds.behavior.selectedActions ++ "}" ++
    ",\"search\":{\"value\":" ++ toString bounds.search.value ++
    ",\"unit\":" ++ quote bounds.search.unit.name ++ "}}"

private def exploredJson (explored : ExploredCounts) : String :=
  "{\"setups\":" ++ toString explored.setups ++
    ",\"traces\":" ++ toString explored.traces ++
    ",\"transitions\":" ++ toString explored.transitions ++
    ",\"propertyEvaluations\":" ++ toString explored.propertyEvaluations ++ "}"

private def checkpointJson (checkpoint : ObservationCheckpoint) : String :=
  "{\"transition\":" ++ toString checkpoint.transition ++
    ",\"observations\":" ++ array (checkpoint.observations.map valueJson) ++ "}"

private def sourceJson (source : SemanticSource) : String :=
  "{\"path\":" ++ quote source.path ++
    ",\"line\":" ++ toString source.line ++
    ",\"column\":" ++ toString source.column ++
    ",\"provenance\":" ++ quote source.provenance ++ "}"

private def provenanceJson (provenance : ArtifactProvenance) : String :=
  "{\"sourceIdentities\":" ++
      array (canonicalIds provenance.sourceIdentities |>.map (quote ∘ DeclarationId.value)) ++
    ",\"sources\":" ++ array (provenance.sources.mergeSort sourceLe |>.eraseDups |>.map sourceJson) ++ "}"

private def propertyJson (property : PortableProperty) : String :=
  "{\"identity\":" ++ quote property.identity.value ++
    ",\"semanticDigest\":" ++ quote property.semanticDigest ++
    ",\"requirements\":" ++
      array (canonicalIds property.requirements |>.map (quote ∘ DeclarationId.value)) ++ "}"

private def drivePlanSemanticJson (plan : DrivePlan) : String :=
  "{\"formatVersion\":" ++ quote plan.formatVersion ++
    ",\"queryIdentity\":" ++ quote plan.queryIdentity.value ++
    ",\"querySemanticDigest\":" ++ quote plan.querySemanticDigest ++
    ",\"behaviorIdentity\":" ++ quote plan.behaviorIdentity.value ++
    ",\"behaviorSemanticDigest\":" ++ quote plan.behaviorSemanticDigest ++
    ",\"targetIdentity\":" ++ quote plan.targetIdentity.value ++
    ",\"targetSemanticDigest\":" ++ quote plan.targetSemanticDigest ++
    ",\"kernelIdentity\":" ++ quote plan.kernelIdentity.value ++
    ",\"kernelSemanticDigest\":" ++ quote plan.kernelSemanticDigest ++
    ",\"bindings\":" ++ array (plan.bindings.mergeSort bindingLe |>.map bindingJson) ++
    ",\"symbolicRoles\":" ++ array (plan.symbolicRoles.map roleJson) ++
    ",\"semanticPreconditions\":" ++ array (plan.semanticPreconditions.map preconditionJson) ++
    ",\"initialState\":" ++ valueJson plan.initialState ++
    ",\"requestedActions\":" ++ array (plan.requestedActions.map valueJson) ++
    ",\"modelOutcomes\":" ++ array (plan.modelOutcomes.map valueJson) ++
    ",\"resultingStates\":" ++ array (plan.resultingStates.map valueJson) ++
    ",\"linearExtension\":" ++
      array (plan.linearExtension.map (quote ∘ DeclarationId.value)) ++
    ",\"selectedChoices\":" ++ array (plan.selectedChoices.map valueJson) ++
    ",\"selectedVariants\":" ++ array (plan.selectedVariants.map valueJson) ++
    ",\"requestedFaults\":" ++ array (plan.requestedFaults.map valueJson) ++
    ",\"capabilityRequirements\":" ++
      array (canonicalIds plan.capabilityRequirements |>.map (quote ∘ DeclarationId.value)) ++
    ",\"expandedBounds\":" ++ boundsJson plan.expandedBounds ++
    ",\"checkpoints\":" ++ array (plan.checkpoints.map checkpointJson) ++
    ",\"selectionReason\":" ++ quote plan.selectionReason.name ++
    ",\"explored\":" ++ exploredJson plan.explored ++
    ",\"omissions\":" ++ array (plan.omissions.mergeSort |>.eraseDups |>.map quote) ++ "}"

def canonicalDrivePlanJson (plan : DrivePlan) : String :=
  let semantic := drivePlanSemanticJson plan
  (semantic.dropEnd 1).toString ++
    ",\"semanticIdentity\":" ++ quote plan.semanticIdentity ++
    ",\"provenance\":" ++ provenanceJson plan.provenance ++ "}"

private def experimentSpecSemanticJson (spec : ExperimentSpec) : String :=
  "{\"formatVersion\":" ++ quote spec.formatVersion ++
    ",\"querySemanticDigest\":" ++ quote spec.querySemanticDigest ++
    ",\"planSemanticIdentity\":" ++ quote spec.plan.semanticIdentity ++
    ",\"plan\":" ++ drivePlanSemanticJson spec.plan ++
    ",\"properties\":" ++ array (spec.properties.mergeSort propertyLe |>.map propertyJson) ++
    ",\"observationRequirements\":" ++
      array (canonicalIds spec.observationRequirements |>.map (quote ∘ DeclarationId.value)) ++ "}"

def canonicalExperimentSpecJson (spec : ExperimentSpec) : String :=
  let semantic := experimentSpecSemanticJson spec
  (semantic.dropEnd 1).toString ++
    ",\"semanticIdentity\":" ++ quote spec.semanticIdentity ++
    ",\"provenance\":" ++ provenanceJson spec.provenance ++ "}"

private def occurrenceLe (left right : NamedOccurrence) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def occurrenceReady
    (ordering : List OccurrenceOrder)
    (remaining : List NamedOccurrence)
    (occurrence : NamedOccurrence) : Bool :=
  ordering.all fun edge =>
    edge.after != occurrence.id ||
      !(remaining.any fun candidate => candidate.id == edge.before)

private def assignLinearExtension
    (ordering : List OccurrenceOrder) :
    List DeclarationId → List NamedOccurrence → Option (List DeclarationId)
  | [], remaining => if remaining.isEmpty then some [] else none
  | action :: actions, remaining =>
      let assignable := remaining.filter fun candidate =>
        candidate.action == action && occurrenceReady ordering remaining candidate
      let rec firstComplete : List NamedOccurrence → Option (List DeclarationId)
        | [] => assignLinearExtension ordering actions remaining
        | candidate :: rest =>
            match assignLinearExtension ordering actions (remaining.erase candidate) with
            | some assigned => some (candidate.id :: assigned)
            | none => firstComplete rest
      firstComplete (assignable.mergeSort occurrenceLe)

private def propertyReference (property : CheckedProperty) : PortableProperty := {
  identity := property.id
  semanticDigest := property.semanticDigest
  requirements := canonicalIds property.requires
}

private def propertyObservationRequirements (property : CheckedProperty) : List DeclarationId :=
  property.access.meanings.filterMap fun meaning =>
    if meaning.kind == .observation || meaning.kind == .relation then
      some meaning.declaration
    else
      none

private def artifactProvenance (query : CheckedQuery LawStatement) : ArtifactProvenance := {
  sourceIdentities := canonicalIds ([
    query.id,
    query.behavior.id,
    query.target.id,
    query.target.kernel.metadata.id
  ] ++ query.form.properties.map CheckedProperty.id)
  sources := (query.source :: query.behavior.source :: query.target.source ::
    query.target.kernel.metadata.source :: query.form.properties.map CheckedProperty.source)
      |>.mergeSort sourceLe |>.eraseDups
}

/-- Artifact construction is a single deep seam over a selected, kernel-produced trace. -/
def artifactOfSelection
    (query : CheckedQuery LawStatement)
    (trace : BehaviorTrace)
    (reason : SelectionReason)
    (explored : ExploredCounts) : ExperimentSpec :=
  let actions := trace.trace.steps.map fun step => step.selectedAction
  let outcomes := trace.trace.steps.map fun step => step.modelOutcome
  let states := trace.trace.steps.map fun step => step.resultingState
  let extension := assignLinearExtension query.behavior.ordering
    (actions.map SemanticValue.identity) query.behavior.requiredOccurrences |>.getD []
  let checkpoints := trace.trace.steps.zipIdx.map fun (step, index) => {
    transition := index + 1
    observations := step.observations
  }
  let provenance := artifactProvenance query
  let planWithoutIdentity : DrivePlan := {
    formatVersion := "umpire-drive-plan/v1"
    semanticIdentity := ""
    queryIdentity := query.id
    querySemanticDigest := query.semanticDigest
    behaviorIdentity := query.behavior.id
    behaviorSemanticDigest := query.behavior.semanticDigest
    targetIdentity := query.target.id
    targetSemanticDigest := query.target.semanticDigest
    kernelIdentity := query.target.kernel.metadata.id
    kernelSemanticDigest := query.target.kernel.metadata.contractDigest
    bindings := trace.setup.mergeSort bindingLe
    symbolicRoles := query.behavior.roles.filter fun role =>
      !(trace.setup.any fun binding => binding.role == role.id)
    semanticPreconditions := query.behavior.setup
    initialState := trace.trace.initialState
    requestedActions := actions
    modelOutcomes := outcomes
    resultingStates := states
    linearExtension := extension
    selectedChoices := []
    selectedVariants := []
    requestedFaults := []
    capabilityRequirements := canonicalIds (query.target.requiredCapabilities ++
      query.behavior.requires ++
      query.form.properties.flatMap CheckedProperty.requires)
    expandedBounds := query.bounds
    checkpoints
    selectionReason := reason
    explored
    omissions := canonicalPlannerOmissions
    provenance
  }
  let plan := {
    planWithoutIdentity with
    semanticIdentity := semanticDigestOf (drivePlanSemanticJson planWithoutIdentity)
  }
  let properties := query.form.properties.map propertyReference |>.mergeSort propertyLe
  let observationRequirements := canonicalIds
    (query.form.properties.flatMap propertyObservationRequirements)
  let specWithoutIdentity : ExperimentSpec := {
    formatVersion := "umpire-experiment/v1"
    semanticIdentity := ""
    querySemanticDigest := query.semanticDigest
    plan
    properties
    observationRequirements
    provenance
  }
  {
    specWithoutIdentity with
    semanticIdentity := semanticDigestOf (experimentSpecSemanticJson specWithoutIdentity)
  }

end Temporal.Experiment
