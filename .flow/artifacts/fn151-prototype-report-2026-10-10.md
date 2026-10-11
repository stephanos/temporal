# fn-151 determinism prototype: results and go/no-go

Date: 2026-10-10. Host: shared linux/arm64 sandbox, 12 cores, 15 GiB RAM, load
average 5–67 from other sessions during measurement. Stock Go 1.27.1 with the
cooperative WASI overlay; Wasmtime 47.0.3 helper. Raw reports and logs:
`.flow/tmp/fn151-prototype/` (local, untracked). Reproduce with
`make -C tools/gomad_wasm prototype` (see the WASM README).

## Question

Does Temporal's unchanged test stack run deterministically under WASM with
controlled scheduling, time skipping and a simulated network, at a cost that
makes further investment worthwhile?

## Answer: yes — recommendation GO

Every gate passed. The same seed reproduces a whole Temporal one-box run byte
for byte (stdout, WASI transcript digest, choice trace, network decisions,
runtime terminal); a fresh guest forced through the recorded tape reproduces it
exactly; a different seed produces a different schedule. Long timers skip in
virtual time, and injected resets and partitions are survived and replayed.

## Gates

| Gate | Workload | Runs | Result |
| --- | --- | --- | --- |
| .5 scheduling | `runtime_probes` (runq, select, map, timers, GC, finalizers, entropy) | 4 seeds × 10 fresh + replay | identical per seed; 4/4 seeds distinct schedules; exact replay |
| .5 tape rejection | `choices` | missing, extra, altered tapes | each rejected with its divergence reason |
| .6 virtual time | `time_probes` | 3 seeds × 5 fresh + replay | 9h16m virtual per guest in 0.1–0.2 s; exact deadlines, ties, cancellation; runnable work never moves the clock |
| .7 network | `network_probes` TCP echo | 3 seeds × 5 fresh + replay + 2 controls | seeded delays; reset at 3 s; partition 6–8.5 s refuses dials; reseeding changes decisions; no model, no host calls |
| .6/.7 Temporal | one-box + SDK worker; workflow with 2 × (1 min sleep, activity failing once, 30 s backoff); reset all at 40 s, partition all 100–110 s | 2 seeds + replay | passes; 3 min virtual; 12 resets + 3 refused dials; replay identical |
| .8 unchanged | `tests/gomadfunctional` `TestFrontendSystemInfo` (default gate) | 2 seeds + replay | passes |
| .8 unchanged (selectable) | `./tests` suites: workflow, activity, child, update, timers | 2 seeds + replay each | all pass, including the update suite that the stock profile left unqualified |

## Cost

Whole default validation (`make prototype`, warm compile cache): **141 s** wall at
load ≈13 (184 s at load ≈5–9). Fixture gates take ~25 s in parallel.

| Guest | Wall per run | Virtual time | Host calls | Decisions | Clock jumps | Guest memory |
| --- | --- | --- | --- | --- | --- | --- |
| Temporal workflow fixture | 94 s (29 s alone) | 180 s | 376 k | 51 k | 3,104 | 169 MiB |
| `./tests` activity suite | 57–61 s | 13 s | 208 k | 39 k | 251 | 221 MiB |
| `./tests` update suite (41 tests) | 176 s | 22 s | 1.06 M | 194 k | 1,022 | 311 MiB |
| `TestFrontendSystemInfo` | 7–11 s | 0 | 18–23 k | 4.8 k | 0 | 142 MiB |

One-time costs per module: Go build 14–30 s; Cranelift compile into the
host-local cache 168–321 s and 4.6–5.6 GiB peak RSS for 240–340 MB modules
(1,008 s at load 67). With cached code a Temporal helper needs ~0.5–1.5 GiB.

Wall time is dominated by host round trips: every scheduler decision, select
result, clock read, poll and log write is a JSON frame between the Wasmtime
helper and the Go model process (~2,100 calls per virtual second in the
workflow). The one-box's own housekeeping (deadlock-detector pings, long-poll
expiry, matching user-data refresh, replication cleanup) wakes the guest about
17 times per virtual second, so cost scales with simulated time, not with
workflow logic.

## Fixes made during the prototype

- Select observations were validated by re-decoding the whole trace (quadratic);
  now linear — 7× faster Temporal guests.
- `choice.MaximumTraceBytes` raised to 1 GiB for whole-program guests; the
  native runner keeps its 64 MiB bound and canonical bytes.
- Helper callback cap raised; digest-only WASI transcripts bound host memory.
- Compiled-module cache in the helper (host-local, never evidence).
- Task 5's earlier watchdog failure is explained by host contention against a
  1-minute per-guest watchdog while every fresh guest recompiled its module;
  same-seed guests were otherwise identical. The compile cache and wider
  watchdogs remove the condition; it was not a determinism defect.

## Open gaps (for fn-156)

- **Performance**: move decision, clock and poll handling into the helper and
  ship traces once; batch or drop log writes. Expected order-of-magnitude gain.
- **Repeat budget**: Temporal gates use 1 fresh guest per seed plus an exact
  replay; the fixture gates' 10-fresh budget and 32×100 qualification move to
  fn-156.1.
- **Scheduling coverage**: map/GC/finalizer stream qualification, independent
  application entropy and the checkpoint profile remain fn-156.1; exploration,
  resume, minimization and CPU-loop non-progress remain fn-156.2. A CPU loop
  without blocking still starves other goroutines (no preemption).
- **Time**: the runtime's idle handshake, not World's arbiter, advances time.
- **Network**: one guest only; delays hold the writer; the model learns of
  connections only through traffic, so a reset affects connections that send
  after it; no backpressure, half-close or deadline modelling; the broker and
  multi-node work is fn-156.5.
- **Resources**: the 338 MB `./tests` module needs ~6 GiB free once to compile.
- Native qualification is untouched and remains deferred (fn-128/fn-149).

## Decision

Recommendation: **GO** — proceed with fn-156, starting with in-helper decision
handling to cut per-call cost, then fn-156.1/.2. Recorded under the owner's
2026-10-10 instruction to follow Claude's recommendations when in doubt; the
owner may override.
