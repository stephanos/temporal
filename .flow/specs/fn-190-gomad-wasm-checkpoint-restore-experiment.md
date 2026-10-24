# WASM checkpoint restore experiment

## Goal & Context

Almost every functional execution under Gomad is cluster boot. The boot-only probe `TestFrontendSystemInfo` records about 4,500 choice decisions and costs as much wall time as whole suites, and forced-prefix exploration re-executes that boot for every candidate. A checkpoint that captures the guest at an inspectable boundary and restores it into a fresh instance would let a search branch from the first test-body decision in milliseconds instead of seconds, and would make a fault matrix over a small workload (start a workflow, complete it) something an engineer runs interactively.

The 2026-10-01 research rated live process snapshots a quarter-plus of platform-specific work. That verdict was about the native backend. Native was frozen on 2026-10-16, and the WASM backend changes the problem: the guest is linear memory, globals and tables with no host pointers; Go's wasm port keeps goroutine stacks in linear memory; and fn-180 already parks every application goroutine at a validated decision boundary. This spec is the bounded experiment that decides whether a checkpoint-backed frontier is worth building. It produces measurements and a go/no-go finding, not a product command.

Prerequisites already delivered: fn-180 stop-and-inspect (prefix cuts, the runtime-state witness, the debug goroutine and the inspection terminal) and the helper's inspection cut material in `wasmhost/src/{model,broker,namespace,engine}.rs`.

## Planning status and prerequisites

Rewritten onto the Rust host on 2026-10-24 (owner request). The `gomadtool` command set and the Go-host replay are gone; `gomad replay --stop=… --dump=…` is the Rust inspection path today (`cli/replay.rs`, `replay/inspection.rs`, helper `linked_inspection`). Acceptance criteria R1–R6 are unchanged from the 2026-10-16 capture. This remains an experiment with a kill switch: a demonstrated impossible re-entry is a valid no-go result retained under R6, not a pass for R1–R5.

Owners:
- Image capture: the Rust replay inspection path, one additional built-in dump `image`.
- Helper snapshot and restore: `wasm-sim` (clock, network, volume, simulation controller) and `wasm-model` (namespace, tape cursor, draw state, descriptors, evidence accumulators) from fn-240. Snapshot and restore are crate-boundary operations: each crate exposes `snapshot() -> Image` and `restore(Image)` for its own state, tested at its boundary. This is why the experiment waits for fn-240.7.
- Restore and probe verb: an experimental developer verb `checkpoint-probe` in the `gomad` bin, compiled only with the determinism-fixture feature set so it is absent from release binaries, never in `gomad replay`.
- Runtime re-entry: the cooperative runtime overlay (`toolchain/runtime_wasm.go.txt`, `toolchain/overlay.go`), dormant unless an image capture is requested.

Sequencing: after fn-240.7 (`wasm-model` extracted; `wasm-sim` exists from .3). Spec-level dependency on fn-240 recorded.

## Architecture & Data Models

- **Image capture** extends the fn-180 inspection stop with one additional built-in dump, `image`, available only for single-guest cooperative artifacts. It writes the guest's linear memory, module globals, tables and remaining fuel, together with a versioned experimental operational helper snapshot (clock run state and pending folds, tape cursor, seeded draw state, descriptor/open-file offsets and rights, namespace contents, network buffers/connections, volume bytes/dirty state and evidence accumulation state). Existing inspection cuts/commitments verify this state but are not sufficient restore inputs. The image is diagnostic output beside the replay result. It is never an artifact, never evidence and never part of any identity.
- **Restore** instantiates a fresh store from the same prepared module, overwrites linear memory, globals and tables from the image, restores the helper cut, and re-enters the guest so that the pending decision at the captured boundary is delivered and application execution continues. Restore lives in the experimental `gomad checkpoint-probe` developer verb (fixture-feature builds only), not in `gomad replay`.
- **Two continuation modes.** *Recorded suffix*: continue under the remainder of the recorded tape and compare the result with an ordinary fresh replay of the same artifact. *Fresh suffix*: continue with a seeded continuation from the cut and compare with a fresh forced-prefix execution (an experimental extension of `ModePrefix` with an explicit continuation-seed boundary) of the same prefix and the same continuation seed. The second mode is the one exploration would use.
- **Comparison** reuses the fn-180 cut verification in the Rust inspection path: choice trace, WASI transcript digests, network and storage digests, stdout/stderr, terminal outcome and the compulsory runtime-state witness. A mismatch names the first differing ordinal and field.
- **Re-entry is the central risk.** At a decision boundary the guest is suspended inside a host import, so the wasm native stack is live and cannot be rebuilt for a fresh instance. Go's wasm port unwinds the native stack to its run loop on every goroutine switch and resumes from `PC_F`/`PC_B` state held in linear memory. The experiment must establish whether the fn-180 handoff can be arranged so that the stop point is reached with the native stack unwound and the pending decision delivered through a reserved linear-memory slot on resume, rather than as the import's return value. This is a guest runtime change under the existing cooperative runtime owner and must be dormant and identical in recording and replay.

