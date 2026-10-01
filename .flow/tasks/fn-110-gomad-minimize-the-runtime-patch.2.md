---
satisfies: [R2, R6]
---
# fn-110-gomad-minimize-the-runtime-patch.2 Move the three scheduler implementations into the runtime overlay

## Description
Move the three scheduler implementations out of the `src/runtime/proc.go` section of the patch into the existing runtime overlay file, leaving hooks in upstream code. Split from the crypto/syscall relocation because this is the behavior-sensitive part: lock state, draw order, and host-timing decisions must not move. The patch stays at three context lines here (the regenerator is unchanged until task 4), which gives the intermediate `-U3` measurement.

**Size:** M
**Files:** `tools/gomad3/toolchain/runtime/go1.27.1.patch`, `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go`, conformance fixture files only if coverage is missing, `docs/research/gomad/GOMAD_PATCH_SIZE.md`
**Touches:** [tools/gomad3/toolchain/runtime/go1.27.1.patch, tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go, tools/gomad3/internal/gomadtool/conformance/**, docs/research/gomad/GOMAD_PATCH_SIZE.md]

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
- No `git commit`, `git add`, stash, or worktrees. The user owns commits; leave changes in the working tree and report them. Earlier fn-110 tasks may therefore be uncommitted working-tree changes — do not revert them.
- Host is `darwin/arm64` only. `linux/amd64` gates cannot run here: record every Linux-dependent check as **incomplete**, never as passing. Cross-compilation is not Linux evidence.
- Disk: about 19 GB was free at planning time and each toolchain build directory under `tools/gomad3/.toolchain/builds/<key>` is 2–6 GB. Run `df -h .` before every rebuild and stop if less than 8 GB is free. Only delete build directories that this spec's own intermediate candidates created, once superseded and not referenced by retained evidence. Never delete the baseline key recorded by task 1 or the active key in `.toolchain/build-key`. Pre-existing directories and `make clean-qualifications` need the user's confirmation.
- Patch and overlay bytes feed the build key (`tools/gomad3/toolchain/buildkey.go:48-58`), so every patch or overlay edit yields a new toolchain identity. Never relabel old artifacts.
- fn-105 D12 (Linux replay divergence) and D14 (Darwin `TestSignalWorkflowTestSuiteChasm`) keep their owners and dispositions. Do not edit qualification expectations to get a passing gate.
- Existing comments move with their code, unchanged. Add no allocations, host reads, dependencies, CLI flags, or capability grants.
- Always pass `-tags test_dep`. In testify code use `require`, not `assert`; plain `testing` files keep their existing style.

**Patch editing workflow (governed, no hand-edited hunks):**
1. Extract the verified archive `tools/gomad3/.toolchain/downloads/go1.27.1.src.tar.gz` (SHA-256 in `toolchain/version/version.json`) into a scratch directory under the gitignored `tools/gomad3/.toolchain/fn-110/`.
2. `go -C tools/gomad3 run ./cmd/gomadtool patch-materialize --root="$PWD/tools/gomad3" --source-root=<scratch>/go`
3. Edit upstream files in `<scratch>/go/src`. Do not copy overlay files into the candidate: `changedFiles` in `toolchain/patch_regenerate.go` rejects added source paths.
4. `go -C tools/gomad3 run ./cmd/gomadtool patch-regenerate --root="$PWD/tools/gomad3" --candidate-root=<scratch>/go`
5. `make -C tools/gomad3 generate validate`, then `make -C tools/gomad3 toolchain` (the build runs the archive-based overlay collision check at `toolchain/build.go:176`).

The descriptor requires `patch_allowlist` and `overlay_allowlist` to equal the checked trees exactly (`toolchain/version/descriptor.go:114-160`), so a task that adds an overlay file or empties a patched file updates `version.json` and regenerates in the same task.

## Acceptance
- [ ] The three implementations live in `overlay/src/runtime/gomad.go` behind private helpers; `proc.go` keeps the hooks, arrival admission, transport accounting, and stock timer-wake path; every moved comment is byte-identical
- [ ] Retry, external, deadlock, and transport-failure responses keep their returns, fatal messages, and `sched.lock` state; no helper adds an allocation, closure, or host read
- [ ] Fixture comparison for seeds 0, 1, 7, 42 and disabled mode matches the baseline toolchain; candidate same-seed repeatability and exact replay pass in the runtime, upstream, live-capability, and process-simulation checks
- [ ] Locked-goroutine syscall resumption, arrival-before/after-timer ordering, and a quiescence/transport race each have named regression coverage that passes before and after
- [ ] Patch still validates with the unchanged allowlists; `runtime.TestSizeof` expectation and timer-presence result are untouched; new `-U3` size is recorded and smaller than baseline
- [ ] Linux execution is recorded as incomplete


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
