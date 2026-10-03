# fn-112 task 5: seeded-stream draw inventory and host-timed check

Host: **linux/arm64 with the development harness**. Everything below is linux/arm64 development
evidence, not darwin/arm64 or linux/amd64 evidence. Source base `d5330bb77` (fn-114.13 follow-up on
top of `9ecd161cf`); toolchain build key `275f92ed3f906dae5ce5563d0d9c693b511350316c9dfe55bc7c5434edb7d7c9`
(harness on; the committed tree hashes differently because the harness adds linux/arm64 to the
descriptor).

## R3 re-anchoring

Line numbers are in the materialized tree of build `275f92ed` unless named as overlay or patch lines.

**Q2 (no structural barrier against host timing): confirmed, then closed by this task.**
`rand` (`runtime/rand.go:172`) and `cheaprand` (`rand.go:246`) serve every M holding the P from the
process-wide states `gomadRuntimeRandom`/`gomadRuntimeCheapRandom` (overlay `gomad.go:629-660`),
seeded from `GOMADSEED`. The four rerouted host-timed sites are where the task file says, re-anchored:
lock-profile sample `lock_spinbit.go:244` -> `gomadLockProfileStart`, anti-starvation wake
`lock_spinbit.go:351`, steal order `proc.go:3896`, pcvalue cache `symtab.go:1103`; all use
`gomadHostCheapRand(N)` (overlay `gomad.go:665`), which draws from `m.cheaprand`. The seeded sites:
runnext flip `proc.go` `runqput`, shuffles in `runqputslow`/`runqputbatch`, run-queue pick
`gomadChoiceRunqIndex` (overlay), select `select.go:193`, timer ties `time.go` (`maybeAdd`,
`updateHeap`, `adjust`). The "remaining sites are unaudited" part was true; the audit is now the
inventory below. No assertion existed; the diagnostic check below adds one.

**Q3 (one-shot syscall wait in `suspendG`): changed.** The structure is as described: the patch hunk
`preempt.go` (patch lines 121-131, tree `preempt.go:118-120`) calls `gomadAwaitHostSyscallExit`
(overlay `gomad.go:1752`) once before the loop, and the loop's `_Gsyscall` case (`preempt.go:168`)
claims a goroutine in a syscall without waiting. The window the assessment names needs the scanned
goroutine to *enter* a syscall after the wait returns. Entering a syscall needs the goroutine to run,
which needs a P; Gomad starts with `GOMAXPROCS` one and the window stays closed while it stays one
(raising it with `runtime.GOMAXPROCS(n)` is unsupported, not prevented), and both `suspendG` callers hold that P while they
scan (`mgcmark.go:298` from a mark worker in `markroot`, `trace.go:523` from the tracer, which is
outside the contract) or run with the world stopped. A goroutine leaving a syscall without a P is
queued as an arrival and becomes `_Grunnable`, which the loop then claims. So, from source reading,
the window is not reachable while `GOMAXPROCS` stays one. Not reproduced or measured; no code
change here. It stays a D12 candidate owned by fn-105 D12, which needs a native linux/amd64 host.

## Inventory

Location: `tools/gomad3/toolchain/draw_inventory_test.go`
(`TestPatchedRuntimeSeededDrawReferencesAreReviewed`, list `reviewedDrawReferences`), beside the clock
and goroutine inventories, in the `test-toolchain` tier.

- Scans every non-test Go file of the built GOROOT that builds for each supported platform with the
  target configuration (`goexperiment.greenteagc` removed, as `target/internal/build` builds with
  `GOEXPERIMENT=nogreenteagc`). Counts identifiers naming a runtime rand helper (upstream `rand`,
  `randn`, `cheaprand*`, `bootstrapRand`, `maps_rand`, `legacy_fastrand*`, and Gomad's seeded and
  M-local helpers) per file, enclosing function, and helper; in other packages it follows each
  `//go:linkname local runtime.<helper>` alias (`math/rand`, `math/rand/v2`, `os`, `net`,
  `hash/maphash`, `sync`, `internal/sync`, `unique`, `internal/runtime/maps`); and it counts the
  compiler's by-name calls in `cmd/compile/internal/walk` (`walkMakeMap` seeds a stack map).
- 69 keys. Classes: `target-ordered`, `host-timed` (must use an M-local helper:
  `gomadHostCheapRand(N)`, or `cheaprand64`/`cheaprandu64`, which stay on `m.cheaprand64` on both
  64-bit platforms), `host-timed-blocked` (finding required), `implementation`, `inactive` (reached
  only with Gomad disabled), `diagnostic` (the fault switch). Every entry needs a reason; an
  unclassified, recounted, or vanished key fails.
- `TestSeededDrawInventoryRejectsUnclassifiedReference` builds a synthetic GOROOT and requires the
  unclassified-reference, seeded-helper-on-host-timed, and missing-finding failures.
- Deliberate unclassified reference in the built tree: failed as expected, then passed after removal
  (`unclassified-reference-demo.log`).

Host-timed sites (8): the four known reroutes plus `mutexSampleContention`,
`mLockProfile.recordUnlock` (x2), `blocksampled`, `mutexevent`, which already use the M-local
`cheaprand64`/`cheaprandu64`.

## Reroutes made

**None.** Every host-timed site outside prohibited files already draws from an M-local stream. Two
host-timed sites stay on the process-wide stream and are recorded as `host-timed-blocked`:

1. `runtime/mgcpacer.go` `gcControllerState.enlistWorker` (`cheaprandn`, line 736): collector file,
   blocked by the collector patch prohibition; raises **spec Open Question 3** (patch-policy
   approval). Unreachable while `GOMAXPROCS` stays one (Gomad starts with one; raising it is
   unsupported): `enlistWorker` returns at `gomaxprocs <= 1` before the draw.
2. `runtime/os_linux.go` `setThreadCPUProfiler` (`cheaprandn`, line 680): in the prohibited `os_`
   runtime area (`toolchain/patch.go`);
   reached only with CPU profiling (SIGPROF), which is outside the deterministic contract (task 6
   exclusion). Linux only. Also referred under Open Question 3.

Not counted: `mgcmark_greenteagc.go` `stealOrder.start(cheaprand())` (collector, host-timed) is not
compiled into targets because they build with `nogreenteagc`; if Green Tea is ever enabled it joins
the blocked list. `crypto/rand` uses Gomad's SHA-256 entropy reader, not the runtime rand helpers.

Because nothing was rerouted, no seed's schedule moved, and the acceptance item asking for core and
smoke qualification after a reroute does not apply. The diagnostic check below is a runtime edit
and moves the toolchain build key and the choice implementation digest; cross-build behavioral
projection was not run (the in-identity diagnostics-off check is in the draw_check test).

## Runtime check (diagnostics only)

- `m.gomadHostTimed` (patch, `runtime2.go`) counts host-timed paths the M is inside, only while a
  diagnostic trace is recorded. `gomadHostTimedEnter`/`Exit` (overlay) bracket the idle steal pass
  (patch, `proc.go` around `stealWork`) and the fault path.
- `gomadSeededDrawCheck` runs first in every seeded helper (`gomadRuntimeRand`,
  `gomadRuntimeCheapRand`, `gomadTimerRand`, `gomadClockTickDraw`, `gomadChoiceRandom`,
  `gomadChoiceSelectSeeded`); inside a bracket it prints
  `runtime: Gomad host-timed path drew from the seeded stream` and exits 125. The diagnostic trace is
  left unclosed, so the launcher reports `diagnostic trace incomplete`.
- Fault switch (task 3's variable): `GOMAD3_DIAGNOSTIC_PERTURB_DRAW=host-timed:N` arms at choice
  record N; the next `gomadHostCheapRand` on any M then draws from the seeded stream inside a
  bracket, as an unrerouted host-timed site would. The plain `=N` form is unchanged and is not
  host-timed (task 4's localiser fixture still passes). Both forms still require a diagnostic trace.
- Negative fixture: `internal/gomadtool/conformance/testdata/draw_check`, run by
  `runner/internal/execution/draw_check_toolchain_test.go`
  (`TestDiagnosticDrawCheckStopsHostTimedSeededDraw`) through the toolchain-test launcher. Controls:
  diagnostics off, on, and on with the plain fault all exit 0, and off/on have equal stdout and
  choice bytes; `host-timed:5` exits 125 with the message and no stdout; `host-timed:5` without a
  trace exits 2 with the configuration error.
- Patch regenerated with `gomadtool patch-regenerate --candidate-root=<materialized tree>`; no hand
  edits to hunks.

## For task 10 (documentation)

- Inventory: `toolchain/draw_inventory_test.go`, test tier `test-toolchain`.
- Contract sentence draft (narrowed after review): "With a diagnostic trace, a draw from a seeded
  stream inside a bracketed host-timed path (the idle steal pass, the injection of netpoll results,
  and the CPU-profiler setup in `execute`) stops the target with exit status 125 and
  `runtime: Gomad host-timed path drew from the seeded stream`. The other host-timed sites in the
  inventory are not bracketed; the inventory, not the runtime check, keeps them on M-local streams. Two host-timed draws remain on the
  seeded stream because rerouting them needs a prohibited file: collector worker enlistment (not
  reached while GOMAXPROCS stays one) and linux CPU-profile timer setup (CPU profiling is outside the
  contract)."
- Reroutes: none.

## Gates (linux/arm64 development harness)

See `gates.md`.

## Owed on qualified platforms

darwin/arm64 and linux/amd64:
`make -C tools/gomad3 validate test-toolchain test-runtime overlay-test`, and
`GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host` (covers
`TestDiagnosticDrawCheckStopsHostTimedSeededDraw`). The inventory test needs no per-platform list:
keys are platform-independent and `os_linux.go` appears only on linux.

## Review follow-up (fn-112.5 review, NEEDS_WORK)

Applied on top of `c2fd6d2f0`; details, commands and results in `review-fixes.md`. This section
supersedes the statements above where they differ:

- Netpoll batches: the shuffle in `runqputbatch` drew from the seeded scheduler stream when a netpoll
  result of two or more goroutines was injected while the M held the P. The six netpoll injection
  sites in `proc.go` now call `gomadInjectHostList` (overlay), which brackets the injection and makes
  `gomadChoiceShuffleSeeded` take the M-local draw. This is a reroute: a schedule that injected such
  a batch changes, under a new toolchain identity. Target-originated batches stay seeded.
- Green Tea: a seeded runtime built with `goexperiment.greenteagc` exits 2 with
  `runtime: GOMADSEED requires GOEXPERIMENT=nogreenteagc`; the runtime campaign builds every
  fixture with `GOEXPERIMENT=nogreenteagc`.
- The host-timed fault no longer brackets itself; only real brackets stop the process.
  `draw_check` gains an idle window before it prints; with the steal-pass bracket removed the fault
  no longer stops it (checked with a `-overlay` build).
- The inventory also counts direct uses of the seeded state variables (allowed only as
  `implementation` entries in their accessors and seeding functions) and scans all of `cmd/compile`
  except the typecheck builtin table. It classifies the containing function, not its callers.
