---
satisfies: [R20, R15, R24]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.9 Decisive slice: the checker as wasm in a Go host (wasmtime-go), on the trait Kettle

## Description
**Location (R29 amended):** there is no `samples` library; `umpire-wasm` includes `umpire/tests/fixtures/kettle_trait/model.rs` by `#[path]`.

**Shared fixtures 2026-10-18 (R29):** end-to-end fixtures live in `umpire/tests/fixtures/<name>/` after .13; their expected files select the consumers (checker, TLC, Apalache).

**Conventions 2026-10-18 (R25, R27):** the wasm report is the `Spell`-based report text, so byte equality with the golden follows; a `Refusal` crosses the ABI as an error code plus its message, and the Go host turns it into a Go error.

**Recut 2026-10-18 (R24, decisive):** this task decides the prototype. Run it right after .2 and .10 on the trait-shaped Kettle fixture (`umpire/tests/fixtures/kettle_trait/`), not the surface-C Kettle; the goldens it must match byte for byte are that fixture's `expected/report.txt` and `rows.txt`. Stock rustc only. Record the result in `model2/DESIGN.md` immediately (a short verdict section), then continue with the remaining tasks.

**Layout (R22):** the wasm host's expected report and rows are Kettle's goldens in `umpire/tests/fixtures/kettle/expected/`; the Go host reads them from there.

New crate `umpire/wasm` exporting `explore(budget)` (returns Kettle's report text) and `rows(state_id)` with a pointer-and-length ABI (`alloc`, `free`, result buffers). First finding: whether wasmtime-go can host a `wasm32-wasip2` component; if not, build a `wasm32-wasip1` core module for the host (R15's wasip2 build of dsl and checker stays in the gate). Go module `model2/hosts/go` (Go pinned by mise) loads the module and its `go test` compares the report with `tests/fixtures/kettle/expected/report.txt` byte for byte and `rows` with `tests/fixtures/kettle/expected/rows.txt`. Measure module size, load time and per-call latency (10^5 `rows` calls) for DESIGN.md.

**Files:** model2/umpire/wasm/, model2/hosts/go/{go.mod,host.go,host_test.go}, model2/umpire/tests/fixtures/kettle/expected/rows.txt, Makefile (`model2-check` runs the Go host test).

**Quick commands:** `mise exec -- cargo build -p umpire-wasm --target wasm32-wasip1 --release`; `cd model2/hosts/go && mise exec -- go test ./...`
## Acceptance
- Go host test passes: report and rows byte-identical to the Rust goldens.
- A trap surfaces as a Go error naming the export.
- Target, module size, load time and per-call latency recorded.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