## API Contracts

- `gomad replay --stop=<choice=N|time=T|failure> --dump=image ARTIFACT` writes `ARTIFACT.inspection/image/` with the guest image (linear memory, globals, tables, remaining fuel), the helper snapshot from each sim and model crate, and a manifest with sizes, the resolved boundary and capture wall time. Refused before execution for cluster artifacts, artifacts without inspection metadata, and stops that are not inspectable boundaries.
- `gomad checkpoint-probe --image DIR --mode recorded|fresh [--seed N] [--repeat K] ARTIFACT` (fixture-feature builds only) restores the image into a fresh store, continues, and writes a comparison report: restore and continuation wall time, matched or first-divergent evidence field and ordinal, and for `fresh` mode the same comparison against a fresh forced-prefix execution of the same prefix and continuation seed. `--repeat` reports medians and ranges. `--baseline` runs the fresh instance to the same boundary and records boot wall time.
- Comparison reuses the existing inspection cut verification: choice trace, WASI transcript digests, network and storage digests, stdout/stderr, terminal outcome and the compulsory runtime-state witness.
- Reports and images are research evidence under `.flow/artifacts/` and `docs/research/gomad/`. No new artifact schema, campaign state, plan field or identity input is introduced.

## Edge Cases & Constraints

- Linear memory may have grown after the cut in the baseline run; the restored instance must be sized from the image, and a `memory.grow` failure after restore remains an infrastructure outcome, not divergence.
- A cut inside a folded WASI clock run must restore the run state exactly; reuse the fn-180 folded-clock cut material and cover this case explicitly.
- Open descriptors for lazy read-only mounts are host state; the cut's descriptor table must be restored, and any read served after restore must come from the same captured entries.
- SQLite in-memory state lives in guest memory and needs nothing extra; file-backed volumes are helper state and require the operational volume snapshot, not a hash-only inspection commitment.
- The debug goroutine, registry, bitmask and emit buffer exist identically in every incarnation, so they are part of the image and need no special handling. Probes enabled during the inspection stop must be disabled again before continuation so the continuation matches a recording.
- Fuel remaining is restored exactly; a continuation that would exceed the recorded fuel limit reports that, as fn-180 does.
- Go's garbage collector state is in linear memory and restores with it. Runtime GC workers created by the inspection stop must not exist in the image; capture happens before any inspection allocation, or the experiment reports that this ordering is impossible and why.
- Nothing in this spec changes what a retained failure means. A failure found through a restored image is a lead; it is retained only after a fresh replay reproduces it.

## Implementation Design

### Re-entry prototype and fixture (.1)

Resolve the kill switch first. Prototype under the runtime overlay owner, using the fn-180 g0 handoff: at a stop requested with an image capture, the scheduler path records the pending decision request in a reserved linear-memory slot, unwinds to the run loop and traps to the host instead of blocking inside the import; resume re-enters the run loop and reads the delivered decision from the slot. The prototype is dormant: a recording and an ordinary replay take the existing import path and produce identical evidence. Fixture: two goroutines handing off through a channel, one timer, one select with two ready cases, a few hundred bytes of fake-network traffic, one read from a read-only mount. Capture at a run-queue decision, a select decision and a boundary inside a clock run, and prove recorded-suffix equivalence at all three before touching Temporal. If re-entry cannot be made to work, skip .2–.4 and run .5 to retain the no-go finding.

### Crate snapshots and image capture (.2)

Add `snapshot`/`restore` at the `wasm-sim` and `wasm-model` boundaries for clock run state and pending folds, tape cursor, seeded draw state, descriptor offsets and rights, namespace contents, network buffers and connections, volume bytes and dirty state, and evidence accumulators; each is a boundary test in that crate (snapshot, mutate, restore, compare commitment). Add the `image` dump to the Rust inspection path. Capture happens before any inspection allocation, or the dump is refused with that reason. Existing replay and recording paths gain no restore entry point.

### Restore, fresh-suffix seed contract and probe verb (.3)

The `checkpoint-probe` verb instantiates a fresh store from the same prepared module, overwrites linear memory, globals and tables, restores each crate from its snapshot, compares the restored commitment with the captured one, and re-enters. Fresh suffix: at the validated cut an experimental runtime hook resets only the scheduling-choice PRNG streams to the continuation seed; clock, entropy, model state and prior evidence are unchanged; the forced-prefix baseline applies the same hook at the same cut. Both paths use the same instrumented module.

### Temporal cell (.4)

Run the one-box cell for `TestFrontendSystemInfo` and one smoke suite with an explicit test-only probe marker just after testcore setup and a controlled handoff to an inspectable decision. Report marker-to-cut distance, boot wall time, image bytes, capture, restore and continuation wall times as medians of at least five warm runs on one host in one window.

