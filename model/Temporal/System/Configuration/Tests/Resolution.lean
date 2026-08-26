import Temporal.System.Configuration.Tests.Fixtures

/-! Deterministic configuration resolution, typed reads, isolation, and view provenance. -/

namespace Temporal.System.ConfigurationTests

open _root_.Umpire
open Temporal.System.Configuration
open Temporal.System.Callback.Configuration
open Temporal.System.Matching.Configuration

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

end Temporal.System.ConfigurationTests
