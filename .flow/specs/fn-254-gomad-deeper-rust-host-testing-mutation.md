# Gomad: deeper Rust host testing (mutation, fuzzing, deterministic concurrency)

## Problem

The Rust host is tested through golden and e2e fixtures plus named-invariant unit tests. Four kinds of defect slip through that:
- **Tests that would pass with the code broken.** fn-242 R1 requires every kept check to name a mutation it catches, but those mutations are written by hand (`source-mutations.json`).
- **Decoders that crash or hang on corrupt input.** About 110 decode/parse sites read retained evidence and wire formats.
- **Interleaving bugs in hand-written threading.** There are ≈55 `thread::scope`/`spawn` sites, ≈300 atomics, ≈90 channel and condvar uses and ≈54 `Mutex`/`RwLock`, concentrated in `wasmhost/src/linked*.rs`, `linked_cluster`, `linked_cancellation`, `event_order`, `gomad/src/backstop.rs`, `slots.rs`, `cycle/process` and `campaign/launcher`. Ordinary tests explore only the schedules the OS happens to pick.
- **Undefined behaviour in the 48 `unsafe` sites.**

Counts are from a grep on 2026-10-25; task .1 replaces them with a real survey.

## Goal

Add the tools the owner rated +1 to +3 on 2026-10-25, each where it catches something the fixtures cannot:

| Tool | Rating | Tier |
|---|---|---|
| cargo-mutants | +3 | scheduled, plus an on-demand run for changed code |
| decoder fuzzing (cargo-fuzz / bolero) | +3 | scheduled |
| shuttle | +2 | scheduled |
| proptest | +2 | inner loop (small case count) and scheduled (large) |
| Miri | +1 | scheduled |
| loom | +1 | scheduled |
| ThreadSanitizer | +1 | occasional, nightly toolchain |

Rejected the same day: Kani (0), turmoil/madsim (-1) and a tokio host (-3).

## Constraints

- **Cleanup first (owner ordering, 2026-10-25).** Start fn-254 only after fn-240 closes, including its fn-242 driver-cutover and fn-247/fn-248/fn-249 cleanup prerequisites. New mutation, fuzzing and concurrency infrastructure targets the resulting crate/API structure. Focused tests needed to validate ongoing cleanup continue in their owning tasks. fn-242 remains parked until explicit owner resume; this ordering does not resume it or waive qualification.
- **Fast gates first (fn-253, MILESTONES).** Nothing here enters the fast inner loop or the pre-merge gate except small proptest runs, and nothing here enters the one-hour cycle. The deep tier runs as `mise run deep` (fn-253.4 entrypoints) on a schedule or on demand.
- **Every finding becomes a regression fixture.** A crash input, a surviving mutant that gets fixed, or a failing seed/schedule is kept as a golden fixture, or as a named-invariant test that the normal suites run.
- **Test-only seams.** The shuttle/loom sync shims are swapped in by `cfg`, so the production build doesn't change. No production code paths are added for testability.
- **Determinism is unaffected.** No tool changes host or guest identities; instrumentation builds use separate target directories.
- **No new Python.** Harnesses are Rust; tool versions are pinned in `tools/gomad/mise.toml`.

## Acceptance Criteria

- **R1:** A survey (`.flow/artifacts/fn-254/survey.md`) lists, per tool, the modules it targets and why, along with what it can't reach (e.g. Miri can't execute wasmtime JIT code). Errors: a tool adopted without named targets is a finding.
- **R2:** cargo-mutants runs over the workspace on schedule and over a diff on demand (`--in-diff`). The surviving mutants of the first full run are triaged: a test is added, or the mutant is recorded as equivalent with a reason. Errors: an untriaged survivor older than one scheduled run is a finding.
- **R3:** Each decoder of retained evidence or wire data has a fuzz target seeded from the golden fixtures. A fuzz run never panics, hangs past a bound, or allocates without bound; malformed input is refused with a typed error. Errors: a decoder without a target, or a crash input not kept as a fixture, is a finding.
- **R4:** The threading modules named in R1 run under shuttle (randomised schedules, replay from seed). Small primitives named in R1 run under loom (exhaustive). Errors: a reported schedule that is not replayable from its seed is a finding.
- **R5:** Codec round-trip, canonical-uniqueness and refusal invariants are proptest properties, each naming its invariant. The default case count fits the fn-253 inner loop, and the scheduled tier runs a large count.
- **R6:** Miri runs over the non-JIT modules containing `unsafe`; ThreadSanitizer runs occasionally over the threading tests. Both are documented with the modules they cover and their known false-positive suppressions.

## Decision Context

Owner, 2026-10-25: asked for "advanced testing tools in Rust ... like Loom or deterministically", then approved every option rated +1 to +3. Tokio was rejected (-3):
- Guest execution is synchronous and CPU-bound.
- It would expose wasmtime's async-path advisories.
- Its scheduler is non-deterministic.
- It conflicts with the fn-232 R1 process-level backstop and with the fn-240 cuts.