### Finding (.5)

Write `docs/research/gomad/<date>-wasm-checkpoint-restore.md`: what the image needed beyond linear memory, every mismatch and its cause, the measurements, and a go/no-go recommendation with either the product design for a checkpoint-backed forced-prefix frontier or the blocking state and the cheapest alternative. Link it from the research index and `.plans/GOMAD_CMP.md`.

### Verification

Fixture and overlay tests, `cargo test --workspace` including the new crate boundary tests, `make -C tools/gomad validate`, and the cooperative runtime qualification once on the integrated candidate to show the dormant path changed no recording or replay evidence. No PR, push or CI run is authorized by this spec.

## Acceptance Criteria

- **R1:** For a single-guest cooperative artifact, `gomad replay --dump=image` at a resolved inspectable boundary writes the guest image and helper cut with a manifest of sizes, boundary and capture time. Errors: cluster artifacts, Go-owned network recordings, artifacts without inspection metadata and non-inspectable stops are refused before execution; a capture that would require inspection allocations before the image is taken is refused with that reason.
- **R2:** For the fixture, restoring an image and continuing under the recorded suffix produces evidence equal to a fresh replay at a run-queue boundary, a select boundary and a boundary inside a clock run: choice trace, WASI transcript digests, network and storage digests, stdout/stderr, terminal outcome and runtime witness. Errors: any mismatch reports the first divergent ordinal and field; a restore into an instance whose memory cannot be sized from the image is an infrastructure outcome.
- **R3:** For the fixture, restoring an image and continuing with a fresh seed produces evidence equal to a fresh forced-prefix execution of the same prefix and seed, for at least eight seeds at each of the three boundaries. No error surface beyond R2.
- **R4:** The dormant re-entry path leaves recording and ordinary replay evidence unchanged: the cooperative runtime qualification and the compulsory runtime-state witness pass on the integrated candidate with identical choice traces and transcript digests before and after the change.
- **R5:** For the Temporal one-box boot probe and one smoke suite on a recorded host with matched baseline/restore measurements in the same window, the report records boot wall time to the first test-body boundary, image bytes, capture, restore and continuation wall times as medians of at least five warm runs, and both continuation modes either match or name their first divergence. No error surface beyond R1 to R3.
- **R6:** A research report records the findings, measurements and a go/no-go recommendation with the follow-on product design or the blocking state, and is linked from the research index and the capability roadmap.

## Boundaries

- No product CLI verb, campaign strategy, artifact schema, plan field or identity change. Images are diagnostic research output and are never published as artifacts or admitted to a corpus.
- Single-guest artifacts only. Multi-instance cluster checkpoints wait for the finding.
- No native backend work; fn-223 removed the native backend and fn-128/fn-149 are closed.
- No visited-state pruning, image deduplication, copy-on-write or memory optimization.
- No change to the meaning of a retained failure: fresh replay remains the evidence contract.
- Performance acceptance is host-independent; retain the host identity and matched same-host/window measurements, repetitions, median/range and gain-versus-spread labels.

## Decision Context

Owner request (2026-10-16): checkpoint a run before an interesting point, freeze it, and restore it to play several scenarios from there, eliminating the start-up cost. Earlier research (2026-10-01 feasibility notes, `.plans/GOMAD_CMP.md` "Checkpoints have four meanings") rated live snapshots as long and platform-specific for the native backend and asked for a prefix-cost study first; the retained reports already show boot dominates `testcore` executions. The native backend was frozen on 2026-10-16 and fn-180 delivered stop-and-inspect, so the freeze half exists and the open question is restore and re-entry. This spec is deliberately an experiment with a kill switch after the fixture, so the runtime re-entry risk is settled before any Temporal or product work.


## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | `--dump=image` writes guest image, helper snapshot and manifest; refusals. | fn-190-gomad-wasm-checkpoint-restore-experiment.2 | — |
| R2 | Fixture recorded-suffix equivalence at three boundaries. | fn-190-gomad-wasm-checkpoint-restore-experiment.1, fn-190-gomad-wasm-checkpoint-restore-experiment.3 | — |
| R3 | Fixture fresh-suffix equivalence, eight seeds per boundary. | fn-190-gomad-wasm-checkpoint-restore-experiment.3 | — |
| R4 | Dormant re-entry path leaves recording and replay evidence unchanged. | fn-190-gomad-wasm-checkpoint-restore-experiment.1, fn-190-gomad-wasm-checkpoint-restore-experiment.5 | — |
| R5 | Temporal one-box measurements, both continuation modes. | fn-190-gomad-wasm-checkpoint-restore-experiment.4 | — |
| R6 | Research report with go/no-go and follow-on design or blocking state. | fn-190-gomad-wasm-checkpoint-restore-experiment.5 | — |

