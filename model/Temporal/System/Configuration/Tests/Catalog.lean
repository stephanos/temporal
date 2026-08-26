import Temporal.System.Configuration.Tests.Fixtures

/-! Catalog composition, constrained-default fixtures, and opaque-default drift checks. -/

namespace Temporal.System.ConfigurationTests

open _root_.Umpire
open Temporal.DynamicConfig
open Temporal.System.Configuration
open Temporal.System.Callback.Configuration
open Temporal.System.Matching.Configuration

example : authoredClassifications.length + matchingClassifications.length = 6 := by native_decide

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
