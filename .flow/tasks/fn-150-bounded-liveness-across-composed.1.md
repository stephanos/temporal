---
satisfies: [R1, R2, R4]
---
# fn-150-bounded-liveness-across-composed.1 Admit composition progress and prove reuse of the checker

## Description
Admit composition progress and prove reuse of the checker. Advances R1, R2, R4 of the parent spec.

**Size:** M
**Files:** `tools/umpire/ir/validate.go`, `tools/umpire/ir/admission_test.go`, `tools/umpire/check/claims.go`, `tools/umpire/check/checking.go`, `tools/umpire/check/composed_progress_test.go`
**Touches:** [tools/umpire/ir/validate.go, tools/umpire/ir/admission_test.go, tools/umpire/check/claims.go, tools/umpire/check/checking.go, tools/umpire/check/composed_progress_test.go]

### Approach
- Extend Progress owner admission from machines to the existing machine-or-composition subject and validate its source/destination predicates against the resolved state shape using existing validator support.
- Drive the existing binding.progress and CheckProgress over a hand-authored two-member composition before changing the Scala surface. Use inherited assumptions only for this proof; explicitly reject claim fairness references that cannot yet resolve for a composition.
- Honor composition subject errors/unsupported flags just as Queries do, including monitored-member compositions; do not accidentally claim monitor support.
- Prove one synchronized step advances both members once, all declared starts feed reachability, and a changed member causes a counterexample. Reuse checking-only bindings; realization metadata must not prevent a valid model check.
- Record pre/post results for existing machine progress and composition safety. If a second checker algorithm is needed, stop and reconsider the binding approach before later tasks.

### Investigation targets
**Required:**
- `tools/umpire/ir/validate.go:1137` - Progress admission.
- `tools/umpire/check/claims.go:966` - generic subject binding.
- `tools/umpire/check/compose.go:44` - all starts and composed state decoding.
- `tools/umpire/check/compose.go:118` - unsupported monitor flag.
- `tools/umpire/check/checking.go:675` - receipts/replay.

### Key context
Re-anchor paths and interfaces against completed fn-140/fn-141 and the approved schema/package moves before editing. Keep this work outside the activity batch and serialize shared regeneration with it and the schema chain; no new spec-close dependency is implied.

### Quick commands
```bash
go test -tags test_dep ./tools/umpire/ir ./tools/umpire/check -run 'Test.*(Progress|Composed|Admission)'
```

## Acceptance
- [ ] Hand-authored composition Progress yields a replayed positive and negative result with the existing engine.
- [ ] Invalid owner/predicate shape/bound and unsupported monitored-member cases retain attributed errors or unsupported results.
- [ ] Synchronized step counting and multiple starts are tested; existing machine progress and composition safety pins remain unchanged.

## Done summary
Blocked:
# fn-150.1 implementation preserved; canonical verification blocked

The worker committed the complete task-local hand-authored composition-progress implementation at `2d643875956c1f90efffb0c0a76cffe2010634db` on `fn-150-bounded-liveness-across-composed`, based on integrated `2d39120b3e`. It changes only the declared reader/binder and admission/composed-progress tests. The checkpoint remains unintegrated and unreviewed; task acceptance and downstream dependencies remain intact.

Focused development checks passed 19 reader and 10 checker top-level tests with new synchronization, all-start, error/type/monitor/fairness and witness-replay controls. Existing targeted machine progress and composition safety pins passed before and after the edit. Changed-package lint, depguard and errortype vet passed. These checks provide scoped evidence, not canonical Quick or full-suite credit.

The original Quick is `go test -tags test_dep ./tools/umpire/ir ./tools/umpire/check -run 'Test.*(Progress|Composed|Admission)'`. It failed before edits with `check` killed after 20.489 seconds and failed once after edits under the shared lock/default memory with `check` killed after 16.403 seconds (outer wall 16 seconds). The reader passed both times. The task worktree's `.flow/tmp/fn150/task1/` retains baseline, pins, TDD, post-edit Quick, lint and handover evidence. An intervening memory-tuned attempt was cancelled and earns no passing credit; runtime-memory tuning is excluded from the accepted strategy.

