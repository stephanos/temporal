# Gomad host-clock escapes investigation

Assessment date: 2026-09-30.

No host-clock escape in the static inventory is read back by runtime control flow: none can change
scheduling, GC pacing, or allocation. Each one hands a host value to the target instead, as reporting
state, as trace output, as entropy input in FIPS mode, or as the result of a pack-gated call. A
target that prints or branches on such a value loses same-seed repeatability. The Temporal
functional workloads link the readers but, by source inspection, neither emit nor consume the
values. A fixture that prints `MemStats.LastGC` is reported by qualification as a stdout divergence
while its choice tape replays exactly. These four escapes are not a cause of the D12 or D14 replay
divergences. One host-time path outside the inventory, `cputicks` on linux/amd64, does steer profile
sampling and is left open for D12. The recommended disposition is a documented limitation now and a stamp overwrite
proposed for a later decision; the dynamic linux/amd64 audit (D11) stays deferred. Nothing in this
report is implemented or fixed.

This is the receipt for
[D21](../../../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.21.md) (fn-105 R21) and the
prerequisite for [D11](../../../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.11.md) (R11).

## Scope and identities

| Item | Value |
| --- | --- |
| Repository revision | `d4d800fb47` plus the shared uncommitted working tree |
| Toolchain | go1.27.1, build key `c0661e38b4e001c8d86912d4ce9e265eb0f33e5f3298015da27a4c97ac4019b9` |
| Host | darwin/arm64, macOS 26.6.2 (25G83) |
| linux/amd64 | Not available. Every linux statement below is source reasoning and is marked as such. |

Source references are `file:line` under the patched tree
`tools/gomad3/.toolchain/builds/<build key>/src`. "Patch" is
`tools/gomad3/toolchain/runtime/go1.27.1.patch`; "overlay" is
`tools/gomad3/toolchain/runtime/overlay/src`. No functional suite, toolchain build, or `make test`
ran for this investigation.

## Classification

The inventory is `reviewedHostClockReferences` in
[`clock_inventory_test.go`](../../../tools/gomad3/toolchain/clock_inventory_test.go). It carries four
escapes, three on both platforms and one on linux only.

| Escape | Platforms | Host value | Class | Read by runtime control flow | Reaches the target when |
| --- | --- | --- | --- | --- | --- |
| `gcMarkTermination` (`runtime/mgc.go:1432`) | both | wall time | Target-visible reporting state | No | The target reads `MemStats.LastGC`, `MemStats.PauseEnd`, `debug.GCStats.LastGC`, or `debug.GCStats.PauseEnd` |
| FIPS `monoTime` (`runtime/time.go:37`) | linux; virtual on darwin | monotonic time | Entropy input, gated by FIPS mode | No | FIPS mode is on and no testing reader is installed; it then seeds the DRBG behind every later random read |
| Tracer clock snapshot (`runtime/tracetime.go:93`) | both | wall time; monotonic too on linux | Trace output | No | Execution tracing is started and the target or harness retains the trace |
| `syscall.Gettimeofday` (`syscall/asm_linux_amd64.s:56`) | linux | wall time | Gated target operation | No | A package admitted by an exact compatibility pack calls it; the result is ordinary target input |

"Runtime control flow" means the runtime's own scheduling, GC pacing, and allocation decisions.
"Evidence" means stdout, stderr, the I/O transcript, World, and coverage. A value that reaches the
target reaches evidence when the target emits it or lets it steer what the target does: a branch on
`LastGC`, a key drawn from a host-seeded DRBG, or a timestamp from `Gettimeofday` all change target
behavior without printing the value. The classification says where a value can enter, not that
targets ignore it.

### `gcMarkTermination`

`runtime/mgc.go:1431-1439` takes two readings. `now := nanotime()` is virtual: the patch returns
`faketime` when Gomad is enabled (`runtime/time_nofake.go:33-35`). `time_now()` is the host read. The
host value goes to `memstats.last_gc_unix` (`mgc.go:1436`) and to one `memstats.pause_end` slot
(`mgc.go:1439`). The virtual value goes to `memstats.last_gc_nanotime` (`mgc.go:1437`), the pause
duration, and `work.tEnd`.

