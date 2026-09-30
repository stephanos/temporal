# Gomad v3: Milestones to a Deterministic Temporal Functional Test

**Plan date:** 2026-09-08 · **Trimmed:** 2026-09-30

## Purpose

This document is the delivery ladder for running Temporal's functional tests (`./tests`, built on
the `testcore` one-box cluster with in-memory SQLite and loopback gRPC) under Gomad v3 so that the
same seed produces the same run and a retained artifact replays byte-exactly.
[GOMAD3_NEXT.md](GOMAD3_NEXT.md) remains the capability roadmap across all four tracks.

Milestones F0–F9 are complete and were removed from this document on 2026-09-30. Their outcomes,
details, and status history are in revision `3e4303807` of this file; manifests and exclusions
that cite a finding as `GOMAD_MILESTONES.md#f3-…` through `#f7-…` refer to the sections of that
revision. What remains here is the open work, the open findings, and the rules that still apply.

## Work tracking

Open work is tracked as flow-next specs under `.flow/specs/`; their tasks and acceptance criteria
(R-IDs) are authoritative, and this document keeps rationale and status. When a spec's scope
changes, change the spec and summarize the change here.

| Milestone | Spec | State |
| --- | --- | --- |
| F7+ | `fn-103-gomad-seeded-virtual-clock-ticks` | open; `.1` done (`forward` and `strict` tick policies), `.2` open (measure the timestamp-tie suites under `forward`, decide the default) |
| F10 | `fn-105-gomad-follow-ups-deferred-scope` | backlog; each item with a revival trigger ([GOMAD_FOLLOWUPS.md](GOMAD_FOLLOWUPS.md)) |

Work a spec with `/flow-next:work <spec>`; list what is ready with `flowctl ready`.

## Open findings

- **Linux seed-17 replay divergence.** Since the FIPS DRBG draw (`6bc11ef7d`) and the mark-start
  greying fix (`440552d2c`) landed, seed 17 of one tier-3 suite per linux run has diverged on
  replay near choice ordinal 9100–9400, a different suite each time (`functional-activity` in fork
  run 36668156879, `functional-query` in 36669836359). The F5 and F6 suites are `intermittent` on
  linux under this finding, the dispatch-only linux gate accepts `nondeterministic` for them, and
  the required smoke gate runs seed 11 only. The darwin representative set stayed fully qualified
  on the same commits. Bisecting the two runtime changes on linux/amd64 is the next step.
- **Host-clock escapes** recorded by the static inventory (`toolchain/clock_inventory_test.go`):
  `gcMarkTermination` stamps `MemStats.LastGC` with host wall time (also visible through
  `debug.GCStats` and the Prometheus `go_memstats_last_gc_time_seconds` gauge), the FIPS entropy
  source's `monoTime`, and the execution tracer's clock snapshot; on linux also
  `syscall.Gettimeofday` behind the `syscall` pack gate. `LastGC` is the one ordinary targets
  reach.
- **`TestDescribeTaskQueueEnhanced_ReportFlags`** (versioning suite) keeps a deterministic failure
  ("poller info should not be reported") that is not yet shown to be a test bug.
- **DTrace clock audit** on darwin needs a root run; CI supplies it on the macOS runner.
- **Downstream cell.** Gomad-side work for a module that embeds the server is complete; its
  in-process cluster test classifies as a capability blocker until the downstream cuts its own
  seams and injects its filesystem and membership transport ([GOMAD_CLOUD.md](GOMAD_CLOUD.md)).

## Constraints

- **No policy widening.** Gomad never grants `syscall`, `os/exec`, `os/signal`, or
  `golang.org/x/sys` generically. Every exception is an exact compatibility pack bound to a
  module version, go.sum hash, per-file SHA-256, owner, and workload, reviewed under the
  existing `discover`, `review`, `generate --approve-review`, `check`, `qualify` flow.
