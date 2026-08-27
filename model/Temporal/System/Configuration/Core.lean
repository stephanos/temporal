import Temporal.DynamicConfig
import Umpire.Core

namespace Temporal.System.Configuration

open _root_.Umpire
open Temporal.DynamicConfig

/-!
# Temporal dynamic-configuration semantics

Handwritten product meaning over the generated Temporal dynamic-config catalog.

The generated catalog records structural setting metadata. This module adds the owner-authored
classification and typed interpretation required to use that metadata safely. A `ConfigUse` seals
those checks for one consumer and context, `checkConfigOverride` binds overrides to checked uses,
and `resolveConfigView` produces an immutable snapshot whose values can only be read through their
originating typed uses.

Resolution is deterministic: uses and overrides are canonicalized before evaluation, constrained
values follow the generated precedence policy, and each stored entry retains the catalog, setting,
interpretation, context, sampling, and change-effect provenance checked by `ConfigView.read`.
-/

/-- Product behavior that a dynamic setting can influence. -/
inductive ImpactClass where
  | feature
  | validation
  | externallyVisibleSemantics
  | timing
  | topology
  | performance
  | observability
  deriving BEq, DecidableEq, Ord, Repr

/-- Owner-authored impact metadata bound to one generated setting identity. -/
structure SettingClassification where
  key : String
  settingIdentity : String
  impacts : List ImpactClass
  deriving BEq, DecidableEq, Repr

/-- The lifecycle boundary at which a consumer observes a setting value. -/
inductive SamplingPoint where
  | liveAccess
  | entityCreation
  | request
  | task
  | processStartup
  deriving BEq, DecidableEq, Ord, Repr

/-- When a configuration change can affect modeled behavior. -/
inductive ChangeEffect where
  | nextRead
  | newEntitiesOnly
  | restartRequired
  deriving BEq, DecidableEq, Ord, Repr

/-- Stable categories for configuration validation and resolution failures. -/
inductive ConfigErrorKind where
  | unknownKey
  | unclassifiedKey
  | emptyClassification
  | missingInterpretation
  | incompatibleInterpretation
  | schemaMismatch
  | defaultDrift
  | illegalConstraints
  | duplicateConstraints
  | missingContext
  | opaqueDefaultSelected
  | malformedUse
  | duplicateUse
  | unknownUse
  | interpretationFailure
  | fixtureMismatch
  deriving BEq, DecidableEq, Ord, Repr

/-- Return the stable diagnostic name for an error category. -/
def ConfigErrorKind.name : ConfigErrorKind → String
  | .unknownKey => "unknown-key"
  | .unclassifiedKey => "unclassified-key"
  | .emptyClassification => "empty-classification"
  | .missingInterpretation => "missing-interpretation"
  | .incompatibleInterpretation => "incompatible-interpretation"
  | .schemaMismatch => "schema-mismatch"
  | .defaultDrift => "default-drift"
  | .illegalConstraints => "illegal-constraints"
  | .duplicateConstraints => "duplicate-constraints"
  | .missingContext => "missing-context"
  | .opaqueDefaultSelected => "opaque-default-selected"
  | .malformedUse => "malformed-use"
  | .duplicateUse => "duplicate-use"
  | .unknownUse => "unknown-use"
  | .interpretationFailure => "interpretation-failure"
  | .fixtureMismatch => "fixture-mismatch"

/-- A deterministic configuration diagnostic with canonicalized related identities. -/
structure ConfigError where
  kind : ConfigErrorKind
  useId : DefinitionId
  key : String
  offendingValue : String
  relatedIdentities : List String
  deriving BEq, DecidableEq, Repr

private def stringLe (left right : String) : Bool := decide (left ≤ right)

private def canonicalStrings (values : List String) : List String :=
  values.mergeSort stringLe |>.eraseDups

private def configError
    (kind : ConfigErrorKind)
    (useId : DefinitionId)
    (key offendingValue : String)
    (relatedIdentities : List String := []) : ConfigError := {
  kind
  useId := if useId.value == "" then DefinitionId.of "umpire.config.anonymous" else useId
  key
  offendingValue
  relatedIdentities := canonicalStrings relatedIdentities
}

/-- An owner-supplied canonical value replacing one expected opaque generated default. -/
structure OpaqueDefaultReplacement where
  expected : OpaqueDefault
  value : CanonicalValue
  deriving BEq, DecidableEq, Repr

/-- Owner-authored metadata and decoder giving a generated setting typed product meaning. -/
structure ConfigInterpretation (α : Type) where
  key : String
  expectedSettingIdentity : String
  expectedSchema : ValueSchema
  expectedDefault : SettingDefault
  opaqueReplacement : Option OpaqueDefaultReplacement := none
  semanticDigest : String
  decode : CanonicalValue → Except String α

/-- The unchecked declaration of one typed setting use at an exact lookup context. -/
structure ConfigUseRequest (α : Type) where
  id : DefinitionId
  key : String
  context : ExactConstraints
  samplingPoint : SamplingPoint
  changeEffect : ChangeEffect
  interpretation : Option (ConfigInterpretation α)

/-- Owner-authored meaning for one generated setting, independent of a concrete lookup context. -/
structure ConfigUseDefinition (α : Type) where
  id : DefinitionId
  classification : SettingClassification
  contextPolicy : PrecedencePolicy
  samplingPoint : SamplingPoint
  changeEffect : ChangeEffect
  interpretation : ConfigInterpretation α