The two host-stamped fields have four readers, all copy-outs in `runtime/mstats.go`: lines 550 and
553 for `ReadMemStats`, lines 600 and 603 for `readGCStats`. `runtime/heapdump.go:601` writes
`MemStats.LastGC` into a heap dump. No branch, pacer input, or allocation decision reads either
field. The periodic collection trigger reads `last_gc_nanotime`, the virtual value
(`mgc.go:718`), and the patch disables that trigger under Gomad anyway (`runtime/proc.go:6820`).

The inventory finding names `MemStats.LastGC` and `debug.GCStats`. `MemStats.PauseEnd` carries the
same host stamp and is not named there.

`time_now` itself cannot perturb the scheduler. On darwin it is `walltime()` plus `nanotime()`
(`runtime/timestub.go:26-29`); on linux/amd64 it is assembly that calls the vDSO or falls back to a
raw `SYSCALL` (`runtime/time_linux_amd64.s:14-96`). Neither path releases the P, takes a runtime
lock, or allocates.

Reporting surfaces in the standard library that expose the stamp:

- `runtime.ReadMemStats` (`runtime/mstats.go:356`) and `debug.ReadGCStats`
  (`runtime/debug/garbage.go:50`, `:61`).
- The text heap profile, which prints `# LastGC` and `# PauseEnd` (`runtime/pprof/pprof.go:740`,
  `:742`). `net/http/pprof` serves it at `/debug/pprof/heap?debug=1`.
- `expvar`, which publishes the whole `MemStats` as `memstats` and registers `/debug/vars` on the
  default mux at package initialization (`expvar/expvar.go:374-387`).
- `debug.WriteHeapDump`.

`runtime/metrics` has no last-collection metric. Its pause and CPU metrics derive from `nanotime`
and are virtual.

### FIPS `monoTime`

`crypto/internal/fips140deps/time.HighPrecisionNow` calls the runtime's
`crypto_internal_fips140deps_time_monoTime` (`runtime/time.go:35-39`), which returns the monotonic
part of `time_now`. Its only caller is the CPU-jitter entropy source
(`crypto/internal/entropy/v1.0.0/entropy.go:106`, `:107`, `:137`), reached through `getEntropy`
(`crypto/internal/fips140/drbg/entropy_fips140.go:30`) from `drbg.Read`
(`crypto/internal/fips140/drbg/rand.go:26`, `:73`).

Two gates stand in front of it. `drbg.Read` returns early when a testing reader is installed
(`rand.go:34-41`) and again when FIPS mode is off (`rand.go:43`). The patch installs Gomad's seeded
reader as that testing reader from `crypto/rand`'s `init` when deterministic I/O is enabled
(`crypto/rand/rand.go:37-44`; patch lines 69-76). The installation therefore exists only in a
program that links `crypto/rand` and runs with deterministic I/O. The standard library also reaches
`drbg.Read` from packages that do not import `crypto/rand`, such as `crypto/hpke`
(`crypto/hpke/pq.go:250`). The path is reachable with `GODEBUG=fips140=on` in any process where the
testing reader is absent: a direct-seed run without deterministic I/O, or a target that never links
`crypto/rand`. When reached, the host reading seeds the DRBG, so it is target input, not reporting.

The inventory finding is wider than the source on darwin. `timestub.go:28` returns `nanotime()` as
the monotonic value, which is `faketime` under Gomad, so `monoTime` is already virtual on
darwin/arm64. Only linux/amd64, where `time·now` is assembly, returns host monotonic time. A frozen
virtual clock gives the jitter source no entropy; its behavior in that configuration was not
exercised.

### Tracer clock snapshot

`traceSyncBatch` writes `time_now`'s wall and monotonic readings into the `EvClockSnapshot` event
(`runtime/tracetime.go:93-103`). Its callers are `StartTrace` and `traceAdvance`
(`runtime/trace.go:427`, `:618`), so it runs only while the execution tracer is on: `runtime/trace`,
a flight recorder, `-test.trace`, or `/debug/pprof/trace`. Event timestamps come from
`traceClockNow`, which is `nanotime()` on both qualified platforms (`tracetime.go:58-63`). The
snapshot is written to the trace buffer and never read back.

