# fn-241-gomad-a-race-profile-for-wasip1-deferred Gomad: a -race profile for wasip1 (deferred)

## Status: deferred

This spec is deferred and not ready. Every task is blocked with the same revival trigger. It exists so the decision in [GOMAD_RACES.md](../../.plans/GOMAD_RACES.md) is recorded as scope with acceptance criteria rather than argued again. No implementation, PR, push or CI run is authorized until the owner revives it.

**Revival trigger.** The persistence fault matrix from [GOMAD_FIND_BUGS.md §3](../../.plans/GOMAD_FIND_BUGS.md) has saturated, the known-bug benchmark table exists, and that table shows a bug in the data-race or lock-discipline family that a workload under Gomad reaches but Gomad does not report. The PR 10549 row is the first candidate. Static checklocks annotations on the history service do not satisfy or cancel the trigger; they are Temporal's work and catch only guarded-field violations.

## Conversation Evidence

> user: "let's write a flow spec based on temporal/.plans/GOMAD_RACES.md (but mark it deferred)"
> user: "should we aim to do the cleanest and simplest approach of 'supporting -race' properly; would that cover the special map case too? and that way we can turn it on/off when needed"

## Goal & Context

Gomad cannot find the bug class behind [temporalio/temporal#10549](https://github.com/temporalio/temporal/pull/10549): `CacheSize()` reading lock-guarded state after `Unlock()`, surfacing as `fatal error: concurrent map iteration and map write` on real hardware and as a silently stale number in a second variant. The cooperative profile keeps one logical processor and asks the helper for a decision only at goroutine creation, run-queue dispatch and select, so a goroutine inside `mapassign` is never descheduled and no reordering of existing decisions reaches the crash. Gomad also has no data-race detection and no lock-discipline oracle, so without a crash the bug has no symptom. [GOMAD_RACES.md §2](../../.plans/GOMAD_RACES.md) records the analysis and the research that already names this class.

That plan sketched four investments: map-write choice points (A), happens-before recording plus map footprints (B), a checklocks-driven lock-ownership oracle (C) and a race detector inside Gomad (D). The owner chose D as the single mechanism on 2026-10-23, because it subsumes the detection value of A, B and C in one switchable profile: stock `runtime/map.go` already calls the race hooks on every map access, assign, delete and iteration when `raceenabled` is set, so a happens-before detector reports the updater's assign and the releaser's iteration as an unordered pair on any schedule where both run, with no crash needed; the compiler instruments plain field reads, so the stale-size variant is reported too; and the runtime's channel, semaphore, `sync` and `sync/atomic` paths already emit acquire and release edges, which removes B's atomics blind spot. A, B and the dynamic half of C are not pursued.

The profile is a switch. Ordinary exploration runs keep the cooperative profile and its exact-replay identity unchanged; the race profile is a separate profile identity with its own qualification, selected per run. This spec covers the Gomad module only. The §3 functional test that makes the workload reach the code, and the `reachable: no` benchmark row, are prerequisites owned by the benchmark-table work in GOMAD_FIND_BUGS.md, not deliverables here.

## Architecture & Data Models

**Toolchain gate.** Both `cmd/go` (`cmd/go/internal/work/init.go:156`) and `cmd/compile` (`cmd/compile/internal/base/flag.go:297`) refuse `-race` through one function, `internal/platform.RaceDetectorSupported`, a goos/goarch switch. The patched tree adds `wasip1/wasm` to that switch and preparation rebuilds the pinned toolchain's `go` and `compile` from the patched tree for the race profile. This widens the overlay surface from runtime and `net` files to one `internal/platform` file plus a toolchain rebuild step. The anchored rewrite table and the pinned stock version `go1.27.1` are unchanged. The rebuilt tool binaries are identity inputs of the race profile.

**Pure-Go `runtime/race` for wasm.** Stock `runtime/race.go` declares about 65 functions (`raceread`, `racewrite`, `racereadrange`, `racewriterange`, `raceacquire`, `racerelease`, `racereleasemerge`, `racemalloc`, `racefree`, `racegostart`, `racegoend`, `racefuncenter`, `racefuncexit`, `raceinit`, `racefini`, the `sync/atomic` wrappers and the PC and object variants) that trampoline through per-architecture assembly into ThreadSanitizer, which does not exist for wasm. A new overlay source `race_wasm.go.txt` under `//go:build race && wasip1 && wasm` implements them in Go, and a `runtime/race` package for the target replaces the `.syso`. Under one P there is no concurrent access to detector state, so no atomics or locks on shadow state are needed.

**Detector.** FastTrack: one vector clock per goroutine, an epoch for the last write and an epoch-or-clock for reads per shadow cell, 8-byte granules. Acquire and release edges come from the stock runtime call sites; `racegostart` seeds the child clock from the parent. A race is an access whose epoch is not ordered before the current goroutine's clock. The detector is in-guest and deterministic, so a race report is a target failure on a seed, retained and replayable like any other.

**Shadow memory.** Shadow is allocated per heap span, driven by `racemalloc` and `racefree`, not for the whole wasm32 address space. Two words per granule is roughly twice the live heap inside a 4 GiB ceiling. If the Temporal test server does not fit, allocation sampling that skips a seeded fraction of spans is the fallback, recorded in the profile so recall loss is visible. Shadow lives outside the Go heap so GC accounting is unaffected; the virtual-memory limit and fuel bounds of the profile account for it.

**Reports.** A race is retained as a classified target failure carrying both program counters, both access kinds, both goroutine creation sites and the stack at each access, symbolized through the inspect path fn-229 built. The report also names the forcing schedule: the seed and the choice ordinal at which the first access ran, so an operator can replay to it. A happens-before fingerprint, the digest of the final vector clocks over the execution, is emitted as an observation record so later work can deduplicate seeds; it is not consumed in this spec.

**Profile.** `gomad.wasi-race/v1` is a third profile beside the stock and cooperative profiles. It composes the cooperative overlay with the race overlay; selection is per run. Its identity binds the rebuilt tool binaries, the race overlay sources, the shadow policy and the recalibrated limits. Artifacts record their profile and replay only on it. Nothing in the cooperative profile's identity, overlay sources or qualified expectations changes.

## API Contracts

- A run flag or profile selector chooses `gomad.wasi-race/v1`; the default remains the cooperative profile. No change to the cooperative profile's CLI, manifest, status, environment variables or wire schema.
- A race report is a new failure class with a stable name, both PCs, both goroutine creation sites, symbolized stacks and the forcing seed and ordinal. Exact field layout is decided in task 3 and documented in CLI.md.
- The happens-before fingerprint is an observation record on the existing choice wire, flagged as observation, never forced.
- The race profile is refused, not silently downgraded, when its rebuilt toolchain, overlay or shadow-policy identity does not match the installation.

## Edge Cases & Constraints

- The runtime itself is not instrumented (`norace`); runtime-internal races are out of scope, as in stock Go.
- `racefuncenter` and `racefuncexit` are called from every instrumented function; they must be cheap no-ops or stack-depth bookkeeping only.
- Reports from shadow sampling carry the sampling policy; a run with sampling cannot claim "no race" for the skipped spans.
- Instrumented code runs several times slower and allocates more; fuel, virtual-time and memory capacities measured for the cooperative profile do not transfer. The race profile has its own measured capacities and qualification cells.
- Any overlay change reopens the determinism inventory. The race overlay must not add nondeterministic inputs; shadow allocation addresses are derived from the deterministic guest heap.
- `sync/atomic` under `-race` routes through the runtime wrappers; on wasm they perform the operation and record the edge, which is correct under one P.
- No PR, push or CI run under this spec.

## Acceptance Criteria

- **R1:** The patched tree lifts the race gate for `wasip1/wasm` through `internal/platform.RaceDetectorSupported` only, preparation rebuilds the pinned `go` and `compile` tools for the race profile, and a stub `runtime/race` lets the toolchain choices fixture build with `-race` and run to its normal result under the cooperative overlay. Errors: any other gate edit, a stock-version change, or an instrumented fixture whose outcome differs from the uninstrumented one is a finding.
- **R2:** The FastTrack detector with span-allocated shadow and stock runtime sync edges reports the PR 10549 map race on the §3 workload from one seed, naming the `mapassign` in `FindOrCreate` and the `mapiterinit` in `GetSize`, with the fix from PR 10549 producing no report. Errors: a report that needs a crash, more than one seed, or that persists after the fix fails this requirement; a false positive on the fixed code is a finding with its PCs.
- **R3:** Race reports are retained as a classified target failure with both PCs, both creation sites, symbolized stacks and the forcing seed and ordinal, and the `sync/atomic` wrappers record edges so the stale-size variant is reported while an atomic-guarded counter is not. Errors: an unsymbolized or non-replayable report, or a report on the atomic counter, is a finding.
- **R4:** `gomad.wasi-race/v1` has its own identity, measured fuel, virtual-time and memory capacities, qualification cells on the one-box workload, and operator documentation for turning it on and off. The cooperative profile's identities, overlay sources and qualified expectations are byte-unchanged. Errors: a cooperative identity drift, an unrecorded sampling policy, or a race artifact replaying on the cooperative profile is a finding.

## Boundaries

- Investments A, B and the dynamic C oracle from GOMAD_RACES.md are not pursued; conflict-directed dispatch and fingerprint-based deduplication are not consumed here.
- The §3 functional test, the benchmark table and its `reachable: no` row, and checklocks annotations are prerequisites owned elsewhere.
- No change to the cooperative profile, its rewrite table or the stock Go version.
- No compiler changes beyond rebuilding from the patched tree; `-race` instrumentation itself is stock.
- No PR, push or CI run.

## Decision Context

The owner asked whether supporting `-race` properly would be cleaner than the incremental investments and whether it covers the map case. It does: the race hooks in stock map code report the pair on any schedule where both accesses run, which the §3 workload guarantees, so the choice-point and footprint machinery becomes unnecessary for detection. The one real design question is shadow memory under wasm32, answered with span-allocated shadow and sampling as a recorded fallback. The feasibility study's L sizing stands, but it replaces four smaller investments that together cost about the same and leave permanent machinery in the exploration runtime. The spec is deferred because GOMAD_RACES.md §9 still holds: for memory races Gomad is the fourth-best detector, and the fault matrix and scheduling bugs rank higher as proof of value. Rejected: Investment A alone (confirmation only, misses the silent variant); Investment B (maps, channels and locks only, atomics blind spot, two M-sized permanent overlay changes); patching `cmd/go` and `cmd/compile` separately (both consult the same platform function).

## Quick commands

```bash
SANDBOX_START_DIR= bash -c 'cd /Users/stephan/Workspace/skunkworks/gomad/temporal && GIT_DISCOVERY_ACROSS_FILESYSTEM=1 python3 /Users/stephan/Workspace/skunkworks/flow-next/plugins/flow-next/scripts/flowctl.py show fn-241-gomad-a-race-profile-for-wasip1-deferred --json'
SANDBOX_START_DIR= bash -c 'cd /Users/stephan/Workspace/skunkworks/gomad/temporal && GIT_DISCOVERY_ACROSS_FILESYSTEM=1 python3 /Users/stephan/Workspace/skunkworks/flow-next/plugins/flow-next/scripts/flowctl.py validate --spec fn-241-gomad-a-race-profile-for-wasip1-deferred --coverage --json'
make -C tools/gomad validate
```

## Early proof point

Task 1: the toolchain choices fixture builds with `-race` for `wasip1/wasm` from the patched tree and runs to its normal result with a stub `runtime/race`. If the rebuilt toolchain or the stub cannot be made to link, the detector design is moot and the spec returns to the owner.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Platform gate lifted, toolchain rebuilt, stub `runtime/race` builds and runs the choices fixture instrumented | fn-241-gomad-a-race-profile-for-wasip1-deferred.1 | — |
| R2 | FastTrack detector with span shadow reports the PR 10549 map race from one seed; fixed code is clean | fn-241-gomad-a-race-profile-for-wasip1-deferred.2 | — |
| R3 | Retained, symbolized, replayable race reports with forcing schedule; atomic edges; stale-size variant reported | fn-241-gomad-a-race-profile-for-wasip1-deferred.3 | — |
| R4 | Race profile identity, capacities, qualification cells, on/off documentation; cooperative profile unchanged | fn-241-gomad-a-race-profile-for-wasip1-deferred.4 | — |
