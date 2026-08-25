import Temporal.System.Matching.Configuration

namespace Temporal.Umpire.Config

open _root_.Umpire
open Temporal.DynamicConfig
open Temporal.System.Configuration

/-! Callback configuration remains staged here until its dedicated System extraction. -/

private def stringLe (left right : String) : Bool := decide (left ≤ right)

private def firstDuplicateString : List String → Option String
  | first :: second :: rest =>
      if first == second then some first else firstDuplicateString (second :: rest)
  | _ => none

private def decodeBool : CanonicalValue → Except String Bool
  | .bool value => pure value
  | value => throw ("expected bool, found " ++ reprStr value)

private def decodeInt : CanonicalValue → Except String Int
  | .int value => pure value
  | value => throw ("expected int, found " ++ reprStr value)

private def decodeDuration : CanonicalValue → Except String Int
  | .duration nanoseconds => pure nanoseconds
  | value => throw ("expected duration, found " ++ reprStr value)

structure CallbackAddressRule where
  pattern : String
  allowInsecure : Bool
  deriving BEq, DecidableEq, Repr

structure CallbackAddressRules where
  rules : List CallbackAddressRule
  deriving BEq, DecidableEq, Repr

private def canonicalValuesToList : CanonicalValues → List CanonicalValue
  | .nil => []
  | .cons value tail => value :: canonicalValuesToList tail

private def canonicalField? (name : String) : CanonicalFields → Option CanonicalValue
  | .nil => none
  | .cons fieldName value tail =>
      if fieldName == name then some value else canonicalField? name tail

private def canonicalFieldsToList : CanonicalFields → List (String × CanonicalValue)
  | .nil => []
  | .cons name value tail => (name, value) :: canonicalFieldsToList tail

private def requireCanonicalFields
    (owner : String)
    (allowed required : List String)
    (fields : CanonicalFields) : Except String Unit := do
  let names := (canonicalFieldsToList fields).map Prod.fst
  let sortedNames := names.mergeSort stringLe
  match firstDuplicateString sortedNames with
  | some duplicate => throw (owner ++ " contains duplicate field " ++ duplicate)
  | none => pure ()
  for name in names do
    if !allowed.contains name then throw (owner ++ " contains unknown field " ++ name)
  for name in required do
    if !names.contains name then throw (owner ++ " requires field " ++ name)

private def decodeAddressRule (value : CanonicalValue) : Except String CallbackAddressRule := do
  let fields ← match value with
    | .object fields => pure fields
    | _ => throw ("address rule must be an object: " ++ reprStr value)
  requireCanonicalFields "address rule" ["Pattern", "AllowInsecure"] ["Pattern"] fields
  let pattern ← match canonicalField? "Pattern" fields with
    | some (.string pattern) => pure pattern
    | _ => throw "address rule requires a string Pattern"
  if pattern == "" then throw "address rule Pattern must be non-empty"
  let allowInsecure ← match canonicalField? "AllowInsecure" fields with
    | none => pure false
    | some (.bool value) => pure value
    | _ => throw "address rule AllowInsecure must be a bool"
  pure { pattern, allowInsecure }

def decodeCallbackAddressRules : CanonicalValue → Except String CallbackAddressRules
  | .object fields => do
      requireCanonicalFields "callback address rules" ["Rules"] ["Rules"] fields
      match canonicalField? "Rules" fields with
      | some .null => pure { rules := [] }
      | some (.list values) =>
          pure { rules := ← canonicalValuesToList values |>.mapM decodeAddressRule }
      | _ => throw "callback address rules require Rules as a list or null"
  | value => throw ("callback address rules must be an object: " ++ reprStr value)

inductive CallbackAddressErrorKind where
  | unknownScheme
  | missingHost
  | malformedAddress
  | unmatchedAddress
  | insecureConnection
  deriving BEq, DecidableEq, Ord, Repr

structure CallbackAddressError where
  kind : CallbackAddressErrorKind
  address : String
  matchedPattern : Option String
  deriving BEq, DecidableEq, Repr

private def wildcardMatchFuel : Nat → List Char → List Char → Bool
  | 0, _, _ => false
  | _ + 1, [], host => host == []
  | fuel + 1, '*' :: pattern, host =>
      wildcardMatchFuel fuel pattern host ||
        match host with
        | [] => false
        | _ :: rest => wildcardMatchFuel fuel ('*' :: pattern) rest
  | fuel + 1, expected :: pattern, actual :: host =>
      expected == actual && wildcardMatchFuel fuel pattern host
  | _ + 1, _ :: _, [] => false

def wholeHostWildcardMatch (pattern host : String) : Bool :=
  wildcardMatchFuel (pattern.length + host.length + 1) pattern.toList host.toList

