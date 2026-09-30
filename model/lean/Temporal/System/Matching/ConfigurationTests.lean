import Temporal.System.Matching.Configuration

namespace Temporal.System.Matching.ConfigurationTests

open Temporal.System.Configuration
open Temporal.System.Matching.Configuration

example : matchingUseDefinitions.map AnyCheckedConfigUseDefinition.key = [
    "matching.updateackinterval",
    "matching.workerregistrynumbuckets"
  ] := by
  native_decide

def matchingUseDefinitionsValid : Bool :=
  match validateConfigUseDefinitions matchingUseDefinitions with
  | .ok _ => true
  | .error _ => false

example : matchingUseDefinitionsValid = true := by native_decide

def matchingDefaults : Except ConfigError (Int × Int) := do
  let updateAck ← matchingUpdateAckIntervalUse "payments" "normal-queue" 1
  let buckets ← matchingWorkerRegistryNumBucketsUse
  let view ← resolveConfigView [] [.of updateAck, .of buckets]
  pure (← view.read updateAck, ← view.read buckets)

def matchingDefaultValues : Option (Int × Int) :=
  matchingDefaults.toOption

example : matchingDefaultValues = some (60000000000, 10) := by native_decide

end Temporal.System.Matching.ConfigurationTests