private structure ConfigUsePayload (α : Type) where
  id : DefinitionId
  setting : Setting
  classification : SettingClassification
  interpretation : ConfigInterpretation α
  context : ExactConstraints
  samplingPoint : SamplingPoint
  changeEffect : ChangeEffect

/-- A validated typed use whose construction and decoder remain sealed in this module. -/
structure ConfigUse (α : Type) where
  private mk ::
  private payload : ConfigUsePayload α

/-- Return the stable owner declaration identifying a checked use. -/
def ConfigUse.id (use : ConfigUse α) : DefinitionId :=
  use.payload.id

/-- Return the generated setting key bound to a checked use. -/
def ConfigUse.key (use : ConfigUse α) : String :=
  use.payload.setting.key

private def ConfigUse.setting (use : ConfigUse α) : Setting :=
  use.payload.setting

private def ConfigUse.classification (use : ConfigUse α) : SettingClassification :=
  use.payload.classification

private def ConfigUse.interpretation (use : ConfigUse α) : ConfigInterpretation α :=
  use.payload.interpretation

private def ConfigUse.matchesSetting (use : ConfigUse α) (setting : Setting) : Bool :=
  setting == use.setting &&
    use.classification.key == setting.key &&
    use.classification.settingIdentity == setting.identity &&
    use.classification.impacts != [] &&
    use.interpretation.key == setting.key &&
    use.interpretation.expectedSettingIdentity == setting.identity &&
    use.interpretation.expectedSchema == setting.schema &&
    use.interpretation.expectedDefault == setting.defaultValue &&
    use.interpretation.semanticDigest != ""

/-- Return the exact lookup context validated for a checked use. -/
def ConfigUse.context (use : ConfigUse α) : ExactConstraints :=
  use.payload.context

/-- Return the lifecycle boundary at which a checked use samples its value. -/
def ConfigUse.samplingPoint (use : ConfigUse α) : SamplingPoint :=
  use.payload.samplingPoint

/-- Return when changes to a checked use can affect modeled behavior. -/
def ConfigUse.changeEffect (use : ConfigUse α) : ChangeEffect :=
  use.payload.changeEffect

private structure CheckedConfigUseDefinitionPayload (α : Type) where
  template : ConfigUse α
  contextPolicy : PrecedencePolicy

/-- A checked owner definition that can instantiate typed uses without repeating owner metadata. -/
structure CheckedConfigUseDefinition (α : Type) where
  private mk ::
  private payload : CheckedConfigUseDefinitionPayload α

/-- Stable, decoded-type-independent metadata for a checked owner definition. -/
structure ConfigUseDefinitionMetadata where
  id : DefinitionId
  key : String
  settingIdentity : String
  impacts : List ImpactClass
  contextPolicy : PrecedencePolicy
  samplingPoint : SamplingPoint
  changeEffect : ChangeEffect
  interpretationDigest : String
  deriving BEq, DecidableEq, Repr

/-- Return the stable metadata sealed into a checked owner definition. -/
def CheckedConfigUseDefinition.metadata
    (definition : CheckedConfigUseDefinition α) : ConfigUseDefinitionMetadata := {
  id := definition.payload.template.id
  key := definition.payload.template.key
  settingIdentity := definition.payload.template.setting.identity
  impacts := definition.payload.template.classification.impacts
  contextPolicy := definition.payload.contextPolicy
  samplingPoint := definition.payload.template.samplingPoint
  changeEffect := definition.payload.template.changeEffect
  interpretationDigest := definition.payload.template.interpretation.semanticDigest
}

/--
An existential wrapper allowing checked owner definitions with different decoded types to share a
registry.
-/
inductive AnyCheckedConfigUseDefinition where
  | of {α : Type} (definition : CheckedConfigUseDefinition α)

namespace AnyCheckedConfigUseDefinition

/-- Return the stable metadata for a wrapped checked owner definition. -/
def metadata : AnyCheckedConfigUseDefinition → ConfigUseDefinitionMetadata
  | .of definition => definition.metadata

/-- Return the stable owner declaration identifying a wrapped checked definition. -/
def id (definition : AnyCheckedConfigUseDefinition) : DefinitionId :=
  definition.metadata.id

/-- Return the generated setting key bound to a wrapped checked definition. -/
def key (definition : AnyCheckedConfigUseDefinition) : String :=
  definition.metadata.key

end AnyCheckedConfigUseDefinition

/-- An existential wrapper allowing checked uses with different decoded types to share a view. -/
inductive AnyConfigUse where
  | of {α : Type} (use : ConfigUse α)

namespace AnyConfigUse

/-- Return the stable owner declaration identifying a wrapped use. -/
def id : AnyConfigUse → DefinitionId
  | .of use => use.id

/-- Return the generated setting key bound to a wrapped use. -/
def key : AnyConfigUse → String
  | .of use => use.key

end AnyConfigUse

private structure ConfigOverridePayload where
  key : String
  constraints : ExactConstraints
  value : CanonicalValue
  deriving BEq, DecidableEq, Repr

