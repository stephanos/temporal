import Umpire.Case.Contract
import Umpire.Core

/-!
Umpire-owned provenance for generated Testpilot Cases.

The protobuf schema owns the Case, Program, and Contract structures. Umpire retains only its
producer-specific definitions, fingerprints, sources, and Known Gaps before encoding them into
opaque Testpilot provenance bytes.
-/

namespace Umpire.Case

/-- The closed categories of source-model incompleteness retained in Umpire provenance. -/
inductive CaseKnownGapKind where
  | capabilityContract
  | input
  | interpretation
  | claim
  deriving BEq, DecidableEq, Repr

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

/-- Compatibility name for the generated Testpilot Case.

Remove this alias when downstream imports use the generated protocol namespace directly.
-/
abbrev Case := temporal.server.api.testpilot.v1.Case

end Umpire
