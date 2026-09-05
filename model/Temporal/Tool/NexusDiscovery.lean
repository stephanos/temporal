import Lean.Data.Json
import Temporal.Feature.Nexus.Operations.AsyncStart
import Temporal.Feature.Nexus.Operations.Cancellation
import Temporal.Feature.Nexus.Operations.SuccessfulCompletion

/-!
# Closed Nexus discovery inventory

This module projects three existing checked Nexus examples into one deterministic inventory. Its
private entry constructor ensures callers can observe only rows whose declaration ownership,
planned Artifact lineage, exact membership, and canonical order were validated together.
The canonical list projection exposes compact summaries of those checked bindings.
The explanation projection reuses one summary and exposes its complete checked plan lineage.
-/

namespace Temporal.Tool.NexusDiscovery

open _root_.Umpire

/-- Declaration roles understood by the closed Nexus discovery adapter. -/
inductive NexusDiscoveryKind where
  | property
  | behavior
  | query
  deriving BEq, DecidableEq, Repr

def NexusDiscoveryKind.name : NexusDiscoveryKind → String
  | .property => "property"
  | .behavior => "behavior"
  | .query => "query"

/-- Identity, source, and Behavior Fingerprint projected from one checked declaration. -/
structure NexusDiscoveryDeclaration where
  id : DefinitionId
  kind : NexusDiscoveryKind
  source : SourceLocation
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

/-- One Property identity and Behavior Fingerprint retained in Query or Artifact lineage. -/
structure NexusDiscoveryPropertyBinding where
  definitionId : DefinitionId
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

/-- Identity lineage projected from one planned `ExperimentSpec`. -/
structure NexusDiscoveryPlan where
  formatVersion : String
  artifactChecksum : ArtifactChecksum
  queryDefinitionId : DefinitionId
  queryBehaviorFingerprint : BehaviorFingerprint
  behaviorDefinitionId : DefinitionId
  behaviorFingerprint : BehaviorFingerprint
  targetDefinitionId : DefinitionId
  targetBehaviorFingerprint : BehaviorFingerprint
  kernelDefinitionId : DefinitionId
  kernelBehaviorFingerprint : BehaviorFingerprint
  properties : List NexusDiscoveryPropertyBinding
  provenanceDefinitionIds : List DefinitionId
  provenanceSources : List SourceLocation
  deriving BEq, DecidableEq, Repr

/-- Untrusted projected row accepted by `checkInventory`. -/
structure NexusDiscoveryCandidate where
  property : NexusDiscoveryDeclaration
  behavior : NexusDiscoveryDeclaration
  query : NexusDiscoveryDeclaration
  queryProperties : List NexusDiscoveryPropertyBinding
  queryBehaviorDefinitionId : DefinitionId
  queryBehaviorFingerprint : BehaviorFingerprint
  plan : Option NexusDiscoveryPlan
  deriving BEq, DecidableEq, Repr

/-- A Nexus discovery row whose closed ownership and plan lineage have been checked. -/
structure NexusDiscoveryEntry where
  private mk ::
  property : NexusDiscoveryDeclaration
  behavior : NexusDiscoveryDeclaration
  query : NexusDiscoveryDeclaration
  plan : NexusDiscoveryPlan
  deriving BEq, DecidableEq, Repr

/-- The validated three-row inventory in canonical query-identity order. -/
structure NexusDiscoveryInventory where
  private mk ::
  entries : List NexusDiscoveryEntry
  deriving BEq, DecidableEq, Repr

/-- Atomic failures produced while admitting the closed inventory. -/
inductive NexusDiscoveryErrorKind where
  | duplicateQuery
  | membershipDrift
  | wrongKind
  | duplicateDeclaration
  | crossedOwner
  | missingSource
  | missingPlan
  | planIdentityDrift
  deriving BEq, DecidableEq, Repr

