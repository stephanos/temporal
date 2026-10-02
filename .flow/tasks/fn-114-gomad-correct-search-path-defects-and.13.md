---
satisfies: [R9]
---
# fn-114-gomad-correct-search-path-defects-and.13 Order runtime-owned goroutines by a fixed rule and offer only user goroutines as alternatives

## Description
E4 (R9): the run-queue choice offers only user goroutines; runtime-owned goroutines are ordered by a fixed rule. Last of the three runtime edits. Depends on task 12 only to keep the overlay, patch, and fixture edits serial.

**Size:** M
**Files:** `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go`, `tools/gomad3/toolchain/runtime/go1.27.1.patch` (the `runqget` hunk, only if the call site must change), a control-probe fixture under `tools/gomad3/internal/gomadtool/conformance/testdata/`, `runtime_scheduling.go`, `tools/gomad3/SPEC.md` (`[RUNTIME.SCHEDULING]`), `.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/control-probe.md`
**Touches:** [tools/gomad3/toolchain/runtime/go1.27.1.patch, tools/gomad3/toolchain/runtime/overlay/src/**, tools/gomad3/toolchain/**, tools/gomad3/choice/**, tools/gomad3/internal/gomadtool/conformance/**, tools/gomad3/SPEC.md, .flow/artifacts/fn-114-gomad-correct-search-path-defects-and/control-probe.md]

### Approach
- First step: confirm the inferred cause from task 1. On the control probe with two user goroutines, show that the extra branching decisions have runtime-owned goroutines among their alternatives. If they do not, record E4 as changed and stop.
- Classify run-queue entries with the runtime's own system-goroutine test. Use the task 5 inventory as the list of runtime-owned kinds, and state how a finalizer or cleanup goroutine running user code is classified.
- The rule: state it once, for example runtime-owned goroutines run before user goroutines in queue order. It must be a function of the queue contents only, and must not starve either class.
- A queue with at most one user goroutine records no decision and still yields a deterministic pick. A queue with two or more user goroutines records one decision whose alternatives are the user goroutines only.
- Replay and forced prefixes apply the same rule; a tape recorded under the old rule is rejected by controller identity, never reinterpreted.
- Collector workers are picked outside the run queue. State in the contract that the rule covers the local run queue and name what it leaves out.
- Write the rule into `[RUNTIME.SCHEDULING]` in `SPEC.md`.
- Measure the control probe's decision counts before and after and retain them.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:778-803` — run-queue choice and its buffers
- `tools/gomad3/toolchain/runtime/go1.27.1.patch:572-592` — `runqget` hunk and the pick it performs
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:495-520` — decision recording, including the single-alternative path
- `tools/gomad3/SPEC.md:218-228` — scheduling and choice contract
- `docs/research/gomad/2026-10-01-feasibility-schedule-search.md:739` — the control probe and its 26 decisions

**Optional** (reference as needed):
- `tools/gomad3/internal/gomadtool/conformance/testdata/runqueue/` — existing run-queue fixture
- `tools/gomad3/toolchain/runtime/go1.27.1.patch:355-362`, `:478-482` — existing uses of the system-goroutine test
- `tools/gomad3/choice/trace.go:100` — controller implementation identity

### Key context
- This changes schedules for every seed. It is a new choice-controller identity, and existing dispositions must not be weakened to obtain a pass in task 14.
- The run-queue choice runs on the system stack with fixed buffers. Classification must not allocate.
- fn-110 task 2 moves the scheduler implementations into the overlay; fn-112 tasks 3 and 5 edit the same file. Check their state first and rebase onto whichever landed.
## Acceptance
- [ ] The cause is confirmed on the control probe, or E4 is recorded as changed with the evidence
- [ ] A program with two user goroutines records only decisions whose alternatives are user goroutines
- [ ] A queue with zero or one user goroutine records no decision and the pick is equal across runs
- [ ] A fixture with a busy runtime-owned goroutine and two user goroutines shows neither class is starved
- [ ] A tape recorded before the change is rejected by identity
- [ ] `[RUNTIME.SCHEDULING]` states the rule for runtime-owned goroutines and what it leaves out
- [ ] Control-probe decision counts before and after are retained in `control-probe.md` in the spec's artifacts directory
- [ ] `make -C tools/gomad3 validate test-toolchain test-runtime overlay-test` pass on darwin/arm64; linux status recorded
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
