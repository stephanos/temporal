import Umpire.Command

/-!
# What every Temporal Model declaration shares

One declaration, read by the Model commands: Temporal hangs its Definition IDs off the `temporal`
root, treats `Temporal.Feature` as scaffolding rather than semantic family, and carries the Known
Gaps below on every Query.

The Known Gaps are the slice's, not the project's; they live here only until a Model file can
author its own.
-/

namespace Temporal.Case

open Umpire

def cancellationKnownGap : KnownGap := {
  kind := .capability
  code := DefinitionId.of "temporal.nexus.success.known-gap.cancellation"
  subject := some (DefinitionId.of "temporal.nexus.success.property.cancellationResolves")
  detail := some "Operation-correlated Nexus cancellation is unsupported by the success slice."
}

def operationCorrelatedProgressKnownGap : KnownGap := {
  kind := .capability
  code := DefinitionId.of "temporal.nexus.success.known-gap.operation-correlated-progress"
  subject := some (DefinitionId.of "temporal.nexus.success.property.cancellationResolves")
  detail := some "Operation-correlated progress counting is unsupported by the success slice."
}

def completionKnownGaps : Except KnownGapError KnownGapSet :=
  KnownGapSet.checkCanonical [cancellationKnownGap, operationCorrelatedProgressKnownGap]

end Temporal.Case

model_conventions root "temporal" under Temporal.Feature
  gaps Temporal.Case.completionKnownGaps