Root keeps the task blocked rather than dispatch implementation review on a RED tree, changing the original Quick selection or marking it done. Canonical verification resumes after resource provisioning or fn-157's measured preservation-safe repair supplies adequate capacity. Run the original Quick with default memory, preserve its original RED results, independently review the integrated diff, and verify task acceptance before completion. Later task .2 remains held. All worker commands are terminal, and the committed workspace is clean.

Blocked:
# fn-150.1 implementation preserved; canonical verification blocked

The worker committed the complete task-local hand-authored composition-progress implementation at `2d643875956c1f90efffb0c0a76cffe2010634db` on `fn-150-bounded-liveness-across-composed`, based on integrated `2d39120b3e`. It changes only the declared reader/binder and admission/composed-progress tests. The checkpoint remains unintegrated and unreviewed; task acceptance and downstream dependencies remain intact.

Focused development checks passed 19 reader and 10 checker top-level tests with new synchronization, all-start, error/type/monitor/fairness and witness-replay controls. Existing targeted machine progress and composition safety pins passed before and after the edit. Changed-package lint, depguard and errortype vet passed. These checks provide scoped evidence, not canonical Quick or full-suite credit.

The original Quick is `go test -tags test_dep ./tools/umpire/ir ./tools/umpire/check -run 'Test.*(Progress|Composed|Admission)'`. It failed before edits with `check` killed after 20.489 seconds and failed once after edits under the shared lock/default memory with `check` killed after 16.403 seconds (outer wall 16 seconds). The reader passed both times. The task worktree's `.flow/tmp/fn150/task1/` retains baseline, pins, TDD, post-edit Quick, lint and handover evidence. An intervening memory-tuned attempt was cancelled and earns no passing credit; runtime-memory tuning is excluded from the accepted strategy.

One unchanged Quick retry after the private fn-155 Bloop daemon's normal exit also failed naturally: exit 1, outer wall 39.102 seconds, checker package killed after 36.698 seconds, reader cached pass. It ran under an exclusive `fcntl.flock` on `/tmp/umpire-heavy-gates.lock`, with no observation timebox, concurrency change or GOGC/GOMEMLIMIT/GOMAXPROCS/GOFLAGS/GOTOOLCHAIN override. Before the command the host had 7,773,312 KiB MemAvailable, 16,593,792 KiB total RAM and no swap; the private daemon was absent and shared daemon remained untouched. Minimum sampled MemAvailable was 319,632 KiB and peak sampled process-tree RSS was 7,358,112 KiB. Both kernel captures exited 0; the new kernel delta records global OOM of `check.test` at 2026-10-10T07:39:21Z with 7,527,312 KiB anonymous RSS. Kernel PID 28894 and tool-namespace checker PID 2456714 are separate observations; their numeric namespace mapping is unproven.

The terminal retry handover and exact receipt are preserved in the task worktree under `.flow/tmp/fn150/task1/revisit-private-exit/`: `handover-summary.md`, `handover-evidence.json`, `result.json`, `quick.log`, samples and complete kernel-before/after/delta logs. Root verified `result.json` SHA256 `6429e5a47cb061700bbf07c52be44efd5204fd5d61343df95ace4b0c7ad564a5` and `handover-evidence.json` SHA256 `a565409baca84a2002698fa6ffa57cac078cce372883dbf8443ed983faa38e7e`. Executable inputs from the pinned base to integrated `de916cc2af` were unchanged: only Flow metadata and MILESTONES differed. No source, lifecycle or commit was changed by the verification worker, and all owned commands are terminal.

Root keeps the task blocked rather than dispatch implementation review on a RED tree, changing the original Quick selection or marking it done. Canonical verification resumes after resource provisioning or fn-157's measured preservation-safe repair supplies adequate capacity. A disk-only improvement or an unchanged environment does not justify repeating this OOM attempt. Run the original Quick with default memory after a material capacity change, preserve all original RED results, independently review the integrated diff, and verify task acceptance before completion. Later task .2 remains held. The committed task workspace is clean.
## Evidence
- Commits:
- Tests:
- PRs:
