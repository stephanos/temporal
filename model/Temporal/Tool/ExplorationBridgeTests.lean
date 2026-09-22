import Temporal.Tool.ExplorationBridge
import Testpilot.Examples.Synthetic
import Testpilot.Authoring

/-!
# What the bridge does at its frame boundary

The campaign has its own tests; these are about the protocol around it. A stub runner with two
candidates stands in for a campaign so that every rejection -- duplicate, out-of-order, crossed,
stale, wrong set, after the end -- is pinned against the frame it answers and the frame the
protocol expects next, without a search per frame. The caller Model's own set is then driven for
real: `initialize`, the first candidate as a Case, a satisfied Run credited to its path, `finish`.
The built executable is run once over the same frames so that what reaches each stream is
measured, not inferred.
-/

namespace Temporal.Tool.ExplorationBridgeTests

open Umpire Umpire.Exploration Temporal.Tool.ExplorationBridge
open Umpire.Exploration (Observation)

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

/-- Serve one script of lines with injected effects, capturing every stream separately. -/
private def runScript (bound : List Bound) (lines : List String)
    (promotion : ArtifactChecksum → Option String := fun _ => none) : IO Captured := do
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
    writeError := fun text => errors.modify (· ++ text)
    promotion } bound
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

private def stringsField (label : String) (json : Lean.Json) (name : String) : IO (List String) := do
  match ← field label json name with
  | .arr items => items.toList.mapM fun item => match item with
      | .str value => pure value
      | other => fail s!"{label}: `{name}` holds a non-string: {other.compress}"
  | other => fail s!"{label}: `{name}` is not an array: {other.compress}"

/-- The frame at one position of the captured output, checked to be of one kind at one sequence
number. -/
private def frameAt (captured : Captured) (index : Nat) (kind : String) (seq : Nat) : IO Lean.Json := do
  let some line := captured.frames[index]?
    | fail s!"frame {index}: no such frame; {captured.frames.length} were written"
  let json ← parseJson s!"frame {index}" line
  let actualKind ← stringField s!"frame {index}" json "frame"
  require (actualKind == kind) s!"frame {index}: is `{actualKind}`, not `{kind}`: {line}"
  let actualSeq ← natField s!"frame {index}" json "seq"
  require (actualSeq == seq) s!"frame {index}: carries seq {actualSeq}, not {seq}: {line}"
  pure json

private def requireRejected (captured : Captured) (index : Nat) (seq : Nat) (mentions : String) :
    IO Unit := do
  let json ← frameAt captured index "rejected" seq
  let reason ← stringField s!"frame {index}" json "reason"
  require ((reason.splitOn mentions).length > 1)
    s!"frame {index}: rejection reason does not mention `{mentions}`: {reason}"

private def requireObject (label : String) (json : Lean.Json) (name : String) : IO Unit := do
  match ← field label json name with
  | .obj _ => pure ()
  | other => fail s!"{label}: `{name}` is not an object: {other.compress}"

private def statusesOf (label : String) (json : Lean.Json) (name : String) :
    IO (List (String × String)) := do
  match ← field label json name with
  | .arr rows => rows.toList.mapM fun row => do
      pure (← stringField label row "target", ← stringField label row "status")
  | other => fail s!"{label}: `{name}` is not an array: {other.compress}"

/-! ### A stub campaign

Two candidates over targets `t1`, `t2` and `t3`: the first covers `t1`, the second `t2` and `t3`;
`t3` is otherwise pending. The runner credits the way the ledger does for the statuses these tests
read: satisfied covers, violated marks violated, anything else attempted. -/

private def stubIdentity (index : Nat) : ArtifactChecksum := planChecksumOf s!"stub-{index}"

private structure StubCandidate where
  identity : ArtifactChecksum
  target : String
  covers : List String

private def stubCandidates : List StubCandidate := [
  { identity := stubIdentity 1, target := "t1", covers := ["t1"] },
  { identity := stubIdentity 2, target := "t2", covers := ["t2", "t3"] }]

private def stubSet : String := "stub"

private def stubView (candidate : StubCandidate) : CandidateView :=
  { identity := candidate.identity
    target := candidate.target
    covers := candidate.covers
    caseId := caseIdOf stubSet candidate.identity
    fixture := fixtureOf stubSet candidate.identity
    produced := .ok Testpilot.Examples.Synthetic.case }

private def stubStatus (observation : Observation) : TargetStatus :=
  match observation with
  | .satisfied => .covered
  | .violated => .violated
  | _ => .attempted

private def stubSummary (ledger : List (String × TargetStatus)) (selected : Nat) : Campaign.Summary :=
  let count (status : TargetStatus) := (ledger.filter (·.2 == status)).length
  { targets := ledger.length
    selected
    covered := count .covered
    unreachable := count .unreachable
    violated := count .violated
    attempted := count .attempted
    unrealizable := count .unrealizable
    pending := count .pending + count .planned
    counterexamples := []
    exhausted := ledger.all fun (_, status) => status != .pending && status != .planned }

/-- The stub runner: `remaining` candidates to hand out, `outstanding` the one out, `ledger` the
statuses so far, `selected` how many were handed out. -/
private partial def stubRunner (remaining : List StubCandidate) (outstanding : Option StubCandidate)
    (ledger : List (String × TargetStatus)) (selected : Nat) : Runner :=
  .mk
    (fun _ => match outstanding, remaining with
      | some _, _ => .outstanding
      | none, [] => .exhausted [] (stubRunner [] none ledger selected)
      | none, candidate :: rest =>
          let planned := ledger.map fun (key, status) =>
            if candidate.covers.contains key then (key, TargetStatus.planned) else (key, status)
          .candidate (stubView candidate) [] (stubRunner rest (some candidate) planned (selected + 1)))
    (fun identity observation => match outstanding with
      | some candidate =>
          if candidate.identity != identity then .rejected
          else
            let after := ledger.map fun (key, status) =>
              if candidate.covers.contains key then (key, stubStatus observation) else (key, status)
            let covered := if observation == .satisfied then candidate.covers else []
            .credited covered (after.filter fun (key, _) => candidate.covers.contains key)
              (stubRunner remaining none after selected)
      | none => .rejected)
    (fun _ => stubSummary ledger selected)
    (fun _ => ledger)

private def stubBound : Bound :=
  { name := stubSet
    machine := "stub.machine"
    budget := "two"
    limits := Limits.bounded 2 2 64
    targets := ["t1", "t2", "t3"]
    campaign := fun _ => .ok (stubRunner stubCandidates none
      [("t1", .pending), ("t2", .pending), ("t3", .pending)] 0) }

/-! ### Frames as the coordinator writes them -/

private def frame (kind : String) (seq : Nat) (setName : String) (extra : String := "") : String :=
  "{\"frame\":\"" ++ kind ++ "\",\"seq\":" ++ toString seq ++ ",\"set\":\"" ++ setName ++ "\"" ++
    extra ++ "}"

private def profile : String := "test-profile"

private def openFrame (seq : Nat) (setName : String) (under : String := profile) : String :=
  frame "initialize" seq setName (",\"profile\":\"" ++ under ++ "\"")

private def observeRejected (seq : Nat) (setName candidate : String) (under : String := profile) : String :=
  frame "observe" seq setName (",\"candidate\":\"" ++ candidate ++ "\",\"profile\":\"" ++ under ++
    "\",\"prepareRejected\":\"no worker\"")

