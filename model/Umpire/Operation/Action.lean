import Umpire.Operation.Canonical

/-!
Checked RPC templates bind exact admitted arguments while retaining the generated owner and
payload witness. Instances contain no outcome. Explicit finite samples and runtime scope are
separate; stronger abstractions are unsupported until preservation evidence has an owner.
-/
namespace Umpire.Operation
open Value Value.Encoding

variable {owner : RpcOwner} {Request Response Failure : Type}

/-- A stable authored Action identity attached to checked generated connectivity. -/
structure ActionTemplate (owner : RpcOwner) (Request Response Failure : Type) where
  private mk ::
  declaration : RpcDeclaration owner Request Response Failure
  identity : DefinitionId
  valid : identity.validate = .ok ()

/-- Admit the model's stable template ID without copying a method name or schema. -/
def ActionTemplate.check (declaration : RpcDeclaration owner Request Response Failure)
    (identity : DefinitionId) : Except Error (ActionTemplate owner Request Response Failure) :=
  match valid : identity.validate with
  | .ok () => .ok ⟨declaration, identity, valid⟩
  | .error _ => .error (.invalidIdentity identity)

variable {template : ActionTemplate owner Request Response Failure} {limits : Limits}

/-- Selecting an instance selects only exact request arguments, never a result alternative. -/
structure ActionInstance (template : ActionTemplate owner Request Response Failure) (limits : Limits) where
  arguments : Checked owner template.declaration.reference .request limits

/-- Equality at fixed owner, witness and schema is exactly canonical request-tree equality. -/
theorem ActionInstance.ext (a b : ActionInstance template limits)
    (same : a.arguments.value = b.arguments.value) : a = b := by
  rcases a with ⟨⟨a, _, _⟩⟩
  rcases b with ⟨⟨b, _, _⟩⟩
  cases same
  rfl

instance : DecidableEq (ActionInstance template limits) := fun a b =>
  if h : a.arguments.value = b.arguments.value then
    isTrue (a.ext b h)
  else isFalse (fun equal => h (congrArg (fun a => a.arguments.value) equal))

/-- Instance identity covers the complete selected schema, through its structural digest, and the
exact canonical value tree. -/
def ActionInstance.canonical (action : ActionInstance template limits) : String :=
  Canonical.key (sequence [textData template.identity.value,
    Canonical.rpcSchema (owner.schema template.declaration.reference), action.arguments.value])

/-- The existing Atom carrier is unchanged; this explicitly selected bridge uses version 1 keys. -/
def ActionInstance.modelValue (action : ActionInstance template limits) : ModelValue :=
  ModelValue.named template.identity action.canonical

/-- Whole-request dimension claims: fixed means one exact value; sampled means only listed values. -/
inductive ParameterCoverage where
  | fixed
  | sampled
  | abstracted (name : String)
  deriving BEq, DecidableEq, Repr

/-- Semantic value bounds contain no search or traversal work budget. -/
structure RuntimeBounds where
  depth : Nat
  bytes : Nat
  collection : Nat
  deriving BEq, DecidableEq, Repr

/-- Runtime admission is independently declared; schema scope never enlarges finite samples. -/
inductive RuntimeScope where
  | samplesOnly
  | schema (semanticBounds : RuntimeBounds)
  deriving BEq, DecidableEq, Repr

/-- Unsupported abstraction is a static admission error, distinct from search exhaustion. -/
inductive ParameterError where
  | value (error : Value.Error)
  | unsupportedAbstraction (name : String)
  | invalidFixedDomain
  | duplicateArgument
  | encodingCollision
  | outOfScope
  | runtimeResource (error : Value.Error)
  deriving BEq, DecidableEq, Repr

/-- Checked finite inputs, with exact identity agreement for their legacy-carrier bridge. -/
structure ParameterDomain (template : ActionTemplate owner Request Response Failure) (limits : Limits) where
  private mk ::
  actions : List (ActionInstance template limits)
  coverage : ParameterCoverage
  runtimeScope : RuntimeScope
  exactIdentity : ∀ a ∈ actions, ∀ b ∈ actions, a.modelValue = b.modelValue → a = b

