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

def matchingUpdateAckIntervalClassification : SettingClassification := {
  key := "matching.updateackinterval"
  settingIdentity := "sha256:58c6db0d991c651b92e007384724788f74236057d53c6814293a5439e216501f"
  impacts := [.timing, .performance]
}

def matchingWorkerRegistryNumBucketsClassification : SettingClassification := {
  key := "matching.workerregistrynumbuckets"
  settingIdentity := "sha256:6369ab31f72b574120e020fe8695290050ce1d2d66b4579e01243bbb4aea5f29"
  impacts := [.topology, .performance]
}

def matchingClassifications : List SettingClassification := [
  matchingUpdateAckIntervalClassification,
  matchingWorkerRegistryNumBucketsClassification
]

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

def matchingUpdateAckIntervalDefinitionResult :
    Except ConfigError (CheckedConfigUseDefinition Int) :=
  checkConfigUseDefinition {
    id := DeclarationId.of "temporal.matching.update-ack-interval"
    classification := matchingUpdateAckIntervalClassification
    contextPolicy := .taskQueue
    samplingPoint := .task
    changeEffect := .nextRead
    interpretation := matchingUpdateAckIntervalInterpretation
  }

def matchingWorkerRegistryNumBucketsDefinitionResult :
    Except ConfigError (CheckedConfigUseDefinition Int) :=
  checkConfigUseDefinition {
    id := DeclarationId.of "temporal.matching.worker-registry-num-buckets"
    classification := matchingWorkerRegistryNumBucketsClassification
    contextPolicy := .global
    samplingPoint := .processStartup
    changeEffect := .restartRequired
    interpretation := matchingWorkerRegistryNumBucketsInterpretation
  }

private theorem matchingUpdateAckIntervalDefinitionResult_isSome :
    matchingUpdateAckIntervalDefinitionResult.toOption.isSome = true := by native_decide
private theorem matchingWorkerRegistryNumBucketsDefinitionResult_isSome :
    matchingWorkerRegistryNumBucketsDefinitionResult.toOption.isSome = true := by native_decide

def matchingUpdateAckIntervalDefinition : CheckedConfigUseDefinition Int :=
  matchingUpdateAckIntervalDefinitionResult.toOption.get
    matchingUpdateAckIntervalDefinitionResult_isSome

def matchingWorkerRegistryNumBucketsDefinition : CheckedConfigUseDefinition Int :=
  matchingWorkerRegistryNumBucketsDefinitionResult.toOption.get
    matchingWorkerRegistryNumBucketsDefinitionResult_isSome

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
