import Testpilot.Authoring
import Umpire.Provenance
import Umpire.Operation.Canonical

/-!
# Case-local names and model value spellings

A checked Model names everything by dotted Definition IDs, and a parameterized model value is a
structural key thousands of characters long. The runtime needs neither: it only compares names and
value strings for equality. `localize` is where a generated Case trades them for short Case-local
names and spellings, and derives the provenance rows that say what each stands for.

**Local names.** The namespace is every Definition ID the Program and Contract name: each Contract
rule id; in the correlated contract its projection, operation field, scope fields, sources, evidence
kinds, field policies, captures, rule ids and the definition of every model value; the Program's
evidence lift rules; and every model value that is itself a Definition ID. A local name is the last
dotted segment of its Definition ID, extended leftward one segment at a time until no other
Definition ID of the Case has the same suffix, so `action.schedule` and `state.schedule` stay apart
where `schedule` alone would merge them.

**Spellings.** A model value keeps its declared spelling. A value that is a Definition ID takes that
ID's local name, and a structural key (`Umpire.Operation.Canonical.isKey`) takes the last segment of
its definition's Definition ID. When two encodings of one definition would share a spelling, each
takes a disambiguator: the shortest prefix, at least eight characters, of its encoding's SHA-256 that
no other encoding of that group shares. The text a correlated step condition compares a definition
with is one of that definition's encodings, so the comparison is renamed with the values it tests.

**Invariant.** Renaming is injective: no local name stands for two Definition IDs, no Definition ID
has two local names, and no spelling stands for two encodings of one definition. Every equality the
runtime tests between names or values therefore has the answer it had before renaming, so no
transition merges and no Verdict moves. `localize` checks the invariant on the table it derives and
rejects the Case, naming both sides, rather than emit a renaming that could merge them.

**Rows.** A `LocalName` row maps each local name that differs from its Definition ID; a name with no
row is its own Definition ID. A `ModelValueFingerprint` row records each value whose spelling is not
the one its name alone gives it, with the SHA-256 of its encoding.
-/

namespace Umpire.Case.LocalNames

open Umpire
open temporal.server.api.testpilot.v1 hiding LocalName ModelValueFingerprint

/-- A renaming that would merge what the Case keeps apart, naming both sides. -/
inductive Error where
  /-- One local name stands for two Definition IDs. -/
  | sharedName (localName first second : String)
  /-- One Definition ID has two local names. -/
  | splitDefinition (definitionId first second : String)
  /-- One spelling stands for two encodings of the same definition. -/
  | ambiguousSpelling (definitionId spelling first second : String)
  deriving BEq, DecidableEq, Repr

/-- The last `count` dotted segments of `id`, or all of `id` when it has fewer. -/
def suffix (id : String) (count : Nat) : String :=
  let segments := id.splitOn "."
  ".".intercalate (segments.drop (segments.length - count))

/-- The local name of `id` in a Case whose namespace is `ids`: its shortest dotted suffix no other
member of `ids` shares at that length, or `id` itself when every suffix is shared. -/
def localName (ids : List String) (id : String) : String :=
  let others := ids.filter (· != id)
  let unique := fun count => others.all fun other => suffix other count != suffix id count
  match (List.range (id.splitOn ".").length).find? (fun extra => unique (extra + 1)) with
  | some extra => suffix id (extra + 1)
  | none => id

/-- Check that a name table is injective both ways. -/
def checkNames (names : List Provenance.LocalName) : Except Error Unit := do
  for name in names do
    if let some other := names.find? fun other =>
        other.localName == name.localName && other.definitionId != name.definitionId then
      throw (.sharedName name.localName name.definitionId other.definitionId)
    if let some other := names.find? fun other =>
        other.definitionId == name.definitionId && other.localName != name.localName then
      throw (.splitDefinition name.definitionId name.localName other.localName)

private def isDefinitionId (encoding : String) : Bool := (DefinitionId.of encoding).isNamespaced

/-- The spelling a name table alone gives an encoding: a Definition ID's local name, or the encoding
itself. A fingerprint row records every other spelling. -/
private def nameSpelling (name : String → String) (encoding : String) : String :=
  if isDefinitionId encoding then name encoding else encoding

