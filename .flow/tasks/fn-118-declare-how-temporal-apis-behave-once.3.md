---
satisfies: [R5]
---
# fn-118-declare-how-temporal-apis-behave-once.3 Make Testpilot wait by condition within a declared bound and report the bound

## Description
Behavior phase, runtime half. Testpilot knows nothing about Models, so this is independent of task 2 and may run in parallel: the Testpilot IR gains whatever task 1 decided it needs to carry a declared bound and its source position, and the Go framework waits by condition within that bound with no default of its own.

**Cross-spec entry gate:** same as task 2 (fn-112.10 done and fn-114 closed). fn-119.1 also edits the Temporal Driver; whichever lands second rebases.

**Size:** M
**Files:** `proto/internal/temporal/server/api/testpilot/v1/{instruction,run,case}.proto` and generated Go; `common/testing/testpilot/internal/execution/{evidence,scheduler}.go`; `temporal/server/session.go`; `common/testing/testpilot/profile.go` and `contract/profile.go` (scale factor); `temporal/profile.go`.
**Touches:** [proto/internal/temporal/server/api/testpilot/v1/**, api/testpilot/v1/**, common/testing/testpilot/**]

### Approach
- A hinted wait carries its own interval/bound and hint position (or a provenance reference, per task 1); preparation rejects a hinted wait missing its bound instead of falling back to `InstructionDefaults` (`profile.go:84`). Unhinted instructions keep today's defaults.
- On expiry, the failure names the condition, the declared bound and the hint's Scala position (extend `rpcFailure` in `session.go`; Run `InstructionOutcome` `run.proto:67`).
- Add a read-once form of `ReadEvidence`: one read, condition checked once, no poll interval (today preparation requires a positive interval <= timeout, `evidence.go:277-281`, and `PollRPC` loops until the context expires, `session.go:134-155`). The lowering emits it for a read after a write visible at once; evidence recording stays the same as a poll's so Contracts do not change.
- Add an optional Profile bound scale factor (`contract/profile.go:78-95` `Resolve`). It scales the declared bounds and the Profile's duration ceilings together (`session.go:207` caps instruction timeouts at `max(MaxTotalDuration, MaxCleanupDuration)`); preparation refuses a scaled bound above the scaled ceiling, naming both, so a reported bound is always the one applied. A scaled Run records the factor.
- No `time.Sleep`; poll with timers/conditions as `session.go:130-160` does.

### Investigation targets
**Required:**
- `common/testing/testpilot/temporal/server/session.go:120-215`
- `common/testing/testpilot/internal/execution/evidence.go:270-290`, `scheduler.go:640-660,840-850`
- `common/testing/testpilot/contract/profile.go:70-100`
**Optional:**
- `proto/internal/temporal/server/api/testpilot/v1/run.proto`

### Quick commands
```bash
make proto
go test -count=1 -tags test_dep ./common/testing/testpilot/...
```

### Execution constraints
- Existing Cases (no hint fields) prepare and run exactly as before; no change to Contract evaluation.
## Acceptance
- [ ] A hinted wait polls by condition within its declared bound; preparation rejects a hinted wait without a bound; no Go default applies to it.
- [ ] A read-once `ReadEvidence` form exists and records evidence like a poll.
- [ ] An expired wait fails naming condition, bound and hint position; a Profile scale factor is applied and recorded in the Run.
- [ ] Existing Cases behave identically; Testpilot tests and `make lint-code-fast` pass.
## Done summary
Runtime half of R5: Testpilot waits by condition within a declared bound, reads once, and reports an expired bound.

What changed
- IR: `InstructionNode.wait_hints` (`WaitHint{hint_id, SourceLocation source, at_most_milliseconds}`), `ReadEvidence.once`, `Run.bound_scale_percent`. `SourceLocation` moved to a new `source.proto` so instruction.proto can use it; the full name and wire form are unchanged. Regenerated with this worktree's linux `.bin` tools (`make protoc`); only `api/testpilot/v1` changed.
- Preparation: wait hints are accepted only on a polling ReadEvidence that writes its own timeout. That timeout must equal the sum of the hints' bounds, so no Profile default applies; each hint needs a valid id, a source path and line, and a positive bound. A read once needs interval 0.
- Driver contract: a zero interval reads once. A false condition is recorded as a poll whose timeout ran out (TIMED_OUT, no evidence), so Contracts are unchanged. Implemented in the server Session, the testsupport fake and the testcore fake.
- Expiry report: built in the scheduler, so it works for every Driver. On a hinted or once read that times out, Detail names the evidence, a deterministic rendering of `until`, the applied bound, each hint as `id (path:line) ms`, and the scale when it is set. Example: `evidence itemSettled: until value.state > 1 did not hold within 75 ms (declared 50 ms, scaled by 150%); hints: visibility.describe (model/Behavior.scala:12) 30 ms, ...`.
- Bound scale: `ProfileSpec.BoundScale` / `temporal.Environment.BoundScale` is an integer percent; 0 means 100, and anything below 100 is refused. It scales hinted bounds and the two duration ceilings together, at the preparation check for a hinted bound, the Session effect cap, the worker callback deadline and the Run's own ceilings. A scaled Run records it, and the binding fingerprint covers it, appended only when the Profile is scaled. Unhinted timeouts and carried durations are admitted against the declared ceilings, so whether a Case admits never depends on the scale.
- The proto change rotated the Driver catalog identity (760017bd -> d1108efd). The pinned control and canary Runs and the receipt goldens were re-recorded live (`make umpire-rerecord-pinned-runs`).

Decisions, and why
- Gate: fn-112 is closed. fn-114's remaining tasks are Scala-model cleanup this Testpilot-only task does not touch, so the conductor started it before fn-114 closed.
- Expiry text lives in the scheduler, not `rpcFailure`: the scheduler synthesizes TIMED_OUT itself when the operation context expires first, so only it sees every expiry. Unhinted, non-once outcomes keep their exact bytes, because Detail is readable as an outcome field.
- The scale is a percent integer, not a float, so Run and fingerprint bytes are deterministic.
- `at_most_milliseconds` keeps the settled spelling, with an api-linter preposition exemption, rather than being renamed.
- Duplicate hint ids are allowed, because a wait may sum the same cause twice (activity-retry).

Review: claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`. Writer and reviewer are the same family (Opus). Round 1 was NEEDS_WORK with three findings, all fixed: source.proto was missing from the protocol check, Lean inputs and authority scan; admission depended on the scale (fixed by option A, declared ceilings); and the fakes refused a zero interval. Round 2 was SHIP.

Existing Cases: no Case carries the new fields, and the lowering, goldens and Case bytes are untouched. Live generated suite: 16 passed and 4 failed. The 4 failures (activity-terminate, activity-pauseResume, activityProtocol.{terminateSettles,cancelIsRequested}) are the known INCONCLUSIVE after the 10 s stop-worker limit, the ShutdownWorker race. The bound report does not cover them: stop-worker is an unhinted InjectFault on the Profile default, so its outcome stays as before. Extending the report to every own-deadline expiry would name that bound.

For the owner / later
- `lint-api` still flags the pre-existing core::0203::optional in umpire/v1/ir.proto:573 (from fn-122.4).
- The canary `iterationBound` (tools/canary/controller/run.go:308) uses unscaled ceilings. Nothing sets a scale there today.
- Nexus schedule-to-close is still derived from the unscaled Profile default (worker/typed.go:68), as task .1 flagged.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e66e54eabd, 0acff41d0e, e352847ffe, 97616e46a2
- Tests: make protoc (exit 0), make umpire-rerecord-pinned-runs (exit 0, 337 s, live), go test -json -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0, 46 packages), go test -tags test_dep -count=1 ./common/testing/testpilot/... after review fixes (exit 0), make umpire-check-testpilot-protocol (exit 0), make lint-protos lint-api (only pre-existing ir.proto:573 finding), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0), go test -tags 'test_dep integration' ./tests -run TestTestpilotGeneratedCases (16 pass; 4 known ShutdownWorker INCONCLUSIVE)
- PRs: