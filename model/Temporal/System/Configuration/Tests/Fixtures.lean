import Temporal.System

/-! Shared generic configuration-use and result-inspection helpers. -/

namespace Temporal.System.ConfigurationTests

open _root_.Umpire
open Temporal.DynamicConfig
open Temporal.System.Configuration
open Temporal.System.Callback.Configuration

def errorKindOf (result : Except ConfigError α) : Option ConfigErrorKind :=
  match result with
  | .error error => some error.kind
  | .ok _ => none

def maxRequest
    (useId namespaceName : String)
    (interpretation : Option (ConfigInterpretation Int) :=
      some callbackMaxPerExecutionInterpretation) : ConfigUseRequest Int := {
  id := DeclarationId.of useId
  key := "callback.maxperexecution"
  context := namespaceContext namespaceName
  samplingPoint := .request
  changeEffect := .nextRead
  interpretation
}

def checkedMaxUse (useId namespaceName : String) : Except ConfigError (ConfigUse Int) :=
  checkConfigUse authoredClassifications
    (maxRequest useId namespaceName)

end Temporal.System.ConfigurationTests