/-- The spelling an encoding takes before disambiguation. -/
private def baseSpelling (name : String → String) (definitionId encoding : String) : String :=
  if Operation.Canonical.isKey encoding then suffix definitionId 1 else nameSpelling name encoding

/-- The shortest prefix of `fingerprint`, at least eight characters long, that no other member of
`fingerprints` starts with. -/
private def disambiguator (fingerprints : List String) (fingerprint : String) : String :=
  let prefixOf := fun (text : String) (length : Nat) => String.ofList (text.toList.take length)
  let unique := fun length => fingerprints.all fun other =>
    other == fingerprint || prefixOf other length != prefixOf fingerprint length
  match (List.range (fingerprint.length - 7)).find? (fun extra => unique (extra + 8)) with
  | some extra => prefixOf fingerprint (extra + 8)
  | none => fingerprint

/-- One encoding of a definition, the spelling the Case gives it, and the SHA-256 of the encoding
when that spelling is not the one its name alone gives it. -/
structure Spelling where
  encoding : String
  spelling : String
  fingerprint : Option String
  deriving BEq, DecidableEq, Repr

/-- Spell every encoding of one definition, disambiguating encodings that share a base spelling and
rejecting a spelling that still stands for two encodings. -/
def spell (name : String → String) (definitionId : String) (encodings : List String) :
    Except Error (List Spelling) := do
  let bases := encodings.eraseDups.map fun encoding =>
    (encoding, baseSpelling name definitionId encoding)
  let colliding := fun (base : String) => (bases.filter (·.2 == base)).length > 1
  let fingerprints := bases.filterMap fun (encoding, base) =>
    if colliding base || base != nameSpelling name encoding then
      some (encoding, Fingerprint.sha256Hex encoding)
    else none
  let fingerprintOf := fun encoding => (fingerprints.find? (·.1 == encoding)).map (·.2)
  let spelled := bases.map fun (encoding, base) =>
    let spelling :=
      match fingerprintOf encoding with
      | some fingerprint =>
          if colliding base then
            base ++ "-" ++ disambiguator
              ((bases.filter (·.2 == base)).filterMap (fingerprintOf ·.1)) fingerprint
          else base
      | none => base
    { encoding, spelling
      fingerprint := if spelling == nameSpelling name encoding then none else fingerprintOf encoding }
  for entry in spelled do
    if let some other := spelled.find? fun other =>
        other.spelling == entry.spelling && other.encoding != entry.encoding then
      throw (.ambiguousSpelling definitionId entry.spelling entry.encoding other.encoding)
  pure spelled

/-- The Case-local name `provenance` gives a Definition ID: the name its `LocalName` row maps, or the
Definition ID itself when no row maps it. -/
def nameIn (provenance : CaseProvenance) (definitionId : String) : String :=
  ((provenance.local_names.find? (·.definition_id == definitionId)).map (·.local_name)).getD
    definitionId

/-! ### Traversal

One traversal visits every name and model value position of a Case. It runs twice: once to collect
the namespace and each definition's encodings, and once to rewrite them through the derived table. -/

/-- What the traversal does at a name position and at a model value position. A value is visited
with its definition's Definition ID before that ID is renamed. -/
structure Visitor (m : Type → Type) where
  name : String → m String
  value : (definitionId encoding : String) → m String

variable {m : Type → Type} [Monad m] (visitor : Visitor m)

private def modelValue (value : temporal.server.api.testpilot.v1.ModelValue) :
    m temporal.server.api.testpilot.v1.ModelValue := do
  let spelling ← visitor.value value.definition_id value.value
  pure { value with definition_id := ← visitor.name value.definition_id, value := spelling }

private def optionalValue :
    Option temporal.server.api.testpilot.v1.ModelValue →
      m (Option temporal.server.api.testpilot.v1.ModelValue)
  | none => pure none
  | some value => some <$> modelValue visitor value

/-- The definition a correlated step reference names, when the expression is one. -/
private def stepDefinition : Option Expression → Option String
  | some { expression := some (.reference { reference := some (.correlated_step step), .. }), .. } =>
      some step.definition_id
  | _ => none

/-- The text of a text literal expression. -/
private def textLiteral : Option Expression → Option String
  | some { expression := some (.literal { value := some (.text_value text), .. }), .. } => some text
  | _ => none