/-- Stable diagnostic label for one inventory-admission failure kind. -/
def NexusDiscoveryErrorKind.name : NexusDiscoveryErrorKind → String
  | .duplicateQuery => "duplicate-query"
  | .membershipDrift => "membership-drift"
  | .wrongKind => "wrong-kind"
  | .duplicateDeclaration => "duplicate-declaration"
  | .crossedOwner => "crossed-owner"
  | .missingSource => "missing-source"
  | .missingPlan => "missing-plan"
  | .planIdentityDrift => "plan-identity-drift"

/-- One structural inventory-admission failure. -/
structure NexusDiscoveryError where
  kind : NexusDiscoveryErrorKind
  queryId : DefinitionId
  deriving BEq, DecidableEq, Repr

private def declaration
    (kind : NexusDiscoveryKind)
    (id : DefinitionId)
    (source : SourceLocation)
    (behaviorFingerprint : BehaviorFingerprint) : NexusDiscoveryDeclaration := {
  id
  kind
  source
  behaviorFingerprint
}

private def propertyBinding
    (property : CheckedProperty) : NexusDiscoveryPropertyBinding := {
  definitionId := property.id
  behaviorFingerprint := property.behaviorFingerprint
}

private def planLineage (spec : ExperimentSpec) : NexusDiscoveryPlan := {
  formatVersion := spec.formatVersion
  artifactChecksum := spec.artifactChecksum
  queryDefinitionId := spec.plan.queryDefinitionId
  queryBehaviorFingerprint := spec.queryBehaviorFingerprint
  behaviorDefinitionId := spec.plan.behaviorDefinitionId
  behaviorFingerprint := spec.plan.behaviorFingerprint
  targetDefinitionId := spec.plan.targetDefinitionId
  targetBehaviorFingerprint := spec.plan.targetBehaviorFingerprint
  kernelDefinitionId := spec.plan.kernelDefinitionId
  kernelBehaviorFingerprint := spec.plan.kernelBehaviorFingerprint
  properties := spec.properties.map fun property => {
    definitionId := property.definitionId
    behaviorFingerprint := property.behaviorFingerprint
  }
  provenanceDefinitionIds := spec.provenance.sourceDefinitionIds
  provenanceSources := spec.provenance.sourceLocations
}

/--
Project one checked Property, Behavior, Query, and optional planned Artifact into an input row.
-/
def candidateOf
    {LawStatement : LawDefinition → Prop}
    (property : CheckedProperty)
    (behavior : CheckedBehavior)
    (query : CheckedQuery LawStatement)
    (plan : Option ExperimentSpec) : NexusDiscoveryCandidate := {
  property := declaration .property property.id property.source property.behaviorFingerprint
  behavior := declaration .behavior behavior.id behavior.source behavior.behaviorFingerprint
  query := declaration .query query.id query.source query.behaviorFingerprint
  queryProperties := query.form.properties.map propertyBinding
  queryBehaviorDefinitionId := query.behavior.id
  queryBehaviorFingerprint := query.behavior.behaviorFingerprint
  plan := plan.map planLineage
}

private def expectedCandidates : List NexusDiscoveryCandidate := [
  candidateOf
    Temporal.Feature.Nexus.Operations.AsyncStart.property
    Temporal.Feature.Nexus.Operations.AsyncStart.behavior
    Temporal.Feature.Nexus.Operations.AsyncStart.query
    Temporal.Feature.Nexus.Operations.AsyncStart.run.artifact,
  candidateOf
    Temporal.Feature.Nexus.Operations.Cancellation.property
    Temporal.Feature.Nexus.Operations.Cancellation.behavior
    Temporal.Feature.Nexus.Operations.Cancellation.query
    Temporal.Feature.Nexus.Operations.Cancellation.run.artifact,
  candidateOf
    Temporal.Feature.Nexus.Operations.SuccessfulCompletion.property
    Temporal.Feature.Nexus.Operations.SuccessfulCompletion.behavior
    Temporal.Feature.Nexus.Operations.SuccessfulCompletion.query
    Temporal.Feature.Nexus.Operations.SuccessfulCompletion.run.artifact
]

private def candidateLe (left right : NexusDiscoveryCandidate) : Bool :=
  decide (left.query.id.value ≤ right.query.id.value)

