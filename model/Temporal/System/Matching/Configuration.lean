import Temporal.System.Configuration

namespace Temporal.System.Matching.Configuration

open _root_.Umpire
open Temporal.DynamicConfig
open Temporal.System.Configuration

/-! Matching-owned meaning over shared dynamic-configuration resolution. -/

private def decodeInt : CanonicalValue → Except String Int
  | .int value => pure value
  | value => throw ("expected int, found " ++ reprStr value)

private def decodeDuration : CanonicalValue → Except String Int
  | .duration nanoseconds => pure nanoseconds
  | value => throw ("expected duration, found " ++ reprStr value)

private def matchingUpdateAckIntervalSpec : ConfigUseSpec Int := {
  id := DefinitionId.of "temporal.matching.update-ack-interval"
  key := "matching.updateackinterval"
  settingIdentity := "sha256:58c6db0d991c651b92e007384724788f74236057d53c6814293a5439e216501f"
  impacts := [.timing, .performance]
  expectedSchema := .duration "time.Duration" false
  expectedDefault := .constrained [
    {
      constraints := emptyConstraints
      value := .concrete (.duration 60000000000)
    },
    {
      constraints := { emptyConstraints with taskQueueName := some "temporal-sys-per-ns-tq" }
      value := .concrete (.duration 300000000000)
    }
  ]
  behaviorFingerprint := behaviorFingerprintOf "temporal.config/matching-update-ack-interval/v1"
  decode := decodeDuration
  contextPolicy := .taskQueue
  samplingPoint := .task
  changeEffect := .nextRead
}

private def matchingWorkerRegistryNumBucketsSpec : ConfigUseSpec Int := {
  id := DefinitionId.of "temporal.matching.worker-registry-num-buckets"
  key := "matching.workerregistrynumbuckets"
  settingIdentity := "sha256:6369ab31f72b574120e020fe8695290050ce1d2d66b4579e01243bbb4aea5f29"
  impacts := [.topology, .performance]
  expectedSchema := .int "int" false
  expectedDefault := .concrete (.int 10)
  behaviorFingerprint := behaviorFingerprintOf "temporal.config/matching-worker-registry-num-buckets/v1"
  decode := decodeInt
  contextPolicy := .global
  samplingPoint := .processStartup
  changeEffect := .restartRequired
}

def taskQueueContext
    (namespaceName taskQueueName : String)
    (taskQueueType : Int) : ExactConstraints :=
  { emptyConstraints with
      namespaceName := some namespaceName
      taskQueueName := some taskQueueName
      taskQueueType := some taskQueueType }

private def matchingUpdateAckIntervalDefinition : CheckedConfigUseDefinition Int :=
  matchingUpdateAckIntervalSpec.checked (by native_decide)

private def matchingWorkerRegistryNumBucketsDefinition : CheckedConfigUseDefinition Int :=
  matchingWorkerRegistryNumBucketsSpec.checked (by native_decide)

def matchingUseDefinitions : List AnyCheckedConfigUseDefinition := [
  .of matchingUpdateAckIntervalDefinition,
  .of matchingWorkerRegistryNumBucketsDefinition
]

def matchingUpdateAckIntervalUse
    (namespaceName taskQueueName : String)
    (taskQueueType : Int) : Except ConfigError (ConfigUse Int) :=
  matchingUpdateAckIntervalDefinition.instantiate
    (taskQueueContext namespaceName taskQueueName taskQueueType)

def matchingWorkerRegistryNumBucketsUse : Except ConfigError (ConfigUse Int) :=
  matchingWorkerRegistryNumBucketsDefinition.instantiate emptyConstraints

end Temporal.System.Matching.Configuration
