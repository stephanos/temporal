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

def expectedUseDefinitionMetadata : List ConfigUseDefinitionMetadata := [
  {
    id := _root_.Umpire.DefinitionId.of "temporal.callback.enable-chasm"
    key := "history.enablechasmcallbacks"
    settingIdentity :=
      "sha256:415f169bb77c82582f2d8f5049648b5b079f4f1047a2f109d4ed9b14037d9c8c"
    impacts := [.feature, .externallyVisibleSemantics]
    contextPolicy := .namespace
    samplingPoint := .entityCreation
    changeEffect := .newEntitiesOnly
    interpretationFingerprint :=
      _root_.Umpire.behaviorFingerprintOf "temporal.config/history-enable-chasm-callbacks/v1"
  },
  {
    id := _root_.Umpire.DefinitionId.of "temporal.callback.max-per-execution"
    key := "callback.maxperexecution"
    settingIdentity :=
      "sha256:6c7f3b78bbbf74a83401b46faedf61250a1c4c2c92d02eab91ec9ebc36b30d71"
    impacts := [.validation]
    contextPolicy := .namespace
    samplingPoint := .request
    changeEffect := .nextRead
    interpretationFingerprint :=
      _root_.Umpire.behaviorFingerprintOf "temporal.config/callback-max-per-execution/v1"
  },
  {
    id := _root_.Umpire.DefinitionId.of "temporal.callback.allowed-addresses"
    key := "callback.allowedaddresses"
    settingIdentity :=
      "sha256:452cd642fac8adb5d5e1e2c0a4ef1d149cfb621ed663842c1bde7dd123faca9b"
    impacts := [.validation, .externallyVisibleSemantics]
    contextPolicy := .namespace
    samplingPoint := .request
    changeEffect := .nextRead
    interpretationFingerprint :=
      _root_.Umpire.behaviorFingerprintOf "temporal.config/callback-allowed-addresses/v1"
  },
  {
    id := _root_.Umpire.DefinitionId.of "temporal.callback.request-timeout"
    key := "callback.request.timeout"
    settingIdentity :=
      "sha256:cd2c7d65a4f41e7edcfa548d7433aeb7cd5a414c6a3258d361676cd3ada8fda9"
    impacts := [.timing]
    contextPolicy := .destination
    samplingPoint := .task
    changeEffect := .nextRead
    interpretationFingerprint :=
      _root_.Umpire.behaviorFingerprintOf "temporal.config/callback-request-timeout/v1"
  },
  {
    id := _root_.Umpire.DefinitionId.of "temporal.matching.update-ack-interval"
    key := "matching.updateackinterval"
    settingIdentity :=
      "sha256:58c6db0d991c651b92e007384724788f74236057d53c6814293a5439e216501f"
    impacts := [.timing, .performance]
    contextPolicy := .taskQueue
    samplingPoint := .task
    changeEffect := .nextRead
    interpretationFingerprint :=
      _root_.Umpire.behaviorFingerprintOf "temporal.config/matching-update-ack-interval/v1"
  },
  {
    id := _root_.Umpire.DefinitionId.of "temporal.matching.worker-registry-num-buckets"
    key := "matching.workerregistrynumbuckets"
    settingIdentity :=
      "sha256:6369ab31f72b574120e020fe8695290050ce1d2d66b4579e01243bbb4aea5f29"
    impacts := [.topology, .performance]
    contextPolicy := .global
    samplingPoint := .processStartup
    changeEffect := .restartRequired
    interpretationFingerprint :=
      _root_.Umpire.behaviorFingerprintOf
        "temporal.config/matching-worker-registry-num-buckets/v1"
  }
]

example :
    (callbackUseDefinitions ++ matchingUseDefinitions).map
        AnyCheckedConfigUseDefinition.metadata = expectedUseDefinitionMetadata := by
  native_decide

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
          entry.settingDigest != "" && entry.interpretationFingerprint.render != "" &&
          entry.key != "" &&
          entry.useId.value != ""

example : representativeMetadataComplete = true := by native_decide

end Temporal.System.ConfigurationIntegrationTests
