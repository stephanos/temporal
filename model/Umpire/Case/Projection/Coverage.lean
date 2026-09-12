import Umpire.Case.Projection
import Umpire.Case.Projection.Coordinates

/-!
The modeled-fields to declared-Observations coverage map.

A model field operand names its value by structural coordinates: a payload root, the selected
operation schema, and the steps that reach the field. A projected Observation names its value by a
declared evidence field id. Neither identity implies the other, so a Case that lowers field
operands or keyed captures needs an explicit, checked correspondence between them. This module is
that correspondence.

One `CoverageEntry` binds one modeled `PropertyFieldPath` to one declared evidence field, and
records the scalar type both sides agree on. A retained occurrence is named by its evidence field
alone, exactly as the portable runtime names it, so the map is by field rather than by evidence
kind: several kinds may declare the same field, and each supplies the same modeled coordinates.
`check` admits a requested map only against the projection declaration it will run under: the field
must be declared and retained by at least one rule, its declared value type must be the modeled
field's scalar type, one retained scalar type must be declared for it everywhere, and both
directions must be injective so a retained occurrence is never aliased.

`Coverage.evidence` is the reverse direction, used by evidence-driven model replay: it rebuilds the
admitted projection one declared evidence value denotes at its covered coordinates. The rebuilt
payload is the smallest admitted value that supplies exactly that scalar there, and the projection
is read back through a real checked cursor, so the retained scalar and its structural denotation are
established by the same admission every authored projection uses. `check` rebuilds each entry once
from a probe value, so coordinates no payload could ever supply reject before any Driver I/O.

A request-rooted operand denotes the very arguments of the selected Action, which no projected
scalar reconstructs. Such coordinates are still coverable -- a history event does observe a
submitted request field, and the portable lowering names it by its declared evidence field -- but
they are not rebuildable, so an evidence-driven model replay skips them and rejects a clause that
reads one.

The rebuildable step vocabulary is field selection, optional presence, oneof selection and keyed map
lookup: each reaches the reported value without supplying anything else. A presence read, a repeated
element and a cardinality are outside it and reject with that reason, because the payload witnessing
them would have to carry sibling data no declared Observation reported. The rebuild itself is
`Projection.rebuild`, the rebuild use of the one coordinate walker in
`Umpire.Case.Projection.Coordinates`.
-/

namespace Umpire.Case.Projection

variable {Law : Law → Prop} {Setup State Action Outcome Fact : Type}
variable {target : CheckedModel Law Setup State Action Outcome Fact}

/-- One requested mapping from modeled field coordinates onto a declared Observation field. -/
structure FieldMapping where
  path : PropertyFieldPath
  field : DefinitionId
  deriving BEq, DecidableEq, Repr

/-- One admitted mapping, carrying the declared scalar type both sides agreed on. -/
structure CoverageEntry extends FieldMapping where
  valueType : ObservationValueType
  deriving BEq, DecidableEq, Repr

/-- A source-owned coverage rejection naming the modeled reference or declared field at fault. -/
structure CoverageError where
  subject : DefinitionId
  reason : String
  deriving BEq, DecidableEq, Repr

/-- The exact scalar a declared evidence value denotes at these modeled coordinates. Only the three
scalar kinds a projected Observation can carry are covered; the modeled integer kind decides the
admitted range, so an out-of-range natural rejects rather than narrowing. -/
private def coverageScalar (path : PropertyFieldPath) (value : EvidenceValue) :
    Except String Operation.Scalar :=
  match value, path.type with
  | .text text, .text => .ok (.text text)
  | .boolean flag, .boolean => .ok (.boolean flag)
  | .natural number, .integer kind =>
      let high : Int := Int.ofNat (2 ^ (kind.bits - if kind.signed then 1 else 0)) - 1
      if Int.ofNat number ≤ high then .ok (.integer kind (Int.ofNat number))
      else .error "declared evidence value is outside the modeled integer range"
  | _, _ => .error "declared evidence value does not match the modeled field type"

/-- Rebuild the admitted projection a declared evidence value denotes at covered coordinates. -/
private def coverageEvidence (limits : Value.Limits) (path : PropertyFieldPath)
    (value : EvidenceValue) (source : SourceLocation) : Except String PropertyFieldEvidence := do
  if request : path.root = .request then
    throw "a request operand denotes the selected Action, not projected evidence"
  else
    let scalar ← coverageScalar path value
    rebuild limits path request scalar source

/-- A checked correspondence between modeled field coordinates and declared Observation fields,
bound to the projection declaration whose evidence supplies them. -/
structure Coverage (plan : Checked target) where
  private mk ::
  entries : List CoverageEntry
  /-- Admission ceilings for rebuilding one covered payload; retained values are bounded by them. -/
  valueLimits : Value.Limits

