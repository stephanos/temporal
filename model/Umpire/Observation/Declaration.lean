import Umpire.Core

/-!
Inert authored Observation vocabulary for mapping typed evidence into Model Values.

These declarations intentionally perform no checking, normalization, registration, or default
selection. `Umpire.Observation.Language` owns the checker boundary that interprets them.
-/

namespace Umpire

/-! Inert definitions and checked plans for mapping typed evidence into Model Values. -/

inductive ObservationValueType where
  | text
  | natural
  | boolean
  deriving BEq, DecidableEq, Ord, Repr

def ObservationValueType.name : ObservationValueType → String
  | .text => "text"
  | .natural => "natural"
  | .boolean => "boolean"

structure EvidenceFieldDeclaration where
  id : DefinitionId
  valueType : ObservationValueType
  deriving BEq, DecidableEq, Repr

structure EvidenceKindDeclaration where
  id : DefinitionId
  fields : List EvidenceFieldDeclaration
  deriving BEq, DecidableEq, Repr

structure EvidenceProfileDeclaration where
  id : DefinitionId
  source : SourceLocation
  version : Nat := 1
  kinds : List EvidenceKindDeclaration
  deriving BEq, DecidableEq, Repr

inductive EvidenceBoundUnit where
  | evidenceRecords
  deriving BEq, DecidableEq, Ord, Repr

def EvidenceBoundUnit.name : EvidenceBoundUnit → String
  | .evidenceRecords => "evidence-records"

/-- Evidence volume is an Observation boundary, never a semantic Property position. -/
structure EvidenceBound where
  value : Nat
  unit : EvidenceBoundUnit
  deriving BEq, DecidableEq, Ord, Repr

structure EvidenceFieldReference where
  kind : DefinitionId
  field : DefinitionId
  deriving BEq, DecidableEq, Ord, Repr

/-- One inert typed field handle shared by explicit Observation authoring records. -/
structure ObservationFieldSpec where
  kind : DefinitionId
  field : DefinitionId
  valueType : ObservationValueType
  deriving BEq, DecidableEq, Repr

/-- Project the field declaration used by an explicit evidence profile. -/
def ObservationFieldSpec.declaration (fieldSpec : ObservationFieldSpec) :
    EvidenceFieldDeclaration := {
  id := fieldSpec.field
  valueType := fieldSpec.valueType
}

/-- Project the field reference used by expressions and dispositions. -/
def ObservationFieldSpec.reference (fieldSpec : ObservationFieldSpec) : EvidenceFieldReference := {
  kind := fieldSpec.kind
  field := fieldSpec.field
}

structure ObservationOperator where
  name : String
  version : Nat
  deriving BEq, DecidableEq, Ord, Repr

structure DigestPolicyDeclaration where
  id : DefinitionId
  name : String
  version : Nat
  deriving BEq, DecidableEq, Repr

/-- Closed, portable expression data with fixed operators and no callback or recursive constructor. -/
inductive ObservationExpression where
  | text (value : String)
  | natural (value : Nat)
  | boolean (value : Bool)
  | field (reference : EvidenceFieldReference)
  | binding (id : DefinitionId)
  | normalize (operator : ObservationOperator) (operand : ObservationExpression)
  | present (operand : ObservationExpression)
  | equals (left right : ObservationExpression)
  | and (left right : ObservationExpression)
  | or (left right : ObservationExpression)
  | not (operand : ObservationExpression)
  | contributionMarker (operand : ObservationExpression)
  | digestToken (policy : DefinitionId) (operand : ObservationExpression)
  deriving BEq, DecidableEq, Repr

/-- Project the existing closed field expression without adding authoring semantics. -/
def ObservationFieldSpec.expression (fieldSpec : ObservationFieldSpec) : ObservationExpression :=
  .field fieldSpec.reference

/-- Inert authoring envelope that makes forbidden expression forms available for typed rejection. -/
inductive ObservationExpressionAuthoring where
  | portable (expression : ObservationExpression)
  | callback (name : String)
  | recursive (id : DefinitionId)
  deriving BEq, DecidableEq, Repr

instance : Coe ObservationExpression ObservationExpressionAuthoring :=
  ⟨ObservationExpressionAuthoring.portable⟩

inductive FieldDisposition where
  | retain
  | redact
  | hash (policy : Option DefinitionId)
  | reject
  deriving BEq, DecidableEq, Repr

def FieldDisposition.name : FieldDisposition → String
  | .retain => "retain"
  | .redact => "redact"
  | .hash _ => "hash"
  | .reject => "reject"

structure FieldDispositionDeclaration where
  field : EvidenceFieldReference
  disposition : FieldDisposition
  deriving BEq, DecidableEq, Repr

/-- Project a disposition declaration chosen explicitly by the Observation author. -/
def ObservationFieldSpec.disposition
    (fieldSpec : ObservationFieldSpec)
    (disposition : FieldDisposition) : FieldDispositionDeclaration := {
  field := fieldSpec.reference
  disposition
}

structure ObservationBinding where
  id : DefinitionId
  valueType : ObservationValueType
  expression : ObservationExpressionAuthoring
  deriving BEq, DecidableEq, Repr

structure ObservationRule where
  id : DefinitionId
  output : DefinitionId
  outputKind : DefinitionKind
  value : ObservationExpressionAuthoring
  condition : Option ObservationExpressionAuthoring := none
  deriving BEq, DecidableEq, Repr

structure ObservationOrdering where
  before : DefinitionId
  after : DefinitionId
  deriving BEq, DecidableEq, Repr

structure EvidenceClosureDeclaration where
  kind : DefinitionId
  deriving BEq, DecidableEq, Repr

/-- One authored profile mapping, including every compile-time structural and retention policy. -/
structure ObservationMappingDeclaration where
  id : DefinitionId
  source : SourceLocation
  version : Nat := 1
  profile : DefinitionId
  digestPolicies : List DigestPolicyDeclaration := []
  bindings : List ObservationBinding := []
  rules : List ObservationRule
  ordering : List ObservationOrdering := []
  closures : List EvidenceClosureDeclaration
  dispositions : List FieldDispositionDeclaration
  evidenceBound : EvidenceBound
  documentation : String := ""
  deriving BEq, DecidableEq, Repr

end Umpire