### `syscall.Gettimeofday`

On linux/amd64 `syscall.Gettimeofday` and `syscall.Time` (`syscall/syscall_linux_amd64.go:82`, `:90`)
reach the vDSO symbol or raw `SYS_gettimeofday` (`syscall/asm_linux_amd64.s:49-68`). No
standard-library package outside `syscall` calls either function. The call is an explicit request by
the calling package and returns a value without touching runtime state; what the caller does with
that value is target behavior. The gate is import-level: a pack rule admits `import:syscall` for a
package bound to exact source digests (for example the `rules` of
`tools/gomad3/internal/compatibilitypack/packs/temporal-functional-tests-darwin-arm64.json`), not
individual functions. Review of the admitted sources is what keeps a clock call out.

### Intentionally retained host reads

`gomadWallNanotime` (overlay `runtime/gomad.go:996-998`) returns `nanotime1()` to the deterministic
I/O packages for host wall bounds (overlay `internal/gomadfs/volume.go:302`, `:323`). The inventory
classifies it host-by-design. Readings taken before activation, during runtime startup, are also
host readings by construction.

### Host-time paths outside the inventory's identifier set

The inventory matches `nanotime1`, `walltime`, `time_now`, and the two linux vDSO symbols. These
paths reach host time under other names. The first is an exposure the inventory does not record; the
others are gated or answered by adapters. A dynamic audit would have to account for each.

| Path | Source | Gate or consequence |
| --- | --- | --- |
| `cputicks` on linux/amd64 | `RDTSC`/`RDTSCP`, `runtime/asm_amd64.s:1225-1250` | Host cycle counts feed the block, mutex, and runtime-lock profiles; see below. On darwin/arm64 `cputicks` is `nanotime()` (`runtime/os_darwin_arm64.go:8-11`) and so virtual. Not a syscall, so no syscall filter observes it. |
| `syscall.Gettimeofday` on darwin | libc trampoline, `syscall/zsyscall_darwin_arm64.go:1924-1934` | Same `syscall` pack gate as linux. The inventory pins only the linux symbol, so toolchain drift in the darwin path is unpinned. |
| `golang.org/x/sys/unix.ClockGettime` and `Gettimeofday` | Raw syscalls in the target's module graph | Pack-gated at import level. |
| gRPC `GetCPUTime` on linux | `unix.ClockGettime(CLOCK_PROCESS_CPUTIME_ID)`, `google.golang.org/grpc@v1.83.2/internal/syscall/syscall_linux.go:38` | The gRPC adapter replaces that file with the module's non-Linux implementation (`tools/gomad3/deterministicio/grpc_adapter.go:39-62`), so a prepared linux target does not contain the call. |
| modernc libc time calls | `Xgettimeofday` on darwin; the syscall dispatcher on linux | The libc adapter answers `Xgettimeofday` on darwin (`tools/gomad3/deterministicio/libc_adapter.go:265`, `adapterdata/modernc_libc_darwin.go.tmpl:227`) and `SYS_clock_gettime`, `SYS_gettimeofday`, and `SYS_time` on linux (`adapterdata/modernc_libc_linux.go.tmpl:160-166`). `Xclock_gettime` on darwin (`modernc.org/libc@v1.72.3/libc_unix3.go:23`) has no rewrite; SQLite's generated darwin source does not call it. |

`cputicks` on linux/amd64 is a fifth escape, reached through the block and mutex profiles. It has
two parts: sampling, gated by the profile rates, and export, reached by writing either profile. With `blockprofilerate` above zero, channel, select, and
semaphore waits time themselves with `cputicks` (`runtime/chan.go:218-220`, `runtime/select.go:156-158`,
`runtime/sema.go:169-177`), and the host cycle count decides whether the event is sampled and whether
a random draw is taken (`blockevent` and `blocksampled`, `runtime/mprof.go:500-518`). With
`mutexprofilerate` above zero, `unlock2` samples contention (`runtime/lock_spinbit.go:282`, `:304`)
and the host cycle count chooses which stack the runtime-lock profile retains
(`lock_spinbit.go:408-411`, `recordUnlock`, `runtime/mprof.go:673-701`). Here host time does steer
runtime profiling decisions, and the block and mutex profiles are target-readable through
`runtime/pprof` and `/debug/pprof`. `SetBlockProfileRate`, `SetMutexProfileFraction`,
`-test.blockprofile`, and `-test.mutexprofile` raise the rates.

