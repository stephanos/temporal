# fn-228.3 independent review: 0c7822bc43..6b137fb3fb

Verdict: **NEEDS_WORK**

Reviewed: 8118630a45 (runner package, umpire.run.v1) and 6b137fb3fb (umpire command, old commands deleted).
Yardstick: spec "Execution contracts resolved by the backlog review" plus R3, task .3, handoff decisions.

## What I verified
- `CGO_ENABLED=0 go test -count=1 ./umpire/runner/` ok; `CGO_ENABLED=1 go test -race -count=3` ok (26.6 s).
- `go vet` clean on `umpire/runner`, `tools/umpire/cmd/umpire` and `-tags 'test_dep integration' ./tests`.
- `model/scripts/proto-go.sh --check` rc 0 (run.pb.go is fresh).
- No Temporal names, Case vocabulary, profiles/receipts/provenance or new deps in the runner or run.proto.
- The loop serializes every Host call on one goroutine in receive order; Done before acceptance,
  premature close, crossed/duplicate/unknown events, blocked/canceled Respond and hung Close are
  all DriverFaults with tests. Publication is exclusive (hard link of a synced temp file) and the
  concurrent, collision, corruption and two-distinct-repeat tests exist and pass.
- I probed replay, budgets and Reset refusal with throwaway tests in a scratch copy of the module
  (not in the repo).

## Findings

1. **blocker: deleting the commands breaks the live CI gate.**
   tests/testpilot_umpire_run_test.go:136-151, tests/testpilot_nexus_control_case_test.go:110-112,
   tests/testpilot_assess_test.go:54-55.
   `make umpire-check-live-tests` runs in .github/workflows/umpire.yml:36. It runs
   `./tests -run '^TestTestpilot'` and requires an empty failure set. Three tests in that selection
   build the deleted commands with `go build go.temporal.io/server/tools/umpire/cmd/<name>`:
   `TestTestpilotUmpireRunRunsACheckedInCaseAgainstAnyEndpoint` and
   `...RejectsAnUnreachableEndpoint` build umpire-run.
   `TestTestpilotNexusControlReplaysThroughTheCommand` builds umpire-run and umpire-replay.
   `TestTestpilotAssessRecordedRuns` builds umpire-run and umpire-assess.
   I confirmed that the build now fails ("no required module provides package .../umpire-run").
   The handoff's model-check run does not cover this gate, and the handoff does not mention these
   tests. The spec makes .6 enumerate dependents and map their coverage before anything is
   deleted. Here, deleting the commands silently removed live coverage instead.
   Fix: in this task, either retire those tests with an explicit coverage note handed to .6, or
   keep the old command directories until .6. Then re-run `umpire-check-live-tests`, or at least
   state in the handoff that it is red. Stale prose remains in
   common/testing/testpilot/README.md:527,560 and tools/umpire/conformance/conformance.go:71.

2. **major: replay accepts a truncated Run with a rewritten verdict, and protocol calls after the terminal event.**
   replay.go:40-44 with session.go:104-115 (`verdictOf` trusts the recorded termination), and
   replay.go:133-170 (`check` has no rule about events after a terminal event).
   Scenario A (verified): take the buggy Run (Unexplained at event 5), keep events[:5], and set
   the verdict to Explained/EVENT_BUDGET. Replay returns a matching Explained verdict, so an
   Unexplained Run is laundered into an Explained one. Any prefix also passes as
   PROPOSAL_BUDGET or QUESTION, including cut points where a live Run could not stop with that
   termination.
   Scenario B (verified): insert a DriverFault before the Unexplained Judge, renumber, and set
   Refused at that index. Replay copies the fault, re-executes the Judge after it and accepts the
   Run.
   The contract says to reject missing events and tampered responses or disposition.
   Fix: (a) reject any protocol-call event after the first terminal event; in a live Run only
   teardown faults can follow it. (b) Record the limit that ended an Explained Run (proposal
   budget, event budget) in `Explained` and check it on replay: the proposal count must equal
   the budget, and the event count must reach the event budget. QUESTION should be accepted only
   from the question mode's shape (Reset or a prefix, then exactly one Propose or Judge). (c) Add
   these tamper cases to `TestReplayRejectsAlteredRuns`.

3. **minor: the event budget is overshot by one.**
   session.go:225-231 and :298.
   `eventsSpent` is checked before Propose and after each served event, but not after Propose
   itself. Verified: EventBudget=2 records 3 events, and EventBudget=4 records 5. Options
   documents it as "caps recorded events per Run".
   Fix: check `eventsSpent()` again after the Propose call (ending EVENT_BUDGET before `Start`),
   or document the budget as a soft cap.

