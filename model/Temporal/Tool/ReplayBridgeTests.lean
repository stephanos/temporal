import Temporal.Tool.ReplayBridge

/-!
# What the replay bridge does at its frame boundary

Every subject here is one the bridge's own binding table recovers, and every functional subject's
identity is taken from its checked-in fixture's bytes, compacted, so admission compares against
what the conformance generator wrote rather than against a second production. The negative
control is irreducible at once: its one prefix step is the schedule, which the Model cannot drop.
The caller's `scheduleToStartTimeout` has one admitted edit, dropping the worker stop, and one
inapplicable one, dropping the schedule; its candidate is driven through each class. An exploration
candidate is a shortest path with nothing to drop, so it is irreducible at once. Every rejection is pinned against the frame it answers, the same frames twice give the same
frames, and the built executable is run once so what reaches each stream is measured.
-/

namespace Temporal.Tool.ReplayBridgeTests

open Temporal.Tool.ReplayBridge

private def fail (reason : String) : IO α :=
  throw <| IO.userError reason

private def require (condition : Bool) (reason : String) : IO Unit :=
  unless condition do fail reason

/-! ### Running the bridge in-process -/

structure Captured where
  status : UInt32
  frames : List String
  progress : List String
  errors : String

private def runScript (lines : List String) : IO Captured := do
  let input ← IO.mkRef lines
  let frames ← IO.mkRef ([] : List String)
  let progress ← IO.mkRef ([] : List String)
  let errors ← IO.mkRef ""
  let status ← serve {
    readLine := do
      match ← input.get with
      | [] => pure none
      | line :: rest => input.set rest; pure (some line)
    writeFrame := fun line => frames.modify (· ++ [line])
    writeProgress := fun line => progress.modify (· ++ [line])
    writeError := fun text => errors.modify (· ++ text) } boundSets
  pure { status, frames := ← frames.get, progress := ← progress.get, errors := ← errors.get }

private def parseJson (label : String) (line : String) : IO Lean.Json :=
  match Lean.Json.parse line with
  | .ok json => pure json
  | .error reason => fail s!"{label}: not JSON ({reason}): {line}"

private def field (label : String) (json : Lean.Json) (name : String) : IO Lean.Json :=
  match json.getObjVal? name with
  | .ok value => pure value
  | .error _ => fail s!"{label}: no `{name}` in {json.compress}"

private def stringField (label : String) (json : Lean.Json) (name : String) : IO String := do
  match ← field label json name with
  | .str value => pure value
  | other => fail s!"{label}: `{name}` is not a string: {other.compress}"

private def natField (label : String) (json : Lean.Json) (name : String) : IO Nat := do
  match ← field label json name with
  | .num value => pure value.mantissa.toNat
  | other => fail s!"{label}: `{name}` is not a number: {other.compress}"

private def arrayField (label : String) (json : Lean.Json) (name : String) : IO (List Lean.Json) := do
  match ← field label json name with
  | .arr items => pure items.toList
  | other => fail s!"{label}: `{name}` is not an array: {other.compress}"

private def frameAt (captured : Captured) (index : Nat) (kind : String) (seq : Nat) : IO Lean.Json := do
  let some line := captured.frames[index]?
    | fail s!"frame {index}: no such frame; {captured.frames.length} were written: {captured.frames}"
  let json ← parseJson s!"frame {index}" line
  let actualKind ← stringField s!"frame {index}" json "frame"
  require (actualKind == kind) s!"frame {index}: is `{actualKind}`, not `{kind}`: {line}"
  let actualSeq ← natField s!"frame {index}" json "seq"
  require (actualSeq == seq) s!"frame {index}: carries seq {actualSeq}, not {seq}: {line}"
  pure json

private def contains (text fragment : String) : Bool := (text.splitOn fragment).length > 1

private def requireRejected (captured : Captured) (index : Nat) (seq : Nat) (mentions : String) :
    IO Unit := do
  let json ← frameAt captured index "rejected" seq
  let reason ← stringField s!"frame {index}" json "reason"
  require (contains reason mentions)
    s!"frame {index}: rejection reason does not mention `{mentions}`: {reason}"