/-- A closed Run for one Case ID with one disposition, cleanup status and Verdict status. -/
private def runFor (caseId : String) (disposition : temporal.server.api.testpilot.v1.RunDisposition)
    (cleanup : temporal.server.api.testpilot.v1.CleanupStatus)
    (verdict : temporal.server.api.testpilot.v1.VerdictStatus) : temporal.server.api.testpilot.v1.Run :=
  Testpilot.Authoring.Run.make "run-1" caseId (caseId ++ ".program") #[] disposition
    (Testpilot.Authoring.Run.cleanup cleanup) (Testpilot.Authoring.Verdict.make verdict #[])

private def encodeRun (run : temporal.server.api.testpilot.v1.Run) : IO String := do
  match ← Testpilot.ProtoJSON.canonical run with
  | .ok encoded => pure encoded
  | .error error => fail s!"run does not encode: {error}"

private def observeRun (seq : Nat) (setName candidate : String) (run : temporal.server.api.testpilot.v1.Run) :
    IO String := do
  pure (frame "observe" seq setName (",\"candidate\":\"" ++ candidate ++ "\",\"profile\":\"" ++ profile ++
    "\",\"run\":" ++ (← encodeRun run)))

private def satisfiedRun (caseId : String) : temporal.server.api.testpilot.v1.Run :=
  runFor caseId .RUN_DISPOSITION_COMPLETED .CLEANUP_STATUS_SUCCEEDED .VERDICT_STATUS_SATISFIED

/-! ### Reading a Run -/

private def checkObservations : IO Unit := do
  let caseId := "temporal.case.stub.abc"
  let expect (label : String) (run : temporal.server.api.testpilot.v1.Run) (expected : Observation) : IO Unit := do
    require (observationOf run == expected) s!"{label}: read as {(observationOf run).name}, not {expected.name}"
    -- The same Run through the codec and the decoder reads the same.
    let json ← parseJson label (← encodeRun run)
    match ← decodeRun json with
    | .error reason => fail s!"{label}: decoded Run rejected: {reason}"
    | .ok decoded =>
        require (observationOf decoded == expected) s!"{label}: decoded Run reads differently"
  expect "completed satisfied" (satisfiedRun caseId) .satisfied
  expect "completed violated"
    (runFor caseId .RUN_DISPOSITION_COMPLETED .CLEANUP_STATUS_SUCCEEDED .VERDICT_STATUS_VIOLATED) .violated
  expect "stopped by monitor violated"
    (runFor caseId .RUN_DISPOSITION_STOPPED_BY_MONITOR .CLEANUP_STATUS_SUCCEEDED .VERDICT_STATUS_VIOLATED)
    .violated
  expect "stopped by monitor satisfied"
    (runFor caseId .RUN_DISPOSITION_STOPPED_BY_MONITOR .CLEANUP_STATUS_SUCCEEDED .VERDICT_STATUS_SATISFIED)
    .inconclusive
  expect "incomplete"
    (runFor caseId .RUN_DISPOSITION_INCOMPLETE .CLEANUP_STATUS_SUCCEEDED .VERDICT_STATUS_SATISFIED)
    .inconclusive
  expect "inconclusive verdict"
    (runFor caseId .RUN_DISPOSITION_COMPLETED .CLEANUP_STATUS_SUCCEEDED .VERDICT_STATUS_INCONCLUSIVE)
    .inconclusive
  expect "cleanup failed"
    (runFor caseId .RUN_DISPOSITION_COMPLETED .CLEANUP_STATUS_FAILED .VERDICT_STATUS_SATISFIED)
    .inconclusive
  expect "cleanup timed out after violation"
    (runFor caseId .RUN_DISPOSITION_STOPPED_BY_MONITOR .CLEANUP_STATUS_TIMED_OUT .VERDICT_STATUS_VIOLATED)
    .inconclusive
  expect "cleanup unspecified"
    (runFor caseId .RUN_DISPOSITION_COMPLETED .CLEANUP_STATUS_UNSPECIFIED .VERDICT_STATUS_SATISFIED)
    .inconclusive
  require (readRun caseId (satisfiedRun caseId) == (.satisfied, "")) "a decisive Run has no detail"
  require ((readRun caseId (satisfiedRun "temporal.case.stub.other")).1 == .inconclusive &&
      (readRun caseId (satisfiedRun "temporal.case.stub.other")).2.startsWith "run run-1 names Case")
    "a Run for another Case says so"
  require ((readRun caseId (runFor caseId .RUN_DISPOSITION_COMPLETED .CLEANUP_STATUS_FAILED
      .VERDICT_STATUS_SATISFIED)).2 == "cleanup is not closed") "an unclosed cleanup says so"
  -- A Run with fields the schema does not declare does not decode.
  match ← decodeRun (Lean.Json.mkObj [("runId", Lean.Json.str "run-1"), ("caseId", Lean.Json.str caseId),
      ("extra", Lean.Json.str "1")]) with
  | .ok _ => fail "a Run with an undeclared field decoded"
  | .error _ => pure ()

/-! ### Frames that are no frame, and frames to reject before any campaign call -/

private def checkParsing : IO Unit := do
  let notFrame (line : String) (mentions : String) : IO Unit :=
    match parseFrame line with
    | .error reason =>
        require ((reason.splitOn mentions).length > 1) s!"`{line}`: reason `{reason}` lacks `{mentions}`"
    | .ok _ => fail s!"`{line}` parsed as a frame"
  let malformed (line : String) (mentions : String) : IO Unit :=
    match parseFrame line with
    | .ok (.error reason) =>
        require ((reason.splitOn mentions).length > 1) s!"`{line}`: reason `{reason}` lacks `{mentions}`"
    | .ok (.ok _) => fail s!"`{line}` parsed as a well-formed frame"
    | .error reason => fail s!"`{line}` is no frame at all: {reason}"
  notFrame "not json" "not JSON"
  notFrame "[1,2]" "not a JSON object"
  malformed "{}" "`frame`"
  malformed "{\"frame\":\"next\"}" "`seq`"
  malformed "{\"frame\":\"next\",\"seq\":1}" "`set`"
  malformed "{\"frame\":\"dance\",\"seq\":1,\"set\":\"s\"}" "unknown frame"
  malformed "{\"frame\":\"initialize\",\"seq\":1,\"set\":\"s\"}" "`profile`"
  malformed "{\"frame\":\"observe\",\"seq\":1,\"set\":\"s\"}" "`candidate`"
  malformed "{\"frame\":\"observe\",\"seq\":1,\"set\":\"s\",\"candidate\":\"c\"}" "`profile`"
  malformed "{\"frame\":\"observe\",\"seq\":1,\"set\":\"s\",\"candidate\":\"c\",\"profile\":\"p\"}" "neither"
  malformed "{\"frame\":\"observe\",\"seq\":1,\"set\":\"s\",\"candidate\":\"c\",\"profile\":\"p\",\"run\":{},\"prepareRejected\":\"x\"}" "both"
  malformed "{\"frame\":\"finish\",\"seq\":1,\"set\":\"s\",\"status\":\"done\"}" "one of"
  -- A frame is exact: a key the bridge does not read, such as one naming a target, rejects.
  malformed "{\"frame\":\"next\",\"seq\":1,\"set\":\"s\",\"target\":\"row:x\"}" "admits no `target`"
  malformed "{\"frame\":\"initialize\",\"seq\":1,\"set\":\"s\",\"profile\":\"p\",\"case\":{}}" "admits no `case`"
  malformed "{\"frame\":\"finish\",\"seq\":1,\"set\":\"s\",\"candidate\":\"c\"}" "admits no `candidate`"
  match parseFrame "{\"frame\":\"finish\",\"seq\":9,\"set\":\"s\",\"status\":\"limit-reached\"}" with
  | .ok (.ok parsed) =>
      let limitReached := match parsed.request with
        | .finish (some "limit-reached") => true
        | _ => false
      require (parsed.seq == 9 && parsed.setName == "s" && limitReached) "finish with limit-reached parsed differently"
  | _ => fail "finish with limit-reached did not parse"

/-! ### The protocol over the stub -/

private def checkProtocol : IO Unit := do
  let first := stubIdentity 1
  let second := stubIdentity 2
  let firstCase := caseIdOf stubSet first
  let secondCase := caseIdOf stubSet second
  let script : List String := [
    frame "next" 1 stubSet,                        -- 0: no campaign open
    openFrame 1 "other",                          -- 1: unknown set
    openFrame 1 stubSet,                          -- 2: initialized
    openFrame 2 stubSet,                          -- 3: already open
    "",                                            --    blank lines are skipped
    frame "next" 2 stubSet,                        -- 4: candidate 1
    frame "next" 3 stubSet,                        -- 5: outstanding
    frame "next" 2 stubSet,                        -- 6: duplicate
    frame "next" 7 stubSet,                        -- 7: out of order
    observeRejected 3 "other" first.render,        -- 8: wrong set
    observeRejected 3 stubSet second.render,       -- 9: crossed
    "{\"frame\":\"observe\",\"seq\":3}",           -- 10: malformed, rejected at seq 0
    observeRejected 3 stubSet first.render "other-profile", -- 11: crossed profile
    frame "observe" 3 stubSet (",\"candidate\":\"" ++ first.render ++ "\",\"profile\":\"" ++ profile ++
      "\",\"run\":{}"),                              -- 12: an empty Run is no observation
    frame "observe" 3 stubSet (",\"candidate\":\"" ++ first.render ++ "\",\"profile\":\"" ++ profile ++
      "\",\"run\":{\"runId\":\"r\",\"caseId\":\"c\",\"extra\":1}"), -- 13: an undecodable Run likewise
    observeRejected 3 stubSet first.render,        -- 14: credited prepare-rejected
    observeRejected 4 stubSet first.render,        -- 15: stale
    frame "next" 4 stubSet,                        -- 16: candidate 2
    ← observeRun 5 stubSet second.render (satisfiedRun "temporal.case.stub.other"), -- 17: another Case's Run
    ← observeRun 6 stubSet second.render (satisfiedRun secondCase), -- 18: stale, already observed
    frame "next" 6 stubSet,                        -- 19: exhausted
    frame "next" 7 stubSet,                        -- 20: after exhaustion
    frame "finish" 7 stubSet]                      -- 21: finished
  let captured ← runScript [stubBound] script
  require (captured.status == 0) s!"bridge exited {captured.status}: {captured.errors}"
  require captured.errors.isEmpty s!"bridge wrote a diagnostic: {captured.errors}"
  require (captured.frames.length == 22) s!"bridge wrote {captured.frames.length} frames, not 22"
  requireRejected captured 0 1 "no campaign is open"
  requireRejected captured 1 1 "unknown set"
  let initialized ← frameAt captured 2 "initialized" 1
  require ((← stringsField "initialized" initialized "targets") == ["t1", "t2", "t3"]) "targets differ"
  require ((← stringField "initialized" initialized "machine") == "stub.machine") "machine differs"
  require ((← stringField "initialized" initialized "budget") == "two") "budget differs"
  require ((← stringField "initialized" initialized "profile") == profile) "profile not echoed"
  let limits ← field "initialized" initialized "limits"
  require ((← natField "limits" limits "steps") == 2 && (← natField "limits" limits "actions") == 2 &&
    (← natField "limits" limits "search") == 64) "limits differ"
  requireRejected captured 3 2 "already open"
  let candidate ← frameAt captured 4 "candidate" 2
  require ((← stringField "candidate" candidate "candidate") == first.render) "candidate identity differs"
  require ((← stringField "candidate" candidate "target") == "t1") "candidate target differs"
  require ((← stringsField "candidate" candidate "covers") == ["t1"]) "candidate covers differ"
  require ((← stringField "candidate" candidate "caseId") == firstCase) "candidate Case ID differs"
  require ((← stringField "candidate" candidate "fixture") == fixtureOf stubSet first) "fixture differs"
  require ((← stringField "candidate" (← field "candidate" candidate "case") "caseId") ==
    "testpilot.synthetic.case") "the frame does not embed the Case"
  -- The embedded Case is the codec's bytes, verbatim.
  let some line := captured.frames[4]? | fail "no candidate frame"
  match ← Testpilot.ProtoJSON.canonical Testpilot.Examples.Synthetic.case with
  | .ok encoded =>
      require ((line.splitOn ("\"case\":" ++ encoded ++ "}")).length == 2)
        "the candidate frame does not carry the canonical Case bytes"
  | .error error => fail s!"synthetic Case does not encode: {error}"
  requireRejected captured 5 3 "outstanding"
  requireRejected captured 6 2 "duplicate"
  requireRejected captured 7 7 "out-of-order"
  requireRejected captured 8 3 "names set other"
  requireRejected captured 9 3 "crossed observe"
  requireRejected captured 10 0 "`set`"
  requireRejected captured 11 3 "crossed profile"
  requireRejected captured 12 3 "names no `runId`"
  requireRejected captured 13 3 "does not decode"
  let credited ← frameAt captured 14 "credited" 3
  require ((← stringField "credited" credited "observation") == "prepare-rejected") "observation differs"
  require ((← stringField "credited" credited "profile") == profile) "profile not echoed on credit"
  require ((← stringsField "credited" credited "credited") == []) "a rejected preparation credited something"
  require ((← statusesOf "credited" credited "statuses") == [("t1", "attempted")]) "statuses differ"
  requireRejected captured 15 4 "stale"
  let candidate2 ← frameAt captured 16 "candidate" 4
  require ((← stringField "candidate" candidate2 "candidate") == second.render) "second identity differs"
  let crossedRun ← frameAt captured 17 "credited" 5
  require ((← stringField "credited" crossedRun "observation") == "inconclusive") "a crossed Run was decisive"
  require (((← stringField "credited" crossedRun "detail").splitOn "names Case").length == 2) "no detail"
  require ((← statusesOf "credited" crossedRun "statuses") == [("t2", "attempted"), ("t3", "attempted")])
    "crossed statuses differ"
  requireRejected captured 18 6 "stale"
  let _ ← frameAt captured 19 "exhausted" 6
  requireRejected captured 20 7 "exhausted"
  let finished ← frameAt captured 21 "finished" 7
  require ((← stringField "finished" finished "status") == "exhausted") "status differs"
  let summary ← field "finished" finished "summary"
  require ((← natField "summary" summary "selected") == 2) "selected differs"
  require ((← natField "summary" summary "attempted") == 3) "attempted differs"
  require ((← statusesOf "finished" finished "ledger") ==
    [("t1", "attempted"), ("t2", "attempted"), ("t3", "attempted")]) "ledger differs"
  require (captured.progress == [
    s!"candidate {first.render} t1", s!"observed {first.render} prepare-rejected",
    s!"candidate {second.render} t2", s!"observed {second.render} inconclusive"]) "progress lines differ"

/-- Satisfied and violated Runs credit along the planned path; `finish` before exhaustion is
`stopped` unless the coordinator names `limit-reached`. -/
private def checkCredit : IO Unit := do
  let first := stubIdentity 1
  let second := stubIdentity 2
  let captured ← runScript [stubBound] [
    openFrame 1 stubSet,
    frame "next" 2 stubSet,
    ← observeRun 3 stubSet first.render (satisfiedRun (caseIdOf stubSet first)),
    frame "next" 4 stubSet,
    ← observeRun 5 stubSet second.render
      (runFor (caseIdOf stubSet second) .RUN_DISPOSITION_STOPPED_BY_MONITOR .CLEANUP_STATUS_SUCCEEDED
        .VERDICT_STATUS_VIOLATED),
    frame "finish" 6 stubSet ",\"status\":\"limit-reached\""]
  require (captured.status == 0) s!"bridge exited {captured.status}: {captured.errors}"
  let credited ← frameAt captured 2 "credited" 3
  require ((← stringField "credited" credited "observation") == "satisfied") "satisfied not read"
  require ((← stringsField "credited" credited "credited") == ["t1"]) "satisfied credited nothing"
  require ((← statusesOf "credited" credited "statuses") == [("t1", "covered")]) "not covered"
  let violated ← frameAt captured 4 "credited" 5
  require ((← stringField "credited" violated "observation") == "violated") "violated not read"
  require ((← stringsField "credited" violated "credited") == []) "violated credited something"
  require ((← statusesOf "credited" violated "statuses") == [("t2", "violated"), ("t3", "violated")])
    "not violated"
  let finished ← frameAt captured 5 "finished" 6
  require ((← stringField "finished" finished "status") == "exhausted")
    "an exhausted ledger reports exhausted whatever finish names"
  -- Stopped early: the requested status stands.
  let early ← runScript [stubBound] [
    openFrame 1 stubSet, frame "finish" 2 stubSet ",\"status\":\"limit-reached\""]
  require (early.status == 0) s!"early finish exited {early.status}"
  require ((← stringField "finished" (← frameAt early 1 "finished" 2) "status") == "limit-reached")
    "limit-reached not reported"
  let stopped ← runScript [stubBound] [openFrame 1 stubSet, frame "finish" 2 stubSet]
  require ((← stringField "finished" (← frameAt stopped 1 "finished" 2) "status") == "stopped")
    "stopped not reported"

/-! ### Failures outside the protocol -/

private def checkOutside : IO Unit := do
  let closed ← runScript [stubBound] [openFrame 1 stubSet, frame "next" 2 stubSet]
  require (closed.status == 1) s!"stdin closing exited {closed.status}"
  require (closed.frames.length == 2) "frames before stdin closed were not written"
  require ((closed.errors.splitOn "stdin closed before `finish`").length == 2) s!"no diagnostic: {closed.errors}"
  require (closed.errors.startsWith diagnosticPrefix) "diagnostic lacks the prefix"
  let garbage ← runScript [stubBound] [openFrame 1 stubSet, "not a frame", frame "finish" 2 stubSet]
  require (garbage.status == 1) s!"a non-frame line exited {garbage.status}"
  require (garbage.frames.length == 1) "frames after a non-frame line were written"
  require ((garbage.errors.splitOn "line is not a frame").length == 2) s!"no diagnostic: {garbage.errors}"

/-! ### The caller Model for real -/

private def callerSet : String := "nexusCallerExploration"

private def checkCaller : IO Unit := do
  let some bound := boundSets.find? (·.name == callerSet) | fail "the bridge does not bind the caller set"
  require (bound.machine == "temporal.nexus.caller.machine.nexusProtocol") s!"machine is {bound.machine}"
  require (bound.budget == "four") s!"budget is {bound.budget}"
  let some firstTarget := bound.targets.head? | fail "the caller set has no targets"
  require (firstTarget.startsWith "row:") s!"the first target is {firstTarget}, not a row"
  let opened ← runScript boundSets [openFrame 1 callerSet, frame "next" 2 callerSet]
  require (opened.frames.length == 2) s!"caller wrote {opened.frames.length} frames: {opened.errors}"
  let initialized ← frameAt opened 0 "initialized" 1
  require ((← stringsField "initialized" initialized "targets") == bound.targets) "targets differ"
  let candidate ← frameAt opened 1 "candidate" 2
  -- The first row target performs a schedule member the realization binds nothing for (every
  -- timeout expiring at once), so it is skipped as prepare-rejected and the first realizable row
  -- is the candidate.
  let skipped ← match ← field "candidate" candidate "skipped" with
    | .arr rows => rows.toList.mapM fun row => do
        pure (← stringField "skipped" row "target", ← stringField "skipped" row "reason")
    | other => fail s!"skipped is not an array: {other.compress}"
  require (skipped.head?.map (·.1) == some firstTarget) s!"the first row target was not skipped: {skipped}"
  require (skipped.all fun (_, reason) => (reason.splitOn "unrealizable").length == 2) "skip reason differs"
  let limits ← field "initialized" initialized "limits"
  require ((← natField "limits" limits "steps") == 4 && (← natField "limits" limits "search") == 32768)
    "the caller budget's limits are not written out"
  require (skipped.any fun (_, reason) => (reason.splitOn "schedule-expires-expires-expires").length == 2)
    "the unbound member is not named"
  let target ← stringField "candidate" candidate "target"
  require (target.startsWith "row:" && bound.targets.contains target) s!"the candidate is not a row target: {target}"
  require (skipped.all fun (key, _) => key != target) "the candidate was also skipped"
  let identity ← stringField "candidate" candidate "candidate"
  let caseId ← stringField "candidate" candidate "caseId"
  require (caseId == "temporal.case." ++ callerSet ++ "." ++ (identity.drop 7)) s!"Case ID {caseId} is not the candidate's"
  let produced ← field "candidate" candidate "case"
  require ((← stringField "case" produced "caseId") == caseId) "the Case does not carry its identity"
  requireObject "case" produced "program"
  requireObject "case" produced "contract"
  let covers ← stringsField "candidate" candidate "covers"
  require (covers.contains target) "the candidate's covers omit its target"
  -- The same first candidate, credited satisfied and finished: identities are the enumeration's.
  let captured ← runScript boundSets [
    openFrame 1 callerSet,
    frame "next" 2 callerSet,
    ← observeRun 3 callerSet identity (satisfiedRun caseId),
    frame "finish" 4 callerSet]
  require (captured.status == 0) s!"caller campaign exited {captured.status}: {captured.errors}"
  let again ← frameAt captured 1 "candidate" 2
  require ((← stringField "candidate" again "candidate") == identity) "the first candidate's identity changed"
  let credited ← frameAt captured 2 "credited" 3
  require ((← stringField "credited" credited "observation") == "satisfied") "the Run was not satisfied"
  require ((← stringsField "credited" credited "credited") == covers) "credit is not the planned path"
  let finished ← frameAt captured 3 "finished" 4
  require ((← stringField "finished" finished "status") == "stopped") "an early finish is stopped"
  let summary ← field "finished" finished "summary"
  require ((← natField "summary" summary "covered") == covers.length) "covered differs"
  require ((← natField "summary" summary "selected") == skipped.length + 1) "selected differs"
  require ((← natField "summary" summary "unrealizable") >= skipped.length) "skipped candidates are not unrealizable"
  require ((← natField "summary" summary "attempted") == 0) "a skipped candidate counted as attempted"
  let ledger ← statusesOf "finished" finished "ledger"
  require (skipped.all fun (key, _) => ledger.contains (key, "unrealizable")) "skipped targets are not unrealizable"

/-! ### The executable -/

private def executable : IO System.FilePath := do
  let path := (← IO.currentDir) / ".lake" / "build" / "bin" / "umpire-explore"
  require (← path.pathExists) s!"bridge executable is missing at {path}"
  pure path

/-- Run the built bridge over frames on stdin, closing stdin after them. -/
private def runExecutable (input : String) : IO IO.Process.Output := do
  let path ← executable
  IO.Process.output { cmd := "sh", args := #["-c", "printf '%s\\n' \"$1\" | \"$2\"", "sh", input, path.toString] }

private def checkExecutable : IO Unit := do
  let script := "\n".intercalate [openFrame 1 callerSet, frame "next" 2 callerSet, frame "finish" 3 callerSet]
  let output ← runExecutable script
  require (output.exitCode == 0) s!"executable exited {output.exitCode}: {output.stderr}"
  let lines := (output.stdout.splitOn "\n").filter (!·.isEmpty)
  require (lines.length == 3) s!"executable wrote {lines.length} lines"
  require (output.stdout.endsWith "\n") "frames do not end with LF"
  let captured : Captured := { status := 0, frames := lines, progress := [], errors := "" }
  let _ ← frameAt captured 0 "initialized" 1
  let _ ← frameAt captured 1 "candidate" 2
  let _ ← frameAt captured 2 "finished" 3
  require ((output.stderr.splitOn "candidate sha256:").length == 2) s!"no progress line: {output.stderr}"
  require ((output.stderr.splitOn "skipped sha256:").length >= 2) s!"no skipped line: {output.stderr}"
  let closed ← runExecutable (openFrame 1 callerSet)
  require (closed.exitCode == 1) s!"stdin closing exited {closed.exitCode}"
  require ((closed.stderr.splitOn diagnosticPrefix).length == 2) s!"no diagnostic: {closed.stderr}"
  let withArguments ← IO.Process.output { cmd := (← executable).toString, args := #["--help"] }
  require (withArguments.exitCode == 2) "arguments were accepted"
  require withArguments.stdout.isEmpty "arguments wrote stdout"

def main : IO UInt32 := do
  let checks : List (String × IO Unit) := [
    ("observations", checkObservations),
    ("parsing", checkParsing),
    ("protocol", checkProtocol),
    ("credit", checkCredit),
    ("outside", checkOutside),
    ("caller", checkCaller),
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

end Temporal.Tool.ExplorationBridgeTests

def main : IO UInt32 := Temporal.Tool.ExplorationBridgeTests.main