/-- A checked canonical override whose key and decoder stay bound to one typed use. -/
structure ConfigOverride where
  private mk ::
  private payload : ConfigOverridePayload
  deriving BEq, DecidableEq, Repr

private def ConfigOverride.key (override : ConfigOverride) : String :=
  override.payload.key

private def ConfigOverride.constraints (override : ConfigOverride) : ExactConstraints :=
  override.payload.constraints

private def ConfigOverride.value (override : ConfigOverride) : CanonicalValue :=
  override.payload.value

private def rawConfigOverride
    (key : String)
    (constraints : ExactConstraints)
    (value : CanonicalValue) : ConfigOverride :=
  .mk { key, constraints, value }

/-- The source selected for a resolved configuration value. -/
inductive ResolutionSource where
  | override
  | constrainedDefault
  | simpleDefault
  | opaqueReplacement
  deriving BEq, DecidableEq, Ord, Repr

/-- Auditable provenance for one value in a resolved configuration view. -/
structure ResolvedEntry where
  private mk ::
  useId : DefinitionId
  key : String
  source : ResolutionSource
  matchedConstraints : ExactConstraints
  context : ExactConstraints
  catalogDigest : String
  settingDigest : String
  interpretationDigest : String
  samplingPoint : SamplingPoint
  changeEffect : ChangeEffect
  deriving BEq, DecidableEq, Repr

private def ResolvedEntry.matchesUse (entry : ResolvedEntry) (use : ConfigUse α) : Bool :=
  entry.key == use.setting.key &&
    entry.settingDigest == use.setting.identity &&
    entry.interpretationDigest == use.interpretation.semanticDigest &&
    entry.context == use.context &&
    entry.samplingPoint == use.samplingPoint &&
    entry.changeEffect == use.changeEffect &&
    entry.catalogDigest == Temporal.DynamicConfig.Settings.catalogIdentity

private structure StoredEntry where
  provenance : ResolvedEntry
  canonicalValue : CanonicalValue
  deriving BEq, DecidableEq, Repr

private structure ConfigViewPayload where
  resolvedEntries : List StoredEntry
  deriving BEq, DecidableEq, Repr

/-- An immutable, use-keyed snapshot resolved before model execution. -/
structure ConfigView where
  private mk ::
  private payload : ConfigViewPayload
  deriving BEq, DecidableEq

/-- Return the canonical, use-keyed provenance of every resolved entry. -/
def ConfigView.provenance (view : ConfigView) : List ResolvedEntry :=
  view.payload.resolvedEntries.map StoredEntry.provenance

/-- Return the number of resolved uses in the view. -/
def ConfigView.entryCount (view : ConfigView) : Nat :=
  view.payload.resolvedEntries.length

/-- Exact constraints representing the unconstrained global lookup level. -/
def emptyConstraints : ExactConstraints := {
  namespaceName := none
  namespaceId := none
  taskQueueName := none
  destination := none
  chasmTaskType := none
  taskQueueType := none
  shardId := none
  taskType := none
}

private def validOptionalString : Option String → Bool
  | none => true
  | some value => value != ""

private def noNamespace (constraints : ExactConstraints) : Bool :=
  constraints.namespaceName.isNone

private def noNamespaceId (constraints : ExactConstraints) : Bool :=
  constraints.namespaceId.isNone

private def noTaskQueue (constraints : ExactConstraints) : Bool :=
  constraints.taskQueueName.isNone && constraints.taskQueueType.isNone

private def noDestination (constraints : ExactConstraints) : Bool :=
  constraints.destination.isNone

private def noChasmTaskType (constraints : ExactConstraints) : Bool :=
  constraints.chasmTaskType.isNone

private def noShardId (constraints : ExactConstraints) : Bool :=
  constraints.shardId.isNone

private def noTaskType (constraints : ExactConstraints) : Bool :=
  constraints.taskType.isNone

private def validConstraintStrings (constraints : ExactConstraints) : Bool :=
  validOptionalString constraints.namespaceName &&
    validOptionalString constraints.namespaceId &&
    validOptionalString constraints.taskQueueName &&
    validOptionalString constraints.destination &&
    validOptionalString constraints.chasmTaskType

private def legalConstraints (policy : PrecedencePolicy) (constraints : ExactConstraints) : Bool :=
  if !validConstraintStrings constraints then false else
  match policy with
  | .global => constraints == emptyConstraints
  | .namespace =>
      noNamespaceId constraints && noTaskQueue constraints && noDestination constraints &&
        noChasmTaskType constraints && noShardId constraints && noTaskType constraints
  | .namespaceId =>
      noNamespace constraints && noTaskQueue constraints && noDestination constraints &&
        noChasmTaskType constraints && noShardId constraints && noTaskType constraints
  | .shardId =>
      noNamespace constraints && noNamespaceId constraints && noTaskQueue constraints &&
        noDestination constraints && noChasmTaskType constraints && noTaskType constraints
  | .taskType =>
      noNamespace constraints && noNamespaceId constraints && noTaskQueue constraints &&
        noDestination constraints && noChasmTaskType constraints && noShardId constraints
  | .chasmTaskType =>
      noNamespace constraints && noNamespaceId constraints && noTaskQueue constraints &&
        noDestination constraints && noShardId constraints && noTaskType constraints
  | .taskQueue =>
      noNamespaceId constraints && noDestination constraints && noChasmTaskType constraints &&
        noShardId constraints && noTaskType constraints &&
        match constraints.namespaceName, constraints.taskQueueName, constraints.taskQueueType with
        | some _, some _, some _ => true
        | some _, some _, none => true
        | none, some _, none => true
        | some _, none, none => true
        | none, none, none => true
        | _, _, _ => false
  | .destination =>
      noNamespaceId constraints && noTaskQueue constraints && noChasmTaskType constraints &&
        noShardId constraints && noTaskType constraints &&
        match constraints.namespaceName, constraints.destination with
        | some _, some _ => true
        | none, some _ => true
        | some _, none => true
        | none, none => true

