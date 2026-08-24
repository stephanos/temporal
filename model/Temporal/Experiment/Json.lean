import Lean.Data.Json
import Temporal.Experiment.DSL

namespace Temporal.Experiment

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def resolvedResourceJson (resource : ResolvedResource) : String :=
  "{\"resourceId\":" ++ quote resource.id.value ++ ",\"value\":" ++ quote resource.value ++ "}"

private def projectedOutcomeJson (projected : ProjectedOutcome) : String :=
  "{\"actionId\":" ++ quote projected.actionId.value ++
    ",\"outcome\":" ++ quote projected.outcome.value ++ "}"

private def propertyObservationJson (expected : ExpectedProperty) : String :=
  "{\"propertyId\":" ++ quote expected.propertyId.value ++
    ",\"contract\":" ++ quote expected.observationContract ++ "}"

def canonicalResolvedSetup (setup : ResolvedSetup) : String :=
  array (setup.resources.map resolvedResourceJson)

def canonicalModelSlice
    (targetId : ModelId)
    (targetDeclaration : String)
    (setup : ResolvedSetup)
    (outcomes : List ProjectedOutcome)
    (properties : List ExpectedProperty) : String :=
  "{\"targetId\":" ++ quote targetId.value ++
    ",\"targetDeclaration\":" ++ quote targetDeclaration ++
    ",\"resolvedSetup\":" ++ canonicalResolvedSetup setup ++
    ",\"projectedOutcomes\":" ++ array (outcomes.map projectedOutcomeJson) ++
    ",\"propertyObservations\":" ++ array (properties.map propertyObservationJson) ++ "}"

def deriveModelIdentity
    (targetId : ModelId)
    (targetDeclaration : String)
    (setup : ResolvedSetup)
    (outcomes : List ProjectedOutcome)
    (properties : List ExpectedProperty) : String :=
  "temporal-model/v1:" ++ canonicalModelSlice targetId targetDeclaration setup outcomes properties

private def precedenceEdgeJson (edge : PrecedenceEdge) : String :=
  "{\"before\":" ++ quote edge.before.value ++ ",\"after\":" ++ quote edge.after.value ++ "}"

private def expectedPropertyJson (expected : ExpectedProperty) : String :=
  "{\"propertyId\":" ++ quote expected.propertyId.value ++
    ",\"observationContract\":" ++ quote expected.observationContract ++ "}"

private def boundsJson (bounds : DeclarationBounds) : String :=
  "{\"resources\":" ++ toString bounds.resources ++
    ",\"actions\":" ++ toString bounds.actions ++
    ",\"precedenceEdges\":" ++ toString bounds.precedenceEdges ++ "}"

private def provenanceJson (provenance : Provenance) : String :=
  "{\"source\":" ++ quote provenance.source ++
    ",\"compiler\":" ++ quote provenance.compiler ++ "}"

def canonicalJson (spec : ExperimentSpec) : String :=
  "{\"formatVersion\":" ++ quote spec.formatVersion ++
    ",\"regressionId\":" ++ quote spec.regressionId.value ++
    ",\"targetId\":" ++ quote spec.targetId.value ++
    ",\"modelIdentity\":" ++ quote spec.modelIdentity ++
    ",\"resources\":" ++ array (spec.resources.map (quote ∘ ResourceId.value)) ++
    ",\"resolvedSetup\":" ++ canonicalResolvedSetup spec.resolvedSetup ++
    ",\"actionAttempts\":" ++ array (spec.actionAttempts.map (quote ∘ ActionId.value)) ++
    ",\"projectedOutcomes\":" ++ array (spec.projectedOutcomes.map projectedOutcomeJson) ++
    ",\"ordering\":" ++ array (spec.ordering.map precedenceEdgeJson) ++
    ",\"expectedProperties\":" ++ array (spec.expectedProperties.map expectedPropertyJson) ++
    ",\"bounds\":" ++ boundsJson spec.bounds ++
    ",\"omissions\":" ++ array (spec.omissions.map quote) ++
    ",\"provenance\":" ++ provenanceJson spec.provenance ++ "}"

end Temporal.Experiment