Export is a second exposure. Writing a block or mutex profile calls `pprof_cyclesPerSecond`
(`runtime/cpuprof.go:221-224`), which calls `ticksPerSecond`. The first call derives a
cycles-per-second rate from `cputicks` against the virtual `nanotime` and caches it; later calls
return the cached value (`runtime/runtime.go:77-96`). The text format always prints that rate
(`runtime/pprof/pprof.go:1007`), whatever the profile rates are. The protobuf format computes it
(`pprof.go:469`) but uses it only to convert the cycle counts of recorded samples
(`pprof.go:474-481`), so an empty protobuf profile carries no host-derived value.

With both rates at zero and no profile written, the only remaining read stores a start tick that
nothing consumes (`lock_spinbit.go:243`). None of this was exercised: it is linux/amd64 only.

## Fixture evidence

Two go-run targets, retained as
[`fn105-d21-probe-lastgc.go.txt`](../../../.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d21-probe-lastgc.go.txt)
and
[`fn105-d21-probe-control.go.txt`](../../../.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d21-probe-control.go.txt),
force the same collections and make the same `ReadMemStats` and `ReadGCStats` calls. `lastgc` prints
the reporting fields. `control` prints the virtual time, `NumGC`, the pause total, and one boolean
derived from the host stamp (`LastGC` later than virtual now), which is true on every run. It
consumes the host value without letting it change output.

| Run (darwin/arm64) | Outcome |
| --- | --- |
| `lastgc`, direct `GOMADSEED=1`, twice | Virtual now is `946684800000000000` both times. `memstats_last_gc`, `memstats_pause_end_latest`, `gcstats_last_gc`, and `gcstats_pause_end_latest` are host wall time and differ (`1790821690329736000` against `1790821690570453000`). `NumGC`, pause durations, and the sampled `runtime/metrics` values are equal. |
| `gomad qualify --seed 1 --repeat 2 go-run ./cmd/lastgc` | `qualified=false deterministic=false`, `first-divergence=stdout.full_sha256` |
| `gomad qualify --seed 1 --repeat 2 go-run ./cmd/control` | `qualified=true deterministic=true` |
| `lastgc` with `--choices --replay-successes`, seeds 11 and 17 | `choice-replay=exact` on all four replays; `divergence=stdout.full_sha256` |
| `control` with `--choices --replay-successes`, seeds 11 and 17 | `match=true choice-replay=exact` on all four replays |

The choice-traced runs used `--success-limit 4 --success-bytes 64MiB`. Outputs and qualification
reports are retained under
`.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d21-*`; the campaigns were deleted.

The pair shows three things. The escape reaches stdout when a target prints it. Qualification
classifies that as nondeterministic without any audit. The runtime's collection and read paths, run
with stamps that differ between runs, change neither the choice tape nor any other evidence. The
pair does not show that a target is safe when it branches on the value: the control's one predicate
has the same outcome on every run by construction. The fixtures were not run on linux/amd64.

## Exposure in the Temporal functional workloads

The closure of `./tests` under the generator's build tags (`disable_grpc_modules`, `gomad`,
`test_dep`) links every reader: `expvar`, `net/http/pprof`, `runtime/pprof`, `runtime/trace`,
`runtime/metrics`, `runtime/debug`, and `github.com/prometheus/client_golang/prometheus`. By source
inspection none of them emits or consumes the host stamp in these workloads. No suite was run to
observe this.

| Reader | State in the functional workloads |
| --- | --- |
| `go_memstats_last_gc_time_seconds` | The Prometheus client registers its Go collector on the default registry at package initialization (`client_golang@v1.21.0/prometheus/registry.go:60-62`). The gauge is computed only inside `Collect` (`go_collector.go:252-260`), which runs on a gather of that registry. Temporal builds its own registry (`common/metrics/opentelemetry_provider.go:66`), and the test cluster uses the capture or no-op handler (`tests/testcore/onebox.go:380-385`). Nothing gathers the default registry. |
| `RuntimeMetricsReporter` | Calls `ReadMemStats` (`common/metrics/runtime.go:44`, `:66`) and records `NumGC`, `PauseTotalNs`, `PauseNs`, and heap sizes (`:68-96`). It reads neither `LastGC` nor `PauseEnd`. |
| pprof and expvar handlers | The test cluster starts the pprof listener on port 7000 (`tests/testcore/test_cluster.go:315-316`), which serves the default mux and therefore `/debug/pprof/heap` and `/debug/vars`. No test dials that port or requests those paths. |
| Execution tracer | No server or test code starts it. |
| `debug.ReadGCStats`, `.LastGC`, `.PauseEnd` | No use in the server or `tests`; the only closure use on either platform is the Prometheus collector above. |
| FIPS entropy | The qualification manifests do not set `fips140`. The closure links `crypto/rand` on both platforms, so Runner-managed targets install the testing reader. |
| Block and mutex profiles (`cputicks`, linux/amd64) | Neither closure calls `SetBlockProfileRate` or `SetMutexProfileFraction`, and the qualification manifests pass no profile flags, so both rates stay zero. Neither closure writes a block or mutex profile; the one `pprof.Lookup` call requests the goroutine profile (`common/deadlock/deadlock.go:131`). The pprof listener would serve both profiles on request, and nothing requests them. |
| `Gettimeofday`, `ClockGettime`, `syscall.Time`, `Sysinfo` | No caller on darwin/arm64 outside modernc libc, which the adapter answers. On linux/amd64 the only caller is the gRPC file the adapter replaces. |

The server's `gomad` build seams do not exclude any of these readers. They cover interrupts,
membership, local IP, the persistence password, cloud archiver and Elasticsearch AWS clients, SQL
errno handling, the worker's WCI component, and the SQL test flag. Exposure is absent because the
readers are pull-only and nothing pulls them. A workload that scrapes the default Prometheus
registry, requests the text heap profile or `/debug/vars`, or logs `LastGC` would become
nondeterministic.

The retained listings
[`fn105-d21-tests-closure-readers-darwin-arm64.txt`](../../../.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d21-tests-closure-readers-darwin-arm64.txt)
and
[`fn105-d21-tests-closure-readers-linux-amd64.txt`](../../../.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d21-tests-closure-readers-linux-amd64.txt)
search the non-standard Go files that `go list -deps -test` selects for each platform with
`CGO_ENABLED=0` and the generator's tags (4,335 and 4,359 files). They honor build constraints and
show module sources before Gomad's adapters rewrite them; the two adapter effects that matter are
stated in the table above. They are source closures computed on darwin/arm64. A prepared or linked
linux/amd64 target was not built or analyzed here, so the linux rows rest on the source closure, the
adapter sources, and the retained `temporal-functional-tests-linux-amd64` pack, not on a fresh
capability analysis.

## Relation to D12 and D14

The four inventoried escapes are not a cause of either divergence. The conclusion does not extend to
`cputicks`.

- The stamped fields have no control-flow reader, so they cannot change scheduling, GC pacing, or
  allocation (source proof above).
- The fixture pair shows the choice tape and replay stay exact while the stamps differ.
- The symptoms differ in kind. D12 diverges at choice ordinals and D14 swaps two heap-span refills. A
  leaked stamp appears as a differing value in output, with the choice tape intact.
- The functional workloads never emit the stamps.
- FIPS entropy and the tracer are unreachable in those workloads. The D12 bisect that ruled out the
  FIPS DRBG draw is independent evidence for the first.

The linux/amd64 half of this conclusion rests on source reasoning: `mgc.go` and `mstats.go` are
platform-neutral, and the platform difference is confined to what `time·now` returns. It was not
confirmed by a linux run. This finding does not resolve D12 or D14.

`cputicks` is left open for D12. By source, with both profile rates at zero and no block or mutex
profile written, its one remaining reading is stored and never consumed, and no code in the
functional closure raises the rates or writes those profiles. That
has not been checked on a linux run, so it is an expectation, not evidence. It cannot bear on D14:
on darwin/arm64 `cputicks` is the virtual clock.

