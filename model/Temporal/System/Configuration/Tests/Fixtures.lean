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

def maxNamespaceContext (namespaceName : String) : ExactConstraints :=
  { emptyConstraints with namespaceName := some namespaceName }

def errorKindOf (result : Except ConfigError α) : Option ConfigErrorKind :=
  match result with
  | .error error => some error.kind
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