private def validContext (policy : PrecedencePolicy) (context : ExactConstraints) : Bool :=
  if !validConstraintStrings context then false else
  match policy with
  | .global => context == emptyConstraints
  | .namespace =>
      context.namespaceName.isSome && legalConstraints .namespace context
  | .namespaceId =>
      context.namespaceId.isSome && legalConstraints .namespaceId context
  | .taskQueue =>
      context.namespaceName.isSome && context.taskQueueName.isSome &&
        context.taskQueueType.isSome && legalConstraints .taskQueue context
  | .shardId => context.shardId.isSome && legalConstraints .shardId context
  | .taskType => context.taskType.isSome && legalConstraints .taskType context
  | .destination =>
      context.namespaceName.isSome && context.destination.isSome &&
        legalConstraints .destination context
  | .chasmTaskType =>
      context.chasmTaskType.isSome && legalConstraints .chasmTaskType context

private def hasRequiredContext (policy : PrecedencePolicy) (context : ExactConstraints) : Bool :=
  match policy with
  | .global => true
  | .namespace => context.namespaceName.isSome
  | .namespaceId => context.namespaceId.isSome
  | .taskQueue =>
      context.namespaceName.isSome && context.taskQueueName.isSome && context.taskQueueType.isSome
  | .shardId => context.shardId.isSome
  | .taskType => context.taskType.isSome
  | .destination => context.namespaceName.isSome && context.destination.isSome
  | .chasmTaskType => context.chasmTaskType.isSome

private def requireContext
    (useId : DefinitionId)
    (setting : Setting)
    (context : ExactConstraints) : Except ConfigError Unit := do
  if !hasRequiredContext setting.policy context then
    throw (configError .missingContext useId setting.key (reprStr context))
  if !validContext setting.policy context then
    throw (configError .illegalConstraints useId setting.key (reprStr context))

private def orderedConstraints
    (policy : PrecedencePolicy)
    (context : ExactConstraints) : List ExactConstraints :=
  match policy with
  | .global => [emptyConstraints]
  | .namespace =>
      [{ emptyConstraints with namespaceName := context.namespaceName }, emptyConstraints]
  | .namespaceId => [{ emptyConstraints with namespaceId := context.namespaceId }, emptyConstraints]
  | .shardId => [{ emptyConstraints with shardId := context.shardId }, emptyConstraints]
  | .taskType => [{ emptyConstraints with taskType := context.taskType }, emptyConstraints]
  | .chasmTaskType =>
      [{ emptyConstraints with chasmTaskType := context.chasmTaskType }, emptyConstraints]
  | .taskQueue =>
      [{ emptyConstraints with
          namespaceName := context.namespaceName
          taskQueueName := context.taskQueueName
          taskQueueType := context.taskQueueType },
       { emptyConstraints with
          namespaceName := context.namespaceName
          taskQueueName := context.taskQueueName },
       { emptyConstraints with taskQueueName := context.taskQueueName },
       { emptyConstraints with namespaceName := context.namespaceName },
       emptyConstraints]
  | .destination =>
      [{ emptyConstraints with
          namespaceName := context.namespaceName
          destination := context.destination },
       { emptyConstraints with destination := context.destination },
       { emptyConstraints with namespaceName := context.namespaceName },
       emptyConstraints]

private def findSetting? (catalog : List Setting) (key : String) : Option Setting :=
  catalog.find? fun setting => setting.key == key

private def requireSetting
    (catalog : List Setting)
    (useId : DefinitionId)
    (key : String) : Except ConfigError Setting :=
  match findSetting? catalog key with
  | none => throw (configError .unknownKey useId key key)
  | some setting => pure setting

private def firstDuplicateString : List String → Option String
  | first :: second :: rest =>
      if first == second then some first else firstDuplicateString (second :: rest)
  | _ => none

/-- Reject duplicate owner identifiers or setting keys in a checked-definition registry. -/
def validateConfigUseDefinitions
    (definitions : List AnyCheckedConfigUseDefinition) : Except ConfigError Unit := do
  let metadata := definitions.map AnyCheckedConfigUseDefinition.metadata
  let ids := metadata.map (fun definition => definition.id.value) |>.mergeSort stringLe
  match firstDuplicateString ids with
  | some duplicate =>
      throw (configError .duplicateUse (DefinitionId.of duplicate) "" duplicate)
  | none => pure ()
  let keys := metadata.map ConfigUseDefinitionMetadata.key |>.mergeSort stringLe
  match firstDuplicateString keys with
  | some duplicate =>
      throw (configError .duplicateUse (DefinitionId.of "umpire.config.definitions")
        duplicate duplicate)
  | none => pure ()

