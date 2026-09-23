import Lean.Data.Json

/-!
# What every Lean bridge shares at its frame boundary

A bridge is one Lean process a Go coordinator drives one frame at a time: a frame in, a frame out,
each one line of canonical JSON naming its kind, its sequence number and the set it is about. The
exploration bridge and the replay bridge answer different questions, and both read and write the
same way: a line that is not a JSON object is no frame and ends the process with a diagnostic; an
object that is not a well-formed frame, or carries a key its kind does not admit, is a frame to
reject under the sequence number it carries; a frame whose sequence number is not the next one is
rejected as a duplicate or out of order before anything else reads it; a frame naming another
Profile than the one the session opened under is crossed; a line longer than the bridge admits is
rejected unread. What each bridge does with an accepted frame is its own.
-/

namespace Temporal.Tool.Bridge

/-- The effects a bridge runs under, injected so a test measures what reached each stream. -/
structure Effects where
  readLine : IO (Option String)
  writeFrame : String → IO Unit
  writeProgress : String → IO Unit
  writeError : String → IO Unit

/-- The effects of the process boundary: frames on stdout, flushed one at a time because the
coordinator waits on each; progress and diagnostics on stderr. -/
def processEffects : IO Effects := do
  let stdin ← IO.getStdin
  let stdout ← IO.getStdout
  let stderr ← IO.getStderr
  pure {
    readLine := do
      let line ← stdin.getLine
      pure (if line.isEmpty then none else some line)
    writeFrame := fun line => do stdout.putStr (line ++ "\n"); stdout.flush
    writeProgress := fun line => do stderr.putStr (line ++ "\n"); stderr.flush
    writeError := fun text => do stderr.putStr text; stderr.flush }

/-! ### Rendering

Frames are rendered by hand, member by member, so their key order is the order written here and an
embedded Case keeps its bytes exactly. -/

def jsonString (value : String) : String := Lean.Json.compress (.str value)

def jsonObject (members : List (String × String)) : String :=
  "{" ++ ",".intercalate (members.map fun (name, rendered) => jsonString name ++ ":" ++ rendered) ++ "}"

def jsonArray (items : List String) : String :=
  "[" ++ ",".intercalate items ++ "]"

def jsonStrings (items : List String) : String := jsonArray (items.map jsonString)

/-- The members every frame a bridge writes opens with: its kind, the sequence number it answers,
the set and the Profile identity the session runs under, echoed. -/
def header (kind : String) (seq : Nat) (setName profile : String) : List (String × String) :=
  [("frame", jsonString kind), ("seq", toString seq), ("set", jsonString setName),
    ("profile", jsonString profile)]

def renderRejected (seq : Nat) (reason : String) : String :=
  jsonObject [("frame", jsonString "rejected"), ("seq", toString seq), ("reason", jsonString reason)]

/-! ### Reading -/

/-- One frame as far as every bridge reads it: its kind, its sequence number, the set it names and
the whole object, whose keys are exactly ones its kind admits. -/
structure Envelope where
  kind : String
  seq : Nat
  setName : String
  json : Lean.Json

/-- Read one line as a frame envelope. A line that is not a JSON object is no frame at all; an
object that is not a well-formed frame is a frame to reject, under the sequence number it carries
where it carries one, so the coordinator can match the rejection to what it sent. The keys each
kind admits are closed: a key the bridge does not read -- one that would name a target, a
coordinate or a Case family -- rejects rather than being dropped. -/
def parseEnvelope (admittedKeys : String → List String) (line : String) :
    Except String (Except (Nat × String) Envelope) := do
  let json ← match Lean.Json.parse line with
    | .ok json => pure json
    | .error reason => throw s!"not JSON: {reason}"
  let .obj members := json | throw "not a JSON object"
  let seqOf : Nat := (json.getObjValAs? Nat "seq").toOption.getD 0
  let envelope : Except String Envelope := do
    let kind ← (json.getObjValAs? String "frame").mapError fun _ => "frame names no `frame`"
    let seq ← (json.getObjValAs? Nat "seq").mapError fun _ => "frame names no `seq`"
    let setName ← (json.getObjValAs? String "set").mapError fun _ => "frame names no `set`"
    let admitted := admittedKeys kind
    if admitted.isEmpty then throw s!"unknown frame `{kind}`"
    for (key, _) in members.toList do
      unless admitted.contains key do
        throw s!"`{kind}` admits no `{key}`; its keys are {admitted}"
    pure { kind, seq, setName, json }
  pure (envelope.mapError fun reason => (seqOf, reason))