/-- The fates an `exhausted` or `finished` frame lists, as (edit, fate). -/
private def fatesOf (label : String) (json : Lean.Json) (name : String) : IO (List (String × String)) := do
  (← arrayField label json name).mapM fun row => do
    pure (← stringField label row "edit", ← stringField label row "fate")

/-! ### Frames -/

private def profile : String := "umpire-replay.test"

private def quoted (value : String) : String := Lean.Json.compress (.str value)

private def admitQuery (seq : Nat) (setName query identity : String) : String :=
  s!"\{\"frame\":\"admit\",\"seq\":{seq},\"set\":{quoted setName},\"profile\":{quoted profile}," ++
    s!"\"query\":{quoted query},\"identity\":{quoted identity}}"

private def admitTarget (seq : Nat) (setName target identity : String) : String :=
  s!"\{\"frame\":\"admit\",\"seq\":{seq},\"set\":{quoted setName},\"profile\":{quoted profile}," ++
    s!"\"target\":{quoted target},\"identity\":{quoted identity}}"

private def plain (kind : String) (seq : Nat) (setName : String) : String :=
  s!"\{\"frame\":{quoted kind},\"seq\":{seq},\"set\":{quoted setName}}"

private def observe (seq : Nat) (setName candidate decided : String) : String :=
  s!"\{\"frame\":\"observe\",\"seq\":{seq},\"set\":{quoted setName},\"candidate\":{quoted candidate}," ++
    s!"\"profile\":{quoted profile},\"class\":{quoted decided}}"

/-! ### Subject identities from the checked-in fixtures -/

/-- The compact form of a JSON text: every byte outside a string that is insignificant whitespace
removed, as Go's `json.Compact` removes it. -/
private def compact (text : String) : String := Id.run do
  let mut out : List Char := []
  let mut inString := false
  let mut escaped := false
  for c in text.toList do
    if inString then
      out := c :: out
      if escaped then escaped := false
      else if c == '\\' then escaped := true
      else if c == '"' then inString := false
    else if c == '"' then
      inString := true
      out := c :: out
    else if c == ' ' || c == '\n' || c == '\t' || c == '\r' then
      pure ()
    else
      out := c :: out
  return String.ofList out.reverse

private def fixtureIdentity (fixture : String) : IO String := do
  let path : System.FilePath := ".." / "tests" / "testcore" / "testpilot" / "testdata" / (fixture ++ "-case.json")
  require (← path.pathExists) s!"fixture {path} is missing"
  pure (caseIdentity (compact (← IO.FS.readFile path)))

private def controlSet : String := "nexusCallerControl"
private def callerSet : String := "nexusCallerTests"
private def explorationSet : String := "nexusCallerExploration"

/-! ### The negative control is irreducible at once -/

private def controlScript (identity : String) : List String :=
  [admitQuery 1 controlSet "forgedCompletion" identity, plain "next" 2 controlSet,
    plain "finish" 3 controlSet]

private def checkControl : IO Unit := do
  let identity ← fixtureIdentity "nexusCallerControl-forgedCompletion"
  let captured ← runScript (controlScript identity)
  require (captured.status == 0) s!"exited {captured.status}: {captured.errors}"
  let admitted ← frameAt captured 0 "admitted" 1
  require ((← stringField "admitted" admitted "caseId") == "temporal.case.nexusCallerControl.forgedCompletion")
    "the subject is the control's registered Case"
  require ((← stringField "admitted" admitted "identity") == identity) "the identity is echoed"
  require ((← stringField "admitted" admitted "profile") == profile) "the Profile is echoed"
  let subject ← stringField "admitted" admitted "subject"
  require (subject.length == 64) s!"the subject's digest is a Plan checksum: {subject}"
  let edits ← arrayField "admitted" admitted "edits"
  require (edits.length == 1) s!"one prefix step: {edits.length}"
  require ((← stringField "edit" edits[0]! "edit") == "dropPrefixStep 0") "the schedule is the edit"
  let exhausted ← frameAt captured 1 "exhausted" 2
  require ((← fatesOf "exhausted" exhausted "skipped") == [("dropPrefixStep 0", "inapplicable")])
    "dropping the schedule is inapplicable"
  let finished ← frameAt captured 2 "finished" 3
  require ((← stringField "finished" finished "status") == "irreducible") "the control is irreducible"
  require ((← stringField "finished" finished "retained") == subject) "the subject is what is retained"
  require ((← fatesOf "finished" finished "edits") == [("dropPrefixStep 0", "inapplicable")])
    "every edit's fate is reported"