private def validateClassifications
    (catalog : List Setting)
    (classifications : List SettingClassification) : Except ConfigError Unit := do
  let owner := DefinitionId.of "umpire.config.classifications"
  let sorted := classifications.mergeSort fun left right => stringLe left.key right.key
  match firstDuplicateString (sorted.map SettingClassification.key) with
  | some key =>
      throw (configError .malformedUse owner key "duplicate classification" [key])
  | none => pure ()
  for classification in sorted do
    if classification.key == "" then
      throw (configError .malformedUse owner "" "empty classification key")
    let setting ← requireSetting catalog owner classification.key
    if classification.settingIdentity != setting.identity then
      throw (configError .incompatibleInterpretation owner classification.key
        (classification.settingIdentity ++ " != " ++ setting.identity))
    if classification.impacts == [] then
      throw (configError .emptyClassification owner classification.key "[]")

private def checkConfigUseInCatalog
    (catalog : List Setting)
    (classifications : List SettingClassification)
    (request : ConfigUseRequest α) : Except ConfigError (ConfigUse α) := do
  validateClassifications catalog classifications
  if request.id.value == "" || !request.id.isNamespaced then
    throw (configError .malformedUse request.id request.key request.id.value)
  if request.key == "" then
    throw (configError .malformedUse request.id request.key "empty key")
  let setting ← requireSetting catalog request.id request.key
  let classification ← match classifications.find? fun item => item.key == request.key with
    | none => throw (configError .unclassifiedKey request.id request.key request.key)
    | some classification => pure classification
  let interpretation ← match request.interpretation with
    | none => throw (configError .missingInterpretation request.id request.key request.key)
    | some interpretation => pure interpretation
  if interpretation.key != request.key || interpretation.semanticDigest == "" then
    throw (configError .incompatibleInterpretation request.id request.key interpretation.key)
  if interpretation.expectedSchema != setting.schema then
    throw (configError .schemaMismatch request.id request.key (reprStr setting.schema))
  if interpretation.expectedDefault != setting.defaultValue then
    throw (configError .defaultDrift request.id request.key (reprStr setting.defaultValue))
  if interpretation.expectedSettingIdentity != setting.identity then
    throw (configError .incompatibleInterpretation request.id request.key
      (interpretation.expectedSettingIdentity ++ " != " ++ setting.identity))
  requireContext request.id setting request.context
  pure (.mk {
    id := request.id
    setting
    classification
    interpretation
    context := request.context
    samplingPoint := request.samplingPoint
    changeEffect := request.changeEffect
  })

private def canonicalMatchesSchema (value : CanonicalValue) (schema : ValueSchema) : Bool :=
  match value, schema with
  | .null, .bool _ nullable
  | .null, .int _ nullable
  | .null, .uint _ nullable
  | .null, .float _ nullable
  | .null, .string _ nullable
  | .null, .duration _ nullable
  | .null, .dynamicValue _ nullable
  | .null, .list _ _ _ nullable
  | .null, .map _ _ nullable
  | .null, .struct _ _ nullable
  | .null, .reference _ nullable
  | .null, .opaque _ nullable => nullable
  | .bool _, .bool _ _ => true
  | .int _, .int _ _ => true
  | .uint _, .uint _ _ => true
  | .float _, .float _ _ => true
  | .string _, .string _ _ => true
  | .duration _, .duration _ _ => true
  | .list _, .list _ _ _ _ => true
  | .object _, .map _ _ _ => true
  | .object _, .struct _ _ _ => true
  | _, .dynamicValue _ _ => true
  | _, .reference _ _ => true
  | _, _ => false

private def decodeConfigValue
    (useId : DefinitionId)
    (key : String)
    (interpretation : ConfigInterpretation α)
    (value : CanonicalValue) : Except ConfigError α :=
  match interpretation.decode value with
  | .ok decoded => pure decoded
  | .error message => throw (configError .interpretationFailure useId key message)

private def opaqueDefaultMetadata : SettingDefault → List OpaqueDefault
  | .opaque metadata => [metadata]
  | .constrained defaults => defaults.filterMap fun candidate =>
      match candidate.value with
      | .opaque metadata => some metadata
      | .concrete _ => none
  | .concrete _ => []

private def validateOpaqueReplacement
    (useId : DefinitionId)
    (setting : Setting)
    (interpretation : ConfigInterpretation α) : Except ConfigError Unit := do
  match interpretation.opaqueReplacement with
  | none => pure ()
  | some replacement =>
      let importedMetadata := opaqueDefaultMetadata setting.defaultValue
      if importedMetadata == [] ||
          !(importedMetadata.all fun metadata => metadata == replacement.expected) then
        throw (configError .defaultDrift useId setting.key (reprStr importedMetadata))
      if !canonicalMatchesSchema replacement.value setting.schema then
        throw (configError .schemaMismatch useId setting.key (reprStr replacement.value))
      let _ ← decodeConfigValue useId setting.key interpretation replacement.value
      pure ()

/-- Validate an owner-authored request against the generated catalog and seal it as a typed use. -/
def checkConfigUse
    (classifications : List SettingClassification)
    (request : ConfigUseRequest α) : Except ConfigError (ConfigUse α) := do
  let use ← checkConfigUseInCatalog Temporal.DynamicConfig.Settings.all classifications request
  validateOpaqueReplacement use.id use.setting use.interpretation
  pure use

