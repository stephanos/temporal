import Temporal.System

/-! Cross-consumer configuration composition and provenance checks. -/

namespace Temporal.System.ConfigurationIntegrationTests

open Temporal.DynamicConfig
open Temporal.System.Configuration
open Temporal.System.Callback.Configuration
open Temporal.System.Matching.Configuration

example : callbackUseDefinitions.map AnyCheckedConfigUseDefinition.key = [
    "history.enablechasmcallbacks",
    "callback.maxperexecution",
    "callback.allowedaddresses",
    "callback.request.timeout"
  ] := by
  native_decide

example : authoredClassifications.length + matchingClassifications.length = 6 := by native_decide

def callbackUseDefinitionsValid : Bool :=
  match validateConfigUseDefinitions callbackUseDefinitions with
  | .ok _ => true
  | .error _ => false

example : callbackUseDefinitionsValid = true := by native_decide

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

end Temporal.System.ConfigurationIntegrationTests