/-! ### A subject no set produces is crossed -/

private def checkCrossed : IO Unit := do
  let wrong := String.ofList (List.replicate 64 '0')
  let captured ← runScript [admitQuery 1 controlSet "forgedCompletion" wrong, plain "next" 2 controlSet]
  require (captured.status == 0) s!"a crossed subject ends the protocol: {captured.status}"
  require (captured.frames.length == 1) s!"nothing is read after `crossed`: {captured.frames}"
  let crossed ← frameAt captured 0 "crossed" 1
  require (contains (← stringField "crossed" crossed "reason") wrong) "the reason names the identity"
  let unknown ← runScript [admitQuery 1 controlSet "nothing" wrong]
  requireRejected unknown 0 1 "has no Query nothing"
  let unbound ← runScript [admitQuery 1 "elsewhere" "forgedCompletion" wrong]
  requireRejected unbound 0 1 "unknown set"
  let wrongKind ← runScript [admitTarget 1 controlSet "row:x" wrong]
  requireRejected wrongKind 0 1 "is functional"

/-! ### The caller's timeout Query reduces by one edit -/

private def callerOpening (identity : String) : List String :=
  [admitQuery 1 callerSet "scheduleToStartTimeout" identity, plain "next" 2 callerSet]

private def candidateOf (captured : Captured) (index seq : Nat) : IO (String × Lean.Json) := do
  let candidate ← frameAt captured index "candidate" seq
  pure (← stringField "candidate" candidate "candidate", candidate)

