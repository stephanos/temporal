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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