private def canonicalCandidates
    (candidates : List NexusDiscoveryCandidate) : List NexusDiscoveryCandidate :=
  candidates.mergeSort candidateLe

private def firstDuplicateQuery : List NexusDiscoveryCandidate → Option DefinitionId
  | first :: second :: rest =>
      if first.query.id == second.query.id then some first.query.id
      else firstDuplicateQuery (second :: rest)
  | _ => none

private def failure
    (kind : NexusDiscoveryErrorKind)
    (candidate : NexusDiscoveryCandidate) : NexusDiscoveryError := {
  kind
  queryId := candidate.query.id
}

private def sourceIsPresent (source : SourceLocation) : Bool :=
  !source.path.trimAscii.isEmpty && !source.provenance.trimAscii.isEmpty

private def declarationIsValid
    (candidate : NexusDiscoveryCandidate)
    (projected : NexusDiscoveryDeclaration)
    (expectedKind : NexusDiscoveryKind) : Except NexusDiscoveryError Unit := do
  if projected.kind != expectedKind then
    throw (failure .wrongKind candidate)
  if !projected.id.isNamespaced then
    throw (failure .membershipDrift candidate)
  if !sourceIsPresent projected.source then
    throw (failure .missingSource candidate)
  if projected.behaviorFingerprint.render.isEmpty then
    throw (failure .membershipDrift candidate)

private def expectedPropertyBinding
    (candidate : NexusDiscoveryCandidate) : NexusDiscoveryPropertyBinding := {
  definitionId := candidate.property.id
  behaviorFingerprint := candidate.property.behaviorFingerprint
}

private def validateOwnerLineage
    (candidate : NexusDiscoveryCandidate) : Except NexusDiscoveryError Unit := do
  if candidate.queryProperties != [expectedPropertyBinding candidate] ||
      candidate.queryBehaviorDefinitionId != candidate.behavior.id ||
      candidate.queryBehaviorFingerprint != candidate.behavior.behaviorFingerprint then
    throw (failure .crossedOwner candidate)

private def validatePlanLineage
    (candidate : NexusDiscoveryCandidate)
    (plan : NexusDiscoveryPlan) : Except NexusDiscoveryError Unit := do
  if plan.queryDefinitionId != candidate.query.id ||
      plan.queryBehaviorFingerprint != candidate.query.behaviorFingerprint ||
      plan.behaviorDefinitionId != candidate.behavior.id ||
      plan.behaviorFingerprint != candidate.behavior.behaviorFingerprint ||
      plan.properties != [expectedPropertyBinding candidate] ||
      plan.formatVersion.trimAscii.isEmpty || plan.artifactChecksum.render.isEmpty ||
      !plan.targetDefinitionId.isNamespaced || !plan.kernelDefinitionId.isNamespaced ||
      plan.targetBehaviorFingerprint.render.isEmpty ||
      plan.kernelBehaviorFingerprint.render.isEmpty ||
      plan.provenanceDefinitionIds.isEmpty ||
      !plan.provenanceDefinitionIds.all DefinitionId.isNamespaced ||
      plan.provenanceSources.isEmpty || !plan.provenanceSources.all sourceIsPresent then
    throw (failure .planIdentityDrift candidate)

private def validateCandidate
    (candidate : NexusDiscoveryCandidate) : Except NexusDiscoveryError NexusDiscoveryEntry := do
  declarationIsValid candidate candidate.property .property
  declarationIsValid candidate candidate.behavior .behavior
  declarationIsValid candidate candidate.query .query
  validateOwnerLineage candidate
  let plan ← match candidate.plan with
    | none => throw (failure .missingPlan candidate)
    | some plan => pure plan
  validatePlanLineage candidate plan
  pure ⟨candidate.property, candidate.behavior, candidate.query, plan⟩