/-- Admit exactly the authored list. No defaults, representative expansion or result selection occurs. -/
def ParameterDomain.check (template : ActionTemplate owner Request Response Failure) (limits : Limits)
    (arguments : List Raw) (coverage : ParameterCoverage) (runtimeScope : RuntimeScope) :
    Except ParameterError (ParameterDomain template limits) := do
  if let .abstracted name := coverage then throw (.unsupportedAbstraction name)
  if coverage == .fixed && arguments.length != 1 then throw .invalidFixedDomain
  let actions ← arguments.mapM fun raw => do
    let value ← Value.check owner template.declaration.reference .request limits raw
      |>.mapError ParameterError.value
    pure (ActionInstance.mk value)
  if !actions.Nodup then throw .duplicateArgument
  if exactIdentity : ∀ a ∈ actions, ∀ b ∈ actions, a.modelValue = b.modelValue → a = b then
    pure ⟨actions, coverage, runtimeScope, exactIdentity⟩
  else throw .encodingCollision

/-- Runtime resource admission and the declared semantic scope must both admit the exact value. -/
def ParameterDomain.admitRuntime (domain : ParameterDomain template limits) (resources : Limits)
    (raw : Raw) : Except ParameterError (ActionInstance template resources) := do
  let value ← Value.check owner template.declaration.reference .request resources raw
    |>.mapError ParameterError.runtimeResource
  match domain.runtimeScope with
    | .samplesOnly =>
      if !domain.actions.any (fun a => decide (a.arguments.value = value.value)) then
        throw .outOfScope
    | .schema semanticBounds =>
      let semanticLimits : Limits := {
        depth := semanticBounds.depth, bytes := semanticBounds.bytes,
        collection := semanticBounds.collection, work := resources.work }
      match Value.check owner template.declaration.reference .request semanticLimits value.value with
      | .ok _ => pure ()
      | .error _ => throw .outOfScope
  pure ⟨value⟩

/-- A bridge decoder searches only the exact admitted finite domain, returning the typed payload. -/
def ParameterDomain.decode (domain : ParameterDomain template limits) (atom : ModelValue) :
    Option (ActionInstance template limits) :=
  domain.actions.find? fun a => decide (a.modelValue = atom)

/-- The actual lookup decoder returns only a member denoting the exact input Atom. -/
theorem ParameterDomain.decode_sound (domain : ParameterDomain template limits)
    (atom : ModelValue) (action : ActionInstance template limits)
    (decoded : domain.decode atom = some action) :
    action ∈ domain.actions ∧ action.modelValue = atom := by
  exact ⟨List.mem_of_find?_eq_some decoded, by simpa using List.find?_some decoded⟩

/-- Exact typed identity makes finite bridge replay recover the same admitted request value. -/
theorem ParameterDomain.decode_encode (domain : ParameterDomain template limits)
    (action : ActionInstance template limits) (member : action ∈ domain.actions) :
    domain.decode action.modelValue = some action := by
  have present : ∃ a ∈ domain.actions, (decide (a.modelValue = action.modelValue)) = true :=
    ⟨action, member, by simp⟩
  obtain ⟨found, decoded⟩ := Option.isSome_iff_exists.mp (List.find?_isSome.mpr present)
  have sound := domain.decode_sound action.modelValue found decoded
  have equal := domain.exactIdentity found sound.1 action member sound.2
  simpa [decode, equal] using decoded

/-- Versioned domain meaning includes scope and all samples, separately from search work limits. -/
def ParameterDomain.canonical (domain : ParameterDomain template limits) : String :=
  let coverage := match domain.coverage with
    | .fixed => "fixed"
    | .sampled => "sampled"
    | .abstracted name => "unsupported-abstraction:" ++ name
  let runtime := match domain.runtimeScope with
    | .samplesOnly => Lean.Json.mkObj [("scope", .str "samples-only")]
    | .schema bounds => Lean.Json.mkObj [
        ("scope", .str "schema"), ("depth", Lean.toJson bounds.depth),
        ("bytes", Lean.toJson bounds.bytes), ("collection", Lean.toJson bounds.collection)]
  Lean.Json.compress <| .mkObj [
    ("formatVersion", .str "umpire-parameter-domain/v1"),
    ("template", .str template.identity.value),
    ("schema", .str (Canonical.key (Canonical.rpcSchema (owner.schema template.declaration.reference)))),
    ("dimension", .str "whole-request"), ("coverage", .str coverage),
    ("values", .arr ((domain.actions.map (·.canonical)).mergeSort.map Lean.Json.str).toArray),
    ("runtime", runtime)]

end Umpire.Operation
