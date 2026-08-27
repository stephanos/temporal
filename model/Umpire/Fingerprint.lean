import Std

namespace Umpire

/-! Pure, typed SHA-256 derivation for model behavior and exact artifacts. -/

namespace Fingerprint

private def rotateRight (value : UInt32) (amount : Nat) : UInt32 :=
  (value >>> UInt32.ofNat amount) ||| (value <<< UInt32.ofNat (32 - amount))

private def smallSigma0 (value : UInt32) : UInt32 :=
  rotateRight value 7 ^^^ rotateRight value 18 ^^^ (value >>> (3 : UInt32))

private def smallSigma1 (value : UInt32) : UInt32 :=
  rotateRight value 17 ^^^ rotateRight value 19 ^^^ (value >>> (10 : UInt32))

private def bigSigma0 (value : UInt32) : UInt32 :=
  rotateRight value 2 ^^^ rotateRight value 13 ^^^ rotateRight value 22

private def bigSigma1 (value : UInt32) : UInt32 :=
  rotateRight value 6 ^^^ rotateRight value 11 ^^^ rotateRight value 25

private def choose (x y z : UInt32) : UInt32 :=
  (x &&& y) ^^^ ((~~~x) &&& z)

private def majority (x y z : UInt32) : UInt32 :=
  (x &&& y) ^^^ (x &&& z) ^^^ (y &&& z)

private def roundConstants : Array UInt32 := #[
  0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5,
  0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
  0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3,
  0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
  0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc,
  0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
  0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7,
  0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
  0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13,
  0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
  0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3,
  0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
  0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5,
  0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
  0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208,
  0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2
]

private structure State where
  a : UInt32
  b : UInt32
  c : UInt32
  d : UInt32
  e : UInt32
  f : UInt32
  g : UInt32
  h : UInt32

private def initialState : State := {
  a := 0x6a09e667
  b := 0xbb67ae85
  c := 0x3c6ef372
  d := 0xa54ff53a
  e := 0x510e527f
  f := 0x9b05688c
  g := 0x1f83d9ab
  h := 0x5be0cd19
}

private def pad (bytes : ByteArray) : Array UInt8 :=
  let bitLength := bytes.size * 8
  let withMarker := bytes.data.push 0x80
  let zeroCount := (64 + 56 - (withMarker.size % 64)) % 64
  let withZeros := (List.range zeroCount).foldl (fun result _ => result.push 0) withMarker
  (List.range 8).foldl (fun result index =>
    let shift := (7 - index) * 8
    result.push (UInt8.ofNat ((bitLength / (2 ^ shift)) % 256))) withZeros

private def wordAt (bytes : Array UInt8) (offset : Nat) : UInt32 :=
  (bytes.getD offset 0).toUInt32 <<< (24 : UInt32) |||
  (bytes.getD (offset + 1) 0).toUInt32 <<< (16 : UInt32) |||
  (bytes.getD (offset + 2) 0).toUInt32 <<< (8 : UInt32) |||
  (bytes.getD (offset + 3) 0).toUInt32

private def schedule (bytes : Array UInt8) (blockStart : Nat) : Array UInt32 :=
  let first := (List.range 16).foldl (fun words index =>
    words.push (wordAt bytes (blockStart + index * 4))) #[]
  (List.range 48).foldl (fun words offset =>
    let index := offset + 16
    words.push (
      smallSigma1 (words.getD (index - 2) 0) +
      words.getD (index - 7) 0 +
      smallSigma0 (words.getD (index - 15) 0) +
      words.getD (index - 16) 0)) first

private def round (state : State) (constant word : UInt32) : State :=
  let first := state.h + bigSigma1 state.e + choose state.e state.f state.g + constant + word
  let second := bigSigma0 state.a + majority state.a state.b state.c
  {
    a := first + second
    b := state.a
    c := state.b
    d := state.c
    e := state.d + first
    f := state.e
    g := state.f
    h := state.g
  }

private def compress (state : State) (bytes : Array UInt8) (blockStart : Nat) : State :=
  let words := schedule bytes blockStart
  let working := (List.range 64).foldl (fun current index =>
    round current (roundConstants.getD index 0) (words.getD index 0)) state
  {
    a := state.a + working.a
    b := state.b + working.b
    c := state.c + working.c
    d := state.d + working.d
    e := state.e + working.e
    f := state.f + working.f
    g := state.g + working.g
    h := state.h + working.h
  }

private def digest (value : String) : State :=
  let bytes := pad value.toUTF8
  (List.range (bytes.size / 64)).foldl (fun state block =>
    compress state bytes (block * 64)) initialState

private def hexDigit (value : Nat) : Char :=
  "0123456789abcdef".toList.getD value '0'

private def byteHex (value : UInt8) : String :=
  String.ofList [hexDigit (value.toNat / 16), hexDigit (value.toNat % 16)]

private def wordHex (value : UInt32) : String :=
  [24, 16, 8, 0].foldl (fun result shift =>
    result ++ byteHex ((value >>> UInt32.ofNat shift).toUInt8)) ""

/-- Return the lowercase hexadecimal SHA-256 digest of a UTF-8 string. -/
def sha256Hex (value : String) : String :=
  let state := digest value
  [state.a, state.b, state.c, state.d, state.e, state.f, state.g, state.h]
    |>.foldl (fun result word => result ++ wordHex word) ""

end Fingerprint

structure BehaviorFingerprint where
  private mk ::
  private value : String
  deriving BEq, DecidableEq, Repr

structure ArtifactChecksum where
  private mk ::
  private value : String
  deriving BEq, DecidableEq, Repr

private def validRendering (value : String) : Bool :=
  value.length == 71 &&
    value.startsWith "sha256:" &&
    (value.toList.drop 7).all fun character =>
      "0123456789abcdef".toList.contains character

def BehaviorFingerprint.render (fingerprint : BehaviorFingerprint) : String :=
  fingerprint.value

def BehaviorFingerprint.parse? (value : String) : Option BehaviorFingerprint :=
  if validRendering value then some ⟨value⟩ else none

instance : ToString BehaviorFingerprint := ⟨BehaviorFingerprint.render⟩

def ArtifactChecksum.render (checksum : ArtifactChecksum) : String :=
  checksum.value

def ArtifactChecksum.parse? (value : String) : Option ArtifactChecksum :=
  if validRendering value then some ⟨value⟩ else none

instance : ToString ArtifactChecksum := ⟨ArtifactChecksum.render⟩

private def derive (domain canonicalContent : String) : String :=
  "sha256:" ++ Fingerprint.sha256Hex (domain ++ "\n" ++ canonicalContent)

/-- Fingerprint already-canonical behavior-relevant content. -/
def behaviorFingerprintOf (canonicalContent : String) : BehaviorFingerprint :=
  ⟨derive "umpire.behavior-fingerprint/v1" canonicalContent⟩

/-- Checksum an already-canonical DrivePlan object without its checksum field. -/
def drivePlanChecksumOf (canonicalContent : String) : ArtifactChecksum :=
  ⟨derive "umpire.drive-plan/v2" canonicalContent⟩

/-- Checksum an already-canonical ExperimentSpec object without its checksum field. -/
def experimentSpecChecksumOf (canonicalContent : String) : ArtifactChecksum :=
  ⟨derive "umpire.experiment-spec/v2" canonicalContent⟩

end Umpire