- **No source translation and no test rewriting.** Determinism comes from the patched
  toolchain and the reviewed boundary. A Temporal test that needs a Gomad-specific overlay of
  its own source, as gomad1 required, is a blocker to record, never a fix to ship.
- **Fail-closed stays.** An unmodeled boundary operation terminates the process. Work that needs
  a new modeled operation adds it with a semantic contract, a resource bound, transcript
  coverage, exact replay, and a negative test, per COMPAT-5 in
  [GOMAD3_NEXT_COMPATIBILITY.md](GOMAD3_NEXT_COMPATIBILITY.md).
- **Evidence over narration.** Work is done when its command produces the stated report on a
  clean checkout. A passing local run that depends on untracked state does not count.
- **Platform.** The boundary manifest qualifies `darwin/arm64` and `linux/amd64`. Each platform
  is its own qualification and artifacts replay only where they were produced. The macOS sandbox
  test and the DTrace clock audit remain `darwin/arm64` only. Each platform's compatibility packs
  are its own; `compatibility-pack-qualification` qualifies the requests that name the host.
- **Server source changes are allowed but bounded.** A change under `common`, `service`,
  `temporal`, or `tests/testcore` is acceptable when it isolates an optional provider behind a
  build tag or an injection seam and the default build is unchanged. A change that alters
  runtime behavior for production builds needs its own review outside this plan.
- **Validation scope.** The full `./tests` set is not run as a gate; a change is validated on the
  smoke selection plus the suites it affects, with `make gomad3-tests-qualification` as the
  on-demand local run.

## F10: follow-ups (deferred scope)

**Spec.** [fn-105-gomad-follow-ups-deferred-scope](../.flow/specs/fn-105-gomad-follow-ups-deferred-scope.md);
items and revival triggers in [GOMAD_FOLLOWUPS.md](GOMAD_FOLLOWUPS.md).

**Outcome.** None required. F10 is a backlog, not a milestone to finish: it keeps the scope cut on
2026-09-29 in one place, each item with its origin, why it was deferred, and the observation that
would revive it.

| Item | Origin | Revive when |
| --- | --- | --- |
| D1–D5 architecture consolidation | F8 R2–R6 | a second consumer or exploration strategy hits the duplication |
| D6 `seeded` and `fixed` tick policies | `fn-103` | a bug class needs deliberate ties or constant quanta |
| D7 macOS smoke job | F7 `.4` | a darwin-only regression escapes to main |
| D8 closure-mode downstream support | F9 C3/R2 | a downstream module needs closure-mode preparation |
| D9 linux/amd64 downstream packs | F9 | a downstream gate must run in linux CI |
| D10 downstream-seam guide | F9 R4 | a second downstream module adopts Gomad |
| D11 dynamic linux clock audit | F7 R5 (pre-amendment) | a linux-only clock escape is observed |

**Constraints.** An item is worked only after its trigger is recorded here; its acceptance is the
origin spec's requirement text. Items may be closed as won't-do.

**Status.** Created on 2026-09-29 with eleven tasks; nothing started.

## Out of scope

- New fault injection, partition, crash-restart, and multi-node capabilities through
  `tools/gomad3sim`. Those are the simulation track and cannot host `testcore`.
- Multi-P scheduling, deterministic GC, DPOR, and preemption bounding. These are BUG-7 research
  items.
- Revival of gomad1 or gomad2. [GOMAD_CMP.md](GOMAD_CMP.md) records why.

## Open risks

- **GC timing** is not controlled and shares the seeded runtime stream. Allocation-heavy suites
  may diverge between repetitions; the open linux seed-17 finding may be one.
- **Spin loops** anywhere in the cluster stall virtual time. Pollers with backoff are fine, but a
  single `for {}` with a non-blocking select is fatal under this runtime.
- **Upstream Go releases** invalidate the patch and the boundary manifest each time; every Go bump
  repeats the port and requalification.
- **Compatibility packs pin exact module versions.** Every dependency bump that touches a packed
  or adapted module invalidates the pack or adapter and reopens the capability closure.
