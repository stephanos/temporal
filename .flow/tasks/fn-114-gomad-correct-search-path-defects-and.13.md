---
satisfies: [R9]
---
# fn-114-gomad-correct-search-path-defects-and.13 Order runtime-owned goroutines by a fixed rule and offer only user goroutines as alternatives

## Description

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implemented scheduler/search behavior, current-source Darwin runtime/full/core/smoke/representative exact replay, measurements, docs and review. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

E4 (R9): the run-queue choice offers only user goroutines; runtime-owned goroutines are ordered by a fixed rule. Last of the three runtime edits. Depends on task 12 only to keep the overlay, patch, and fixture edits serial.

**Size:** M
**Files:** `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go`, `tools/gomad3/toolchain/runtime/go1.27.1.patch` (the `runqget` hunk, only if the call site must change), a control-probe fixture under `tools/gomad3/internal/gomadtool/conformance/testdata/`, `runtime_scheduling.go`, `tools/gomad3/SPEC.md` (`[RUNTIME.SCHEDULING]`), `.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/control-probe.md`
**Touches:** [tools/gomad3/toolchain/runtime/go1.27.1.patch, tools/gomad3/toolchain/runtime/overlay/src/**, tools/gomad3/toolchain/**, tools/gomad3/choice/**, tools/gomad3/internal/gomadtool/conformance/**, tools/gomad3/SPEC.md, .flow/artifacts/fn-114-gomad-correct-search-path-defects-and/control-probe.md]

### Approach
- First step: confirm the inferred cause on a fixture that deliberately starts two user goroutines. Show that the extra branching decisions have runtime-owned goroutines among their alternatives. The historical D21 control source starts no goroutine explicitly; its reported peak of 2 is not a substitute for this fixture. If the cause is absent, record the narrowed E4 finding with evidence and stop.
- Classify run-queue entries with the runtime's own system-goroutine test. Use the task 5 inventory as the list of runtime-owned kinds, and state how a finalizer or cleanup goroutine running user code is classified.
- The rule: state it once, for example runtime-owned goroutines run before user goroutines in queue order. It must be a function of the queue contents only, and must not starve either class.
- A queue with at most one user goroutine records no decision and still yields a deterministic pick. A user dispatch with two or more user goroutines records one decision whose alternatives are the user goroutines only; a runtime-owned queue head runs deterministically without recording a decision.
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
- Task 1 re-anchor (2026-10-02): E4 is changed in its historical probe premise. `.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d21-probe-control.go.txt` has no `go` statement. Its seed-11 report still records 26 branching Runnable decisions and peak goroutines 2 (seed 17 records 29). `gomadChoiceRunqIndex` still includes every local run-queue identity without a system-goroutine filter, but the reported peak does not establish two deliberately started user goroutines or attribute each extra decision. Keep the explicit two-user fixture, cause check, fairness, and before/after acceptance; no runtime correction was made in task 1.
- This changes schedules for every seed. It is a new choice-controller identity, and existing dispositions must not be weakened to obtain a pass in task 14.
- The run-queue choice runs on the system stack with fixed buffers. Classification must not allocate.
- fn-110 task 2 moves the scheduler implementations into the overlay; fn-112 tasks 3 and 5 edit the same file. Check their state first and rebase onto whichever landed.
## Acceptance

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] The cause is confirmed on the control probe, or E4 is recorded as changed with the evidence
- [ ] A program with two user goroutines records only decisions whose alternatives are user goroutines
- [ ] A queue with zero or one user goroutine records no decision and the pick is equal across runs
- [ ] A fixture with a busy runtime-owned goroutine and two user goroutines shows neither class is starved
- [ ] A tape recorded before the change is rejected by identity
- [ ] `[RUNTIME.SCHEDULING]` states the rule for runtime-owned goroutines and what it leaves out
- [ ] Control-probe decision counts before and after are retained in `control-probe.md` in the spec's artifacts directory
- [ ] `make -C tools/gomad3 validate test-toolchain test-runtime overlay-test` pass on darwin/arm64; linux status recorded
## Done summary
Runtime-owned local-queue heads dispatch in queue order without a decision; user heads choose only user identities. The two-user cause probe confirms the old controller selected a runtime helper in31 branching decisions across32 seeds. A same-source comparison reduces2261 runnable decisions to2080 and runtime selections31 to0. Zero/one-user, all-alternative-set, busy-runtime mutual-progress, exact replay/prefix and legacy-controller rejection fixtures cover the rule. Finalizer/cleanup callback classification uses isSystemGoroutine; SPEC names the local-queue boundary and excluded channels.

Native Darwin validate, test-toolchain, test-runtime and all nine overlay packages pass on build key4a6e5b695ea538f0a56eb70874ff945693223b53d0fcee56cc555f89e1a9ac0e. The final strengthened fixture has supplemental focused evidence. The integrated full host gate passes all45 packages. Runner fixture repairs preserve strict tape assertions, refresh only seven controller-derived hashes in the traced diagnostics baseline, and pin two-outcome search exhaustion in two executions. The100-job positive retention characterization has a one-minute bound after a retained96-of100 ten-second timeout under broad-suite contention; its correctness assertions are unchanged. See control-probe.md, runner-fixture-repair.md, source bindings and logs in this directory.

Independent codex review returned SHIP after a pin-impact input guard outside this runtime task was corrected. Native Linux remains unverified. Root lint cannot load nested-module paths; scoped vet, gofmt and diff checks pass. No implementation commits or pushes were made; the user owns commits. Fn-114 task14 and R12 qualification remain open.
## Evidence
- Commits:
- Tests: make -C tools/gomad3 validate test-toolchain test-runtime overlay-test, GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host, go -C tools/gomad3 vet -tags test_dep ./runner ./runner/internal/execution, TestRuntimeOwnedControlProbe and TestRuntimeOwnedRejectsPreviousController
- PRs:

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implemented scheduler/search behavior, current-source Darwin runtime/full/core/smoke/representative exact replay, measurements, docs and review. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