### Adjacent lead for D12 and D14

This is not a host-clock path and it is untested. It surfaced while tracing host-timed inputs.

The overlay gives an M that holds a P a process-wide `cheaprand` stream; an M without a P keeps
its own (`runtime/rand.go:243-247`, overlay `runtime/gomad.go:400-404`). It moves draws made at host-timed
moments onto per-M streams, because such draws "moved every later type-assertion cache fill and
semaphore ticket between same-seed runs" (overlay `runtime/gomad.go:406-420`). The patch applies that
to `unlock2Wake` (patch lines 81-92).

One draw of the same class can remain on the process-wide stream. When `lock2` is about to sleep on
a contended runtime lock it calls `gp.m.mLockProfile.start()` (`runtime/lock_spinbit.go:244`), which
calls `cheaprandn(gTrackingPeriod)` (`runtime/mprof.go:654-659`). The draw comes from the
process-wide stream only when the contending M holds a P at that moment; otherwise it is M-local.
Whether a runtime lock is contended at that instant depends on host timing. Later consumers of the stream include the type-assertion
cache refresh, which allocates (`runtime/iface.go:501-508`), semaphore tickets
(`runtime/sema.go:381`), and goroutine tracking sequences (`runtime/proc.go:5485`).

`mprof.go` is a prohibited file, but the call site is in `lock_spinbit.go`, which the patch already
modifies. Whether this draw occurs in a diverging run, and whether it explains either symptom, has
not been measured. Owner: D12 and D14.

## Remedies within the current prohibitions

The prohibitions are the file rules in
[`patch.go`](../../../tools/gomad3/toolchain/patch.go) (`prohibitedRuntimeArea`, which covers every
`mgc*` and `mstats*` file, `mprof.go`, and `heapdump.go`) and the rule against changing platform
assembly. The virtual time of the last collection is already stored in `memstats.last_gc_nanotime`
(`mgc.go:1437`); Gomad's `faketime` counts nanoseconds since 1970, so that field is the value a
remedy would report. A remedy must not call `gomadTimeNow`, which draws a tick under the `forward`
policy.

| Option | Covers | Cost and limits |
| --- | --- | --- |
| A. Document the limitation | Nothing at run time | Names the affected fields and surfaces in the contract. Qualification already reports an emitting target as nondeterministic. |
| B. Guard FIPS `monoTime` in `runtime/time.go` | FIPS escape on linux | One hunk in a file the patch already modifies. A frozen clock starves the jitter source, so FIPS mode with no testing reader would need its own decision. Low value for the inspected Temporal closure, which links `crypto/rand` and does not enable FIPS; the path stays reachable for a FIPS target that does not link `crypto/rand`. |
| C. Patch `debug.ReadGCStats` in `runtime/debug/garbage.go` | `GCStats.LastGC`, `GCStats.PauseEnd`, and the Prometheus gauge | Not a prohibited file, but it adds a patch-allowlist entry and leaves `MemStats`, the heap profile, and `expvar` uncovered. `PauseEnd` has no stored virtual counterpart. |
| D. Overwrite the stamps from `startTheWorldWithSema` in `runtime/proc.go` | All `LastGC` and `PauseEnd` readers on both platforms; the host read itself remains | `gcMarkTermination` stamps at `mgc.go:1436-1439` and restarts the world at `mgc.go:1531`, inside one stop-the-world. A hook there can copy `last_gc_nanotime` over both host stamps before any reader runs. `proc.go` is already patched. The hook writes collector-owned reporting fields from outside the collector files. |
| E. Guard `time_now` in `runtime/timestub.go` | All three runtime escapes on darwin only | Not a prohibited file. Leaves linux/amd64 unchanged, so the two qualified platforms would differ. |
| F. Report readers in closure analysis | Detection only | A new capability-analysis finding for field reads of `LastGC` and `PauseEnd`. Compatibility-pack facts bind modules and symbols; field selectors have no existing representation. |
| G. Deny the readers in packs | Would block the workloads | `ReadMemStats` is called by the server's metrics reporter and the Prometheus client is linked, so a symbol-level denial removes the Temporal functional closure. Rejected. |
| H. Adapter for the Prometheus client | The gauge only | The libc adapter's `Xgettimeofday` rewrite is the precedent. It would pin another exact module version for a value no workload gathers. Rejected. |

