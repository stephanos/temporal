import Temporal.System.Configuration.Tests.Fixtures

/-! Validation failures for checked configuration uses, overrides, and setting structure. -/

namespace Temporal.System.ConfigurationTests

open _root_.Umpire
open Temporal.DynamicConfig
open Temporal.System.Configuration
open Temporal.System.Callback.Configuration

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

end Temporal.System.ConfigurationTests
