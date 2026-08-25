import Temporal.System.Callback.Configuration
import Temporal.System.Matching.Configuration

namespace Temporal.System.ConfigurationTests

open _root_.Umpire
open Temporal.DynamicConfig
open Temporal.System.Configuration
open Temporal.System.Callback.Configuration
open Temporal.System.Matching.Configuration

def errorKindOf (result : Except ConfigError α) : Option ConfigErrorKind :=
  match result with
  | .error error => some error.kind
  | .ok _ => none

def maxRequest
    (useId namespaceName : String)
    (interpretation : Option (ConfigInterpretation Int) :=
      some callbackMaxPerExecutionInterpretation) : ConfigUseRequest Int := {
  id := DeclarationId.of useId
  key := "callback.maxperexecution"
  context := namespaceContext namespaceName
  samplingPoint := .request
  changeEffect := .nextRead
  interpretation
}

def checkedMaxUse (useId namespaceName : String) : Except ConfigError (ConfigUse Int) :=
  checkConfigUse authoredClassifications
    (maxRequest useId namespaceName)

example : authoredClassifications.length + matchingClassifications.length = 6 := by native_decide

def unknownUseResult : Except ConfigError (ConfigUse Unit) :=
  checkConfigUse authoredClassifications {
    id := DeclarationId.of "test.config.unknown"
    key := "does.not.exist"
    context := emptyConstraints
    samplingPoint := .request
    changeEffect := .nextRead
    interpretation := none
  }

def unclassifiedUseResult : Except ConfigError (ConfigUse Unit) :=
  checkConfigUse authoredClassifications {
    id := DeclarationId.of "test.config.unclassified"
    key := "admin.enablelisthistorytasks"
    context := emptyConstraints
    samplingPoint := .request
    changeEffect := .nextRead
    interpretation := none
  }

def emptyClassificationResult : Except ConfigError (ConfigUse Int) :=
  let classification : SettingClassification := {
    key := "callback.maxperexecution"
    settingIdentity := Temporal.DynamicConfig.Settings.callback_maxperexecution.identity
    impacts := []
  }
  checkConfigUse [classification]
    (maxRequest "test.config.empty-classification" "payments")

def missingInterpretationResult : Except ConfigError (ConfigUse Int) :=
  checkConfigUse authoredClassifications
    (maxRequest "test.config.missing-interpretation" "payments" none)

def incompatibleInterpretationResult : Except ConfigError (ConfigUse Int) :=
  let interpretation := { callbackMaxPerExecutionInterpretation with key := "other.key" }
  checkConfigUse authoredClassifications
    (maxRequest "test.config.incompatible-interpretation" "payments" (some interpretation))

def schemaDriftResult : Except ConfigError (ConfigUse Int) :=
  let interpretation := {
    callbackMaxPerExecutionInterpretation with expectedSchema := .bool "bool" false
  }
  checkConfigUse authoredClassifications
    (maxRequest "test.config.schema-drift" "payments" (some interpretation))

def defaultDriftResult : Except ConfigError (ConfigUse Int) :=
  let interpretation := {
    callbackMaxPerExecutionInterpretation with expectedDefault := .concrete (.int 1999)
  }
  checkConfigUse authoredClassifications
    (maxRequest "test.config.default-drift" "payments" (some interpretation))

def missingContextResult : Except ConfigError (ConfigUse Int) :=
  checkConfigUse authoredClassifications {
    maxRequest "test.config.missing-context" "payments" with context := emptyConstraints
  }

def illegalContextResult : Except ConfigError (ConfigUse Int) :=
  checkConfigUse authoredClassifications {
    maxRequest "test.config.illegal-context" "payments" with
      context := { namespaceContext "payments" with destination := some "callback-api" }
  }

def malformedUseResult : Except ConfigError (ConfigUse Int) :=
  checkConfigUse authoredClassifications
    (maxRequest "not-namespaced" "payments")

