import Temporal.DynamicConfig
import Umpire.Core

namespace Temporal.Umpire.Config

open _root_.Umpire
open Temporal.DynamicConfig

/-! Handwritten product meaning over the generated Temporal dynamic-config catalog. -/

inductive ImpactClass where
  | feature
  | validation
  | externallyVisibleSemantics
  | timing
  | topology
  | performance
  | observability
  deriving BEq, DecidableEq, Ord, Repr

structure SettingClassification where
  key : String
  settingIdentity : String
  impacts : List ImpactClass
  deriving BEq, DecidableEq, Repr

inductive SamplingPoint where
  | liveAccess
  | entityCreation
  | request
  | task
  | processStartup
  deriving BEq, DecidableEq, Ord, Repr

inductive ChangeEffect where
  | nextRead
  | newEntitiesOnly
  | restartRequired
  deriving BEq, DecidableEq, Ord, Repr

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

structure ConfigError where
  kind : ConfigErrorKind
  useId : DeclarationId
  key : String
  offendingValue : String
  relatedIdentities : List String
  deriving BEq, DecidableEq, Repr

private def stringLe (left right : String) : Bool := decide (left ≤ right)

private def canonicalStrings (values : List String) : List String :=
  values.mergeSort stringLe |>.eraseDups

private def configError
    (kind : ConfigErrorKind)
    (useId : DeclarationId)
    (key offendingValue : String)
    (relatedIdentities : List String := []) : ConfigError := {
  kind
  useId := if useId.value == "" then DeclarationId.of "umpire.config.anonymous" else useId
  key
  offendingValue
  relatedIdentities := canonicalStrings relatedIdentities
}

structure OpaqueDefaultReplacement where
  expected : OpaqueDefault
  value : CanonicalValue
  deriving BEq, DecidableEq, Repr

structure ConfigInterpretation (α : Type) where
  key : String
  expectedSettingIdentity : String
  expectedSchema : ValueSchema
  expectedDefault : SettingDefault
  opaqueReplacement : Option OpaqueDefaultReplacement := none
  semanticDigest : String
  decode : CanonicalValue → Except String α

structure ConfigUseRequest (α : Type) where
  id : DeclarationId
  key : String
  context : ExactConstraints
  samplingPoint : SamplingPoint
  changeEffect : ChangeEffect
  interpretation : Option (ConfigInterpretation α)

private structure ConfigUsePayload (α : Type) where
  id : DeclarationId
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

def ConfigUse.id (use : ConfigUse α) : DeclarationId :=
  use.payload.id

def ConfigUse.key (use : ConfigUse α) : String :=
  use.payload.setting.key

private def ConfigUse.setting (use : ConfigUse α) : Setting :=
  use.payload.setting

private def ConfigUse.classification (use : ConfigUse α) : SettingClassification :=
  use.payload.classification

private def ConfigUse.interpretation (use : ConfigUse α) : ConfigInterpretation α :=
  use.payload.interpretation

def ConfigUse.context (use : ConfigUse α) : ExactConstraints :=
  use.payload.context

def ConfigUse.samplingPoint (use : ConfigUse α) : SamplingPoint :=
  use.payload.samplingPoint

def ConfigUse.changeEffect (use : ConfigUse α) : ChangeEffect :=
  use.payload.changeEffect

inductive AnyConfigUse where
  | of {α : Type} (use : ConfigUse α)

namespace AnyConfigUse

def id : AnyConfigUse → DeclarationId
  | .of use => use.id

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

inductive ResolutionSource where
  | override
  | constrainedDefault
  | simpleDefault
  | opaqueReplacement
  deriving BEq, DecidableEq, Ord, Repr

structure ResolvedEntry where
  private mk ::
  useId : DeclarationId
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

def ConfigView.provenance (view : ConfigView) : List ResolvedEntry :=
  view.payload.resolvedEntries.map StoredEntry.provenance

def ConfigView.entryCount (view : ConfigView) : Nat :=
  view.payload.resolvedEntries.length

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
    (useId : DeclarationId)
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
  | .namespace => [{ emptyConstraints with namespaceName := context.namespaceName }, emptyConstraints]
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

private def firstDuplicateString : List String → Option String
  | first :: second :: rest =>
      if first == second then some first else firstDuplicateString (second :: rest)
  | _ => none