/-- Validate exact membership and return entries in canonical query-identity order. -/
def checkInventory
    (candidates : List NexusDiscoveryCandidate) :
    Except NexusDiscoveryError NexusDiscoveryInventory := do
  let canonical := canonicalCandidates candidates
  match firstDuplicateQuery canonical with
  | some queryId => throw { kind := .duplicateQuery, queryId }
  | none => pure ()
  let entries ← canonical.mapM validateCandidate
  let declarationIds := canonical.flatMap fun candidate =>
    [candidate.property.id, candidate.behavior.id, candidate.query.id]
  if !declarationIds.Nodup then
    let queryId := canonical.head?.map (fun candidate => candidate.query.id)
      |>.getD (DefinitionId.of "temporal.nexus.discovery")
    throw { kind := .duplicateDeclaration, queryId }
  if canonical != canonicalCandidates expectedCandidates then
    let queryId := canonical.head?.map (fun candidate => candidate.query.id)
      |>.getD (DefinitionId.of "temporal.nexus.discovery")
    throw { kind := .membershipDrift, queryId }
  pure ⟨entries⟩

private def frame (value : String) : String :=
  toString value.length ++ ":" ++ value

private def sourceBinding (source : SourceLocation) : String :=
  String.intercalate "" [
    frame source.path,
    frame (toString source.line),
    frame (toString source.column),
    frame source.provenance
  ]

private def declarationBinding (projected : NexusDiscoveryDeclaration) : String :=
  String.intercalate "" [
    frame projected.id.value,
    frame projected.kind.name,
    frame (sourceBinding projected.source),
    frame projected.behaviorFingerprint.render
  ]

private def propertyBindingBytes (property : NexusDiscoveryPropertyBinding) : String :=
  frame property.definitionId.value ++ frame property.behaviorFingerprint.render

private def planBinding (plan : NexusDiscoveryPlan) : String :=
  String.intercalate "" [
    frame plan.formatVersion,
    frame plan.artifactChecksum.render,
    frame plan.queryDefinitionId.value,
    frame plan.queryBehaviorFingerprint.render,
    frame plan.behaviorDefinitionId.value,
    frame plan.behaviorFingerprint.render,
    frame plan.targetDefinitionId.value,
    frame plan.targetBehaviorFingerprint.render,
    frame plan.kernelDefinitionId.value,
    frame plan.kernelBehaviorFingerprint.render,
    frame (String.intercalate "" (plan.properties.map propertyBindingBytes)),
    frame (String.intercalate "" (plan.provenanceDefinitionIds.map fun id => frame id.value)),
    frame (String.intercalate "" (plan.provenanceSources.map fun source =>
      frame (sourceBinding source)))
  ]

/-- Canonical internal bytes used to compare validated inventory bindings across input order. -/
def NexusDiscoveryInventory.canonicalBindingBytes
    (inventory : NexusDiscoveryInventory) : String :=
  String.intercalate "" (inventory.entries.map fun entry => frame <| String.intercalate "" [
    frame (declarationBinding entry.property),
    frame (declarationBinding entry.behavior),
    frame (declarationBinding entry.query),
    frame (planBinding entry.plan)
  ])

/-- Find one validated row by exact canonical Query identity. -/
def NexusDiscoveryInventory.findEntry?
    (inventory : NexusDiscoveryInventory)
    (queryId : String) : Option NexusDiscoveryEntry :=
  inventory.entries.find? fun entry => entry.query.id.value == queryId

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def sourceJson (source : SourceLocation) : String :=
  "{\"path\":" ++ quote source.path ++
    ",\"line\":" ++ toString source.line ++
    ",\"column\":" ++ toString source.column ++
    ",\"provenance\":" ++ quote source.provenance ++ "}"

private def declarationSummaryJson (declaration : NexusDiscoveryDeclaration) : String :=
  "{\"definitionId\":" ++ quote declaration.id.value ++
    ",\"kind\":" ++ quote declaration.kind.name ++
    ",\"source\":" ++ sourceJson declaration.source ++
    ",\"behaviorFingerprint\":" ++ quote declaration.behaviorFingerprint.render ++ "}"

private def planSummaryJson (plan : NexusDiscoveryPlan) : String :=
  "{\"formatVersion\":" ++ quote plan.formatVersion ++
    ",\"artifactChecksum\":" ++ quote plan.artifactChecksum.render ++ "}"

