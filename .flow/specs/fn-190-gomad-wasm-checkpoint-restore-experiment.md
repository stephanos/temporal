# WASM checkpoint restore experiment

## Goal & Context

Almost every functional execution under Gomad is cluster boot. The boot-only probe `TestFrontendSystemInfo` records about 4,500 choice decisions and costs as much wall time as whole suites, and forced-prefix exploration re-executes that boot for every candidate. A checkpoint that captures the guest at an inspectable boundary and restores it into a fresh instance would let a search branch from the first test-body decision in milliseconds instead of seconds, and would make a fault matrix over a small workload (start a workflow, complete it) something an engineer runs interactively.

The 2026-10-01 research rated live process snapshots a quarter-plus of platform-specific work. That verdict was about the native backend. Native was frozen on 2026-10-16, and the WASM backend changes the problem: the guest is linear memory, globals and tables with no host pointers; Go's wasm port keeps goroutine stacks in linear memory; and fn-180 already parks every application goroutine at a validated decision boundary. This spec is the bounded experiment that decides whether a checkpoint-backed frontier is worth building. It produces measurements and a go/no-go finding, not a product command.

Prerequisites already delivered: fn-180 stop-and-inspect (prefix cuts, the runtime-state witness, the debug goroutine and the inspection terminal) and the helper's inspection cut material in `wasmhost/src/{model,broker,namespace,engine}.rs`.

## Architecture & Data Models

- **Image capture** extends the fn-180 inspection stop with one additional built-in dump, `image`, available only for single-guest cooperative artifacts. It writes the guest's linear memory, module globals, tables and remaining fuel, together with the helper's existing inspection cut (clock run state, tape cursor, seeded draw counters, WASI descriptor table for read-only mounts, namespace, network and volume inspection material). The image is diagnostic output beside the replay result. It is never an artifact, never evidence and never part of any identity.
- **Restore** instantiates a fresh store from the same prepared module, overwrites linear memory, globals and tables from the image, restores the helper cut, and re-enters the guest so that the pending decision at the captured boundary is delivered and application execution continues. Restore lives in an experimental `gomadtool` subcommand in the WASM backend, not in `gomad replay`.
- **Two continuation modes.** *Recorded suffix*: continue under the remainder of the recorded tape and compare the result with an ordinary fresh replay of the same artifact. *Fresh suffix*: continue with a seeded continuation from the cut and compare with a fresh forced-prefix execution (the existing `ModePrefix` path) of the same prefix and the same continuation seed. The second mode is the one exploration would use.
- **Comparison** reuses the fn-180 cut verification and `gomadtool diagnostic-diff`: choice trace, WASI transcript digests, network and storage digests, stdout/stderr, terminal outcome and the compulsory runtime-state witness. A mismatch names the first differing ordinal and field.
- **Re-entry is the central risk.** At a decision boundary the guest is suspended inside a host import, so the wasm native stack is live and cannot be rebuilt for a fresh instance. Go's wasm port unwinds the native stack to its run loop on every goroutine switch and resumes from `PC_F`/`PC_B` state held in linear memory. The experiment must establish whether the fn-180 handoff can be arranged so that the stop point is reached with the native stack unwound and the pending decision delivered through a reserved linear-memory slot on resume, rather than as the import's return value. This is a guest runtime change under the existing cooperative runtime owner and must be dormant and identical in recording and replay.

## API Contracts

- `gomad replay --stop=<choice=N|time=T|failure> --dump=image ARTIFACT` writes `ARTIFACT.inspection/image/` containing the guest image, the helper cut and a manifest with sizes, the resolved boundary and the capture wall time. Refused before execution for cluster artifacts, Go-owned network recordings, artifacts without fn-180 inspection metadata, and stops that are not inspectable boundaries.
- `gomadtool checkpoint-probe --image DIR --mode recorded|fresh [--seed N] [--repeat K] ARTIFACT` restores the image, continues, and writes a comparison report: restore wall time, continuation wall time, the matched or first-divergent evidence field and ordinal, and for `fresh` mode the same comparison against a fresh forced-prefix execution. `--repeat` re-runs the restore and reports medians and ranges.
- `gomadtool checkpoint-probe --baseline` runs the fresh instance to the same boundary and records boot wall time, so every report carries the ratio the decision needs.
- Reports and images are research evidence under `.flow/artifacts/` and `docs/research/gomad/`. No new artifact schema, campaign state, plan field or identity input is introduced.

## Edge Cases & Constraints

- Linear memory may have grown after the cut in the baseline run; the restored instance must be sized from the image, and a `memory.grow` failure after restore remains an infrastructure outcome, not divergence.
- A cut inside a folded WASI clock run must restore the run state exactly; reuse the fn-180 folded-clock cut material and cover this case explicitly.
- Open descriptors for lazy read-only mounts are host state; the cut's descriptor table must be restored, and any read served after restore must come from the same captured entries.
- SQLite in-memory state lives in guest memory and needs nothing extra; file-backed volumes are helper state and are restored from the volume inspection material.
- The debug goroutine, registry, bitmask and emit buffer exist identically in every incarnation, so they are part of the image and need no special handling. Probes enabled during the inspection stop must be disabled again before continuation so the continuation matches a recording.
- Fuel remaining is restored exactly; a continuation that would exceed the recorded fuel limit reports that, as fn-180 does.
- Go's garbage collector state is in linear memory and restores with it. Runtime GC workers created by the inspection stop must not exist in the image; capture happens before any inspection allocation, or the experiment reports that this ordering is impossible and why.
- Nothing in this spec changes what a retained failure means. A failure found through a restored image is a lead; it is retained only after a fresh replay reproduces it.