private def definitionContext : PrecedencePolicy → ExactConstraints
  | .global => emptyConstraints
  | .namespace => { emptyConstraints with namespaceName := some "definition" }
  | .namespaceId => { emptyConstraints with namespaceId := some "definition" }
  | .taskQueue => {
      emptyConstraints with
      namespaceName := some "definition"
      taskQueueName := some "definition"
      taskQueueType := some 0
    }
  | .shardId => { emptyConstraints with shardId := some 0 }
  | .taskType => { emptyConstraints with taskType := some 0 }
  | .destination => {
      emptyConstraints with
      namespaceName := some "definition"
      destination := some "definition"
    }
  | .chasmTaskType => { emptyConstraints with chasmTaskType := some "definition" }

/-- Check owner metadata once before any concrete context is instantiated. -/
def checkConfigUseDefinition
    (definition : ConfigUseDefinition α) : Except ConfigError (CheckedConfigUseDefinition α) := do
  let setting ← requireSetting Temporal.DynamicConfig.Settings.all definition.id
    definition.classification.key
  if definition.contextPolicy != setting.policy then
    throw (configError .incompatibleInterpretation definition.id definition.classification.key
      (reprStr definition.contextPolicy ++ " != " ++ reprStr setting.policy))
  let template ← checkConfigUse [definition.classification] {
    id := definition.id
    key := definition.classification.key
    context := definitionContext definition.contextPolicy
    samplingPoint := definition.samplingPoint
    changeEffect := definition.changeEffect
    interpretation := some definition.interpretation
  }
  pure (.mk { template, contextPolicy := definition.contextPolicy })

/-- Instantiate a checked owner definition at one concrete lookup context. -/
def CheckedConfigUseDefinition.instantiate
    (definition : CheckedConfigUseDefinition α)
    (context : ExactConstraints) : Except ConfigError (ConfigUse α) := do
  let template := definition.payload.template
  requireContext template.id template.setting context
  pure (.mk { template.payload with context })

/-- Bind a canonical override to a checked typed use before it can enter resolution. -/
def checkConfigOverride
    (use : ConfigUse α)
    (constraints : ExactConstraints)
    (value : CanonicalValue) : Except ConfigError ConfigOverride := do
  if !legalConstraints use.setting.policy constraints then
    throw (configError .illegalConstraints use.id use.setting.key (reprStr constraints))
  if !canonicalMatchesSchema value use.setting.schema then
    throw (configError .schemaMismatch use.id use.setting.key (reprStr value))
  let _ ← decodeConfigValue use.id use.setting.key use.interpretation value
  pure (rawConfigOverride use.setting.key constraints value)

private def firstDuplicateOverride : List ConfigOverride → Option ConfigOverride
  | [] => none
  | first :: rest =>
      if rest.any fun item => item.key == first.key && item.constraints == first.constraints then
        some first
      else
        firstDuplicateOverride rest

private def defaultLeafMatchesSchema (schema : ValueSchema) : DefaultLeaf → Bool
  | .concrete value => canonicalMatchesSchema value schema
  | .opaque _ => true

private def constrainedDefaultLe (left right : ConstrainedDefault) : Bool :=
  stringLe
    (reprStr left.constraints ++ "\u0000" ++ reprStr left.value)
    (reprStr right.constraints ++ "\u0000" ++ reprStr right.value)

private def firstDuplicateDefault : List ConstrainedDefault → Option ConstrainedDefault
  | [] => none
  | first :: rest =>
      if rest.any fun item => item.constraints == first.constraints then
        some first
      else
        firstDuplicateDefault rest

/-- Validate the schema, constraints, and default structure of one generated setting. -/
def validateSettingStructure (setting : Setting) : Except ConfigError Unit := do
  let owner := DefinitionId.of "umpire.config.catalog"
  match setting.defaultValue with
  | .concrete value =>
      if !canonicalMatchesSchema value setting.schema then
        throw (configError .schemaMismatch owner setting.key (reprStr value))
  | .opaque _ => pure ()
  | .constrained defaults =>
    if defaults == [] then
      throw (configError .defaultDrift owner setting.key "empty constrained defaults")
    let canonicalDefaults := defaults.mergeSort constrainedDefaultLe
    if !(canonicalDefaults.any fun candidate => candidate.constraints == emptyConstraints) then
      throw (configError .defaultDrift owner setting.key "missing unconstrained default")
    match firstDuplicateDefault canonicalDefaults with
      | some duplicate =>
          throw (configError .duplicateConstraints owner setting.key
            (reprStr duplicate.constraints) [setting.key])
      | none => pure ()
    for candidate in canonicalDefaults do
      if !legalConstraints setting.policy candidate.constraints then
        throw (configError .illegalConstraints owner setting.key
          (reprStr candidate.constraints))
      if !defaultLeafMatchesSchema setting.schema candidate.value then
        throw (configError .schemaMismatch owner setting.key (reprStr candidate.value))

private def overrideLe (left right : ConfigOverride) : Bool :=
  stringLe
    (left.key ++ "\u0000" ++ reprStr left.constraints ++ "\u0000" ++ reprStr left.value)
    (right.key ++ "\u0000" ++ reprStr right.constraints ++ "\u0000" ++ reprStr right.value)

