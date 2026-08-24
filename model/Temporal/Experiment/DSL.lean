import Std

namespace Temporal.Experiment

structure ResourceId where
  value : String
  deriving BEq, DecidableEq, Ord, Repr

structure ActionId where
  value : String
  deriving BEq, DecidableEq, Ord, Repr

structure PropertyId where
  value : String
  deriving BEq, DecidableEq, Ord, Repr

structure ModelId where
  value : String
  deriving BEq, DecidableEq, Ord, Repr

structure RegressionId where
  value : String
  deriving BEq, DecidableEq, Ord, Repr

structure DeclarationBounds where
  resources : Nat
  actions : Nat
  precedenceEdges : Nat
  deriving BEq, DecidableEq, Repr

structure PrecedenceEdge where
  before : ActionId
  after : ActionId
  deriving BEq, DecidableEq, Ord, Repr

structure ExpectedProperties where
  items : List PropertyId
  deriving BEq, DecidableEq, Repr

structure Regression where
  id : RegressionId
  target : ModelId
  resources : List ResourceId
  actionAttempts : List ActionId
  ordering : List PrecedenceEdge
  expectedProperties : ExpectedProperties
  bounds : DeclarationBounds
  omissions : List String
  deriving BEq, DecidableEq, Repr

structure ResolvedResource where
  id : ResourceId
  value : String
  deriving BEq, DecidableEq, Ord, Repr

structure ResolvedSetup where
  resources : List ResolvedResource
  deriving BEq, DecidableEq, Repr

structure ModelOutcome where
  value : String
  deriving BEq, DecidableEq, Ord, Repr

structure ResourceBinding where
  id : ResourceId
  value : String
  deriving BEq, DecidableEq, Repr

structure ActionProjection where
  id : ActionId
  project : ResolvedSetup → Option ModelOutcome

structure PropertyObservation where
  id : PropertyId
  contract : String
  deriving BEq, DecidableEq, Repr

structure Provenance where
  source : String
  compiler : String
  deriving BEq, DecidableEq, Repr

structure ModelTarget where
  id : ModelId
  declaration : String
  resources : List ResourceBinding
  actionProjections : List ActionProjection
  propertyObservations : List PropertyObservation
  provenance : Provenance

structure ProjectedOutcome where
  actionId : ActionId
  outcome : ModelOutcome
  deriving BEq, DecidableEq, Ord, Repr

structure ExpectedProperty where
  propertyId : PropertyId
  observationContract : String
  deriving BEq, DecidableEq, Ord, Repr

structure ExperimentSpec where
  formatVersion : String
  regressionId : RegressionId
  targetId : ModelId
  modelIdentity : String
  resources : List ResourceId
  resolvedSetup : ResolvedSetup
  actionAttempts : List ActionId
  projectedOutcomes : List ProjectedOutcome
  ordering : List PrecedenceEdge
  expectedProperties : List ExpectedProperty
  bounds : DeclarationBounds
  omissions : List String
  provenance : Provenance
  deriving BEq, DecidableEq, Repr

inductive CompileErrorKind where
  | missingIdentity
  | duplicateIdentity
  | emptyExpectations
  | unresolvedResource
  | unresolvedAction
  | unresolvedProperty
  | targetMismatch
  | unmappedAction
  | impossibleAction
  | duplicateOrdering
  | selfOrdering
  | cyclicOrdering
  | invalidBound
  | boundExceeded
  deriving BEq, DecidableEq, Repr

structure CompileError where
  kind : CompileErrorKind
  subject : String
  context : String
  deriving BEq, DecidableEq, Repr

end Temporal.Experiment
