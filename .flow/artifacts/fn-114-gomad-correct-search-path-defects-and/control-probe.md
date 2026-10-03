# Runtime-owned local queue choices (fn-114 task 13)

Native darwin/arm64 evidence, 2026-10-02. The explicit control starts two user
goroutines; each completes 64 iterations with a scheduler yield. Unlike the
historical D21 probe, its source contains two deliberate user starts.

## Cause and measured reduction

The unchanged controller selected the runtime parentless ordinal 1 identity
among two or three alternatives in 31 decisions over 32 seeds. Runtime init's
first parentless creation is `forcegchelper`; the creation inventory and pinned
`runtime/proc.go` establish that attribution. The original regression fails with
`runtime-owned goroutines selected in 31 branching decisions` in
`task-13/baseline-final-fixture.log`. No runtime correction preceded this test.

`task-13/measure.go` then builds exactly the same control source with both
retained native toolchains, records seeds 1–32, and validates every payload and
terminal using `choice.DecodeTrace`. The source snapshot is
`task-13/measurement-main.go`; `same-source-counts.json` binds its hash and each
trace to its toolchain build key.

| Measurement | Before | After |
| --- | ---: | ---: |
| Runnable decisions per seed | 70–72 | 65 |
| Runnable decisions over 32 seeds | 2,261 | 2,080 |
| Decisions selecting a runtime parentless identity | 31 | 0 |
| Trace payload bytes over 32 seeds | 217,056 | 199,680 |

The saving is 181 records, or 17,376 payload bytes (8.01%). Terminal and backing
capacity bounds are unchanged. Before build key:
`2008ea81459afbd1ee019beb4a231968c1af8a24f4b46d4a8426b3188b1a9b87`.
After build key:
`4a6e5b695ea538f0a56eb70874ff945693223b53d0fcee56cc555f89e1a9ac0e`.

## Dispatch and classification

The queue head determines the class. A runtime-owned head runs deterministically;
a user head selects only user alternatives. Selecting a user swaps it with the
head and advances the head as before. Runtime entries advance without competing
in seeded decisions, while yielding runtime work rejoins behind queued users.
Always giving runtime work priority would fail this class fairness condition.
A user dispatch with two or more users records one decision; every other local
dispatch records none. Recording disabled, replay, and prefixes use the same rule.

Classification uses `isSystemGoroutine(gp, false)` without allocation. The task 5
inventory covers force-GC, sweep, scavenge, GOMAXPROCS updater, mark-worker,
finalizer, cleanup, signal, tracing, and timer callback creation sites.
Finalizer and cleanup goroutines executing user callbacks are users under the
runtime's dynamic classification; timer callbacks are users too. This does not
qualify finalizer, cleanup, signal, or tracing workloads. Collector workers picked
outside the local queue, run-next, the global queue, and timer delivery retain
their existing rules. The contract states those limits in `[RUNTIME.SCHEDULING]`.

## Acceptance evidence

`task-13/focused-fairness-final.log` passes the final fixture and preceding
controller rejection tests. `task-13/final-runtime-owned.json` records 32 seeds
for each of zero, one, two, and busy modes. Zero and one record no decisions and
equal traces. The two-user modes validate every alternative-set digest against
subsets of main and the two observed user identities, including alternatives
that were never selected.

The busy worker has a fixture-only runtime symbol and no arguments, so its
start PC is a runtime function, rather than a generated user wrapper. Runtime
and both users must each progress through 64 mutually gated steps before any
can finish. This passes all 32 seeds under the subprocess watchdog. The earlier
argument-taking worker compiled through a user wrapper; the alternative test
caught the resulting third user identity (`task-13/focused.log`).

Ordinary two-user replay also succeeds under a different seed. Busy replay uses
the recorded seed, as Runner replay does: global/system interleaving is outside
the Decision Tape, and the mutual-progress fixture's cross-seed experiment
diverged (`task-13/focused-fairness.log`). Both modes pass a forced prefix that
changes the first user choice. The retained pre-change tape validates under its
original identity and is rejected when only the controller identity is changed;
the test holds target, platform, and build key fixed to isolate that rejection.
Protocol generation incorporates the edited runtime overlay and patch in the
controller source hash.

`task-13/probes.tar.gz` retains all raw probe backings, terminal frames, tapes,
binaries, and compact same-source measurements. Every archived file was verified
against `task-13/probes-manifest.json` before removing the loose copies.
`task-13/source-hashes.json` binds the final source snapshot.

Native full gates passed: `make -C tools/gomad3 validate test-toolchain
test-runtime overlay-test` returned status 0, recorded in
`task-13/required-gates.log`. All nine overlay packages passed.
The gate started before the busy fixture's mutual-progress strengthening; the
final focused run supplements that fixture snapshot without changing runtime
source or the toolchain key. The stock Go 1.27.1 PATH is selected explicitly and
inherited GOROOT/GOBIN are unset. Native linux/amd64 has not been executed and
remains unverified.