private def validateOverrides
    (catalog : List Setting)
    (overrides : List ConfigOverride) : Except ConfigError Unit := do
  let owner := DefinitionId.of "umpire.config.overrides"
  match firstDuplicateOverride overrides with
  | some duplicate =>
      throw (configError .duplicateConstraints owner duplicate.key
        (reprStr duplicate.constraints) [duplicate.key])
  | none => pure ()
  for override in overrides do
    let setting ← requireSetting catalog owner override.key
    validateSettingStructure setting
    if !legalConstraints setting.policy override.constraints then
      throw (configError .illegalConstraints owner override.key (reprStr override.constraints))
    if !canonicalMatchesSchema override.value setting.schema then
      throw (configError .schemaMismatch owner override.key (reprStr override.value))

private structure CanonicalResolution where
  value : CanonicalValue
  source : ResolutionSource
  matchedConstraints : ExactConstraints
  deriving Repr

private def replacementFor
    (useId : DefinitionId)
    (key : String)
    (interpretation : Option (ConfigInterpretation α))
    (metadata : OpaqueDefault) : Except ConfigError CanonicalValue := do
  match interpretation.bind ConfigInterpretation.opaqueReplacement with
  | some replacement =>
      if replacement.expected == metadata then
        pure replacement.value
      else
        throw (configError .defaultDrift useId key (reprStr metadata))
  | none => throw (configError .opaqueDefaultSelected useId key (reprStr metadata))

private def resolutionFromLeaf
    (useId : DefinitionId)
    (key : String)
    (interpretation : Option (ConfigInterpretation α))
    (constraints : ExactConstraints)
    (leaf : DefaultLeaf) : Except ConfigError CanonicalResolution := do
  match leaf with
  | .concrete value => pure {
      value
      source := .constrainedDefault
      matchedConstraints := constraints
    }
  | .opaque metadata =>
      let value ← replacementFor useId key interpretation metadata
      pure { value, source := .opaqueReplacement, matchedConstraints := constraints }

private def resolveLevels
    (useId : DefinitionId)
    (setting : Setting)
    (interpretation : Option (ConfigInterpretation α))
    (overrides : List ConfigOverride)
    (defaults : List ConstrainedDefault) :
    List ExactConstraints → Except ConfigError (Option CanonicalResolution)
  | [] => pure none
  | constraints :: rest => do
      match overrides.find? fun override => override.constraints == constraints with
      | some override =>
          pure (some {
            value := override.value
            source := .override
            matchedConstraints := constraints
          })
      | none =>
          match defaults.find? fun candidate => candidate.constraints == constraints with
          | some candidate =>
              return some (← resolutionFromLeaf useId setting.key interpretation
                constraints candidate.value)
          | none => resolveLevels useId setting interpretation overrides defaults rest

private def resolveCanonical
    (useId : DefinitionId)
    (setting : Setting)
    (interpretation : Option (ConfigInterpretation α))
    (context : ExactConstraints)
    (overrides : List ConfigOverride) : Except ConfigError CanonicalResolution := do
  requireContext useId setting context
  validateSettingStructure setting
  let matchingOverrides := overrides.filter fun override => override.key == setting.key
  let defaults := match setting.defaultValue with
    | .constrained values => values
    | _ => []
  let resolution? ← resolveLevels useId setting interpretation matchingOverrides defaults
    (orderedConstraints setting.policy context)
  match resolution? with
  | some resolution => pure resolution
  | none =>
      match setting.defaultValue with
      | .concrete value => pure {
          value
          source := .simpleDefault
          matchedConstraints := emptyConstraints
        }
      | .opaque metadata =>
          let value ← replacementFor useId setting.key interpretation metadata
          pure { value, source := .opaqueReplacement, matchedConstraints := emptyConstraints }
      | .constrained _ =>
          throw (configError .incompatibleInterpretation useId setting.key
            "constrained defaults have no applicable exact constraint")

private def validateOverrideInterpretations
    (use : ConfigUse α)
    (overrides : List ConfigOverride) : Except ConfigError Unit := do
  for override in overrides do
    if override.key == use.setting.key then
      let _ ← decodeConfigValue use.id use.setting.key use.interpretation override.value
      pure ()

private def resolveUse
    (overrides : List ConfigOverride)
    (use : ConfigUse α) : Except ConfigError StoredEntry := do
  validateOverrideInterpretations use overrides
  let resolution ←
    resolveCanonical use.id use.setting (some use.interpretation) use.context overrides
  if !canonicalMatchesSchema resolution.value use.setting.schema then
    throw (configError .schemaMismatch use.id use.setting.key (reprStr resolution.value))
  let _ ← decodeConfigValue use.id use.setting.key use.interpretation resolution.value
  pure {
    provenance := {
      useId := use.id
      key := use.setting.key
      source := resolution.source
      matchedConstraints := resolution.matchedConstraints
      context := use.context
      catalogDigest := Temporal.DynamicConfig.Settings.catalogIdentity
      settingDigest := use.setting.identity
      interpretationDigest := use.interpretation.semanticDigest
      samplingPoint := use.samplingPoint
      changeEffect := use.changeEffect
    }
    canonicalValue := resolution.value
  }

