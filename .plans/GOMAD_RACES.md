# Detecting lock-discipline and data-race bugs with Gomad

**Date:** 2026-10-22. **Scope:** the WASM cooperative backend. **Worked example:**
[temporalio/temporal#10549](https://github.com/temporalio/temporal/pull/10549),
`CacheSize()` reading lock-guarded state after `Unlock()`.

This plan records why Gomad cannot find the bug class behind PR 10549 today, what
the research on disk already says about it, and four investments in dependency
order with the smallest version of each. It does not track tasks; Flow-Next specs
own those. [GOMAD_FIND_BUGS.md](GOMAD_FIND_BUGS.md) holds the bug-finding loop this
extends, [GOMAD_NEXT.md](GOMAD_NEXT.md) the roadmap, and
[GOMAD_CMP.md](GOMAD_CMP.md) the research comparison.

## 1. The worked example

### The bug

With `history.cacheSizeBasedLimit` enabled (default off), the history service can
die with `fatal error: concurrent map iteration and map write`. Go treats this as a
runtime fatal, not a panic, so the `recover()` in `makeReleaseFunc` cannot catch it.

The release path in `service/history/workflow/cache/cache.go` calls
`wfContext.Unlock()` and then `c.Release(cacheKey)`. The LRU's `Release`
(`common/cache/lru.go:256-281`) recomputes the entry size through `getSize`, which
calls `ContextImpl.CacheSize()`, which calls `updateRegistry.GetSize()`, which
ranges over the registry's `updates` map
(`service/history/workflow/update/registry.go:492-499`). The registry has no lock
of its own; the only guard is the context's `lock locks.PrioritySemaphore`
(`service/history/workflow/context.go:51`). A waiter that acquires the workflow
lock between the `Unlock()` and the range loop writes to that map through
`FindOrCreate`. The author reproduced it on SQLite with 2,000 concurrent updates
to one workflow.

A second variant never crashes. `lockWorkflowExecution` on a failed acquisition
releases the cache entry, so `CacheSize()` reads `MutableState` size while another
goroutine holds the lock. The only symptom is a stale number.

### The fix

`ContextImpl.RefreshCacheSize()` recomputes the estimate into an `atomic.Int64`
under the lock; `CacheSize()` returns the stored value and touches no shared state.
`makeReleaseFunc` calls `RefreshCacheSize()` before `Unlock()` on every path and
`NewContext` seeds the value. The method is added to the `WorkflowContext`
interface and its mock.

### Why the broken and fixed variants matter to Gomad

The feasibility study
([§7, Bug benchmark](../docs/research/gomad/2026-10-01-feasibility-schedule-search.md))
records that Gomad has no "known bug plus corrected counterpart" pairs and no table
of which bug families are reachable. PR 10549 is a clean pair: one method differs,
the trigger is a dynamic-config flag plus two concurrent updates, and the family
is one the research already predicts Gomad cannot reach. It belongs in that table
as a `reachable: no` row until one of the investments below lands.

## 2. Why Gomad cannot find it today

### The crash needs an interleaving the scheduler never produces

Go's swiss map toggles `m.writing` around every write
(`internal/runtime/maps/map.go:514, 684, 764` in the pinned Go 1.27.1 tree) and
`Iter.Next`, `mapaccess` and `mapassign` fatal when they observe it set. The
fatal therefore requires the iterator to step **while the writer is inside
`mapassign`**, a window of a few dozen instructions. On real hardware with many
Ps parallelism reaches it; that is why 2,000 updates were needed.

Gomad's cooperative profile keeps one logical processor, disables asynchronous
preemption, and asks the helper for a decision only at goroutine creation,
run-queue dispatch, and select poll/result
([ARCHITECTURE.md, Scheduling](../tools/gomad/docs/ARCHITECTURE.md)). The overlay's
rewrite list in `tools/gomad/internal/backends/wasm/toolchain/overlay.go:255`
touches `proc.go`, `select.go`, `rand.go`, `time.go`, `runtime2.go`,
`note_other.go`, `lock_wasip1.go` and the WASI files. It never touches
`malloc.go`, `mgc*.go` or `internal/runtime/maps`. A goroutine inside `mapassign`
can therefore never be descheduled. The only theoretical park on that path is a
GC assist (`gcParkAssist` in `mgcmark.go`), and
[MILESTONES.md](../MILESTONES.md) lists GC timing as uncontrolled. That is luck,
not exploration.

The consequence is sharper than "low probability". The racing accesses are not
separated by any choice point, so **no reordering of existing decisions reaches
the crash**. A seed sweep has probability zero.

### Gomad has no other way to see the bug

- No data-race detection. `ARCHITECTURE.md` lists it under "What is not claimed";
  `-race` does not exist for `wasip1`. Without a crash the bug has no symptom.
- No lock-discipline oracle. Nothing in Gomad knows that `updateRegistry` is
  guarded by `lock`, so the stale-size variant is invisible by construction.
- The bug sits behind a dynamic-config flag that is off by default, and the
  qualified one-box cells
  ([QUALIFICATION.md](../tools/gomad/docs/wasm/QUALIFICATION.md))
  run frontend smoke tests, not concurrent updates on one workflow.

### The research already names this class

- Academic survey, Uber data-race study entry
  ([2026-10-01-academic-concurrency-testing.md:296](../docs/research/gomad/2026-10-01-academic-concurrency-testing.md)):
  "these are memory-level races. Under Gomad's voluntary-switch model they cannot
  manifest unless the racing accesses are separated by a blocking operation. They
  need the race detector or added yield points, not better scheduling." The
  unsafe built-in map is one of the study's named causes.
- Concurrency-tools note
  ([2026-09-30-concurrency-tools.md:47](../docs/research/gomad/2026-09-30-concurrency-tools.md)):
  "Recording an ordered schedule alone does not establish happens-before edges."
- Feasibility study §6 inventories why `-race` is unreachable and sizes a race
  profile as **L**, "probably not worth it relative to keeping stock `-race` as a
  separate lane." Its file references point at the retired `tools/gomad3` layout;
  its cgo objection is moot on WASM, where ThreadSanitizer does not exist at all.
- The deferral is recorded three times: [GOMAD_FIND_BUGS.md §8](GOMAD_FIND_BUGS.md),
  [the vision note](../docs/research/gomad/2026-10-01-gomad-vision.md) under
  "Deliberately not pursued", and [GOMAD_CLOUD.md](GOMAD_CLOUD.md) ("race detection
  is a separate unsupported execution profile"). The BUG-7 table in
  [GOMAD_NEXT.md](GOMAD_NEXT.md) frames the same gap as "single-P exploration
  does not cover racy reads or weak-memory behavior."

### Why stock `-race` as a separate lane did not catch it either

Temporal runs `-race` in CI, with a dedicated `notify-data-race` job in
`.github/workflows/run-tests.yml`. It missed this bug because no test enabled the
size-based limit and sent concurrent updates to one workflow. The PR's new tests
exist to give `-race` something to look at. Any detector, static or dynamic, still
needs a workload that reaches the code. Section 3 below is therefore required by
every other section.

## 3. Workload reach: a functional test that exercises the path

A functional test on the one-box cell that overrides `history.cacheSizeBasedLimit`
to `true` and sets a small `history.hostLevelCacheMaxSizeBytes` through
`testcore`'s `OverrideDynamicConfig`
(`tests/testcore/dynamic_config_overrides.go`), then issues a handful of
concurrent `UpdateWorkflowExecution` calls against one workflow using the helpers
in `tests/update_workflow_test.go` and `tests/update_workflow_utils.go`. The
small cache limit makes releases recompute sizes and evict.

This test is the carrier for every detector below. On its own, under stock
`-race` natively, it finds the bug. Under Gomad today it passes.

**Size:** S. **Owner:** Temporal server test suite, selectable by Gomad through
the per-test support list.

## 4. Investment A: choice points inside map writes

### Design

Add a `gomadChoiceMapWrite` hook in the runtime overlay at the `m.writing ^= 1`
sites of `internal/runtime/maps/map.go`. While the flag is set, record a
Runnable-style decision whose alternatives are "continue" and "yield to another
user goroutine". The stock checks then do the detection: the iterator or reader
hits the existing `fatal("concurrent map iteration and map write")`, and Gomad
retains it as an ordinary target failure with an exact replay tape. The same
hook exposes "concurrent map read and map write" and "concurrent map writes".

Constraints:

- User goroutines only (`isSystemGoroutine(gp, false)`); skip when runtime locks
  are held; never inside `systemstack`.
- Bound the choice space: a seeded per-write probability or a per-execution
  budget, recorded in the tape so replay forces the same yields. Every map write
  in Temporal becomes a potential decision, so the frontier rules in
  [SPEC.md `[RUNTIME.CHOICES]`](../tools/gomad/docs/SPEC.md) and the frozen-plan
  `Frontier` need the bound.
- Any overlay change reopens the determinism inventory and requalification.

### Effect on the example

Two concurrent updates suffice instead of 2,000. The updater parks at its
`mapassign`; the releaser runs `Release` and iterates; the fatal fires. A seed
sweep of a few dozen should find it with the yield probability tuned toward the
releaser. That is an estimate, not a measurement; the benchmark row in §1 is how
it gets measured.

### Limits

Only bugs that pass through Go's built-in map checks. The stale-size variant and
any plain-field race stay invisible. It is a confirmation tool, and in §5 it
becomes a *targeted* one.

**Size:** S to M, overlay-only, a few hundred lines plus bounding and
requalification.

## 5. Investment B: happens-before and runtime-owned footprints

### What the trace lacks

Choice kinds today are goroutine creation, runnable dispatch, select poll and
select result (`ChoiceKind*` in
`tools/gomad/internal/model/runtimesource/gomad_choicewire_generated.go.txt`).
There are no records of lock acquire/release, channel operations, or memory
accesses. Without happens-before every pair of decisions looks dependent, which
is why the feasibility study rules out DPOR: tens of thousands of decisions per
execution with no footprint to tell which matter.

### Step one: record happens-before edges

The runtime already sees the synchronization that matters. Temporal's workflow
lock is a `sync.Mutex` plus a ready channel
(`common/locks/priority_semaphore_impl.go:49, 118`), so the edges are
`semacquire`/`semrelease` keyed by semaphore address, `chansend`/`chanrecv`
keyed by channel address, and `newproc`. Record them as observation records on
the existing choice wire (which has `ChoiceFlagObservation` for records that are
evidence but never forced) and keep a vector clock per goroutine. Thousands of
goroutines make naive vector clocks expensive; FastTrack epochs keep the common
case O(1).

Blind spot: `sync/atomic`. The runtime does not see it, so edges through atomics
are missing and produce false positives. The feasibility study §1 notes atomic
hooks are possible overlay-only; they are a follow-up, not a prerequisite.

### Step two: footprints for data structures the runtime owns

Full shadow memory needs compiler instrumentation. The built-in map is runtime
code, and the overlay already rewrites runtime files. Record
`(goroutine, map pointer, read|write, pc)` at `mapaccess*`, `mapassign*`,
`mapdelete*` and `mapiterinit`, and keep a last-write epoch plus read set per
map in a field added to the map header, the same way the overlay already added
`gomadID` and friends to `g` (`overlay.go:270`). This is a FastTrack detector
restricted to maps: a few loads per map operation instead of a shadow word per
8 bytes of heap. Channels and runtime-visible locks get the same treatment for
free since their operations are already the edges.

### Effect on the example

One seed with the §3 workload. The releaser's `mapiterinit` in `GetSize` and the
updater's `mapassign` in `FindOrCreate` touch the same map. The releaser's
`Unlock` orders it before the updater's `Lock`; nothing orders the updater's
write before the releaser's later read. The detector reports the pair with both
program counters and both goroutine creation sites. No crash is needed. The
report also names the forcing schedule, which turns Investment A into a targeted
confirmation: park the updater at the reported `mapassign`, run the releaser.
This is the predict-then-rewrite loop the academic survey singles out for
ADVOCATE as the closest existing Go tool to Gomad.

### Effect on search cost in general

- **Happens-before fingerprint per execution.** Two seeds with the same relation
  explored the same equivalence class; stop spending seeds on it. The survey
  already proposes this as a corpus deduplication key.
- **Conflict-directed decisions.** With footprints the scheduler knows which
  runnable goroutines touch the same object. Morpheus reports that more than 90%
  of operations never conflict and can be dispatched without a choice. Applied to
  the Runnable decision this shrinks the frontier to the decisions adjacent to a
  conflicting access, which is the "enabled sets and dependency relation"
  prerequisite the BUG-7 table lists for PCT and DPOR.
- **Near-miss worklist.** Unordered access pairs to the same map that happened
  not to interleave badly are candidates, ranked by trace distance, for targeted
  replay instead of a seed sweep.

### Limits

Only maps, channels and runtime-visible locks get footprints; a plain struct
field read without the lock, like the `MutableState` size read in the
lock-failure path, stays invisible until compiler instrumentation exists. The
recording changes allocation and timing, so it is its own profile identity, like
the diagnostic trace, and not part of exact-replay identity. The full detector
the feasibility study rated **L** stays **L**.

**Size:** M for happens-before recording, M for map footprints, both
overlay-only.

## 6. Investment C: checklocks annotations as a static gate and a dynamic oracle

gVisor's `checklocks` (`gvisor.dev/gvisor/tools/checklocks`) is a `go/analysis`
pass driven by comments: `// +checklocks:mu` on a struct field means the field is
only accessed with `mu` held; `// +checklocks:c.mu` on a function is a
precondition; `// +checklocksacquire` / `// +checklocksrelease` mark the methods
of any type as a lock; `// +checklocksignore` and `// +checklocksalias` handle
what the analysis cannot see. Temporal does not use it today (no reference in
`go.mod`, `Makefile` or `.golangci.yml`).

### As a static gate

Annotate `MutableState` and `updateRegistry` in `ContextImpl` with
`// +checklocks:lock`, and `PrioritySemaphore.Lock`/`Unlock` with acquire and
release markers. `CacheSize()` reads both fields with no lock held and no
precondition; checklocks reports it at analysis time, with **no test, no
workload and no execution**. The analysis is per-function, so the fact that the
LRU reaches `CacheSize` through the `SizeGetter` interface does not matter. For
this bug the static gate is the cheapest possible detector.

Costs and gaps, from gVisor's experience and this code:

- Annotation effort is the main cost, and `MutableState` is an exported field
  that tests assign directly; those sites need `+checklocksignore`.
- Acquire and release are split across functions here: `lockWorkflowExecution`
  acquires, the closure from `makeReleaseFunc` releases. checklocks cannot carry
  lock state into a closure, so that closure needs an ignore or a forced
  assertion, and the fixed code's `RefreshCacheSize()` precondition is checked
  only where the analyzer can see the lock.
- `PrioritySemaphore.Lock(ctx, priority)` can fail. checklocks models a plain
  acquire; the error path is a false-positive pattern to annotate around.
- It finds *guarded-field* violations only, where someone wrote the guard down.
  It says nothing about maps shared by convention, atomics, or channels.

### As a dynamic oracle inside Gomad

The annotations are machine-readable facts: "field F of type T is guarded by
lock L". Gomad can consume them instead of inventing its own lock-discipline
model:

- A `sync` overlay (the build overlay already replaces non-runtime files such as
  `net/net_fake.go`) records the owning goroutine on `Mutex.Lock`/`Unlock`,
  including the uncontended CAS fast path the runtime never sees. Custom lock
  types marked `+checklocksacquire`/`release` get the same treatment through a
  Gomad-only hook.
- A source-level rewrite, driven by the analyzer's own facts, inserts
  `gomad.AssertHeld(&c.lock)` before each guarded-field access in Gomad builds.
  The analyzer already computes the access sites; the rewrite only emits them.
- The assertion fires on the **first sequential execution** that reaches
  `CacheSize()` with the flag on. No concurrency, no interleaving, no second
  updater. Where the static pass is unsure (closures, interface calls, aliases)
  the dynamic check is exact, because it tests actual ownership.

### As a dependency relation for search

Guarded-by facts say which goroutines can conflict: those contending the same
lock on the same object. That is a free, static independence relation for the
conflict-directed dispatch in §5, weaker than footprints but available before any
runtime recording exists.

### Verdict

checklocks is complementary, not an alternative. It catches the lock-discipline
family, which includes both variants of PR 10549, earlier and cheaper than any
dynamic tool, and it only catches that family. Investment B catches the
unannotated and convention-guarded cases, and the search-cost gains in §5 do not
depend on annotations. The order that pays soonest is: §3 test, checklocks on the
history service's workflow context and update registry, then A and B.

**Size:** S for the analyzer in CI once the annotation debt on the history
service is paid; that debt is M and is Temporal's, not Gomad's. The dynamic
oracle is M, overlay-only plus a rewrite tool.

## 7. Investment D: a race detector inside Gomad (deferred, restated)

A general-purpose happens-before detector finds this bug on almost any schedule,
because it reports the unsynchronized pair rather than waiting for the crash,
and it catches the silent variant too. The portable half exists: the gc
compiler's `-race` instrumentation only inserts calls to `raceread`,
`racewrite`, `raceacquire`, `racerelease` and friends. The non-portable half,
`runtime/race`, is ThreadSanitizer as a `.syso`, absent for wasm.

What it would take:

- Lift two gates that reject `-race` on `wasip1/wasm`:
  `cmd/go/internal/work/init.go:156` and
  `cmd/compile/internal/base/flag.go:297`. Gomad patches only the runtime
  today, so this means patching the pinned compiler or driving
  `go tool compile -race` from preparation.
- A pure-Go `runtime/race` for wasm. Under one P the algorithm is simpler than
  TSan's: no atomics on shadow state, no real parallelism. Shadow memory for a
  4GiB wasm32 heap is the design question; a per-8-byte shadow word quadruples
  memory, so sparse shadow over heap spans or allocation sampling is needed and
  both trade recall for cost.
- Instrumented code runs several times slower and allocates more, which changes
  measured capacities under virtual time and fuel and needs its own
  qualification cells.

Scope and acceptance criteria for this profile live in
[fn-241](../.flow/specs/fn-241-gomad-a-race-profile-for-wasip1-deferred.md),
deferred with all tasks blocked (owner decision, 2026-10-23: one switchable
profile is preferred over Investments A, B and the dynamic half of C).

The deferral stands. Investments B and C deliver most of the value for this bug
family at a fraction of the cost. Revival trigger: a bug in the §1 benchmark
table that B and C both miss, with a workload that reaches it.

**Size:** L.

## 8. Order of work and what each step proves

| Step | Investment | Proves | Size |
| --- | --- | --- | --- |
| 1 | §1 benchmark row plus §3 functional test | PR 10549 is `reachable: no` under Gomad today and `reachable: yes` under native `-race` | S |
| 2 | §6 static checklocks on `ContextImpl` and the update registry | The lock-discipline family is caught at analysis time; annotation debt measured | S plus Temporal annotation debt M |
| 3 | §4 map-write choice points | The crash variant becomes a replayable Gomad failure; seeds-to-first-failure measured on the benchmark row | S to M |
| 4 | §5 happens-before edges and map footprints | One-seed prediction of the pair, happens-before fingerprint dedup, conflict-directed dispatch | M plus M |
| 5 | §6 dynamic lock-ownership oracle | Stale-size variant found on a sequential execution | M |
| 6 | §7 race profile | Only if the benchmark table shows misses after steps 2 to 5 | L |

Every step after the first reports its result as a change to the benchmark
row, so the deferred-detector decision in §7 is measured rather than argued.

## 9. Where this sits against GOMAD_FIND_BUGS.md

This plan is not the strongest way to prove Gomad's value, and it should not be
scheduled as if it were.

### Why the race class is a weak showcase

- For memory races Gomad is the fourth-best detector. checklocks finds PR 10549
  statically; stock `-race` finds it with one reaching test. Gomad needs
  Investment B, two M-sized overlay changes, to match them. A proof of value
  should pick bugs that only Gomad finds.
- The research on disk says workload quality and decision-space shape limit
  results before runtime sophistication. Building footprint recording before
  fixing either is the order it warns against.
- There is no known-bug table, so no value claim is measurable yet. That is
  finding number one in GOMAD_FIND_BUGS.md.
- The map race is the hard case for a cooperative scheduler by construction: the
  racing accesses sit between choice points. Hard-for-Gomad is the wrong place
  to demonstrate Gomad.

### What ranks higher

1. **The persistence fault matrix crossed with schedule exploration**
   ([GOMAD_FIND_BUGS.md §3](GOMAD_FIND_BUGS.md)). Fail call k of store method M
   with ShardOwnershipLost, timeout, conditional-failed or unavailable during a
   start-and-complete workflow, with bounded schedule exploration around the
   fault. Days, no toolchain change, existing oracles (history equality, soft
   assertions, no leaked goroutines). No other tool has a notion of "a
   shard-ownership-lost on the third update during task completion". The
   calibration step, reintroducing the update-admission regression and counting
   seeds to catch it, is the first row of the missing table.
2. **The wake-up yield after `Unlock`, `close` and `cancel()`** (finding two in
   GOMAD_FIND_BUGS.md and the vision note). Today a woken goroutine never yields
   there, so unlock-wake-order, close-versus-send and cancel-versus-complete bugs
   are unreachable under any search policy. Those are scheduling bugs, the class
   Gomad should own outright, and three of the six benchmark fixtures the
   research lists. Cheaper and more general than Investment A. It does not reach
   PR 10549, because the writer completes before the reader runs; that is what
   marks the map race as the hard case.
3. **An Omes scenario body as an in-process target** (GOMAD_FIND_BUGS.md §1).
   throughput_stress against a `testcore` cluster through the SDK adapter Gomad
   already carries, with the mixedbrain log scanners as oracles. This is how the
   workload gets thick enough for any of the above to find something.

### What to keep from this plan now

- Steps 1 and 2 of §8: the `reachable: no` benchmark row and checklocks on the
  workflow context. Both are cheap, and a negative row that states plainly what
  Gomad cannot do is worth more to credibility than another positive claim.
- The happens-before fingerprint from §5, on its own merits, once seeds are
  visibly being wasted on equivalent schedules. It cuts cost in every campaign,
  not only race hunting.

Defer Investments A, B and the dynamic lock-ownership oracle until the fault
matrix saturates and the benchmark table shows schedule bugs being missed.

## 10. Decision context

Owner conversation on 2026-10-22, in order: whether Gomad could find PR 10549
(no, and why); what would need to change (map-write yield points, workload
reach, a lock-discipline oracle); whether a general-purpose race detector would
help (yes, with the caveat that Temporal's own `-race` CI missed it for lack of a
reaching test); whether the question was already covered on disk (it is, in the
documents cited in §2, which defer the detector and name this exact bug class);
whether potential race interleavings or the realistically explorable scenarios
can be noticed from an execution to cut search time (§5); and whether gVisor
checklocks annotations would improve the result (§6); what unique advantage Gomad
holds over those tools (forcing, exact replay, oracles over retained evidence,
virtual time, composition with faults); and whether this is among the most
promising ways to prove Gomad's value (no, see §9). This document was written at
the owner's request to collect that deliberation.