private def propertyBindingJson (property : NexusDiscoveryPropertyBinding) : String :=
  "{\"definitionId\":" ++ quote property.definitionId.value ++
    ",\"behaviorFingerprint\":" ++ quote property.behaviorFingerprint.render ++ "}"

private def planLineageJson (plan : NexusDiscoveryPlan) : String :=
  "{\"formatVersion\":" ++ quote plan.formatVersion ++
    ",\"artifactChecksum\":" ++ quote plan.artifactChecksum.render ++
    ",\"queryDefinitionId\":" ++ quote plan.queryDefinitionId.value ++
    ",\"queryBehaviorFingerprint\":" ++ quote plan.queryBehaviorFingerprint.render ++
    ",\"behaviorDefinitionId\":" ++ quote plan.behaviorDefinitionId.value ++
    ",\"behaviorFingerprint\":" ++ quote plan.behaviorFingerprint.render ++
    ",\"targetDefinitionId\":" ++ quote plan.targetDefinitionId.value ++
    ",\"targetBehaviorFingerprint\":" ++ quote plan.targetBehaviorFingerprint.render ++
    ",\"kernelDefinitionId\":" ++ quote plan.kernelDefinitionId.value ++
    ",\"kernelBehaviorFingerprint\":" ++ quote plan.kernelBehaviorFingerprint.render ++
    ",\"properties\":" ++ array (plan.properties.map propertyBindingJson) ++
    ",\"provenanceDefinitionIds\":" ++
      array (plan.provenanceDefinitionIds.map fun id => quote id.value) ++
    ",\"provenanceSources\":" ++ array (plan.provenanceSources.map sourceJson) ++ "}"

/-- Encode the compact list summary shared by discovery commands. -/
def NexusDiscoveryEntry.canonicalSummaryJson (entry : NexusDiscoveryEntry) : String :=
  "{\"queryDefinitionId\":" ++ quote entry.query.id.value ++
    ",\"property\":" ++ declarationSummaryJson entry.property ++
    ",\"behavior\":" ++ declarationSummaryJson entry.behavior ++
    ",\"query\":" ++ declarationSummaryJson entry.query ++
    ",\"experimentSpec\":" ++ planSummaryJson entry.plan ++ "}"

/-- Encode one validated row and its complete checked plan lineage as canonical explanation JSON. -/
def NexusDiscoveryEntry.canonicalExplanationJson (entry : NexusDiscoveryEntry) : String :=
  "{\"formatVersion\":\"umpire-nexus-explanation/v1\",\"summary\":" ++
    entry.canonicalSummaryJson ++ ",\"lineage\":" ++ planLineageJson entry.plan ++ "}"

/-- Encode one canonical Nexus explanation followed by one line feed. -/
def NexusDiscoveryEntry.canonicalExplanationBytes (entry : NexusDiscoveryEntry) : String :=
  entry.canonicalExplanationJson ++ "\n"

/-- Encode one validated inventory as the canonical version-one discovery JSON value. -/
def NexusDiscoveryInventory.canonicalListJson (inventory : NexusDiscoveryInventory) : String :=
  "{\"formatVersion\":\"umpire-nexus-discovery/v1\",\"entries\":" ++
    array (inventory.entries.map NexusDiscoveryEntry.canonicalSummaryJson) ++ "}"

/-- Encode one validated inventory as canonical discovery JSON followed by one line feed. -/
def NexusDiscoveryInventory.canonicalListBytes (inventory : NexusDiscoveryInventory) : String :=
  inventory.canonicalListJson ++ "\n"

private def inventoryResult : Except NexusDiscoveryError NexusDiscoveryInventory :=
  checkInventory expectedCandidates

private theorem inventoryResult_isSome : inventoryResult.toOption.isSome = true := by
  native_decide

/-- The sole validated input for retained Nexus discovery projections. -/
def inventory : NexusDiscoveryInventory :=
  inventoryResult.toOption.get inventoryResult_isSome

end Temporal.Tool.NexusDiscovery