private def isHexCharacter (character : Char) : Bool :=
  "0123456789abcdefABCDEF".toList.contains character

private def validPercentEscapes : List Char → Bool
  | [] => true
  | '%' :: first :: second :: rest =>
      isHexCharacter first && isHexCharacter second && validPercentEscapes rest
  | '%' :: _ => false
  | _ :: rest => validPercentEscapes rest

private def isForbiddenAuthorityCharacter (character : Char) : Bool :=
  [' ', '\t', '\n', '\r', '\\'].contains character

private def lastString : List String → String
  | [] => ""
  | [value] => value
  | _ :: rest => lastString rest

private def validPort (characters : List Char) : Bool :=
  characters.all fun character => "0123456789".toList.contains character

private def validIPv4Octet (octet : String) : Bool :=
  octet != "" && (octet.length == 1 || octet.toList.head? != some '0') &&
    match octet.toNat? with
    | some value => decide (value ≤ 255)
    | none => false

private def validIPv4Address (address : String) : Bool :=
  match address.splitOn "." with
  | [first, second, third, fourth] =>
      [first, second, third, fourth].all validIPv4Octet
  | _ => false

private def validIPv6Group (group : String) : Bool :=
  group.length > 0 && group.length ≤ 4 && group.toList.all isHexCharacter

private def ipv6Units : List String → Option Nat
  | [] => some 0
  | [last] =>
      if last.toList.contains '.' then
        if validIPv4Address last then some 2 else none
      else if validIPv6Group last then some 1 else none
  | group :: rest =>
      if validIPv6Group group then
        (ipv6Units rest).map Nat.succ
      else
        none

private def ipv6Side (side : String) : Option Nat :=
  if side == "" then some 0 else ipv6Units (side.splitOn ":")

private def validIPv6Zone (zone : String) : Bool :=
  zone != "" && zone.toList.all fun character =>
    character.isAlphanum || ['.', '_', '-'].contains character

private def validIPv6Address (literal : String) : Bool :=
  let address? := match literal.splitOn "%25" with
    | [address] => some address
    | [address, zone] => if validIPv6Zone zone then some address else none
    | _ => none
  match address? with
  | none => false
  | some address =>
      if address.toList.contains '%' then false else
      match address.splitOn "::" with
      | [whole] => ipv6Side whole == some 8
      | [left, right] =>
          match ipv6Side left, ipv6Side right with
          | some leftUnits, some rightUnits => leftUnits + rightUnits < 8
          | _, _ => false
      | _ => false

private def validBracketHostPort (hostPort : String) : Bool :=
  match hostPort.toList with
  | '[' :: rest =>
      match (String.ofList rest).splitOn "]" with
      | [inside, suffix] =>
          validIPv6Address inside &&
            match suffix.toList with
            | [] => true
            | ':' :: port => validPort port
            | _ => false
      | _ => false
  | _ => false

private def validHostPort (hostPort : String) : Bool :=
  if hostPort == "" || hostPort.toList.any isForbiddenAuthorityCharacter then
    false
  else if hostPort.toList.head? == some '[' then
    validBracketHostPort hostPort
  else if hostPort.toList.any fun character => character == '[' || character == ']' || character == '%' then
    false
  else
    match hostPort.splitOn ":" with
    | [host] => host != ""
    | [host, port] => host != "" && validPort port.toList
    | _ => false

private def urlHost? (rawAddress : String) : Except CallbackAddressError (String × String) := do
  match rawAddress.splitOn "://" with
  | [scheme, remainder] =>
      if scheme != "http" && scheme != "https" then
        throw { kind := .unknownScheme, address := rawAddress, matchedPattern := none }
      if !validPercentEscapes rawAddress.toList then
        throw { kind := .malformedAddress, address := rawAddress, matchedPattern := none }
      let authority := String.ofList (remainder.toList.takeWhile fun character =>
        character != '/' && character != '?' && character != '#')
      let hostPort := lastString (authority.splitOn "@")
      if hostPort == "" then
        throw { kind := .missingHost, address := rawAddress, matchedPattern := none }
      if !validHostPort hostPort then
        throw { kind := .malformedAddress, address := rawAddress, matchedPattern := none }
      pure (scheme, hostPort)
  | _ => throw { kind := .unknownScheme, address := rawAddress, matchedPattern := none }