/-- A Case that requests no field coverage keeps its existing meaning and reads no evidence value. -/
def Coverage.empty (plan : Checked target) : Coverage plan := ⟨[], ⟨0, 0, 0, 0⟩⟩

/-- Whether an evidence-driven model replay can rebuild this covered projection. A request operand
denotes the selected Action's own arguments, so no declared evidence scalar reconstructs it. -/
def CoverageEntry.rebuildable (entry : CoverageEntry) : Bool := entry.path.root != .request

/-- The scalar kind this covered field carries, in the shared portable numbering. -/
def CoverageEntry.scalarKind (entry : CoverageEntry) : Nat :=
  match entry.valueType with
  | .text => 1
  | .natural => 2
  | .boolean => 3

private def coverageSource : SourceLocation := { path := "", provenance := "coverage" }

private def typeAgrees (declared : ObservationValueType) (modeled : Operation.Singular) : Bool :=
  match declared, modeled with
  | .text, .text | .boolean, .boolean => true
  | .natural, .integer _ => true
  | _, _ => false

private def probeValue : ObservationValueType → EvidenceValue
  | .text => .text ""
  | .natural => .natural 0
  | .boolean => .boolean false

/-- Admit a requested coverage map against the projection declaration it will run under.
Every mapping must name a declared retained field of a declared kind whose declared scalar type is
the modeled field's own, both directions must be injective, and the payload rebuilding this
coverage performs at admission must already succeed for a probe value of the declared type. -/
def Coverage.check (plan : Checked target) (valueLimits : Value.Limits)
    (mappings : List FieldMapping) : Except CoverageError (Coverage plan) := do
  let declaration := plan.sourceDeclaration
  let retained := declaration.rules.flatMap fun rule =>
    rule.fields.filterMap fun field =>
      if field.2 == .retain then some (field.1.id, field.1.valueType) else none
  let mut entries : List CoverageEntry := []
  for mapping in mappings do
    if mapping.path.capture.isSome then
      throw ⟨mapping.field, "coverage names field coordinates, never a retained occurrence"⟩
    -- A retained occurrence is named by its evidence field alone, so one field declared at two
    -- scalar types has no single type a comparison could be checked against.
    let declared := ((retained.filter (·.1 == mapping.field)).map Prod.snd).eraseDups
    let [valueType] := declared
      | throw ⟨mapping.field, if declared.isEmpty then "unknown or unretained declared evidence field"
          else "declared evidence field has no single retained scalar type"⟩
    if !typeAgrees valueType mapping.path.type then
      throw ⟨mapping.field, "declared evidence type is not the modeled field type"⟩
    if entries.any (·.field == mapping.field) then
      throw ⟨mapping.field, "declared evidence field is already covered"⟩
    if entries.any (·.path == mapping.path) then
      throw ⟨mapping.field, "modeled field coordinates are already covered"⟩
    -- Rebuilding is probed once here, so coordinates no admitted payload could ever supply reject
    -- before any Driver I/O rather than at the first event that carries them.
    if mapping.path.root != .request then
      match coverageEvidence valueLimits mapping.path (probeValue valueType) coverageSource with
      | .error reason => throw ⟨mapping.field, reason⟩
      | .ok _ => pure ()
    entries := entries ++ [{ toFieldMapping := mapping, valueType }]
  pure ⟨entries, valueLimits⟩

/-- The admitted entry covering these modeled coordinates, if any. -/
def Coverage.entryOf? {plan : Checked target} (coverage : Coverage plan)
    (path : PropertyFieldPath) : Option CoverageEntry :=
  coverage.entries.find? (·.path == path)

/-- Rebuild every covered projection this event's declared fields supply. A field the event left
absent contributes nothing; a field it supplies at a value the modeled coordinates cannot denote
rejects the whole append. -/
def Coverage.evidence {plan : Checked target} (coverage : Coverage plan)
    (fields : List Field) : Except CoverageError (List PropertyFieldEvidence) := do
  let mut values : List PropertyFieldEvidence := []
  for entry in coverage.entries do
    -- Request coordinates are covered for portable lowering only; skipping them here keeps a
    -- lowering-only mapping from failing an unrelated evidence-driven replay.
    if entry.rebuildable then
      if let some field := fields.find? (·.id == entry.field) then
      if let some value := field.value then
        match coverageEvidence coverage.valueLimits entry.path value coverageSource with
        | .error reason => throw ⟨entry.field, reason⟩
        | .ok evidence => values := values ++ [evidence]
  pure values

end Umpire.Case.Projection