private def validateClassifications
    (catalog : List Setting)
    (classifications : List SettingClassification) : Except ConfigError Unit := do
  let sorted := classifications.mergeSort fun left right => stringLe left.key right.key
  match firstDuplicateString (sorted.map SettingClassification.key) with
  | some key =>
      throw (configError .malformedUse (DeclarationId.of "umpire.config.classifications") key
        "duplicate classification" [key])
  | none => pure ()
  for classification in sorted do
    if classification.key == "" then
      throw (configError .malformedUse (DeclarationId.of "umpire.config.classifications") ""
        "empty classification key")
    match findSetting? catalog classification.key with
    | none =>
        throw (configError .unknownKey (DeclarationId.of "umpire.config.classifications")
          classification.key classification.key)
    | some setting =>
        if classification.settingIdentity != setting.identity then
          throw (configError .incompatibleInterpretation
            (DeclarationId.of "umpire.config.classifications") classification.key
            (classification.settingIdentity ++ " != " ++ setting.identity))
        if classification.impacts == [] then
          throw (configError .emptyClassification
            (DeclarationId.of "umpire.config.classifications") classification.key "[]")

private def checkConfigUseInCatalog
    (catalog : List Setting)
    (classifications : List SettingClassification)
    (request : ConfigUseRequest α) : Except ConfigError (ConfigUse α) := do
  validateClassifications catalog classifications
  if request.id.value == "" || !request.id.isNamespaced then
    throw (configError .malformedUse request.id request.key request.id.value)
  if request.key == "" then
    throw (configError .malformedUse request.id request.key "empty key")
  let setting ← match findSetting? catalog request.key with
    | none => throw (configError .unknownKey request.id request.key request.key)
    | some setting => pure setting
  let classification ← match classifications.find? fun item => item.key == request.key with
    | none => throw (configError .unclassifiedKey request.id request.key request.key)
    | some classification => pure classification
  if classification.impacts == [] then
    throw (configError .emptyClassification request.id request.key "[]")
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

private def opaqueDefaultMetadata : SettingDefault → List OpaqueDefault
  | .opaque metadata => [metadata]
  | .constrained defaults => defaults.filterMap fun candidate =>
      match candidate.value with
      | .opaque metadata => some metadata
      | .concrete _ => none
  | .concrete _ => []

private def validateOpaqueReplacement
    (useId : DeclarationId)
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
      match interpretation.decode replacement.value with
      | .ok _ => pure ()
      | .error message =>
          throw (configError .interpretationFailure useId setting.key message)

def checkConfigUse
    (classifications : List SettingClassification)
    (request : ConfigUseRequest α) : Except ConfigError (ConfigUse α) := do
  let use ← checkConfigUseInCatalog Temporal.DynamicConfig.Settings.all classifications request
  validateOpaqueReplacement use.id use.setting use.interpretation
  pure use

/-- Bind a canonical override to a checked typed use before it can enter resolution. -/
def checkConfigOverride
    (use : ConfigUse α)
    (constraints : ExactConstraints)
    (value : CanonicalValue) : Except ConfigError ConfigOverride := do
  if !legalConstraints use.setting.policy constraints then
    throw (configError .illegalConstraints use.id use.setting.key (reprStr constraints))
  if !canonicalMatchesSchema value use.setting.schema then
    throw (configError .schemaMismatch use.id use.setting.key (reprStr value))
  match use.interpretation.decode value with
  | .error message =>
      throw (configError .interpretationFailure use.id use.setting.key message)
  | .ok _ => pure (rawConfigOverride use.setting.key constraints value)

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

def validateSettingStructure (setting : Setting) : Except ConfigError Unit := do
  let owner := DeclarationId.of "umpire.config.catalog"
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
  match firstDuplicateOverride overrides with
  | some duplicate =>
      throw (configError .duplicateConstraints (DeclarationId.of "umpire.config.overrides")
        duplicate.key (reprStr duplicate.constraints) [duplicate.key])
  | none => pure ()
  for override in overrides do
    let setting ← match findSetting? catalog override.key with
      | none => throw (configError .unknownKey (DeclarationId.of "umpire.config.overrides")
          override.key override.key)
      | some setting => pure setting
    validateSettingStructure setting
    if !legalConstraints setting.policy override.constraints then
      throw (configError .illegalConstraints (DeclarationId.of "umpire.config.overrides")
        override.key (reprStr override.constraints))
    if !canonicalMatchesSchema override.value setting.schema then
      throw (configError .schemaMismatch (DeclarationId.of "umpire.config.overrides")
        override.key (reprStr override.value))

private structure CanonicalResolution where
  value : CanonicalValue
  source : ResolutionSource
  matchedConstraints : ExactConstraints
  deriving Repr

