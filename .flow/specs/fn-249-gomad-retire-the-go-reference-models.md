# Gomad: retire the Go reference models and the stdio helper protocol

## Goal & Context

The Rust host runs the engine in-process through `gomad_wasmhost::linked` and `linked_cluster`. Two parallel structures remain from the Go-host era.

- **Go reference models.** `tools/gomad/internal/model` holds about 37k production and 28k test lines of Go (2026-10-24): world, volume, network, cluster, wasi, clock, choice. Outside `internal/model` only two non-test Go files import it: `internal/backends/wasm/toolchain/overlay.go` (uses `runtimesource`) and `internal/support/temporalinput/schema.go` (uses `capture`). Everything else is a test oracle that conformance tests compare against the Rust models.
- **The stdio helper protocol path in `wasmhost`.** JSON/base64 frames (`protocol.rs` `read_frame`/`send_frame`/callbacks), `cluster/execution/manual.rs`, the `transport*` modules and their corpora, broker and storage frame I/O, the `reference-helper` binary (`main.rs`) and the `conformance` binary (`conformance.rs`, copied to `gomad-model-conformance`). Their consumers are the Go tests (`GOMAD_WASM_TEST_HELPER` in `internal/model/wasi/measure_process_test.go`, `make test-wasm`) and `wasmhost/tests/helper_corpus.rs`. `linked` still builds the wire struct `protocol::Execute` for in-process calls.

Owner, 2026-10-24: file the complexity review's items 1 and 2. A model change is already "a change to a corpus, reviewed as an expectation diff" (`ARCHITECTURE.md`). Checked-in corpora can carry that role without a second implementation.

## Architecture & Data Models

- **Kept Go:** `internal/model/runtimesource`, `internal/model/capture` and whatever the toolchain overlay or `temporalinput` reach transitively. Also `protocolgen` and the schemas for guest-boundary wires (`choicewire`, `timewire`, `net_stream_status`, iowire if the overlay uses it), `internal/archcheck`, and `internal/backends/wasm/toolchain`.
- **Conformance becomes goldens.** Each Go↔Rust differential test becomes a Rust corpus test over checked-in inputs and expected outputs, generated once from the current Go oracle and reviewed as expectation diffs afterwards. Corpus formats follow fn-247 where it applies.
- **In-process API only.** `linked` and `linked_cluster` take typed in-process requests and return typed outcomes. Helper corpora that drove the stdio binary drive `linked` instead, with the same inputs and expectations. The `reference-helper` and `conformance` Cargo features and binaries are deleted, so fn-240 R8's remaining features shrink to `guest-profile`.

## API Contracts

- Public `gomad` verbs, records and outputs are unchanged.
- **Stage plan change (needs owner acceptance in fn-246):** cycle stages and leaf commands that build `reference-helper`/`conformance`, run the Go model tests or run `helper_corpus` are removed or replaced by their Rust corpus equivalents. The change is recorded as an explicit stage-plan delta (`.flow/artifacts/fn-249/stage-delta.md`) mapping each removed stage to its replacement coverage. fn-246 receives the reduced plan, and its "same 96 stage contracts" wording is amended by that delta, not silently bypassed.

## Edge Cases & Constraints

- No behaviour is lost. Every deleted Go test and every helper corpus case maps to a Rust test that still catches the same failure, or to a recorded reason it's obsolete. A mutation that the old test caught must still be caught.
- Capture the goldens from the current Go oracle at an exact revision before deleting the Go model, and record that revision.
- The determinism inventory, `internal/archcheck` inventories, Makefile targets, `ARCHITECTURE.md`, `README.md`, `CLI.md` and `wasmhost/protocol.md` are updated. `protocol.md` shrinks to the in-process contract or is deleted.

## Acceptance Criteria

- **R1:** An inventory lists every `internal/model` package with its consumers and every Go↔Rust conformance test with the failure it catches, marked kept-Go (overlay, capture, guest wires), converted to a Rust corpus, or obsolete with a reason. Errors: a deleted test without a mapping, or a kept package without a production consumer, is a finding.
- **R2:** Go reference models with no production consumer are deleted, and their conformance roles are carried by checked-in Rust corpus tests captured at a recorded revision. Errors: a surviving test-only Go model, or a converted test that no longer catches its named mutation, is a finding.
- **R3:** The stdio helper protocol path, the `reference-helper` and `conformance` binaries and features, and the frame I/O in broker, storage and cluster execution are deleted. `linked` uses in-process request and outcome types, and helper corpora run through `linked`. Errors: a surviving `read_frame`/`send_frame` call, or a `protocol::` wire type in the `linked` API, is a finding.
- **R4:** The cycle stage plan, Makefile, docs and inventories reflect the deletions. The stage-plan delta maps every removed stage to replacement coverage, and before/after line counts are recorded with exact revisions. Errors: an unmapped removed stage or a count without revisions is a finding.

## Boundaries

- No change to the guest overlay, guest-boundary wires or public CLI.
- No crate moves (fn-240), and no protobuf migration (fn-247 does that on what remains).
- No CI run, PR or push is authorized by this spec.

## Decision Context

Owner, 2026-10-24 complexity review: accepted retiring the test-only Go models and the second engine entry path. Cost accepted: no independent second implementation; reviewed corpora are the oracle. Sequencing: after fn-242 (the cycle verb exists and owns the stage plan), before fn-247 (so fn-247 needs no Go protobuf side) and before fn-240 (less to extract).