4. **minor: a Run whose Reset was refused cannot be replayed.**
   replay.go:141.
   Verified: exploring with a world the guest refuses records `[refusal:UNSUPPORTED]`, Refused.
   Replaying it gives "replay: event 0: a Run begins with its one Reset" instead of reproducing
   the Reset refusal.
   Fix: accept a METHOD_RESET Refusal at position 0, with the world check applied to its decoded
   request.

5. **minor: reduce exits 0 when nothing reproduces, and admits the recorded Run only by model hash.**
   runner.go:208-238.
   When no candidate is Unexplained, reduce returns the last candidate's own verdict (normally
   Explained). The command then prints "explained: …" and exits 0, which a script cannot tell
   apart from success. The recorded Run's protocol version is not checked, and neither is whether
   its proposal seeds follow `ProposalSeed`.
   Fix: return a Refused Run with "not reproduced within n candidates", or a distinct exit code,
   and reuse `r.admit` (or a replay) on the recorded Run first.

6. **minor: `Driver.Start` is synchronous, unbounded by the read timeout, and not documented as non-blocking.**
   driver.go:14-16, session.go:250.
   The contract says the Runner "never waits for Start/RPC completion before answering a worker".
   With this interface, that holds only if `Start` returns before the RPC completes. Suppose a
   Driver blocks in `Start` until an activity RPC finishes, and that RPC needs a worker callback.
   The Run then deadlocks until RunTimeout (5 min) rather than EventTimeout.
   Fix: state in the `Start` doc that it must return once the request is sent, and bound it with
   `context.WithTimeout(ctx, EventTimeout)`.

7. **minor: two deferrals are not recorded where the next task will see them.**
   .flow/tasks/fn-228-runner-and-run-the-one-go-host-loop.6.md.
   Decision 11 (canary reduced to lease + policy) and decision 9 (testpilot/publish duplicated; 8
   importers remain in tools/canary, tools/umpire/internal/cli, testpilot replay/evaluation) are
   deferred to .6, but the .6 spec mentions neither.
   Fix: add both to the .6 description via flowctl so they are not dropped.

8. **nit: proto-go `--exclude=runner` matches by basename anywhere.**
   model/scripts/proto-go.sh:33.
   A future generated package or file named `runner` would silently escape the freshness diff.
   The hand-written package also lives inside the generated tree.
   Acceptable; consider `--exclude` on a more specific marker, or note it in proto/README.

9. **nit: a guest trap loses the partial Run.**
   session.go:120-127, 283-286.
   A Host failure that is not a refusal returns an error and records nothing (exit 3). The
   partial Run would be useful evidence of a Model or runtime defect.
   Consider publishing the prefix with a Refused "host failure" verdict. Decision 3 is defensible
   as is.

## Decisions made on the owner's behalf
1. Binary in the server module with no Driver until .4: consistent (staging).
2. Exit 3 for "could not run": a reasonable extension of the spec's 0/1/2. Record it in the
   spec's API Contracts.
3. A Host trap aborts with no verdict: defensible (see nit 9).
4. `Run` is an alias, and the deciding Run is the first Unexplained, then the first Refused, then
   the last: consistent.
5. splitmix64 proposal seeds, with reduce trying proposal budgets 1..n: consistent. Budget k
   replays exactly the first k recorded proposal seeds, and the candidate count is bounded by the
   original proposal count. The non-reproduction outcome is weak (finding 5).
6. Annotate as a Driver event: consistent. It is served by the one Host and recorded.
7. Budgets end Explained with a termination, and a run-time expiry is a DriverFault: consistent
   with "Explained budget-limited … record the budget termination reason". However, replay
   cannot verify the termination (finding 2).
8. `question` without `--run` Resets with `--world`: acceptable. It equals a one-event Reset
   prefix and cannot reach an arbitrary state.
9. Publish duplicated in runner: forced by the module boundary. Removing testpilot/publish must
   be tracked (finding 7).
10. Deleting the old commands now: violates the gate (finding 1).
11. Canary deferred to .6: acceptable only if recorded (finding 7).
12. proto-go `--exclude=runner`: acceptable (nit 8).
13. Redaction as a hook that strips in place and marks the event: consistent. It runs after
    judgment and before publication, and replay refuses a redacted Run.
