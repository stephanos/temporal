import Lean.Elab.Term
import Umpire.Model.Check

/-!
Target syntax capture and TermElabM integration. The frontend calls pure admission and locates
its typed diagnostics using captured occurrences, retaining the fallback when no syntax matches.
-/

namespace Umpire

/-- Compiler-only syntax is paired with the pure occurrence row and never enters checked data. -/
structure CapturedSourceRef where
  occurrence : SourceRef
  reference : Lean.Syntax

/-- Capture one syntax occurrence as a nonsemantic source-span/ordinal token. -/
def captureSourceRef
    (reference : Lean.Syntax)
    (definitionId : DefinitionId)
    (path : SourceRefPath)
    (localOrdinal : Nat) : Lean.Elab.Term.TermElabM CapturedSourceRef := do
  let fileMap ← Lean.getFileMap
  let sourcePath ← Lean.getFileName
  let startOffset := reference.getPos?.getD 0
  let endOffset := reference.getTailPos?.getD startOffset
  let startPosition := fileMap.toPosition startOffset
  let endPosition := fileMap.toPosition endOffset
  pure {
    occurrence := {
      id := {
        sourcePath
        line := startPosition.line
        column := startPosition.column
        endLine := endPosition.line
        endColumn := endPosition.column
        localOrdinal
      }
      definitionId
      path
    }
    reference
  }

/-- Run the ordinary adapter once and emit its typed failure at the selected captured occurrence. -/
def elabModel
    (authored : DraftModel LawStatement Setup State Action Outcome Observation)
    (captured : List CapturedSourceRef) :
    Lean.Elab.Term.TermElabM
      (CheckedModel LawStatement Setup State Action Outcome Observation) := do
  let authored := authored.withOccurrences (captured.map CapturedSourceRef.occurrence)
  match checkModel authored with
  | .ok checked => pure checked
  | .error diagnostic =>
      let message := s!"target authoring failed: {canonicalLocatedErrorJson diagnostic}"
      match captured.find? (fun item => item.occurrence.id == diagnostic.offending) with
      | some item => Lean.throwErrorAt item.reference message
      | none => Lean.throwError message

end Umpire