## Implementation Design

### Fixture first (.1)

Use a tiny cooperative fixture with two goroutines handing off through a channel, one timer, one select with two ready cases, a few hundred bytes of fake-network traffic and one read from a read-only mount. Capture at three boundaries: a run-queue decision, a select decision and a boundary inside a clock run. Prove recorded-suffix equivalence at all three before touching Temporal. If re-entry cannot be made to work for the fixture, stop and write the finding; .2 to .4 are not started.

### Runtime re-entry (.1)

Prototype under the cooperative runtime owner (`toolchain/runtime_wasm.go.txt`, `toolchain/overlay.go`) using the fn-180 g0 handoff: at a stop requested with an image capture, the scheduler path records the pending decision request in a reserved slot, unwinds to the run loop and traps to the host instead of blocking inside the import. Resume re-enters the run loop, reads the delivered decision from the slot and continues through the ordinary decision path. The prototype must be dormant: a recording and an ordinary replay take the existing import path and produce identical evidence, which the compulsory witness checks.

### Helper restore (.2)

Add restore counterparts to the helper's inspection cut material for clock, tape cursor, draws, WASI descriptors, namespace, network and volume state. Restore is only ever applied to a fresh store created for the probe command. Existing replay and recording paths do not gain a restore entry point.

### Temporal cell (.3)

Run the WASM one-box qualification cell for `./tests/gomadfunctional TestFrontendSystemInfo` and one smoke suite. Identify the first test-body boundary by a reviewed method (the first decision whose site is in the test function, found with `inspect --choices`) and record it. Capture, restore in both modes, and measure on linux/arm64 with a warm cache, median of at least five runs, as the 2026-10-14 performance host decision requires. Report boot wall to the boundary, image bytes, capture and restore wall time, continuation wall time, and the ratio of boot to restore.

### Finding (.4)

Write `docs/research/gomad/<date>-wasm-checkpoint-restore.md`: what the image needed beyond linear memory, every mismatch found and its cause, the measurements, and a go/no-go recommendation. If go, include the product design the next spec would implement: a checkpoint-backed forced-prefix frontier that caches one image per prefix, binds image provenance to the prepared module and helper identities, and keeps retained failures fresh-replay verified. If no-go, name the state that cannot be captured and the cheapest alternative. Link the report from `docs/research/gomad/README.md` and `.plans/GOMAD_NEXT.md`.

### Verification

Fixture and runtime tests run with `-tags test_dep`, the Rust helper suites, `make -C tools/gomad validate` and the cooperative runtime qualification once on the integrated candidate, to show the dormant re-entry path changed no recording or replay evidence. Retain commands, exits and identities in the batch evidence handover. Plan review authorizes no implementation, push, PR or CI action.

## Acceptance Criteria

- **R1:** For a single-guest cooperative artifact, `gomad replay --dump=image` at a resolved inspectable boundary writes the guest image and helper cut with a manifest of sizes, boundary and capture time. Errors: cluster artifacts, Go-owned network recordings, artifacts without inspection metadata and non-inspectable stops are refused before execution; a capture that would require inspection allocations before the image is taken is refused with that reason.
- **R2:** For the fixture, restoring an image and continuing under the recorded suffix produces evidence equal to a fresh replay at a run-queue boundary, a select boundary and a boundary inside a clock run: choice trace, WASI transcript digests, network and storage digests, stdout/stderr, terminal outcome and runtime witness. Errors: any mismatch reports the first divergent ordinal and field; a restore into an instance whose memory cannot be sized from the image is an infrastructure outcome.
- **R3:** For the fixture, restoring an image and continuing with a fresh seed produces evidence equal to a fresh forced-prefix execution of the same prefix and seed, for at least eight seeds at each of the three boundaries. No error surface beyond R2.
- **R4:** The dormant re-entry path leaves recording and ordinary replay evidence unchanged: the cooperative runtime qualification and the compulsory runtime-state witness pass on the integrated candidate with identical choice traces and transcript digests before and after the change.
- **R5:** For the Temporal one-box boot probe and one smoke suite on linux/arm64, the report records boot wall time to the first test-body boundary, image bytes, capture, restore and continuation wall times as medians of at least five warm runs, and both continuation modes either match or name their first divergence. No error surface beyond R1 to R3.
- **R6:** A research report records the findings, measurements and a go/no-go recommendation with the follow-on product design or the blocking state, and is linked from the research index and the capability roadmap.

## Boundaries

- No product CLI verb, campaign strategy, artifact schema, plan field or identity change. Images are diagnostic research output and are never published as artifacts or admitted to a corpus.
- Single-guest artifacts only. Multi-instance cluster checkpoints wait for the finding.
- No native backend work; native stays frozen under fn-184 and deferred under fn-128 and fn-149.
- No visited-state pruning, image deduplication, copy-on-write or memory optimization.
- No change to the meaning of a retained failure: fresh replay remains the evidence contract.
- Performance acceptance is linux/arm64 only; Darwin or emulated measurements are informational.

## Decision Context

Owner request (2026-10-16): checkpoint a run before an interesting point, freeze it, and restore it to play several scenarios from there, eliminating the start-up cost. Earlier research (2026-10-01 feasibility notes, `.plans/GOMAD_CMP.md` "Checkpoints have four meanings") rated live snapshots as long and platform-specific for the native backend and asked for a prefix-cost study first; the retained reports already show boot dominates `testcore` executions. The native backend was frozen on 2026-10-16 and fn-180 delivered stop-and-inspect, so the freeze half exists and the open question is restore and re-entry. This spec is deliberately an experiment with a kill switch after the fixture, so the runtime re-entry risk is settled before any Temporal or product work.
