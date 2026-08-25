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

def matchingClassifications : List SettingClassification :=
  [{ key := "matching.updateackinterval"
     settingIdentity := "sha256:58c6db0d991c651b92e007384724788f74236057d53c6814293a5439e216501f"
     impacts := [.timing, .performance] },
   { key := "matching.workerregistrynumbuckets"
     settingIdentity := "sha256:6369ab31f72b574120e020fe8695290050ce1d2d66b4579e01243bbb4aea5f29"
     impacts := [.topology, .performance] }]

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

def taskQueueContext
    (namespaceName taskQueueName : String)
    (taskQueueType : Int) : ExactConstraints :=
  { emptyConstraints with
      namespaceName := some namespaceName
      taskQueueName := some taskQueueName
      taskQueueType := some taskQueueType }

private def checkedMatchingUse
    (request : ConfigUseRequest α) : Except ConfigError (ConfigUse α) :=
  checkConfigUse matchingClassifications request

def matchingUpdateAckIntervalUse
    (namespaceName taskQueueName : String)
    (taskQueueType : Int) : Except ConfigError (ConfigUse Int) :=
  checkedMatchingUse {
    id := DeclarationId.of "temporal.matching.update-ack-interval"
    key := matchingUpdateAckIntervalInterpretation.key
    context := taskQueueContext namespaceName taskQueueName taskQueueType
    samplingPoint := .task
    changeEffect := .nextRead
    interpretation := some matchingUpdateAckIntervalInterpretation
  }

def matchingWorkerRegistryNumBucketsUse : Except ConfigError (ConfigUse Int) :=
  checkedMatchingUse {
    id := DeclarationId.of "temporal.matching.worker-registry-num-buckets"
    key := matchingWorkerRegistryNumBucketsInterpretation.key
    context := emptyConstraints
    samplingPoint := .processStartup
    changeEffect := .restartRequired
    interpretation := some matchingWorkerRegistryNumBucketsInterpretation
  }

end Temporal.System.Matching.Configuration