private def replaceText (text : String) : Option Expression → Option Expression
  | some operand =>
      match operand.expression with
      | some (.literal literal) =>
          some { operand with expression := some (.literal { literal with value := some (.text_value text) }) }
      | _ => some operand
  | none => none

mutual
private def expression : Expression → m Expression
  | ⟨none, unknown⟩ => pure ⟨none, unknown⟩
  | ⟨some arm, unknown⟩ => do pure ⟨some (← expressionArm arm), unknown⟩

private def expressionArm : Expression.expression_Type → m Expression.expression_Type
  | .reference reference => do
      let arm ← match reference.reference with
        | some (.correlated_step step) =>
            pure (some (.correlated_step { step with definition_id := ← visitor.name step.definition_id }))
        | some (.evidence_field_id id) => pure (some (.evidence_field_id (← visitor.name id)))
        | some (.correlated_capture capture) =>
            pure (some (.correlated_capture { capture with capture_id := ← visitor.name capture.capture_id }))
        | some (.model_value value) => pure (some (.model_value (← modelValue visitor value)))
        | other => pure other
      pure (.reference { reference with reference := arm })
  | .path ⟨operand, path, unknown⟩ => do pure (.path ⟨← operand? operand, path, unknown⟩)
  | .present ⟨operand, unknown⟩ => do pure (.present ⟨← operand? operand, unknown⟩)
  | .not ⟨operand, unknown⟩ => do pure (.not ⟨← operand? operand, unknown⟩)
  | .compare ⟨operator, left, right, unknown⟩ => do
      -- A step condition's text is an encoding of the step's definition, so it takes that
      -- encoding's spelling rather than staying the text the renamed values no longer carry.
      let right ← match stepDefinition left, textLiteral right with
        | some definitionId, some text => do
            pure (replaceText (← visitor.value definitionId text) right)
        | _, _ => operand? right
      pure (.compare ⟨operator, ← operand? left, right, unknown⟩)
  | .all ⟨⟨operands⟩, unknown⟩ => do pure (.all ⟨⟨← operandList operands⟩, unknown⟩)
  | .any ⟨⟨operands⟩, unknown⟩ => do pure (.any ⟨⟨← operandList operands⟩, unknown⟩)
  | .literal literal => pure (.literal literal)

private def operand? : Option Expression → m (Option Expression)
  | none => pure none
  | some operand => some <$> expression operand

private def operandList : List Expression → m (List Expression)
  | [] => pure []
  | operand :: rest => do pure ((← expression operand) :: (← operandList rest))
end

private def transition (row : CorrelatedTransition) : m CorrelatedTransition := do
  pure { row with
    prior_state := ← optionalValue visitor row.prior_state
    action := ← optionalValue visitor row.action
    state := ← optionalValue visitor row.state
    outcome := ← optionalValue visitor row.outcome
    facts := ← row.facts.mapM (modelValue visitor) }

private def correlatedRule (rule : CorrelatedRule) : m CorrelatedRule := do
  pure { rule with
    rule_id := ← visitor.name rule.rule_id
    trigger := ← operand? visitor rule.trigger
    response := ← operand? visitor rule.response
    captures := ← rule.captures.mapM fun capture => do
      pure { capture with
        capture_id := ← visitor.name capture.capture_id
        field_id := ← visitor.name capture.field_id }
    correlation := ← operand? visitor rule.correlation }

private def correlatedContract (contract : CorrelatedContract) : m CorrelatedContract := do
  pure { contract with
    projection_id := ← visitor.name contract.projection_id
    scope_fields := ← contract.scope_fields.mapM visitor.name
    operation_field := ← visitor.name contract.operation_field
    sources := ← contract.sources.mapM visitor.name
    initial_state := ← optionalValue visitor contract.initial_state
    transitions := ← contract.transitions.mapM (transition visitor)
    projection_rules := ← contract.projection_rules.mapM fun rule => do
      pure { rule with
        kind := ← visitor.name rule.kind
        submission := ← optionalValue visitor rule.submission
        outputs := ← rule.outputs.mapM (transition visitor)
        fields := ← rule.fields.mapM fun field => do
          pure { field with field_id := ← visitor.name field.field_id } }
    rules := ← contract.rules.mapM (correlatedRule visitor) }

private def namedExpression (named : NamedExpression) : m NamedExpression := do
  pure { named with field_id := ← visitor.name named.field_id }

