---
satisfies: [R20, R15]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.9 Slice 2: the checker as wasm in a Go host (wasmtime-go)

## Description
New crate `umpire/wasm` exporting `explore(budget)` (returns Kettle's report text) and `rows(state_id)` with a pointer-and-length ABI (`alloc`, `free`, result buffers). First finding: whether wasmtime-go can host a `wasm32-wasip2` component; if not, build a `wasm32-wasip1` core module for the host (R15's wasip2 build of dsl and checker stays in the gate). Go module `model2/hosts/go` (Go pinned by mise) loads the module and its `go test` compares the report with `cases/kettle/expected/report.txt` byte for byte and `rows` with `cases/kettle/expected/rows.txt`. Measure module size, load time and per-call latency (10^5 `rows` calls) for DESIGN.md.

**Files:** model2/umpire/wasm/, model2/hosts/go/{go.mod,host.go,host_test.go}, model2/umpire/samples/cases/kettle/expected/rows.txt, Makefile (`model2-check` runs the Go host test).

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
