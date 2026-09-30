import Temporal.System.Callback.Configuration
import Temporal.System.Configuration.Tests.Fixtures

namespace Temporal.System.Callback.ConfigurationTests

open _root_.Umpire
open Temporal.DynamicConfig
open Temporal.System.Configuration
open Temporal.System.ConfigurationTests
open Temporal.System.Callback.Configuration

def callbackRules : CallbackAddressRules := {
  rules := [
    { pattern := "api.*.example.com", allowInsecure := false },
    { pattern := "*.insecure.test:80", allowInsecure := true },
    { pattern := "*", allowInsecure := true }]
}

def callbackValidationKind (address : String) : Option CallbackAddressErrorKind :=
  match callbackRules.validate address with
  | .ok _ => none
  | .error error => some error.kind

example :
    [callbackValidationKind "temporal://system",
     callbackValidationKind "temporal://internal",
     callbackValidationKind "temporal://system/path",
     callbackValidationKind "https://api.a.b.example.com/path",
     callbackValidationKind "http://api.a.example.com",
     callbackValidationKind "http://hooks.insecure.test:80/path",
     callbackValidationKind "ftp://api.a.example.com",
     callbackValidationKind "https:///missing-host",
     callbackValidationKind "https://user:secret@api.a.b.example.com/path",
     callbackValidationKind "https://api.a.example.com:invalid",
     callbackValidationKind "https://api.%zz.example.com",
     callbackValidationKind "https://[2001:db8::1]/path",
     callbackValidationKind "https://[not:ipv6]"] =
    [none,
     none,
     some .unknownScheme,
     none,
     some .insecureConnection,
     none,
     some .unknownScheme,
     some .missingHost,
     none,
     some .malformedAddress,
     some .malformedAddress,
     none,
     some .malformedAddress] := by native_decide

def secureRuleWins : CallbackAddressRules := {
  rules := [
    { pattern := "api.example.com", allowInsecure := false },
    { pattern := "*", allowInsecure := true }]
}

def secureRuleResult : Option CallbackAddressErrorKind :=
  match secureRuleWins.validate "http://api.example.com" with
  | .ok _ => none
  | .error error => some error.kind

example : secureRuleResult = some .insecureConnection := by native_decide

def restrictedRules : CallbackAddressRules := {
  rules := [{ pattern := "api.example.com", allowInsecure := false }]
}

def unmatchedRuleResult : Option CallbackAddressErrorKind :=
  match restrictedRules.validate "https://other.example.com" with
  | .ok _ => none
  | .error error => some error.kind

example : unmatchedRuleResult = some .unmatchedAddress := by native_decide

example :
    [wholeHostWildcardMatch "*.example.com" "api.example.com",
     wholeHostWildcardMatch "*.example.com" "api.example.com.evil",
     wholeHostWildcardMatch "api.*.example.com" "api.a.b.example.com"] =
    [true, false, true] := by native_decide

def valuesOfList : List CanonicalValue → CanonicalValues
  | [] => .nil
  | value :: rest => .cons value (valuesOfList rest)

def addressRuleValue (pattern : String) (allowInsecure : Bool) : CanonicalValue :=
  .object (.cons "Pattern" (.string pattern)
    (.cons "AllowInsecure" (.bool allowInsecure) .nil))

def addressRulesValue (rules : List CanonicalValue) : CanonicalValue :=
  .object (.cons "Rules" (.list (valuesOfList rules)) .nil)

def malformedUnselectedAddressOverrideResult : Except ConfigError ConfigView := do
  let use ← callbackAllowedAddressesUse "payments"
  let selected ← checkConfigOverride use (namespaceContext "payments")
    (addressRulesValue [addressRuleValue "api.example.com" false])
  let malformed ← checkConfigOverride use emptyConstraints
    (addressRulesValue [addressRuleValue "" false])
  resolveConfigView [selected, malformed] [.of use]

example : errorKindOf malformedUnselectedAddressOverrideResult =
    some .interpretationFailure := by native_decide

def callbackConsumerRulesValue : CanonicalValue :=
  addressRulesValue [
    addressRuleValue "api.*.example.com" false,
    addressRuleValue "*.insecure.test:80" true]