def CallbackAddressRules.validate
    (rules : CallbackAddressRules)
    (rawAddress : String) : Except CallbackAddressError Unit := do
  if rawAddress == "temporal://system" || rawAddress == "temporal://internal" then
    pure ()
  else
    let (scheme, host) ← urlHost? rawAddress
    let rec validateRules : List CallbackAddressRule → Except CallbackAddressError Unit
      | [] => throw { kind := .unmatchedAddress, address := rawAddress, matchedPattern := none }
      | rule :: rest =>
          if wholeHostWildcardMatch rule.pattern host then
            if scheme == "https" || rule.allowInsecure then
              pure ()
            else
              throw {
                kind := .insecureConnection
                address := rawAddress
                matchedPattern := some rule.pattern
              }
          else
            validateRules rest
    validateRules rules.rules

def authoredClassifications : List SettingClassification :=
  [{ key := "history.enablechasmcallbacks"
     settingIdentity := "sha256:415f169bb77c82582f2d8f5049648b5b079f4f1047a2f109d4ed9b14037d9c8c"
     impacts := [.feature, .externallyVisibleSemantics] },
   { key := "callback.maxperexecution"
     settingIdentity := "sha256:6c7f3b78bbbf74a83401b46faedf61250a1c4c2c92d02eab91ec9ebc36b30d71"
     impacts := [.validation] },
   { key := "callback.request.timeout"
     settingIdentity := "sha256:cd2c7d65a4f41e7edcfa548d7433aeb7cd5a414c6a3258d361676cd3ada8fda9"
     impacts := [.timing] },
   { key := "callback.allowedaddresses"
     settingIdentity := "sha256:452cd642fac8adb5d5e1e2c0a4ef1d149cfb621ed663842c1bde7dd123faca9b"
     impacts := [.validation, .externallyVisibleSemantics] }]

def historyEnableChasmCallbacksInterpretation : ConfigInterpretation Bool := {
  key := "history.enablechasmcallbacks"
  expectedSettingIdentity := "sha256:415f169bb77c82582f2d8f5049648b5b079f4f1047a2f109d4ed9b14037d9c8c"
  expectedSchema := .bool "bool" false
  expectedDefault := .concrete (.bool true)
  semanticDigest := semanticDigestOf "temporal.config/history-enable-chasm-callbacks/v1"
  decode := decodeBool
}

def callbackMaxPerExecutionInterpretation : ConfigInterpretation Int := {
  key := "callback.maxperexecution"
  expectedSettingIdentity := "sha256:6c7f3b78bbbf74a83401b46faedf61250a1c4c2c92d02eab91ec9ebc36b30d71"
  expectedSchema := .int "int" false
  expectedDefault := .concrete (.int 2000)
  semanticDigest := semanticDigestOf "temporal.config/callback-max-per-execution/v1"
  decode := decodeInt
}

def callbackRequestTimeoutInterpretation : ConfigInterpretation Int := {
  key := "callback.request.timeout"
  expectedSettingIdentity := "sha256:cd2c7d65a4f41e7edcfa548d7433aeb7cd5a414c6a3258d361676cd3ada8fda9"
  expectedSchema := .duration "time.Duration" false
  expectedDefault := .concrete (.duration 10000000000)
  semanticDigest := semanticDigestOf "temporal.config/callback-request-timeout/v1"
  decode := decodeDuration
}

def callbackAllowedAddressesInterpretation : ConfigInterpretation CallbackAddressRules := {
  key := "callback.allowedaddresses"
  expectedSettingIdentity := "sha256:452cd642fac8adb5d5e1e2c0a4ef1d149cfb621ed663842c1bde7dd123faca9b"
  expectedSchema := Temporal.DynamicConfig.Settings.callback_allowedaddresses.schema
  expectedDefault := .concrete (.object (.cons "Rules" .null .nil))
  semanticDigest := semanticDigestOf "temporal.config/callback-allowed-addresses/v1"
  decode := decodeCallbackAddressRules
}

def namespaceContext (namespaceName : String) : ExactConstraints :=
  { emptyConstraints with namespaceName := some namespaceName }

def destinationContext (namespaceName destination : String) : ExactConstraints :=
  { emptyConstraints with namespaceName := some namespaceName, destination := some destination }

private def checkedAuthoredUse
    (request : ConfigUseRequest α) : Except ConfigError (ConfigUse α) :=
  checkConfigUse authoredClassifications request

def historyEnableChasmCallbacksUse (namespaceName : String) : Except ConfigError (ConfigUse Bool) :=
  checkedAuthoredUse {
    id := DeclarationId.of "temporal.callback.enable-chasm"
    key := historyEnableChasmCallbacksInterpretation.key
    context := namespaceContext namespaceName
    samplingPoint := .entityCreation
    changeEffect := .newEntitiesOnly
    interpretation := some historyEnableChasmCallbacksInterpretation
  }