private def firstDuplicateUse : List AnyConfigUse → Option AnyConfigUse
  | [] => none
  | first :: rest =>
      if rest.any fun item => item.id == first.id then some first else firstDuplicateUse rest

private def anyUseLe (left right : AnyConfigUse) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def validateCheckedUse
    (catalog : List Setting)
    (use : ConfigUse α) : Except ConfigError Unit := do
  if use.id.value == "" || !use.id.isNamespaced || use.setting.key == "" then
    throw (configError .malformedUse use.id use.setting.key use.id.value)
  let setting ← requireSetting catalog use.id use.setting.key
  if !use.matchesSetting setting then
    throw (configError .incompatibleInterpretation use.id setting.key setting.identity)
  validateOpaqueReplacement use.id setting use.interpretation
  requireContext use.id setting use.context

private def validateCheckedUses
    (catalog : List Setting) : List AnyConfigUse → Except ConfigError Unit
  | [] => pure ()
  | .of use :: rest => do
      validateCheckedUse catalog use
      validateCheckedUses catalog rest

private def resolveUses
    (overrides : List ConfigOverride) :
    List AnyConfigUse → Except ConfigError (List StoredEntry)
  | [] => pure []
  | .of use :: rest => do
      let entry ← resolveUse overrides use
      return entry :: (← resolveUses overrides rest)

private def resolveConfigViewInCatalog
    (catalog : List Setting)
    (overrides : List ConfigOverride)
    (uses : List AnyConfigUse) : Except ConfigError ConfigView := do
  let sortedUses := uses.mergeSort anyUseLe
  match firstDuplicateUse sortedUses with
  | some duplicate =>
      throw (configError .duplicateUse duplicate.id duplicate.key duplicate.id.value
        [duplicate.id.value])
  | none => pure ()
  validateCheckedUses catalog sortedUses
  let canonicalOverrides := overrides.mergeSort overrideLe
  validateOverrides catalog canonicalOverrides
  pure (.mk { resolvedEntries := (← resolveUses canonicalOverrides sortedUses) })

/-- Resolve checked uses and their bound overrides into a deterministic immutable snapshot. -/
def resolveConfigView
    (overrides : List ConfigOverride)
    (uses : List AnyConfigUse) : Except ConfigError ConfigView :=
  resolveConfigViewInCatalog Temporal.DynamicConfig.Settings.all overrides uses

/-- Decode a view entry through the exact checked use from which it was resolved. -/
def ConfigView.read (view : ConfigView) (use : ConfigUse α) : Except ConfigError α := do
  let stored ← match view.payload.resolvedEntries.find? fun entry =>
      entry.provenance.useId == use.id with
    | none => throw (configError .unknownUse use.id use.setting.key use.id.value)
    | some entry => pure entry
  let entry := stored.provenance
  if !entry.matchesUse use then
    throw (configError .incompatibleInterpretation use.id use.setting.key (reprStr entry))
  decodeConfigValue use.id use.setting.key use.interpretation stored.canonicalValue

/-- Catalog identity against which the retained cross-language fixtures were generated. -/
def expectedFixtureCatalogIdentity : String :=
  "sha256:22be68647d91a7249ac5fab0ef87a9e77cbcc391df54076dabdbfe9070f9832f"

/-- Check that a fixture catalog identity matches the imported generated catalog. -/
def checkFixtureCatalogIdentity (expected : String) : Except ConfigError Unit := do
  if expected != Temporal.DynamicConfig.Settings.catalogIdentity then
    throw (configError .fixtureMismatch (DefinitionId.of "umpire.config.fixture.catalog")
      "<catalog>" (expected ++ " != " ++ Temporal.DynamicConfig.Settings.catalogIdentity))

private def fixtureSource : FixtureSource → ResolutionSource
  | .override => .override
  | .constrainedDefault => .constrainedDefault
  | .simpleDefault => .simpleDefault

/--
Verify one retained Go-computed resolver fixture without creating a model-facing string lookup.
-/
def checkResolutionFixture (fixture : ResolutionFixture) : Except ConfigError Unit := do
  checkFixtureCatalogIdentity expectedFixtureCatalogIdentity
  let useId := DefinitionId.of ("umpire.config.fixture." ++ fixture.name)
  let setting ← requireSetting Temporal.DynamicConfig.Settings.all useId fixture.settingKey
  if setting.policy != fixture.policy then
    throw (configError .fixtureMismatch useId fixture.settingKey (reprStr fixture.policy))
  let overrides := fixture.overrides.map fun override =>
    rawConfigOverride fixture.settingKey override.constraints override.value
  validateOverrides Temporal.DynamicConfig.Settings.all overrides
  let resolution ← resolveCanonical useId setting (α := Unit) none fixture.context overrides
  if resolution.value != fixture.result ||
      resolution.source != fixtureSource fixture.selectedSource ||
      resolution.matchedConstraints != fixture.selectedConstraint then
    throw (configError .fixtureMismatch useId fixture.settingKey (reprStr resolution))

/-- Verify every retained Go-computed resolver fixture against the Lean resolver. -/
def checkAllResolutionFixtures : Except ConfigError Unit := do
  for fixture in Temporal.DynamicConfig.Settings.fixtures do
    checkResolutionFixture fixture

end Temporal.System.Configuration