def callbackConsumerView
    (enabled : Bool)
    (maximum timeoutNanoseconds : Int) : Except ConfigError ConfigView := do
  let plan : CallbackConfigPlan := { namespaceName := "payments", destination := "callback-api" }
  let enableUse ← historyEnableChasmCallbacksUse "payments"
  let maximumUse ← callbackMaxPerExecutionUse "payments"
  let addressesUse ← callbackAllowedAddressesUse "payments"
  let timeoutUse ← callbackRequestTimeoutUse "payments" "callback-api"
  let enableOverride ← checkConfigOverride enableUse
    (namespaceContext "payments") (.bool enabled)
  let maximumOverride ← checkConfigOverride maximumUse
    (namespaceContext "payments") (.int maximum)
  let addressesOverride ← checkConfigOverride addressesUse
    (namespaceContext "payments") callbackConsumerRulesValue
  let timeoutOverride ← checkConfigOverride timeoutUse
    (destinationContext "payments" "callback-api") (.duration timeoutNanoseconds)
  plan.resolve [enableOverride, maximumOverride, addressesOverride, timeoutOverride]

def callbackConsumerConfig
    (enabled : Bool)
    (maximum timeoutNanoseconds : Int) : Except ConfigError CallbackDomainConfig := do
  let view ← callbackConsumerView enabled maximum timeoutNanoseconds
  ({ namespaceName := "payments", destination := "callback-api" } : CallbackConfigPlan).project view

def callbackRequest
    (existingCallbacks newCallbacks : Nat)
    (address : String)
    (elapsedNanoseconds : Int) : CallbackRequest := {
  existingCallbacks
  newCallbacks
  address
  elapsedNanoseconds
}

def callbackCountBoundaries : Except ConfigError (List CallbackAdmission) := do
  let config ← callbackConsumerConfig false 2 10
  pure [
    (runCallbackTrace config (callbackRequest 1 1 "https://api.a.example.com" 1)).admission,
    (runCallbackTrace config (callbackRequest 2 1 "https://api.a.example.com" 1)).admission]

def callbackCountBoundariesMatch : Bool :=
  match callbackCountBoundaries with
  | .ok [.admitted, .rejectedOverflow] => true
  | _ => false

example : callbackCountBoundariesMatch = true := by native_decide

def zeroCallbackRequests : Except ConfigError
    (List (Option CallbackRoute × CallbackAdmission × CallbackDispatch)) := do
  let config ← callbackConsumerConfig true 2 10
  let belowLimit := runCallbackTrace config (callbackRequest 1 0 "ftp://invalid" 20)
  let aboveLimit := runCallbackTrace config (callbackRequest 3 0 "ftp://invalid" 20)
  pure [
    (belowLimit.route, belowLimit.admission, belowLimit.dispatch),
    (aboveLimit.route, aboveLimit.admission, aboveLimit.dispatch)]

def zeroCallbackRequestsMatch : Bool :=
  match zeroCallbackRequests with
  | .ok [(none, .notRequested, .notDispatched),
         (none, .notRequested, .notDispatched)] => true
  | _ => false

example : zeroCallbackRequestsMatch = true := by native_decide

def callbackAddressAdmissions : Except ConfigError (List CallbackAdmission) := do
  let config ← callbackConsumerConfig true 10 10
  let request := callbackRequest 0 1 "" 1
  pure [
    (runCallbackTrace config { request with address := "temporal://system" }).admission,
    (runCallbackTrace config { request with address := "temporal://internal" }).admission,
    (runCallbackTrace config { request with address := "temporal://system/path" }).admission,
    (runCallbackTrace config { request with address := "temporal://internal?query" }).admission,
    (runCallbackTrace config { request with address := "temporal://system#fragment" }).admission,
    (runCallbackTrace config { request with address := "https://api.a.b.example.com/path" }).admission,
    (runCallbackTrace config { request with address := "http://api.a.example.com" }).admission,
    (runCallbackTrace config { request with address := "http://hooks.insecure.test:80/path" }).admission,
    (runCallbackTrace config { request with address := "http://hooks.insecure.test:81/path" }).admission,
    (runCallbackTrace config { request with address := "http://hooks.insecure.test.evil:80/path" }).admission,
    (runCallbackTrace config { request with address := "https://other.example.com" }).admission,
    (runCallbackTrace config { request with address := "ftp://api.a.example.com" }).admission,
    (runCallbackTrace config { request with address := "https:///missing-host" }).admission]