private def replacementFor
    (useId : DeclarationId)
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
    (useId : DeclarationId)
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
    (useId : DeclarationId)
    (setting : Setting)
    (interpretation : Option (ConfigInterpretation α))
    (overrides : List ConfigOverride)
    (defaults : List ConstrainedDefault) :
    List ExactConstraints → Except ConfigError (Option CanonicalResolution)
  | [] => pure none
  | constraints :: rest => do
      match overrides.find? fun override => override.constraints == constraints with
      | some override =>
          pure (some { value := override.value, source := .override, matchedConstraints := constraints })
      | none =>
          match defaults.find? fun candidate => candidate.constraints == constraints with
          | some candidate =>
              return some (← resolutionFromLeaf useId setting.key interpretation
                constraints candidate.value)
          | none => resolveLevels useId setting interpretation overrides defaults rest

private def resolveCanonical
    (useId : DeclarationId)
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
  match ← resolveLevels useId setting interpretation matchingOverrides defaults
      (orderedConstraints setting.policy context) with
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
      match use.interpretation.decode override.value with
      | .ok _ => pure ()
      | .error message =>
          throw (configError .interpretationFailure use.id use.setting.key message)

private def resolveUse
    (overrides : List ConfigOverride)
    (use : ConfigUse α) : Except ConfigError StoredEntry := do
  validateOverrideInterpretations use overrides
  let resolution ← resolveCanonical use.id use.setting (some use.interpretation) use.context overrides
  if !canonicalMatchesSchema resolution.value use.setting.schema then
    throw (configError .schemaMismatch use.id use.setting.key (reprStr resolution.value))
  match use.interpretation.decode resolution.value with
  | .error message =>
      throw (configError .interpretationFailure use.id use.setting.key message)
  | .ok _ => pure {
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
  let setting ← match findSetting? catalog use.setting.key with
    | none => throw (configError .unknownKey use.id use.setting.key use.setting.key)
    | some setting => pure setting
  if setting != use.setting || use.classification.key != setting.key ||
      use.classification.settingIdentity != setting.identity || use.classification.impacts == [] ||
      use.interpretation.key != setting.key ||
      use.interpretation.expectedSettingIdentity != setting.identity ||
      use.interpretation.expectedSchema != setting.schema ||
      use.interpretation.expectedDefault != setting.defaultValue ||
      use.interpretation.semanticDigest == "" then
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
      throw (configError .duplicateUse duplicate.id duplicate.key duplicate.id.value [duplicate.id.value])
  | none => pure ()
  validateCheckedUses catalog sortedUses
  let canonicalOverrides := overrides.mergeSort overrideLe
  validateOverrides catalog canonicalOverrides
  pure (.mk { resolvedEntries := (← resolveUses canonicalOverrides sortedUses) })

def resolveConfigView
    (overrides : List ConfigOverride)
    (uses : List AnyConfigUse) : Except ConfigError ConfigView :=
  resolveConfigViewInCatalog Temporal.DynamicConfig.Settings.all overrides uses

def ConfigView.read (view : ConfigView) (use : ConfigUse α) : Except ConfigError α := do
  let stored ← match view.payload.resolvedEntries.find? fun entry => entry.provenance.useId == use.id with
    | none => throw (configError .unknownUse use.id use.setting.key use.id.value)
    | some entry => pure entry
  let entry := stored.provenance
  if entry.key != use.setting.key || entry.settingDigest != use.setting.identity ||
      entry.interpretationDigest != use.interpretation.semanticDigest ||
      entry.context != use.context || entry.samplingPoint != use.samplingPoint ||
      entry.changeEffect != use.changeEffect ||
      entry.catalogDigest != Temporal.DynamicConfig.Settings.catalogIdentity then
    throw (configError .incompatibleInterpretation use.id use.setting.key (reprStr entry))
  match use.interpretation.decode stored.canonicalValue with
  | .ok value => pure value
  | .error message => throw (configError .interpretationFailure use.id use.setting.key message)

def expectedFixtureCatalogIdentity : String :=
  "sha256:22be68647d91a7249ac5fab0ef87a9e77cbcc391df54076dabdbfe9070f9832f"

def checkFixtureCatalogIdentity (expected : String) : Except ConfigError Unit := do
  if expected != Temporal.DynamicConfig.Settings.catalogIdentity then
    throw (configError .fixtureMismatch (DeclarationId.of "umpire.config.fixture.catalog")
      "<catalog>" (expected ++ " != " ++ Temporal.DynamicConfig.Settings.catalogIdentity))

private def fixtureSource : FixtureSource → ResolutionSource
  | .override => .override
  | .constrainedDefault => .constrainedDefault
  | .simpleDefault => .simpleDefault

/-- Verify one retained Go-computed resolver fixture without creating a model-facing string lookup. -/
def checkResolutionFixture (fixture : ResolutionFixture) : Except ConfigError Unit := do
  checkFixtureCatalogIdentity expectedFixtureCatalogIdentity
  let useId := DeclarationId.of ("umpire.config.fixture." ++ fixture.name)
  let setting ← match findSetting? Temporal.DynamicConfig.Settings.all fixture.settingKey with
    | none => throw (configError .unknownKey useId fixture.settingKey fixture.settingKey)
    | some setting => pure setting
  if setting.policy != fixture.policy then
    throw (configError .fixtureMismatch useId fixture.settingKey (reprStr fixture.policy))
  let overrides := fixture.overrides.map fun override =>
    rawConfigOverride fixture.settingKey override.constraints override.value
  validateOverrides Temporal.DynamicConfig.Settings.all overrides
  let resolution ← resolveCanonical useId setting (α := Unit) none fixture.context overrides
  if resolution.value != fixture.result || resolution.source != fixtureSource fixture.selectedSource ||
      resolution.matchedConstraints != fixture.selectedConstraint then
    throw (configError .fixtureMismatch useId fixture.settingKey (reprStr resolution))

def checkAllResolutionFixtures : Except ConfigError Unit := do
  for fixture in Temporal.DynamicConfig.Settings.fixtures do
    checkResolutionFixture fixture

private def decodeBool : CanonicalValue → Except String Bool
  | .bool value => pure value
  | value => throw ("expected bool, found " ++ reprStr value)

private def decodeInt : CanonicalValue → Except String Int
  | .int value => pure value
  | value => throw ("expected int, found " ++ reprStr value)

private def decodeDuration : CanonicalValue → Except String Int
  | .duration nanoseconds => pure nanoseconds
  | value => throw ("expected duration, found " ++ reprStr value)

structure CallbackAddressRule where
  pattern : String
  allowInsecure : Bool
  deriving BEq, DecidableEq, Repr

structure CallbackAddressRules where
  rules : List CallbackAddressRule
  deriving BEq, DecidableEq, Repr

private def canonicalValuesToList : CanonicalValues → List CanonicalValue
  | .nil => []
  | .cons value tail => value :: canonicalValuesToList tail

private def canonicalField? (name : String) : CanonicalFields → Option CanonicalValue
  | .nil => none
  | .cons fieldName value tail =>
      if fieldName == name then some value else canonicalField? name tail

private def canonicalFieldsToList : CanonicalFields → List (String × CanonicalValue)
  | .nil => []
  | .cons name value tail => (name, value) :: canonicalFieldsToList tail

private def requireCanonicalFields
    (owner : String)
    (allowed required : List String)
    (fields : CanonicalFields) : Except String Unit := do
  let names := (canonicalFieldsToList fields).map Prod.fst
  let sortedNames := names.mergeSort stringLe
  match firstDuplicateString sortedNames with
  | some duplicate => throw (owner ++ " contains duplicate field " ++ duplicate)
  | none => pure ()
  for name in names do
    if !allowed.contains name then throw (owner ++ " contains unknown field " ++ name)
  for name in required do
    if !names.contains name then throw (owner ++ " requires field " ++ name)

private def decodeAddressRule (value : CanonicalValue) : Except String CallbackAddressRule := do
  let fields ← match value with
    | .object fields => pure fields
    | _ => throw ("address rule must be an object: " ++ reprStr value)
  requireCanonicalFields "address rule" ["Pattern", "AllowInsecure"] ["Pattern"] fields
  let pattern ← match canonicalField? "Pattern" fields with
    | some (.string pattern) => pure pattern
    | _ => throw "address rule requires a string Pattern"
  if pattern == "" then throw "address rule Pattern must be non-empty"
  let allowInsecure ← match canonicalField? "AllowInsecure" fields with
    | none => pure false
    | some (.bool value) => pure value
    | _ => throw "address rule AllowInsecure must be a bool"
  pure { pattern, allowInsecure }

def decodeCallbackAddressRules : CanonicalValue → Except String CallbackAddressRules
  | .object fields => do
      requireCanonicalFields "callback address rules" ["Rules"] ["Rules"] fields
      match canonicalField? "Rules" fields with
      | some .null => pure { rules := [] }
      | some (.list values) =>
          pure { rules := ← canonicalValuesToList values |>.mapM decodeAddressRule }
      | _ => throw "callback address rules require Rules as a list or null"
  | value => throw ("callback address rules must be an object: " ++ reprStr value)

inductive CallbackAddressErrorKind where
  | unknownScheme
  | missingHost
  | malformedAddress
  | unmatchedAddress
  | insecureConnection
  deriving BEq, DecidableEq, Ord, Repr

structure CallbackAddressError where
  kind : CallbackAddressErrorKind
  address : String
  matchedPattern : Option String
  deriving BEq, DecidableEq, Repr

private def wildcardMatchFuel : Nat → List Char → List Char → Bool
  | 0, _, _ => false
  | _ + 1, [], host => host == []
  | fuel + 1, '*' :: pattern, host =>
      wildcardMatchFuel fuel pattern host ||
        match host with
        | [] => false
        | _ :: rest => wildcardMatchFuel fuel ('*' :: pattern) rest
  | fuel + 1, expected :: pattern, actual :: host =>
      expected == actual && wildcardMatchFuel fuel pattern host
  | _ + 1, _ :: _, [] => false

def wholeHostWildcardMatch (pattern host : String) : Bool :=
  wildcardMatchFuel (pattern.length + host.length + 1) pattern.toList host.toList

private def isHexCharacter (character : Char) : Bool :=
  "0123456789abcdefABCDEF".toList.contains character

private def validPercentEscapes : List Char → Bool
  | [] => true
  | '%' :: first :: second :: rest =>
      isHexCharacter first && isHexCharacter second && validPercentEscapes rest
  | '%' :: _ => false
  | _ :: rest => validPercentEscapes rest

private def isForbiddenAuthorityCharacter (character : Char) : Bool :=
  [' ', '\t', '\n', '\r', '\\'].contains character

private def lastString : List String → String
  | [] => ""
  | [value] => value
  | _ :: rest => lastString rest

private def validPort (characters : List Char) : Bool :=
  characters.all fun character => "0123456789".toList.contains character

private def validIPv4Octet (octet : String) : Bool :=
  octet != "" && (octet.length == 1 || octet.toList.head? != some '0') &&
    match octet.toNat? with
    | some value => decide (value ≤ 255)
    | none => false

private def validIPv4Address (address : String) : Bool :=
  match address.splitOn "." with
  | [first, second, third, fourth] =>
      [first, second, third, fourth].all validIPv4Octet
  | _ => false

private def validIPv6Group (group : String) : Bool :=
  group.length > 0 && group.length ≤ 4 && group.toList.all isHexCharacter

private def ipv6Units : List String → Option Nat
  | [] => some 0
  | [last] =>
      if last.toList.contains '.' then
        if validIPv4Address last then some 2 else none
      else if validIPv6Group last then some 1 else none
  | group :: rest =>
      if validIPv6Group group then
        (ipv6Units rest).map Nat.succ
      else
        none

private def ipv6Side (side : String) : Option Nat :=
  if side == "" then some 0 else ipv6Units (side.splitOn ":")

private def validIPv6Zone (zone : String) : Bool :=
  zone != "" && zone.toList.all fun character =>
    character.isAlphanum || ['.', '_', '-'].contains character

private def validIPv6Address (literal : String) : Bool :=
  let address? := match literal.splitOn "%25" with
    | [address] => some address
    | [address, zone] => if validIPv6Zone zone then some address else none
    | _ => none
  match address? with
  | none => false
  | some address =>
      if address.toList.contains '%' then false else
      match address.splitOn "::" with
      | [whole] => ipv6Side whole == some 8
      | [left, right] =>
          match ipv6Side left, ipv6Side right with
          | some leftUnits, some rightUnits => leftUnits + rightUnits < 8
          | _, _ => false
      | _ => false

private def validBracketHostPort (hostPort : String) : Bool :=
  match hostPort.toList with
  | '[' :: rest =>
      match (String.ofList rest).splitOn "]" with
      | [inside, suffix] =>
          validIPv6Address inside &&
            match suffix.toList with
            | [] => true
            | ':' :: port => validPort port
            | _ => false
      | _ => false
  | _ => false

private def validHostPort (hostPort : String) : Bool :=
  if hostPort == "" || hostPort.toList.any isForbiddenAuthorityCharacter then
    false
  else if hostPort.toList.head? == some '[' then
    validBracketHostPort hostPort
  else if hostPort.toList.any fun character => character == '[' || character == ']' || character == '%' then
    false
  else
    match hostPort.splitOn ":" with
    | [host] => host != ""
    | [host, port] => host != "" && validPort port.toList
    | _ => false

private def urlHost? (rawAddress : String) : Except CallbackAddressError (String × String) := do
  match rawAddress.splitOn "://" with
  | [scheme, remainder] =>
      if scheme != "http" && scheme != "https" then
        throw { kind := .unknownScheme, address := rawAddress, matchedPattern := none }
      if !validPercentEscapes rawAddress.toList then
        throw { kind := .malformedAddress, address := rawAddress, matchedPattern := none }
      let authority := String.ofList (remainder.toList.takeWhile fun character =>
        character != '/' && character != '?' && character != '#')
      let hostPort := lastString (authority.splitOn "@")
      if hostPort == "" then
        throw { kind := .missingHost, address := rawAddress, matchedPattern := none }
      if !validHostPort hostPort then
        throw { kind := .malformedAddress, address := rawAddress, matchedPattern := none }
      pure (scheme, hostPort)
  | _ => throw { kind := .unknownScheme, address := rawAddress, matchedPattern := none }

def CallbackAddressRules.validate
    (rules : CallbackAddressRules)
    (rawAddress : String) : Except CallbackAddressError Unit := do
  if rawAddress == "temporal://system" || rawAddress == "temporal://internal" then
    pure ()
  else
    let (scheme, host) ← urlHost? rawAddress
    let rec validateRules : List CallbackAddressRule → Except CallbackAddressError Unit
      | [] => throw { kind := .unmatchedAddress, address := rawAddress, matchedPattern := none }
      | rule :: rest =>
          if wholeHostWildcardMatch rule.pattern host then
            if scheme == "https" || rule.allowInsecure then
              pure ()
            else
              throw {
                kind := .insecureConnection
                address := rawAddress
                matchedPattern := some rule.pattern
              }
          else
            validateRules rest
    validateRules rules.rules

def authoredClassifications : List SettingClassification :=
  [{ key := "history.enablechasmcallbacks"
     settingIdentity := "sha256:415f169bb77c82582f2d8f5049648b5b079f4f1047a2f109d4ed9b14037d9c8c"
     impacts := [.feature, .externallyVisibleSemantics] },
   { key := "callback.maxperexecution"
     settingIdentity := "sha256:6c7f3b78bbbf74a83401b46faedf61250a1c4c2c92d02eab91ec9ebc36b30d71"
     impacts := [.validation] },
   { key := "callback.request.timeout"
     settingIdentity := "sha256:cd2c7d65a4f41e7edcfa548d7433aeb7cd5a414c6a3258d361676cd3ada8fda9"
     impacts := [.timing] },
   { key := "callback.allowedaddresses"
     settingIdentity := "sha256:452cd642fac8adb5d5e1e2c0a4ef1d149cfb621ed663842c1bde7dd123faca9b"
     impacts := [.validation, .externallyVisibleSemantics] },
   { key := "matching.updateackinterval"
     settingIdentity := "sha256:58c6db0d991c651b92e007384724788f74236057d53c6814293a5439e216501f"
     impacts := [.timing, .performance] },
   { key := "matching.workerregistrynumbuckets"
     settingIdentity := "sha256:6369ab31f72b574120e020fe8695290050ce1d2d66b4579e01243bbb4aea5f29"
     impacts := [.topology, .performance] }]

def historyEnableChasmCallbacksInterpretation : ConfigInterpretation Bool := {
  key := "history.enablechasmcallbacks"
  expectedSettingIdentity := "sha256:415f169bb77c82582f2d8f5049648b5b079f4f1047a2f109d4ed9b14037d9c8c"
  expectedSchema := .bool "bool" false
  expectedDefault := .concrete (.bool true)
  semanticDigest := semanticDigestOf "temporal.config/history-enable-chasm-callbacks/v1"
  decode := decodeBool
}

def callbackMaxPerExecutionInterpretation : ConfigInterpretation Int := {
  key := "callback.maxperexecution"
  expectedSettingIdentity := "sha256:6c7f3b78bbbf74a83401b46faedf61250a1c4c2c92d02eab91ec9ebc36b30d71"
  expectedSchema := .int "int" false
  expectedDefault := .concrete (.int 2000)
  semanticDigest := semanticDigestOf "temporal.config/callback-max-per-execution/v1"
  decode := decodeInt
}

def callbackRequestTimeoutInterpretation : ConfigInterpretation Int := {
  key := "callback.request.timeout"
  expectedSettingIdentity := "sha256:cd2c7d65a4f41e7edcfa548d7433aeb7cd5a414c6a3258d361676cd3ada8fda9"
  expectedSchema := .duration "time.Duration" false
  expectedDefault := .concrete (.duration 10000000000)
  semanticDigest := semanticDigestOf "temporal.config/callback-request-timeout/v1"
  decode := decodeDuration
}

def callbackAllowedAddressesInterpretation : ConfigInterpretation CallbackAddressRules := {
  key := "callback.allowedaddresses"
  expectedSettingIdentity := "sha256:452cd642fac8adb5d5e1e2c0a4ef1d149cfb621ed663842c1bde7dd123faca9b"
  expectedSchema := Temporal.DynamicConfig.Settings.callback_allowedaddresses.schema
  expectedDefault := .concrete (.object (.cons "Rules" .null .nil))
  semanticDigest := semanticDigestOf "temporal.config/callback-allowed-addresses/v1"
  decode := decodeCallbackAddressRules
}

def matchingUpdateAckIntervalInterpretation : ConfigInterpretation Int := {
  key := "matching.updateackinterval"
  expectedSettingIdentity := "sha256:58c6db0d991c651b92e007384724788f74236057d53c6814293a5439e216501f"
  expectedSchema := .duration "time.Duration" false
  expectedDefault := Temporal.DynamicConfig.Settings.matching_updateackinterval.defaultValue
  semanticDigest := semanticDigestOf "temporal.config/matching-update-ack-interval/v1"
  decode := decodeDuration
}

def matchingWorkerRegistryNumBucketsInterpretation : ConfigInterpretation Int := {
  key := "matching.workerregistrynumbuckets"
  expectedSettingIdentity := "sha256:6369ab31f72b574120e020fe8695290050ce1d2d66b4579e01243bbb4aea5f29"
  expectedSchema := .int "int" false
  expectedDefault := .concrete (.int 10)
  semanticDigest := semanticDigestOf "temporal.config/matching-worker-registry-num-buckets/v1"
  decode := decodeInt
}

def namespaceContext (namespaceName : String) : ExactConstraints :=
  { emptyConstraints with namespaceName := some namespaceName }

def destinationContext (namespaceName destination : String) : ExactConstraints :=
  { emptyConstraints with namespaceName := some namespaceName, destination := some destination }

def taskQueueContext
    (namespaceName taskQueueName : String)
    (taskQueueType : Int) : ExactConstraints :=
  { emptyConstraints with
      namespaceName := some namespaceName
      taskQueueName := some taskQueueName
      taskQueueType := some taskQueueType }

private def checkedAuthoredUse
    (request : ConfigUseRequest α) : Except ConfigError (ConfigUse α) :=
  checkConfigUse authoredClassifications request

def historyEnableChasmCallbacksUse (namespaceName : String) : Except ConfigError (ConfigUse Bool) :=
  checkedAuthoredUse {
    id := DeclarationId.of "temporal.callback.enable-chasm"
    key := historyEnableChasmCallbacksInterpretation.key
    context := namespaceContext namespaceName
    samplingPoint := .entityCreation
    changeEffect := .newEntitiesOnly
    interpretation := some historyEnableChasmCallbacksInterpretation
  }

def callbackMaxPerExecutionUse (namespaceName : String) : Except ConfigError (ConfigUse Int) :=
  checkedAuthoredUse {
    id := DeclarationId.of "temporal.callback.max-per-execution"
    key := callbackMaxPerExecutionInterpretation.key
    context := namespaceContext namespaceName
    samplingPoint := .request
    changeEffect := .nextRead
    interpretation := some callbackMaxPerExecutionInterpretation
  }

def callbackAllowedAddressesUse
    (namespaceName : String) : Except ConfigError (ConfigUse CallbackAddressRules) :=
  checkedAuthoredUse {
    id := DeclarationId.of "temporal.callback.allowed-addresses"
    key := callbackAllowedAddressesInterpretation.key
    context := namespaceContext namespaceName
    samplingPoint := .request
    changeEffect := .nextRead
    interpretation := some callbackAllowedAddressesInterpretation
  }

def callbackRequestTimeoutUse
    (namespaceName destination : String) : Except ConfigError (ConfigUse Int) :=
  checkedAuthoredUse {
    id := DeclarationId.of "temporal.callback.request-timeout"
    key := callbackRequestTimeoutInterpretation.key
    context := destinationContext namespaceName destination
    samplingPoint := .task
    changeEffect := .nextRead
    interpretation := some callbackRequestTimeoutInterpretation
  }

def matchingUpdateAckIntervalUse
    (namespaceName taskQueueName : String)
    (taskQueueType : Int) : Except ConfigError (ConfigUse Int) :=
  checkedAuthoredUse {
    id := DeclarationId.of "temporal.matching.update-ack-interval"
    key := matchingUpdateAckIntervalInterpretation.key
    context := taskQueueContext namespaceName taskQueueName taskQueueType
    samplingPoint := .task
    changeEffect := .nextRead
    interpretation := some matchingUpdateAckIntervalInterpretation
  }

def matchingWorkerRegistryNumBucketsUse : Except ConfigError (ConfigUse Int) :=
  checkedAuthoredUse {
    id := DeclarationId.of "temporal.matching.worker-registry-num-buckets"
    key := matchingWorkerRegistryNumBucketsInterpretation.key
    context := emptyConstraints
    samplingPoint := .processStartup
    changeEffect := .restartRequired
    interpretation := some matchingWorkerRegistryNumBucketsInterpretation
  }

inductive CallbackRoute where
  | legacyHsm
  | chasm
  deriving BEq, DecidableEq, Repr

inductive CallbackAdmission where
  | notRequested
  | admitted
  | rejectedOverflow
  | rejectedAddress (kind : CallbackAddressErrorKind)
  deriving BEq, DecidableEq, Repr

inductive CallbackDispatch where
  | notDispatched
  | succeeded
  | timedOut
  deriving BEq, DecidableEq, Repr

structure CallbackRequest where
  existingCallbacks : Nat
  newCallbacks : Nat
  address : String
  elapsedNanoseconds : Int
  deriving BEq, DecidableEq, Repr

private structure CallbackDomainConfigPayload where
  route : CallbackRoute
  maximumCallbacks : Int
  addressRules : CallbackAddressRules
  timeoutNanoseconds : Int
  deriving BEq, DecidableEq, Repr

/-- The four callback settings projected once from one validated immutable view. -/
structure CallbackDomainConfig where
  private mk ::
  private payload : CallbackDomainConfigPayload
  deriving BEq, DecidableEq, Repr

structure CallbackTrace where
  route : Option CallbackRoute
  admission : CallbackAdmission
  dispatch : CallbackDispatch
  deriving BEq, DecidableEq, Repr

def projectCallbackDomainConfig
    (view : ConfigView)
    (namespaceName destination : String) : Except ConfigError CallbackDomainConfig := do
  if destination == "" then
    throw (configError .missingContext (DeclarationId.of "temporal.callback.snapshot")
      callbackRequestTimeoutInterpretation.key "destination")
  let enableUse ← historyEnableChasmCallbacksUse namespaceName
  let maximumUse ← callbackMaxPerExecutionUse namespaceName
  let addressesUse ← callbackAllowedAddressesUse namespaceName
  let timeoutUse ← callbackRequestTimeoutUse namespaceName destination
  let enabled ← view.read enableUse
  let maximumCallbacks ← view.read maximumUse
  let addressRules ← view.read addressesUse
  let timeoutNanoseconds ← view.read timeoutUse
  pure (.mk {
    route := if enabled then .chasm else .legacyHsm
    maximumCallbacks
    addressRules
    timeoutNanoseconds
  })

/-- Evaluate callback admission and dispatch against only the captured callback projection. -/
def runCallbackTrace
    (config : CallbackDomainConfig)
    (request : CallbackRequest) : CallbackTrace :=
  if request.newCallbacks == 0 then
    { route := none, admission := .notRequested, dispatch := .notDispatched }
  else
    let route := some config.payload.route
    match config.payload.addressRules.validate request.address with
    | .error error => {
        route
        admission := .rejectedAddress error.kind
        dispatch := .notDispatched
      }
    | .ok _ =>
        if Int.ofNat (request.existingCallbacks + request.newCallbacks) >
            config.payload.maximumCallbacks then
          { route, admission := .rejectedOverflow, dispatch := .notDispatched }
        else
          let dispatch :=
            if config.payload.timeoutNanoseconds <= 0 ||
                request.elapsedNanoseconds >= config.payload.timeoutNanoseconds then
              .timedOut
            else
              .succeeded
          { route, admission := .admitted, dispatch }

end Temporal.Umpire.Config