/-- A string member the frame's kind requires. -/
def Envelope.string (envelope : Envelope) (name : String) : Except String String :=
  (envelope.json.getObjValAs? String name).mapError fun _ => s!"`{envelope.kind}` names no `{name}`"

/-- A string member the frame's kind admits and may leave out. -/
def Envelope.string? (envelope : Envelope) (name : String) : Except String (Option String) :=
  match envelope.json.getObjVal? name with
  | .error _ => pure none
  | .ok (.str value) => pure (some value)
  | .ok _ => throw s!"`{envelope.kind}` `{name}` is not a string"

/-- Why a frame's sequence number is not the one the session expects next, or none. -/
def sequenceRejection (expected seq : Nat) : Option String :=
  if seq == expected then none
  else if seq + 1 == expected then some s!"duplicate frame: seq {seq} was already accepted"
  else some s!"out-of-order frame: expected seq {expected}, got {seq}"

/-- Why a frame naming `named` is not the session's, which runs under `expected`, or none. -/
def profileRejection (expected named : String) : Option String :=
  if named == expected then none
  else some s!"crossed profile: the session runs under {expected}, not {named}"

/-! ### Serving -/

/-- What one accepted frame produces: the frame to write and the state after, or the last frame. -/
inductive Outcome (State : Type) where
  | reply (line : String) (state : State)
  | finished (line : String)

/-- How one bridge reads, checks and applies its frames. `reject` is decided before any campaign
call and leaves the state as it was. -/
structure Protocol (State Frame : Type) where
  diagnosticPrefix : String
  /-- The longest line the bridge reads, in bytes; a longer one is rejected unread. -/
  maxLineBytes : Option Nat := none
  parse : String → Except String (Except (Nat × String) Frame)
  seqOf : Frame → Nat
  reject : State → Frame → Option String
  step : State → Frame → IO (Outcome State)

/-- Serve frames until the protocol finishes. A line that is no frame, or stdin closing before the
last frame, is a failure outside the protocol: a diagnostic on stderr and a non-zero exit. -/
partial def serve {State Frame : Type} (effects : Effects) (protocol : Protocol State Frame)
    (initial : State) : IO UInt32 := do
  let rec loop (state : State) : IO UInt32 := do
    match ← effects.readLine with
    | none =>
        effects.writeError s!"{protocol.diagnosticPrefix} stdin closed before `finish`\n"
        pure 1
    | some line =>
        if line.all Char.isWhitespace then loop state
        else if protocol.maxLineBytes.any (line.utf8ByteSize > ·) then
          effects.writeFrame (renderRejected 0
            s!"oversized frame: {line.utf8ByteSize} bytes exceeds {protocol.maxLineBytes.getD 0}")
          loop state
        else match protocol.parse line with
          | .error reason =>
              effects.writeError s!"{protocol.diagnosticPrefix} line is not a frame: {reason}\n"
              pure 1
          | .ok (.error (seq, reason)) =>
              effects.writeFrame (renderRejected seq reason)
              loop state
          | .ok (.ok frame) =>
              match protocol.reject state frame with
              | some reason =>
                  effects.writeFrame (renderRejected (protocol.seqOf frame) reason)
                  loop state
              | none =>
                  match ← protocol.step state frame with
                  | .reply line next =>
                      effects.writeFrame line
                      loop next
                  | .finished line =>
                      effects.writeFrame line
                      pure 0
  loop initial

end Temporal.Tool.Bridge
