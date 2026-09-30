import Temporal.System.Configuration.Tests.Fixtures

/-! Catalog composition, constrained-default fixtures, and opaque-default drift checks. -/

namespace Temporal.System.ConfigurationTests

open _root_.Umpire
open Temporal.DynamicConfig
open Temporal.System.Configuration

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
  behaviorFingerprint := behaviorFingerprintOf "temporal.config/frontend-http-allowed-hosts/v1"
  decode := pure
}

def opaqueSpec
    (replacement : Option OpaqueDefaultReplacement) : ConfigUseSpec CanonicalValue := {
  id := DefinitionId.of "test.config.opaque-default"
  key := opaqueClassification.key
  settingIdentity := opaqueClassification.settingIdentity
  impacts := opaqueClassification.impacts
  expectedSchema := Temporal.DynamicConfig.Settings.frontend_httpallowedhosts.schema
  expectedDefault := .opaque opaqueMetadata
  opaqueReplacement := replacement
  behaviorFingerprint := behaviorFingerprintOf "temporal.config/frontend-http-allowed-hosts/v1"
  decode := pure
  contextPolicy := .global
  samplingPoint := .processStartup
  changeEffect := .restartRequired
}

def checkedOpaqueUse
    (replacement : Option OpaqueDefaultReplacement) : Except ConfigError (ConfigUse CanonicalValue) :=
  checkConfigUse [opaqueClassification] {
    id := DefinitionId.of "test.config.opaque-default"
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

def replacedOpaqueSpecResult : Except ConfigError CanonicalValue := do
  let replacement : OpaqueDefaultReplacement := {
    expected := opaqueMetadata
    value := .object .nil
  }
  let definition ← (opaqueSpec (some replacement)).check
  let use ← definition.instantiate emptyConstraints
  let view ← resolveConfigView [] [.of use]
  view.read use

def staleOpaqueSpecResult : Except ConfigError (CheckedConfigUseDefinition CanonicalValue) :=
  let replacement : OpaqueDefaultReplacement := {
    expected := { opaqueMetadata with reason := "stale" }
    value := .object .nil
  }
  (opaqueSpec (some replacement)).check

def malformedOpaqueSpecResult : Except ConfigError (CheckedConfigUseDefinition CanonicalValue) :=
  let replacement : OpaqueDefaultReplacement := {
    expected := opaqueMetadata
    value := .int 1
  }
  (opaqueSpec (some replacement)).check

def opaqueDecoderFailureSpecResult :
    Except ConfigError (CheckedConfigUseDefinition CanonicalValue) :=
  let replacement : OpaqueDefaultReplacement := {
    expected := opaqueMetadata
    value := .object .nil
  }
  ({ opaqueSpec (some replacement) with
      decode := fun _ => throw "intentional opaque decoder failure" } :
    ConfigUseSpec CanonicalValue).check

example : errorKindOf selectedOpaqueDefaultResult = some .opaqueDefaultSelected := by native_decide

def replacedOpaqueDefaultMatches : Bool :=
  match replacedOpaqueDefaultResult with
  | .ok (.object .nil) => true
  | _ => false

example : replacedOpaqueDefaultMatches = true := by native_decide

def replacedOpaqueSpecMatches : Bool :=
  match replacedOpaqueSpecResult with
  | .ok (.object .nil) => true
  | _ => false

example : replacedOpaqueSpecMatches = replacedOpaqueDefaultMatches := by native_decide

example : errorKindOf staleOpaqueReplacementResult = some .defaultDrift := by native_decide

example : errorKindOf malformedOpaqueReplacementResult = some .schemaMismatch := by native_decide

example :
    [configErrorOf staleOpaqueSpecResult,
     configErrorOf malformedOpaqueSpecResult,
     configErrorOf opaqueDecoderFailureSpecResult] =
    [some {
       kind := .defaultDrift
       useId := DefinitionId.of "test.config.opaque-default"
       key := opaqueClassification.key
       offendingValue := reprStr [opaqueMetadata]
       relatedIdentities := []
     },
     some {
       kind := .schemaMismatch
       useId := DefinitionId.of "test.config.opaque-default"
       key := opaqueClassification.key
       offendingValue := reprStr (CanonicalValue.int 1)
       relatedIdentities := []
     },
     some {
       kind := .interpretationFailure
       useId := DefinitionId.of "test.config.opaque-default"
       key := opaqueClassification.key
       offendingValue := "intentional opaque decoder failure"
       relatedIdentities := []
     }] := by native_decide

end Temporal.System.ConfigurationTests