Option D stays within the file rules. Whether writing `memstats` reporting fields from `proc.go`
honors the intent that the patch never touches the collector is a judgement for the patch-policy
owner. It is listed here because no prohibited file changes, not because that judgement is settled.

## Options that need a policy decision

| Option | Change | Policy affected |
| --- | --- | --- |
| P1. Guard the read in `gcMarkTermination` | One hunk at `mgc.go:1432` replacing `time_now()` with the virtual value under Gomad | Relaxes the `mgc*` prohibition for one hunk. Smallest change, both platforms, every reader. [fn-106.2](../../../.flow/tasks/fn-106-gomad-close-the-remaining-tests-gaps.2.md) rejected it. |
| P2. Move linux/amd64 to the Go `time_now` | Build constraints in `timeasm.go`, `timestub.go`, and `time_linux_amd64.s`, a new linux/amd64 `walltime`, then the option E guard | linux/amd64 has no `walltime`: `timestub2.go` excludes the platform and `time_linux_amd64.s` defines only `time·now`. The Go `time_now` calls `walltime()` (`timestub.go:27`), so this option adds or refactors platform assembly, beyond changing constraints. Closes the tracer and FIPS escapes on linux as well. |
| P3. Guard inside `time·now` assembly | Activation check in `time_linux_amd64.s` | Adds logic to platform assembly. |

## Recommendation

1. Accept the limitation for now and state it in the contract (option A). Outside the contract:
   `MemStats.LastGC`, `MemStats.PauseEnd`, `debug.GCStats.LastGC`, `debug.GCStats.PauseEnd`, the
   surfaces that print them (text heap profile, `expvar` `memstats`, heap dumps, the Prometheus
   `go_memstats_last_gc_time_seconds` gauge), execution traces, block and mutex profiles on
   linux/amd64 (their sampling when a rate is set, the cycles-per-second line of the text format,
   and the converted durations of protobuf samples), and FIPS mode without the seeded reader.
   Affected claim: "supported runtime-controlled choices repeat" holds; "same-seed evidence repeats"
   does not hold for a target that emits these values.
2. Correct and widen the inventory: add `MemStats.PauseEnd` to the `gcMarkTermination` finding,
   record that FIPS `monoTime` is host time on linux/amd64 only and seeds the DRBG, and pin the two
   host-time paths the identifier set misses, darwin's `libc_gettimeofday` trampoline and `cputicks`
   (`RDTSC` on linux/amd64), recording `cputicks` as an escape that steers sampling when a profile
   rate is set and that text profiles, and protobuf profiles with samples, carry. Until then both
   are unpinned against toolchain drift.
3. Put option D to the patch-policy owner as the proposed fix. It closes every `LastGC` and
   `PauseEnd` reader on both platforms without changing a prohibited file, but it writes
   collector-owned state and needs that owner's judgement before any work starts. If the owner judges
   that it touches the collector in substance, the choice is between P1 and keeping the limitation.
4. Do not pursue B, C, E, F, G, or H. Each is partial, platform-asymmetric, or costs more than the
   exposure justifies.
5. Hand the adjacent lead to D12 and D14.

Owner of the subsequent decision: the fn-105 owner, acting as patch-policy owner. Items 1 and 2 are
documentation and inventory edits with no runtime change. Items 1 to 3 are open work in fn-105 until
decided; nothing here implements them.

## D11 determination

The evidence does not establish a need for the dynamic linux/amd64 clock audit. D11 stays deferred.

- Every standard-library reference to the five inventoried clock symbols on linux/amd64 is pinned
  and classified, and `TestPatchedRuntimeClockEntryPointsCheckActivationFirst` proves the guard
  order for `nanotime` and `time_runtimeNow`. The inventory does not pin `cputicks`, and the audit
  could not observe it either: `RDTSC` is not a syscall.
