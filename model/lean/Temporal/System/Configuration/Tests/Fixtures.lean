import Temporal.System.Configuration

/-! Shared generic configuration-use and result-inspection helpers. -/

namespace Temporal.System.ConfigurationTests

open _root_.Umpire
open Temporal.DynamicConfig
open Temporal.System.Configuration

private def decodeInt : CanonicalValue → Except String Int
  | .int value => pure value
  | value => throw ("expected int, found " ++ reprStr value)

def maxClassification : SettingClassification := {
  key := "callback.maxperexecution"
  settingIdentity := Temporal.DynamicConfig.Settings.callback_maxperexecution.identity
  impacts := [.validation]
}

def maxInterpretation : ConfigInterpretation Int := {
  key := maxClassification.key
  expectedSettingIdentity := maxClassification.settingIdentity
  expectedSchema := Temporal.DynamicConfig.Settings.callback_maxperexecution.schema
  expectedDefault := Temporal.DynamicConfig.Settings.callback_maxperexecution.defaultValue
  behaviorFingerprint := behaviorFingerprintOf "test.config/callback-max-per-execution/v1"
  decode := decodeInt
}

def maxDefinition : ConfigUseDefinition Int := {
  id := DefinitionId.of "test.config.max-definition"
  classification := maxClassification
  contextPolicy := .namespace
  samplingPoint := .request
  changeEffect := .nextRead
  interpretation := maxInterpretation
}

def maxSpec : ConfigUseSpec Int := {
  id := DefinitionId.of "test.config.max-definition"
  key := "callback.maxperexecution"
  settingIdentity :=
    "sha256:6c7f3b78bbbf74a83401b46faedf61250a1c4c2c92d02eab91ec9ebc36b30d71"
  impacts := [.validation]
  expectedSchema := .int "int" false
  expectedDefault := .concrete (.int 2000)
  behaviorFingerprint := behaviorFingerprintOf "test.config/callback-max-per-execution/v1"
  decode := decodeInt
  contextPolicy := .namespace
  samplingPoint := .request
  changeEffect := .nextRead
}

private theorem maxSpecCheck_isSome : maxSpec.check.toOption.isSome = true := by native_decide

def checkedMaxSpec : CheckedConfigUseDefinition Int :=
  maxSpec.checked maxSpecCheck_isSome

def maxNamespaceContext (namespaceName : String) : ExactConstraints :=
  { emptyConstraints with namespaceName := some namespaceName }

def errorKindOf (result : Except ConfigError α) : Option ConfigErrorKind :=
  match result with
  | .error error => some error.kind
  | .ok _ => none

def configErrorOf (result : Except ConfigError α) : Option ConfigError :=
  match result with
  | .error error => some error
  | .ok _ => none

def maxRequest
    (useId namespaceName : String)
    (interpretation : Option (ConfigInterpretation Int) :=
      some maxInterpretation) : ConfigUseRequest Int := {
  id := DefinitionId.of useId
  key := "callback.maxperexecution"
  context := maxNamespaceContext namespaceName
  samplingPoint := .request
  changeEffect := .nextRead
  interpretation
}

def checkedMaxUse (useId namespaceName : String) : Except ConfigError (ConfigUse Int) :=
  checkConfigUse [maxClassification]
    (maxRequest useId namespaceName)

end Temporal.System.ConfigurationTests