private def checkCaller : IO Unit := do
  let identity ← fixtureIdentity "nexusCallerTests-scheduleToStartTimeout"
  -- Reproduced: the candidate is retained, the schedule is still inapplicable, minimized.
  let opening ← runScript (callerOpening identity)
  let admitted ← frameAt opening 0 "admitted" 1
  let subject ← stringField "admitted" admitted "subject"
  let edits ← (← arrayField "admitted" admitted "edits").mapM fun edit => stringField "edit" edit "edit"
  require (edits == ["dropPrefixStep 1", "dropPrefixStep 0"]) s!"the sweep is last first: {edits}"
  let (digest, candidate) ← candidateOf opening 1 2
  require ((← stringField "candidate" candidate "edit") == "dropPrefixStep 1") "the worker stop is dropped first"
  require ((← stringField "candidate" candidate "caseId") == s!"temporal.case.{callerSet}.{digest}")
    "the candidate is named by its digest"
  require ((← stringField "candidate" candidate "fixture") == s!"{callerSet}-{digest}") "its fixture too"
  require (digest != subject) "the candidate's Plan is its own"
  let caseJson ← field "candidate" candidate "case"
  require ((← stringField "case" caseJson "caseId") == s!"temporal.case.{callerSet}.{digest}")
    "the whole Case carries the candidate's identity"
  require ((← stringField "candidate" candidate "identity").length == 64) "the Case checksum travels beside it"
  let reproduced ← runScript (callerOpening identity ++
    [observe 3 callerSet digest "reproduced", plain "next" 4 callerSet, plain "finish" 5 callerSet])
  let settled ← frameAt reproduced 2 "settled" 3
  require ((← stringField "settled" settled "fate") == "retained") "a reproduced candidate is retained"
  require ((← stringField "settled" settled "retained") == digest) "and is what later edits apply to"
  let exhausted ← frameAt reproduced 3 "exhausted" 4
  require ((← fatesOf "exhausted" exhausted "skipped") == [("dropPrefixStep 0", "inapplicable")])
    "the schedule cannot be dropped"
  let finished ← frameAt reproduced 4 "finished" 5
  require ((← stringField "finished" finished "status") == "minimized") "one retained edit is minimized"
  require ((← stringField "finished" finished "retained") == digest) "the retained candidate is reported"
  require ((← fatesOf "finished" finished "edits") ==
    [("dropPrefixStep 1", "retained"), ("dropPrefixStep 0", "inapplicable")]) "every fate, in order"
  -- Not reproduced: nothing is retained, irreducible.
  let notReproduced ← runScript (callerOpening identity ++
    [observe 3 callerSet digest "not-reproduced", plain "next" 4 callerSet, plain "finish" 5 callerSet])
  let finished ← frameAt notReproduced 4 "finished" 5
  require ((← stringField "finished" finished "status") == "irreducible") "nothing retained is irreducible"
  require ((← stringField "finished" finished "retained") == subject) "the subject stays retained"
  -- Indeterminate after the retry: the reduction ends incomplete, naming the edit.
  let undecided ← runScript (callerOpening identity ++
    [observe 3 callerSet digest "indeterminate", plain "next" 4 callerSet, plain "finish" 4 callerSet])
  let settled ← frameAt undecided 2 "settled" 3
  require ((← stringField "settled" settled "fate") == "undecided") "an indeterminate candidate is undecided"
  requireRejected undecided 3 4 "the reduction ended"
  let finished ← frameAt undecided 4 "finished" 4
  require ((← stringField "finished" finished "status") == "incomplete") "an undecided edit is incomplete"
  require (contains (← stringField "finished" finished "reason") "dropPrefixStep 1") "naming the edit"
  -- A preparation rejection is `rejected`, never rerun.
  let prepared ← runScript (callerOpening identity ++
    [s!"\{\"frame\":\"observe\",\"seq\":3,\"set\":\"{callerSet}\",\"candidate\":\"{digest}\",\"profile\":\"{profile}\",\"prepareRejected\":\"unsupported opcode\"}",
      plain "next" 4 callerSet, plain "finish" 5 callerSet])
  let settled ← frameAt prepared 2 "settled" 3
  require ((← stringField "settled" settled "fate") == "rejected") "a rejected preparation is rejected"
  let finished ← frameAt prepared 4 "finished" 5
  require ((← stringField "finished" finished "status") == "irreducible") "a rejected edit retains nothing"
  -- Stopped early with a candidate outstanding: incomplete, and every edit is still listed.
  let stopped ← runScript (callerOpening identity ++ [plain "finish" 3 callerSet])
  let finished ← frameAt stopped 2 "finished" 3
  require ((← stringField "finished" finished "status") == "incomplete") "an unsettled candidate is incomplete"
  require ((← fatesOf "finished" finished "edits") ==
    [("dropPrefixStep 1", "unsettled"), ("dropPrefixStep 0", "not-tried")]) "every edit's fate, even untried"
  let unsettled := (← arrayField "finished" finished "edits")[0]!
  require ((← stringField "edit" unsettled "candidate") == digest) "the outstanding candidate is named"

/-! ### An exploration candidate is irreducible at once -/

private def checkExploration : IO Unit := do
  -- The subject is the exploration bridge's own first candidate: its target key, and the SHA-256
  -- of the Case `umpire-explore` hands out for it, so admission compares the two bridges.
  let binding := Temporal.Tool.ExplorationBridge.nexusCallerBinding
  let some campaign := (Umpire.Exploration.Campaign.check
      Temporal.Feature.Nexus.Caller.nexusProtocol binding.set binding.limits).toOption
    | fail "the caller's exploratory set is not a campaign"
  let runner := Temporal.Tool.ExplorationBridge.Runner.ofSession binding
    (Umpire.Exploration.Session.begin campaign)
  let .candidate view _ _ := runner.next | fail "the exploration campaign hands out no candidate"
  let key := view.target
  let encoded ← match view.produced with
    | .ok value =>
        match ← Testpilot.ProtoJSON.canonical value with
        | .ok encoded => pure encoded
        | .error error => fail s!"encoding: {error}"
    | .error error => fail s!"production: {error.construct}"
  let some recovered := (Bound.exploratory binding).recover (.target key) |>.toOption
    | fail s!"the replay bridge does not recover target {key}"
  let captured ← runScript [admitTarget 1 explorationSet key (caseIdentity encoded),
    plain "next" 2 explorationSet, plain "finish" 3 explorationSet]
  let admitted ← frameAt captured 0 "admitted" 1
  let subject ← stringField "admitted" admitted "subject"
  require ((← stringField "admitted" admitted "caseId") == view.caseId)
    "the exploration subject is the Case the exploration bridge hands out"
  require ((← stringField "admitted" admitted "caseId") == s!"temporal.case.{explorationSet}.{subject}")
    "an exploration subject is named by its own digest"
  let exhausted ← frameAt captured 1 "exhausted" 2
  let fates ← fatesOf "exhausted" exhausted "skipped"
  require (fates.all (·.2 == "inapplicable")) s!"every edit of a shortest path is inapplicable: {fates}"
  require (fates.length == recovered.reduction.sweep.length) "every edit is listed once"
  let finished ← frameAt captured 2 "finished" 3
  require ((← stringField "finished" finished "status") == "irreducible") "irreducible at once"
  let wrongKind ← runScript [admitQuery 1 explorationSet "retry" (caseIdentity encoded)]
  requireRejected wrongKind 0 1 "is exploratory"

/-! ### Rejections before any Query is admitted -/

private def checkRejections : IO Unit := do
  let identity ← fixtureIdentity "nexusCallerTests-scheduleToStartTimeout"
  let captured ← runScript [
    plain "next" 1 callerSet,
    admitQuery 1 callerSet "scheduleToStartTimeout" identity,
    admitQuery 1 callerSet "scheduleToStartTimeout" identity,
    plain "next" 5 callerSet,
    admitQuery 2 callerSet "retry" identity,
    plain "next" 2 controlSet,
    observe 2 callerSet "nothing" "reproduced",
    plain "next" 2 callerSet,
    plain "next" 3 callerSet,
    observe 3 callerSet "nothing" "reproduced",
    "{\"frame\":\"observe\",\"seq\":3,\"set\":\"nexusCallerTests\",\"candidate\":\"x\",\"profile\":\"other\",\"class\":\"reproduced\"}",
    "{\"frame\":\"next\",\"seq\":3,\"set\":\"nexusCallerTests\",\"edit\":\"dropPrefixStep 0\"}",
    "{\"frame\":\"observe\",\"seq\":3,\"set\":\"nexusCallerTests\",\"candidate\":\"x\",\"profile\":\"p\",\"class\":\"maybe\"}",
    "{\"frame\":\"admit\",\"seq\":3,\"set\":\"s\",\"profile\":\"p\",\"query\":\"q\",\"identity\":\"ABC\"}",
    String.ofList (List.replicate maxLineBytes ' ') ++ "x",
    plain "finish" 3 callerSet]
  require (captured.status == 0) s!"exited {captured.status}: {captured.errors}"
  requireRejected captured 0 1 "send `admit` first"
  let _ ← frameAt captured 1 "admitted" 1
  requireRejected captured 2 1 "duplicate frame"
  requireRejected captured 3 5 "out-of-order"
  requireRejected captured 4 2 "already admitted"
  requireRejected captured 5 2 "names set nexusCallerControl"
  requireRejected captured 6 2 "no candidate is outstanding"
  let (digest, _) ← candidateOf captured 7 2
  requireRejected captured 8 3 s!"candidate {digest} is outstanding"
  requireRejected captured 9 3 "crossed observe"
  requireRejected captured 10 3 "crossed profile"
  requireRejected captured 11 3 "admits no `edit`"
  requireRejected captured 12 3 "class `maybe`"
  requireRejected captured 13 3 "lowercase hex"
  requireRejected captured 14 0 "oversized frame"
  let _ ← frameAt captured 15 "finished" 3
  -- A settled candidate's observe is stale.
  let stale ← runScript (callerOpening identity ++ [observe 3 callerSet digest "not-reproduced",
    observe 4 callerSet digest "reproduced", plain "finish" 4 callerSet])
  requireRejected stale 3 4 "stale observe"

/-! ### The same frames give the same frames -/

private def checkDeterminism : IO Unit := do
  let control ← fixtureIdentity "nexusCallerControl-forgedCompletion"
  let first ← runScript (controlScript control)
  let second ← runScript (controlScript control)
  require (first.frames == second.frames) "the control's frames differ between runs"
  let identity ← fixtureIdentity "nexusCallerTests-scheduleToStartTimeout"
  let opening ← runScript (callerOpening identity)
  let (digest, _) ← candidateOf opening 1 2
  let script := callerOpening identity ++
    [observe 3 callerSet digest "reproduced", plain "next" 4 callerSet, plain "finish" 5 callerSet]
  let once ← runScript script
  let twice ← runScript script
  require (once.frames == twice.frames) "the caller's frames differ between runs"
  require (once.progress == twice.progress) "the caller's progress differs between runs"

/-! ### The executable -/

private def executable : IO System.FilePath := do
  let path := (← IO.currentDir) / ".lake" / "build" / "bin" / "umpire-replay-bridge"
  require (← path.pathExists) s!"bridge executable is missing at {path}"
  pure path

private def runExecutable (input : String) : IO IO.Process.Output := do
  let path ← executable
  IO.Process.output { cmd := "sh", args := #["-c", "printf '%s\\n' \"$1\" | \"$2\"", "sh", input, path.toString] }

private def checkExecutable : IO Unit := do
  let identity ← fixtureIdentity "nexusCallerControl-forgedCompletion"
  let output ← runExecutable ("\n".intercalate (controlScript identity))
  require (output.exitCode == 0) s!"executable exited {output.exitCode}: {output.stderr}"
  let lines := (output.stdout.splitOn "\n").filter (!·.isEmpty)
  require (lines.length == 3) s!"executable wrote {lines.length} lines"
  require (output.stdout.endsWith "\n") "frames do not end with LF"
  let captured : Captured := { status := 0, frames := lines, progress := [], errors := "" }
  let _ ← frameAt captured 0 "admitted" 1
  let _ ← frameAt captured 1 "exhausted" 2
  let _ ← frameAt captured 2 "finished" 3
  require (contains output.stderr "admitted ") s!"no progress line: {output.stderr}"
  let closed ← runExecutable (admitQuery 1 controlSet "forgedCompletion" identity)
  require (closed.exitCode == 1) s!"stdin closing exited {closed.exitCode}"
  require (contains closed.stderr diagnosticPrefix) s!"no diagnostic: {closed.stderr}"
  let withArguments ← IO.Process.output { cmd := (← executable).toString, args := #["--help"] }
  require (withArguments.exitCode == 2) "arguments were accepted"
  require withArguments.stdout.isEmpty "arguments wrote stdout"

def main : IO UInt32 := do
  let checks : List (String × IO Unit) := [
    ("control", checkControl),
    ("crossed", checkCrossed),
    ("caller", checkCaller),
    ("exploration", checkExploration),
    ("rejections", checkRejections),
    ("determinism", checkDeterminism),
    ("executable", checkExecutable)]
  let mut failed := 0
  for (name, check) in checks do
    try
      check
      IO.println s!"ok {name}"
    catch failure =>
      IO.eprintln s!"FAIL {name}: {failure}"
      failed := failed + 1
  pure (if failed == 0 then 0 else 1)

end Temporal.Tool.ReplayBridgeTests

def main : IO UInt32 := Temporal.Tool.ReplayBridgeTests.main