def callbackAddressAdmissionsMatch : Bool :=
  match callbackAddressAdmissions with
  | .ok [.admitted,
         .admitted,
         .rejectedAddress .unknownScheme,
         .rejectedAddress .unknownScheme,
         .rejectedAddress .unknownScheme,
         .admitted,
         .rejectedAddress .insecureConnection,
         .admitted,
         .rejectedAddress .unmatchedAddress,
         .rejectedAddress .unmatchedAddress,
         .rejectedAddress .unmatchedAddress,
         .rejectedAddress .unknownScheme,
         .rejectedAddress .missingHost] => true
  | _ => false

example : callbackAddressAdmissionsMatch = true := by native_decide

def callbackTimeoutBoundaries : Except ConfigError (List CallbackDispatch) := do
  let positive ← callbackConsumerConfig true 10 10
  let zero ← callbackConsumerConfig true 10 0
  let negative ← callbackConsumerConfig true 10 (-1)
  let request := callbackRequest 0 1 "https://api.a.example.com" 0
  pure [
    (runCallbackTrace positive { request with elapsedNanoseconds := 9 }).dispatch,
    (runCallbackTrace positive { request with elapsedNanoseconds := 10 }).dispatch,
    (runCallbackTrace positive { request with elapsedNanoseconds := 11 }).dispatch,
    (runCallbackTrace zero { request with elapsedNanoseconds := -1 }).dispatch,
    (runCallbackTrace negative { request with elapsedNanoseconds := -2 }).dispatch]

def callbackTimeoutBoundariesMatch : Bool :=
  match callbackTimeoutBoundaries with
  | .ok [.succeeded, .timedOut, .timedOut, .timedOut, .timedOut] => true
  | _ => false

example : callbackTimeoutBoundariesMatch = true := by native_decide

def callbackSnapshotPairs : Except ConfigError
    ((Option CallbackRoute × CallbackAdmission × CallbackDispatch) ×
     (Option CallbackRoute × CallbackAdmission × CallbackDispatch) ×
     (CallbackDispatch × CallbackDispatch) × Bool) := do
  let chasm ← callbackConsumerConfig true 1 5
  let legacy ← callbackConsumerConfig false 2 10
  let boundaryRequest := callbackRequest 1 1 "https://api.a.example.com" 4
  let dispatchRequest := callbackRequest 0 1 "https://api.a.example.com" 5
  let legacyBoundary := runCallbackTrace legacy boundaryRequest
  let chasmBoundary := runCallbackTrace chasm boundaryRequest
  let legacyDispatch := runCallbackTrace legacy dispatchRequest
  let chasmDispatch := runCallbackTrace chasm dispatchRequest
  let chasmAfterDisableSnapshot := runCallbackTrace chasm boundaryRequest
  pure ((legacyBoundary.route, legacyBoundary.admission, legacyBoundary.dispatch),
    (chasmBoundary.route, chasmBoundary.admission, chasmBoundary.dispatch),
    (legacyDispatch.dispatch, chasmDispatch.dispatch),
    chasmAfterDisableSnapshot == chasmBoundary)

def callbackSnapshotPairsMatch : Bool :=
  match callbackSnapshotPairs with
  | .ok ((some .legacyHsm, .admitted, .succeeded),
      (some .chasm, .rejectedOverflow, .notDispatched),
      (.succeeded, .timedOut), true) => true
  | _ => false

example : callbackSnapshotPairsMatch = true := by native_decide

def missingDestinationCallbackConfig : Except ConfigError CallbackDomainConfig := do
  let view ← callbackConsumerView true 10 10
  projectCallbackDomainConfig view "payments" ""

example : errorKindOf missingDestinationCallbackConfig = some .missingContext := by
  native_decide

def malformedCallbackConfig : Except ConfigError CallbackDomainConfig := do
  let view ← malformedUnselectedAddressOverrideResult
  projectCallbackDomainConfig view "payments" "callback-api"

example : errorKindOf malformedCallbackConfig = some .interpretationFailure := by
  native_decide

end Temporal.System.Callback.ConfigurationTests