- No inventoried escape is read by runtime control flow, and qualification already detects a target
  that emits one. `cputicks` does steer profile sampling when a profile rate is set and reaches
  written block and mutex profiles, but no syscall audit can see it, so it gives no reason for this
  audit.
- The audit is a bounded runtime fixture. It does not examine a workload's closure; that is the
  capability analysis and the packs. The workload findings above therefore neither require nor
  replace it.
- The audit as specified would not observe the known escapes unless its seeded run completes a
  collection, starts the tracer, or enables FIPS. The darwin audit target calls `time.Now` 1,001
  times and never collects
  (`tools/gomad3/internal/gomadtool/conformance/testdata/clock_audit`), which is why it passes with
  the `gcMarkTermination` escape present.
- Its remaining value is dynamic confirmation on linux of what the static checks already assert.

Evidence that would revive it:

- A fix that removes or guards a linux clock read is selected (P1, P2, or P3). The fixture then
  becomes that fix's regression gate. Option D is not such a fix: it overwrites the stamps and leaves
  the `time_now` read at `mgc.go:1432` in place.
- D12's native linux investigation attributes a divergent event to a host-clock read.
- A toolchain upgrade adds a host-clock reference the inventory cannot classify.
- A compatibility pack admits a raw clock syscall for a qualified target.

Feasible scope under the current restrictions, if revived:

- **vDSO.** `runtime.vdsoClockgettimeSym` and `runtime.vdsoGettimeofdaySym` are push-linknamed
  (`runtime/badlinkname_linux.go`, `runtime/vdso_linux_amd64.go:29-30`). A direct-seed fixture can
  zero both after startup, which sends `nanotime1`, `time·now`, and `syscall.gettimeofday` to their
  raw `SYSCALL` fallbacks (`sys_linux_amd64.s:295-298`, `time_linux_amd64.s:88-96`,
  `asm_linux_amd64.s:66-68`). No runtime or assembly patch is needed.
- **Detection.** A seccomp filter installed by the fixture after its activation marker, or an
  external syscall tracer windowed on that marker, observes `clock_gettime`, `gettimeofday`, and
  `time`. The fixture is a conformance target run by direct seed, as the darwin one is, so it needs
  no compatibility-pack change and no generic syscall widening.
- **Controls.** An unseeded positive control must trip on `time.Now`. The seeded run must pass.
- **Retained paths the fixture must account for.**
  - Startup readings before the marker.
  - `gcMarkTermination`: keep collections out of the audited window, or assert the read as a second
    positive control. With the vDSO disabled every collection reaches the raw `clock_gettime`
    fallback (`time_linux_amd64.s:88-96`). That stays true under option D, so the read remains a
    retained path; only P1, P2, or P3 would let the expectation invert.
  - `gomadWallNanotime`: reached only with deterministic I/O active, so a direct-seed fixture avoids
    it; otherwise allow `CLOCK_MONOTONIC` from that caller.
  - The tracer and FIPS paths: leave both off.
  - `cputicks`: `RDTSC` is invisible to a syscall filter and stays outside the audit.
  - Timed waits such as futex timeouts and sleeps are not clock reads.
- **Limits.** The audit sees syscalls only. `cputicks` stays outside it, and so does any value the
  fixture does not exercise.
- **Platform.** Development and verification need a native linux/amd64 host and `core-linux` CI. No
  part of this was run. Seccomp and ptrace under a Rosetta container are unverified.

Next action: none for D11 until one of the revival conditions holds. An unavailable audit is not
passing evidence; the linux escape gate remains the static inventory.

## Limits

- No linux/amd64 run. Linux conclusions are source reasoning over platform-neutral Go and the cited
  assembly.
- No functional suite ran. Workload exposure is established by source and closure inspection, not by
  observing a suite's evidence.
- The closure reader listings honor build constraints but precede adapter rewrites, and no prepared
  or linked linux/amd64 target was analyzed.
- The fixtures do not cover a target that branches on a host stamp.
- The linux/amd64 `cputicks` profile paths were read, not run.
- The adjacent lead is a source observation without a reproducer.
- The jitter entropy source's behavior under a frozen clock was not exercised.
