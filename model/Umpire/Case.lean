import Umpire.Case.Contract

/-!
The versioned standalone Umpire Case envelope.

A Case pairs exactly one bounded Program with one Contract and retains the source definitions and
behavior fingerprints that produced the executable artifact.
-/

namespace Umpire.Case

/-- The source definition classes retained in Case provenance. -/
inductive CaseDefinitionKind where
  | setup
  | state
  | action
  | outcome
  | observation
  | relation
  | capability
  | property
  | query
  | behavior
  | target
  | compiler
  | provider
  | law
  | connector
  | kernel
  | experimentSpace
  | variationAxis
  | choice
  | fault
  | coverageGoal
  deriving BEq, DecidableEq, Repr

/-- One source Definition ID and the behavior fingerprint used for this Case. -/
structure CaseDefinitionBinding where
  definitionId : String
  behaviorFingerprint : String
  kind : CaseDefinitionKind
  deriving BEq, DecidableEq, Repr

/-- One explicit coverage or portability gap retained by the compiler. -/
structure CaseKnownGap where
  kind : CaseKnownGapKind
  code : String
  subject : Option String := none
  detail : Option String := none
  deriving BEq, DecidableEq, Repr

/-- Compiler and source provenance for one Case artifact. -/
structure CaseMetadata where
  producerId : String
  producerVersion : String := ""
  definitions : List CaseDefinitionBinding := []
  sources : List SourceLocation := []
  knownGaps : List CaseKnownGap := []
  deriving BEq, Repr

end Umpire.Case

namespace Umpire

/-- A versioned standalone pairing of exactly one bounded Program and one Contract. -/
structure Case where
  version : Case.FormatVersion
  caseId : String
  metadata : Case.CaseMetadata
  program : Case.Program
  contract : Case.Contract
  deriving BEq, Repr

end Umpire