/-- The evidence lift rules of one instruction; every other instruction names no Definition ID. -/
private def instruction (node : InstructionNode) : m InstructionNode := do
  let some wrapped := node.instruction | pure node
  let some (.invoke_rpc invoke) := wrapped.instruction | pure node
  let reads ← invoke.response_reads.mapM fun read => do
    pure { read with targets := ← read.targets.mapM fun target => do
      match target.target with
      | some (.correlated_evidence lift) =>
          pure { target with target := some (.correlated_evidence { lift with
            rules := ← lift.rules.mapM fun rule => do
              pure { rule with
                evidence_source := ← visitor.name rule.evidence_source
                kind := ← visitor.name rule.kind
                scope := ← rule.scope.mapM (namedExpression visitor)
                fields := ← rule.fields.mapM (namedExpression visitor) } }) }
      | _ => pure target }
  pure { node with instruction := some { wrapped with instruction := some (.invoke_rpc { invoke with
    response_reads := reads }) } }

private def program (value : Program) : m Program := do
  pure { value with
    entrypoints := ← value.entrypoints.mapM fun entrypoint => do
      pure { entrypoint with instructions := ← entrypoint.instructions.mapM (instruction visitor) }
    cleanup := ← value.cleanup.mapM fun cleanup => do
      pure { cleanup with instructions := ← cleanup.instructions.mapM (instruction visitor) } }

/-- Visit every name and model value position of a Case's Program, Contract rules and correlated
contract, in that order. -/
private def visitCase (value : Program) (rules : Array ContractRule)
    (capability : Option CorrelatedContract) :
    m (Program × Array ContractRule × Option CorrelatedContract) := do
  let value ← program visitor value
  let rules ← rules.mapM fun rule => do pure { rule with rule_id := ← visitor.name rule.rule_id }
  pure (value, rules, ← capability.mapM (correlatedContract visitor))

/-! ### Localization -/

/-- A Case's Program and Contract under Case-local names and spellings, with the provenance rows
that map them back. -/
structure Localized where
  program : Program
  rules : Array ContractRule
  capability : Option CorrelatedContract
  localNames : List Provenance.LocalName
  modelValueFingerprints : List Provenance.ModelValueFingerprint

private structure Collected where
  ids : Array String := #[]
  values : Array (String × String) := #[]

private def collector : Visitor (StateM Collected) where
  name id := modifyGet fun collected => (id, { collected with ids := collected.ids.push id })
  value definitionId encoding := modifyGet fun collected =>
    (encoding, { collected with
      ids := if isDefinitionId encoding then collected.ids.push encoding else collected.ids
      values := collected.values.push (definitionId, encoding) })

/-- Rename a Case's Program, Contract rules and correlated contract to Case-local names and
spellings, rejecting a renaming that would merge two Definition IDs or two encodings. -/
def localize (value : Program) (rules : Array ContractRule)
    (capability : Option CorrelatedContract) : Except Error Localized := do
  let collected := ((visitCase collector value rules capability).run {}).2
  let ids := collected.ids.toList.eraseDups
  let table := ids.map fun id => ({ localName := localName ids id, definitionId := id } :
    Provenance.LocalName)
  checkNames table
  let name := fun id => ((table.find? (·.definitionId == id)).map (·.localName)).getD id
  let values := collected.values.toList
  let spellings ← (values.map (·.1)).eraseDups.mapM fun definitionId => do
    pure (definitionId, ← spell name definitionId
      ((values.filter (·.1 == definitionId)).map (·.2)))
  let spellingOf := fun definitionId encoding =>
    (((spellings.find? (·.1 == definitionId)).bind fun entry =>
      entry.2.find? (·.encoding == encoding)).map (·.spelling)).getD encoding
  let renamer : Visitor Id := { name, value := spellingOf }
  let (value, rules, capability) := visitCase renamer value rules capability
  pure {
    program := value, rules, capability
    localNames := table.filter fun row => row.localName != row.definitionId
    modelValueFingerprints := spellings.flatMap fun (definitionId, entries) =>
      entries.filterMap fun entry => entry.fingerprint.map fun fingerprint =>
        { localName := name definitionId, spelling := entry.spelling, fingerprint } }

end Umpire.Case.LocalNames
