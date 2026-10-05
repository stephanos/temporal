---
satisfies: [R2, R6]
---
# fn-110-gomad-minimize-the-runtime-patch.2 Move the three scheduler implementations into the runtime overlay

## Description

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Representation/overlay implementation, byte equivalence, pinned checks, preservation, review, Darwin gates and unmet R8 size reduction. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Move the three scheduler implementations out of the `src/runtime/proc.go` section of the patch into the existing runtime overlay file, leaving hooks in upstream code. Split from the crypto/syscall relocation because this is the behavior-sensitive part: lock state, draw order, and host-timing decisions must not move. The patch stays at three context lines here (the regenerator is unchanged until task 4), which gives the intermediate `-U3` measurement.

**Size:** M
**Files:** `tools/gomad3/toolchain/runtime/go1.27.1.patch`, `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go`, conformance fixture files only if coverage is missing, `docs/research/gomad/GOMAD_PATCH_SIZE.md`
**Touches:** [tools/gomad3/toolchain/runtime/go1.27.1.patch, tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go, tools/gomad3/internal/gomadtool/conformance/**, docs/research/gomad/GOMAD_PATCH_SIZE.md, tools/gomad3/choice/internal/wire/wire_generated.go, tools/gomad3/target/internal/livecap/protocol_generated.go, tools/gomad3/toolchain/runtime/overlay/src/cmd/internal/gomadcap/protocol_generated.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated.go, tools/gomad3/runner/testdata/diagnostic-identity-choices.json, .flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task-2/gfield-compact-20261005/**]

### Approach
Hunk locations are line numbers in the baseline patch; re-find them by `@@` header if they shifted.
1. **`gomadSimulationTimeQuiescenceChanged`** — patch lines 459-493 (`@@ -6519,6 +6657,34 @@ func checkdead()`). Move the whole function verbatim into `overlay/src/runtime/gomad.go`; the call inside `checkdead` stays. Place it near `gomadSimulationTimeQuiesce` (`gomad.go:1068`).
2. **`checkdead` quiescence and time-advance selection** — the body added inside `if faketime != 0 {` at patch lines 380-447 (`@@ -6479,12 +6561,64 @@`). Move it behind one private helper that reports "wake a timer" or "wait" (the investigation used `wakeTimer, waiting := gomadCheckDeadTime()`; the name is guidance). Keep in `proc.go`: the transport-syscall term in `run` (patch 336-344), the `forEachG` transport/runnable handling (345-379), the stock `pidleget`/`mget` timer-wake path, and its two `gomadSimulationTimeEnabled` guards (448-458).
3. **`exitsyscallNoP` idle-P resumption** — the `if gomadEnabled { … }` block at patch lines 302-324 (`@@ -5139,6 +5189,22 @@`). Move it behind a helper called inside the existing activation guard (investigation: `gomadResumeSyscall(gp, pp)`). Keep in `proc.go`: the quiescing guard on `pidleget` and the `gomadArrivals.pushBack`/`globrunqput` choice (patch 279-301).

Preservation contract for the helpers:
- `checkdead` is entered with `sched.lock` held. Every normal return from the helper holds the lock; the round-trip keeps its unlock → `gomadSimulationTimeQuiesce(when)` → lock order and the `gomadSimulationTimeQuiescing` set/clear around it; both fatal paths unlock first.
- Keep evaluation order: external-request check, awaiting-external check, `timeSleepUntil()`, quiescing check, round-trip, the arrival recheck that starts an idle P, then the response switch. Keep `wakeTimer = timer && when <= current` and the non-simulation branch `when < maxWhen || gomadEnabled && timer` with its `netpollAnyWaiters` guard; disabled mode must reduce to upstream behavior.
- `exitsyscallNoP` is `//go:nowritebarrierrec` and runs on g0 via `mcall`; the helper is checked transitively. Keep `pushBack` before `acquirep`, `locked` read under `sched.lock`, and the `stoplockedm`/`execute`/`schedule` calls that never return.
- No new allocations, closures, or host reads; the only closure is the existing `forEachG` one, moved verbatim.

Behavior pin (this is a refactor; "tests pass" alone is not the pin):
- For seeds 0, 1, 7, 42 and disabled mode, run the timer/scheduler fixtures (`clock`, `clock_io`, `clock_tick`, `clock_deadlock`, `clock_synctest`, `io_net_races`, `scheduler`, `runqueue`, `select`, `preemption` under `tools/gomad3/internal/gomadtool/conformance/testdata/`) on the baseline toolchain and on the candidate; target-visible output and result classification must match. Caller-site values are text offsets and may differ between builds, so replay is verified within each build, not across them.
- Same-seed repeatability and exact replay within the candidate: `make -C tools/gomad3 test-runtime test-upstream test-live-capability`, plus `TestRootProcessSimulationUsesRunnerTransport` (clock synchronization, crash drain, exact replay).
- Planning found no `LockOSThread` use in the conformance fixtures or `tools/gomad3sim`. Locked-goroutine syscall resumption therefore needs new regression coverage; add the smallest fixture following an existing one's registration, and show it passes on the baseline toolchain before the extraction.

Regenerate through the governed workflow, then record the new `-U3` bytes/lines and `proc.go` added/deleted counts beside the baseline (investigation estimate: about 3,059 bytes and 95 lines saved).

### Investigation targets
**Required:**
- `tools/gomad3/toolchain/runtime/go1.27.1.patch:123-589` — the `proc.go` section
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:20-70,905-930,1060-1110` — quiescing state, response kinds, arrival queue, quiesce transport
- `docs/research/gomad/GOMAD_PATCH_SIZE.md` — "Scheduler extraction" and "Changes that need to remain"
- `tools/gomad3/runner/internal/execution/simulation_root_integration_test.go` — process-simulation entrypoint
- `tools/gomad3/internal/gomadtool/conformance/runtime_repeatability.go`, `driver.go` — fixture registration

**Optional:**
- `tools/gomad3sim/cluster_toolchain_test.go:177-400` — clock, crash-drain, digest tests

### Quick commands
```bash
make -C tools/gomad3 generate validate
df -h . && make -C tools/gomad3 toolchain
make -C tools/gomad3 test-toolchain test-runtime test-upstream test-live-capability
(cd tools/gomad3 && GOWORK=off .toolchain/bin/go test -tags test_dep,integration -count=1 -run TestRootProcessSimulationUsesRunnerTransport ./runner/internal/execution)
wc -c -l tools/gomad3/toolchain/runtime/go1.27.1.patch
```

### Key context

**Working constraints (apply to every fn-110 task):**
- Commit verified progress in bounded batches, as requested by the user. The conductor owns staging and commits; preserve unrelated changes and leave active shared-source edits unstaged until their verification boundary. Do not push, stash, create worktrees or rewrite history without separate authorization.
- Recheck the actual host before gates. This development session is `linux/arm64`; neither qualified native `darwin/arm64` nor native `linux/amd64` execution is available here. Keep source-owned Darwin gates incomplete until source-bound native evidence exists; Linux execution belongs to fn-128.1/.4/.7; cross-compilation, emulation and developmental stock-host checks qualify neither.
- Disk: about 19 GB was free at planning time and each toolchain build directory under `tools/gomad3/.toolchain/builds/<key>` is 2–6 GB. Run `df -h .` before every rebuild and stop if less than 8 GB is free. Only delete build directories that this spec's own intermediate candidates created, once superseded and not referenced by retained evidence. Never delete the baseline key recorded by task 1 or the active key in `.toolchain/build-key`. Pre-existing directories and `make clean-qualifications` need the user's confirmation.
- Patch and overlay bytes feed the build key (`tools/gomad3/toolchain/buildkey.go:48-58`), so every patch or overlay edit yields a new toolchain identity. Never relabel old artifacts.
- fn-128.2 owns the transferred D12 Linux replay fix; fn-105 D14 retains Darwin `TestSignalWorkflowTestSuiteChasm`. Keep their existing dispositions until their own qualification passes. Do not edit qualification expectations to get a passing gate.
- Existing comments move with their code, unchanged. Add no allocations, host reads, dependencies, CLI flags, or capability grants.
- Always pass `-tags test_dep`. In testify code use `require`, not `assert`; plain `testing` files keep their existing style.

**Patch editing workflow (governed, no hand-edited hunks):**
1. Extract the verified archive `tools/gomad3/.toolchain/downloads/go1.27.1.src.tar.gz` (SHA-256 in `toolchain/version/version.json`) into a scratch directory under the gitignored `tools/gomad3/.toolchain/fn-110/`.
2. `go -C tools/gomad3 run ./cmd/gomadtool patch-materialize --root="$PWD/tools/gomad3" --source-root=<scratch>/go`
3. Edit upstream files in `<scratch>/go/src`. Do not copy overlay files into the candidate: `changedFiles` in `toolchain/patch_regenerate.go` rejects added source paths.
4. `go -C tools/gomad3 run ./cmd/gomadtool patch-regenerate --root="$PWD/tools/gomad3" --candidate-root=<scratch>/go`
5. `make -C tools/gomad3 generate validate`, then `make -C tools/gomad3 toolchain` (the build runs the archive-based overlay collision check at `toolchain/build.go:176`).

The descriptor requires `patch_allowlist` and `overlay_allowlist` to equal the checked trees exactly (`toolchain/version/descriptor.go:114-160`), so a task that adds an overlay file or empties a patched file updates `version.json` and regenerates in the same task.

### Host-draw checkpoint measurement (2026-10-05)

The producer-owned fn-112.5 checkpoint removes the alignment overhead of its private per-M diagnostic field while preserving diagnostics and scheduler behavior. Governed canonical output measures U1 24,894 bytes / 706 lines and U3 34,148 bytes / 1,040 lines. Against the immediate pre-edit candidate, U1 saves 4,121 bytes and U3 saves 4,214 bytes. Against the original task-1 U3 baseline of 32,652 bytes, the candidate remains 1,496 bytes larger. R8 is still unmet; the comparator, scope and acceptance are unchanged.

The checkpoint's task-local RED exposed 36 unchanged m fields realigned by the long name; GREEN measures zero alignment edits. Both supported source-set inventories pass. The original descriptor-pinned regeneration, checksum/rejection and U1/U3 zero-fuzz equivalence checks pass. Fresh materialization and alpha-renaming plus pinned gofmt prove preservation across all 20 patched files and the overlay. The descriptor's 20 patched / 79 overlay allowlists are unchanged. Evidence lives under `.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-5/host-draw-field-compact-20261005/`.

These source measurements supersede earlier current-candidate size statements without altering historic receipts. Native Darwin runtime/full/affected and formal qualification remain open. Transferred native Linux execution remains nonblocking under fn-128. U1 representation savings do not satisfy the separate R8 U3 reduction.

### Private-g-field representation checkpoint (2026-10-05)

The conductor admits the bounded alpha-rename in `.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task-2/gfield-compact-20261005/admission.md` to remove avoidable upstream field-alignment edits. Preserve the original fn-114.5 timer identity derivation and every runtime field type/order/behavior. The function/linkname gomadSimulationDomain and global gomadSimulationTransportSyscalls stay unchanged. Regeneration-derived identities and the choices-only golden are included in Touches solely for their exact input consequences. Retain RED/GREEN real canonical measurements and full alpha-gofmt preservation; original Acceptance, historic evidence, the 32,652-byte U3 comparator and required Darwin/full/review gates remain unchanged. This is verified source progress, not completed acceptance.

### Current-source measurement after private-g-field compaction (2026-10-05)

The fn-110.2 source checkpoint shortens five private goroutine fields and their selectors without changing field types/order, identity derivation, counters, transport accounting, comments or linked symbols. Governed outputs measure U1 24,117 bytes / 692 lines and U3 33,294 bytes / 1,026 lines. Against 1b0bc27758, U1 saves 777 bytes and U3 saves 854 bytes. The original task-1 U3 comparator stays 32,652 bytes; the current U3 remains 642 bytes larger, so R8 is unmet. This measurement supersedes earlier current-candidate size statements without changing their historical source scope.

The task-local RED exposed seven otherwise unchanged g fields realigned by the long names; the stable frozen GREEN finds zero. Complete alpha-renaming plus pinned gofmt equality covers all 20 patched files and the runtime overlay, with exactly 30 field sites and unchanged 20/79 allowlists. The independent calculation validates precisely seven fixture identity/digest updates while preserving the plain fixture, owner contracts and native guard. The frozen focused pinned-regeneration, checksum/rejection, U1/U3 zero-fuzz equivalence and both-source-set draw/seeded/clock/goroutine inventories and negative checks pass (207.808 seconds). Artifacts are under `.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task-2/gfield-compact-20261005/`.

These are source-progress measurements. Current-source native Darwin runtime/upstream/live-capability/process-simulation, baseline/candidate behavioral comparisons, full/affected and formal qualification remain open. Transferred Linux execution remains nonblocking under fn-128. The private field names necessarily change runtime debug metadata and build-derived identities; no binary identity or native acceptance is claimed. A fixed-source audit found no further authorized alignment correction or remaining approved scheduler extraction to close the gap. Any additional relocation requires its own bounded admission.
## Acceptance

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] The three implementations live in `overlay/src/runtime/gomad.go` behind private helpers; `proc.go` keeps the hooks, arrival admission, transport accounting, and stock timer-wake path; every moved comment is byte-identical
- [ ] Retry, external, deadlock, and transport-failure responses keep their returns, fatal messages, and `sched.lock` state; no helper adds an allocation, closure, or host read
- [ ] Fixture comparison for seeds 0, 1, 7, 42 and disabled mode matches the baseline toolchain; candidate same-seed repeatability and exact replay pass in the runtime, upstream, live-capability, and process-simulation checks
- [ ] Locked-goroutine syscall resumption, arrival-before/after-timer ordering, and a quiescence/transport race each have named regression coverage that passes before and after
- [ ] Patch still validates with the unchanged allowlists; `runtime.TestSizeof` expectation and timer-presence result are untouched; new `-U3` size is recorded and smaller than baseline
- [ ] Linux execution is recorded as incomplete


## Done summary
Blocked:
Implementation and local structural verification are complete, but the task's required native qualification cannot run in this environment. The host is linux/arm64; Gomad qualifies darwin/arm64 and linux/amd64, and the available Docker builder is also linux/arm64. Emulated or cross-compiled execution is not accepted as native Linux evidence. GitHub Actions cannot be triggered because this checkout has no usable GitHub credentials.

Outstanding gates: native toolchain build on darwin/arm64 and linux/amd64; runtime, upstream, live-capability, and process-simulation tests; the full baseline/candidate fixture comparison and exact replay on qualified hosts.

Blocked:
The scheduler extraction source candidate is integrated and its retained
structural/developmental checks remain evidence for their stated source scope.
Required native qualification remains unavailable on this linux/arm64 development
host. Gomad qualifies darwin/arm64 and linux/amd64; emulated, cross-compiled or
stock-host execution does not qualify either platform. No authorized source-bound
CI run is available for the current dirty combined candidate.

Fresh source measurements also contradict the original extraction-size acceptance:
the original task1 -U3 is 32,652 bytes, while the current combined final -U3 is
38,362 bytes, 5,710 bytes larger. Canonical -U1 at 29,015 bytes demonstrates
context reduction only; it cannot close R8's separate extraction reduction.
The evidence is retained under task-5/source-size-verification.md and independently
checked in task-5/conductor-source-size-verification.md. Keep the original
comparator and preserve all integrated behavior and comments.

All three approved scheduler bodies are already fully extracted. Task 2 requires
the remaining upstream integration hooks and scheduler machinery to stay; no
additional scoped extraction has been identified. Reconciliation must remain
with this extraction-size owner and the owners of introduced runtime inputs,
especially fn-112.5's diagnostic fields/alignment. Do not remove those capabilities,
embed goroutine state, move protected machinery or widen scope to manufacture
size acceptance. Task5 owns final verification, not a source fix for this gap.

Outstanding gates: native toolchain builds on darwin/arm64 and linux/amd64;
runtime, upstream, live-capability and process-simulation checks; full baseline/
candidate fixture comparisons and exact replay; and the original extraction-size
acceptance, which is presently unmet rather than merely unmeasured.

Blocked:
Native Linux execution stays transferred to fn-128 and does not block this source task. Current-source Darwin build/runtime/upstream/live-capability/process-simulation, baseline/candidate fixture comparisons, full/affected and formal review gates remain incomplete. R8 is independently unmet. Canonical U3 now measures 34,148 bytes after fn-112.5 private host-draw field compaction, against the unchanged original task-1 U3 baseline of 32,652 bytes, a 1,496-byte increase. U1 measures 24,894 bytes and its representation savings do not satisfy R8. Source preservation, pinned regeneration/checksum/rejection and both-source-set inventory receipts are in the fn-112.5 host-draw-field-compact-20261005 checkpoint. Historical larger candidate measurements retain their original source scope.

Blocked:
Private-g-field compaction is verified source progress. Canonical U1 measures
24,117 bytes / 692 lines; U3 measures 33,294 bytes / 1,026 lines. This removes
777 U1 bytes and 854 U3 bytes from 1b0bc27758. The original U3 baseline remains
32,652 bytes, leaving R8 unmet by 642 bytes. Seven otherwise unchanged g fields
no longer carry alignment-only edits. The five private field names and 30 sites
preserve complete alpha-gofmt source equivalence, field layout and 20/79 allowlists.

The frozen original pinned-regeneration, checksum/rejection, U1/U3 zero-fuzz
equivalence, both-source-set inventories and negative checks pass. Independent
identity calculation preserves all owner contracts and exactly seven derived
fixture pointers. These checks do not qualify runtime behavior on a native host.
Runtime debug field names and build-derived identities necessarily change.

Current-source native Darwin build/runtime/upstream/live-capability/process-
simulation, baseline/candidate behavioral fixture comparisons, full/affected and
formal qualification remain unproved. The developmental host is Linux/arm64.
The narrowed pure-host selection passes, but three diagnostic controls cannot
reach runtime assertions without the patched driver and the native identity
test skips. Existing full-host failures and 317 full-lint findings remain open;
changed-line lint is a separate source scope. No unchanged unsupported-host
full-gate timeout is retried or relabeled as passing evidence.

A fixed-source audit found no further authorized alignment correction or remaining
approved scheduler body extraction that closes R8. Further relocation needs a
separate bounded admission preserving protected scheduler/GC machinery and the
original comparator. Native Linux execution remains transferred to fn-128,
nonblocking here and explicitly unverified. Retain all original acceptance and
historical evidence. Evidence is in task-2/gfield-compact-20261005; source-only
review does not constitute formal SHIP or completed native acceptance.
## Evidence
- Commits:
- Tests:
- PRs:

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Representation/overlay implementation, byte equivalence, pinned checks, preservation, review, Darwin gates and unmet R8 size reduction. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
R8 remains independently unmet: baseline U3 patch 32,652 bytes, final U3 38,362 bytes (5,710-byte increase); U1 context reduction does not satisfy the extraction-size reduction.