example :
    [errorKindOf unknownUseResult,
     errorKindOf unclassifiedUseResult,
     errorKindOf emptyClassificationResult,
     errorKindOf missingInterpretationResult,
     errorKindOf incompatibleInterpretationResult,
     errorKindOf schemaDriftResult,
     errorKindOf defaultDriftResult,
     errorKindOf missingContextResult,
     errorKindOf illegalContextResult,
     errorKindOf malformedUseResult] =
    [some .unknownKey,
     some .unclassifiedKey,
     some .emptyClassification,
     some .missingInterpretation,
     some .incompatibleInterpretation,
     some .schemaMismatch,
     some .defaultDrift,
     some .missingContext,
     some .illegalConstraints,
     some .malformedUse] := by native_decide

def duplicateOverrideResult : Except ConfigError ConfigView := do
  let use ← checkedMaxUse "test.config.duplicate-override" "payments"
  let override ← checkConfigOverride use (namespaceContext "payments") (.int 20)
  resolveConfigView [override, override] [.of use]

def illegalOverrideResult : Except ConfigError ConfigView := do
  let use ← checkedMaxUse "test.config.illegal-override" "payments"
  let override ← checkConfigOverride use
    { emptyConstraints with destination := some "callback-api" } (.int 20)
  resolveConfigView [override] [.of use]

def schemaMismatchOverrideResult : Except ConfigError ConfigView := do
  let use ← checkedMaxUse "test.config.schema-mismatch-override" "payments"
  let override ← checkConfigOverride use (namespaceContext "payments") (.bool true)
  resolveConfigView [override] [.of use]

def duplicateUseResult : Except ConfigError ConfigView := do
  let use ← checkedMaxUse "test.config.duplicate-use" "payments"
  resolveConfigView [] [.of use, .of use]

example :
    [errorKindOf duplicateOverrideResult,
     errorKindOf illegalOverrideResult,
     errorKindOf schemaMismatchOverrideResult,
     errorKindOf duplicateUseResult] =
    [some .duplicateConstraints,
     some .illegalConstraints,
     some .schemaMismatch,
     some .duplicateUse] := by native_decide

def duplicateConstrainedDefaultSetting : Setting := {
  Temporal.DynamicConfig.Settings.matching_updateackinterval with
    defaultValue := .constrained [
      { constraints := emptyConstraints, value := .concrete (.duration 1) },
      { constraints := emptyConstraints, value := .concrete (.duration 2) }]
}

example : errorKindOf (validateSettingStructure duplicateConstrainedDefaultSetting) =
    some .duplicateConstraints := by native_decide

def valuesOfList : List CanonicalValue → CanonicalValues
  | [] => .nil
  | value :: rest => .cons value (valuesOfList rest)

def addressRuleValue (pattern : String) (allowInsecure : Bool) : CanonicalValue :=
  .object (.cons "Pattern" (.string pattern)
    (.cons "AllowInsecure" (.bool allowInsecure) .nil))

def addressRulesValue (rules : List CanonicalValue) : CanonicalValue :=
  .object (.cons "Rules" (.list (valuesOfList rules)) .nil)

def malformedUnselectedAddressOverrideResult : Except ConfigError ConfigView := do
  let use ← callbackAllowedAddressesUse "payments"
  let selected ← checkConfigOverride use (namespaceContext "payments")
    (addressRulesValue [addressRuleValue "api.example.com" false])
  let malformed ← checkConfigOverride use emptyConstraints
    (addressRulesValue [addressRuleValue "" false])
  resolveConfigView [selected, malformed] [.of use]

example : errorKindOf malformedUnselectedAddressOverrideResult =
    some .interpretationFailure := by native_decide

def sameKeyView (reverseInput : Bool) : Except ConfigError ConfigView := do
  let first ← checkedMaxUse "test.config.consumer-a" "alpha"
  let second ← checkedMaxUse "test.config.consumer-b" "beta"
  let firstOverride ← checkConfigOverride first (namespaceContext "alpha") (.int 11)
  let secondOverride ← checkConfigOverride second (namespaceContext "beta") (.int 22)
  if reverseInput then
    resolveConfigView [secondOverride, firstOverride] [.of second, .of first]
  else
    resolveConfigView [firstOverride, secondOverride] [.of first, .of second]

def sameKeyTypedReads : Except ConfigError (Int × Int) := do
  let view ← sameKeyView false
  let first ← checkedMaxUse "test.config.consumer-a" "alpha"
  let second ← checkedMaxUse "test.config.consumer-b" "beta"
  pure (← view.read first, ← view.read second)

def sameKeyViewsEqual : Bool :=
  match sameKeyView false, sameKeyView true with
  | .ok first, .ok second => decide (first = second)
  | _, _ => false

def sameKeyTypedReadsMatch : Bool :=
  match sameKeyTypedReads with
  | .ok (11, 22) => true
  | _ => false

example : sameKeyViewsEqual = true := by native_decide

example : sameKeyTypedReadsMatch = true := by native_decide

def originatingUseReadResult : Except ConfigError Int := do
  let original ← checkedMaxUse "test.config.originating-use" "alpha"
  let otherContext ← checkedMaxUse "test.config.originating-use" "beta"
  let view ← resolveConfigView [] [.of original]
  view.read otherContext

example : errorKindOf originatingUseReadResult = some .incompatibleInterpretation := by
  native_decide

def immutableViewReads : Except ConfigError (Int × Int × Int) := do
  let use ← checkedMaxUse "test.config.immutable" "payments"
  let view ← resolveConfigView [] [.of use]
  let before ← view.read use
  let override ← checkConfigOverride use (namespaceContext "payments") (.int 10)
  let changed ← resolveConfigView [override] [.of use]
  let changedValue ← changed.read use
  let after ← view.read use
  pure (before, changedValue, after)

def immutableViewReadsMatch : Bool :=
  match immutableViewReads with
  | .ok (2000, 10, 2000) => true
  | _ => false

example : immutableViewReadsMatch = true := by native_decide

def representativeView : Except ConfigError ConfigView := do
  let enable ← historyEnableChasmCallbacksUse "payments"
  let maximum ← callbackMaxPerExecutionUse "payments"
  let addresses ← callbackAllowedAddressesUse "payments"
  let timeout ← callbackRequestTimeoutUse "payments" "callback-api"
  let updateAck ← matchingUpdateAckIntervalUse "payments" "normal-queue" 1
  let buckets ← matchingWorkerRegistryNumBucketsUse
  resolveConfigView []
    [.of enable, .of maximum, .of addresses, .of timeout, .of updateAck, .of buckets]

def representativeMetadataComplete : Bool :=
  match representativeView with
  | .error _ => false
  | .ok view =>
      view.entryCount == 6 && view.provenance.all fun entry =>
        entry.catalogDigest == Temporal.DynamicConfig.Settings.catalogIdentity &&
          entry.settingDigest != "" && entry.interpretationDigest != "" && entry.key != "" &&
          entry.useId.value != ""

example : representativeMetadataComplete = true := by native_decide

def constrainedDefaultInterleaving : Except ConfigError (Int × ResolutionSource) := do
  let use ← matchingUpdateAckIntervalUse "fixture-namespace" "temporal-sys-per-ns-tq" 1
  let namespaceOverride ← checkConfigOverride use
    (namespaceContext "fixture-namespace") (.duration 120000000000)
  let view ← resolveConfigView [namespaceOverride] [.of use]
  let value ← view.read use
  match view.provenance with
  | [entry] => pure (value, entry.source)
  | _ => throw {
      kind := .fixtureMismatch
      useId := use.id
      key := use.key
      offendingValue := "unexpected entry count"
      relatedIdentities := []
    }

def constrainedDefaultInterleavingMatches : Bool :=
  match constrainedDefaultInterleaving with
  | .ok (300000000000, .constrainedDefault) => true
  | _ => false

example : constrainedDefaultInterleavingMatches = true := by native_decide

example : Temporal.DynamicConfig.Settings.fixtures.length = 13 := by native_decide

example : errorKindOf checkAllResolutionFixtures = none := by native_decide

example : errorKindOf (checkFixtureCatalogIdentity "sha256:stale") =
    some .fixtureMismatch := by native_decide

def mismatchedFixtureResult : Except ConfigError Unit :=
  match Temporal.DynamicConfig.Settings.fixtures with
  | [] => checkAllResolutionFixtures
  | fixture :: _ => checkResolutionFixture { fixture with result := .duration (-1) }

example : errorKindOf mismatchedFixtureResult = some .fixtureMismatch := by native_decide

def opaqueMetadata : OpaqueDefault := {
  goType := "*regexp.Regexp"
  reason := "default contains unsupported unexported mutable value at *.prog"
}

def opaqueClassification : SettingClassification := {
  key := "frontend.httpallowedhosts"
  settingIdentity := "sha256:a7cbb5378c5ac8aabe60fe8e0f4bd1559152659f01b384233cd97bfd89323f1c"
  impacts := [.validation]
}

def opaqueInterpretation
    (replacement : Option OpaqueDefaultReplacement) : ConfigInterpretation CanonicalValue := {
  key := opaqueClassification.key
  expectedSettingIdentity := opaqueClassification.settingIdentity
  expectedSchema := Temporal.DynamicConfig.Settings.frontend_httpallowedhosts.schema
  expectedDefault := .opaque opaqueMetadata
  opaqueReplacement := replacement
  semanticDigest := semanticDigestOf "temporal.config/frontend-http-allowed-hosts/v1"
  decode := pure
}

def checkedOpaqueUse
    (replacement : Option OpaqueDefaultReplacement) : Except ConfigError (ConfigUse CanonicalValue) :=
  checkConfigUse [opaqueClassification] {
    id := DeclarationId.of "test.config.opaque-default"
    key := opaqueClassification.key
    context := emptyConstraints
    samplingPoint := .processStartup
    changeEffect := .restartRequired
    interpretation := some (opaqueInterpretation replacement)
  }

def selectedOpaqueDefaultResult : Except ConfigError ConfigView := do
  let use ← checkedOpaqueUse none
  resolveConfigView [] [.of use]

def replacedOpaqueDefaultResult : Except ConfigError CanonicalValue := do
  let replacement : OpaqueDefaultReplacement := {
    expected := opaqueMetadata
    value := .object .nil
  }
  let use ← checkedOpaqueUse (some replacement)
  let view ← resolveConfigView [] [.of use]
  view.read use

def staleOpaqueReplacementResult : Except ConfigError ConfigView := do
  let replacement : OpaqueDefaultReplacement := {
    expected := { opaqueMetadata with reason := "stale" }
    value := .object .nil
  }
  let use ← checkedOpaqueUse (some replacement)
  resolveConfigView [] [.of use]

def malformedOpaqueReplacementResult : Except ConfigError ConfigView := do
  let replacement : OpaqueDefaultReplacement := {
    expected := opaqueMetadata
    value := .int 1
  }
  let use ← checkedOpaqueUse (some replacement)
  resolveConfigView [] [.of use]

example : errorKindOf selectedOpaqueDefaultResult = some .opaqueDefaultSelected := by native_decide

def replacedOpaqueDefaultMatches : Bool :=
  match replacedOpaqueDefaultResult with
  | .ok (.object .nil) => true
  | _ => false

example : replacedOpaqueDefaultMatches = true := by native_decide

example : errorKindOf staleOpaqueReplacementResult = some .defaultDrift := by native_decide

example : errorKindOf malformedOpaqueReplacementResult = some .schemaMismatch := by native_decide

end Temporal.System.ConfigurationTests