def callbackMaxPerExecutionUse (namespaceName : String) : Except ConfigError (ConfigUse Int) :=
  checkedAuthoredUse {
    id := DeclarationId.of "temporal.callback.max-per-execution"
    key := callbackMaxPerExecutionInterpretation.key
    context := namespaceContext namespaceName
    samplingPoint := .request
    changeEffect := .nextRead
    interpretation := some callbackMaxPerExecutionInterpretation
  }

def callbackAllowedAddressesUse
    (namespaceName : String) : Except ConfigError (ConfigUse CallbackAddressRules) :=
  checkedAuthoredUse {
    id := DeclarationId.of "temporal.callback.allowed-addresses"
    key := callbackAllowedAddressesInterpretation.key
    context := namespaceContext namespaceName
    samplingPoint := .request
    changeEffect := .nextRead
    interpretation := some callbackAllowedAddressesInterpretation
  }

def callbackRequestTimeoutUse
    (namespaceName destination : String) : Except ConfigError (ConfigUse Int) :=
  checkedAuthoredUse {
    id := DeclarationId.of "temporal.callback.request-timeout"
    key := callbackRequestTimeoutInterpretation.key
    context := destinationContext namespaceName destination
    samplingPoint := .task
    changeEffect := .nextRead
    interpretation := some callbackRequestTimeoutInterpretation
  }
inductive CallbackRoute where
  | legacyHsm
  | chasm
  deriving BEq, DecidableEq, Repr

inductive CallbackAdmission where
  | notRequested
  | admitted
  | rejectedOverflow
  | rejectedAddress (kind : CallbackAddressErrorKind)
  deriving BEq, DecidableEq, Repr

inductive CallbackDispatch where
  | notDispatched
  | succeeded
  | timedOut
  deriving BEq, DecidableEq, Repr

structure CallbackRequest where
  existingCallbacks : Nat
  newCallbacks : Nat
  address : String
  elapsedNanoseconds : Int
  deriving BEq, DecidableEq, Repr

private structure CallbackDomainConfigPayload where
  route : CallbackRoute
  maximumCallbacks : Int
  addressRules : CallbackAddressRules
  timeoutNanoseconds : Int
  deriving BEq, DecidableEq, Repr

/-- The four callback settings projected once from one validated immutable view. -/
structure CallbackDomainConfig where
  private mk ::
  private payload : CallbackDomainConfigPayload
  deriving BEq, DecidableEq, Repr

structure CallbackTrace where
  route : Option CallbackRoute
  admission : CallbackAdmission
  dispatch : CallbackDispatch
  deriving BEq, DecidableEq, Repr

def projectCallbackDomainConfig
    (view : ConfigView)
    (namespaceName destination : String) : Except ConfigError CallbackDomainConfig := do
  if destination == "" then
    throw {
      kind := .missingContext
      useId := DeclarationId.of "temporal.callback.snapshot"
      key := callbackRequestTimeoutInterpretation.key
      offendingValue := "destination"
      relatedIdentities := []
    }
  let enableUse ← historyEnableChasmCallbacksUse namespaceName
  let maximumUse ← callbackMaxPerExecutionUse namespaceName
  let addressesUse ← callbackAllowedAddressesUse namespaceName
  let timeoutUse ← callbackRequestTimeoutUse namespaceName destination
  let enabled ← view.read enableUse
  let maximumCallbacks ← view.read maximumUse
  let addressRules ← view.read addressesUse
  let timeoutNanoseconds ← view.read timeoutUse
  pure (.mk {
    route := if enabled then .chasm else .legacyHsm
    maximumCallbacks
    addressRules
    timeoutNanoseconds
  })

/-- Evaluate callback admission and dispatch against only the captured callback projection. -/
def runCallbackTrace
    (config : CallbackDomainConfig)
    (request : CallbackRequest) : CallbackTrace :=
  if request.newCallbacks == 0 then
    { route := none, admission := .notRequested, dispatch := .notDispatched }
  else
    let route := some config.payload.route
    match config.payload.addressRules.validate request.address with
    | .error error => {
        route
        admission := .rejectedAddress error.kind
        dispatch := .notDispatched
      }
    | .ok _ =>
        if Int.ofNat (request.existingCallbacks + request.newCallbacks) >
            config.payload.maximumCallbacks then
          { route, admission := .rejectedOverflow, dispatch := .notDispatched }
        else
          let dispatch :=
            if config.payload.timeoutNanoseconds <= 0 ||
                request.elapsedNanoseconds >= config.payload.timeoutNanoseconds then
              .timedOut
            else
              .succeeded
          { route, admission := .admitted, dispatch }

end Temporal.Umpire.Config
