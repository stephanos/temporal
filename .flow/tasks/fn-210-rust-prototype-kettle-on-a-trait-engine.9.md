---
satisfies: [R20, R15, R24, R31, R32]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.9 Decisive slice: the checker as wasm in a wazero Go host, generated protobuf interface, on the trait Kettle

## Description
**Current owner recut (R31, R32):** build the decisive stock-rustc slice using generated protobuf over a three-function core-wasm ABI, hosted by wazero v1.12 with `CGO_ENABLED=0`. This supersedes all former wasmtime-go/component instructions. The hosted module and dsl/checker portability gate target `wasm32-wasip1`.

One schema at `model2/umpire/proto/umpire/oracle/v1/oracle.proto` (package `umpire.oracle.v1`) defines Explore, Rows and Refusal messages. Generate Rust with admitted `prost`/`prost-build` 0.14 and mise-pinned protoc; generate Go with the repository's pinned `protoc-gen-go`. Only application exports are `alloc(len) -> ptr`, `free(ptr, len)` and `call(method, ptr, len) -> u64` (packed response pointer/length). Unknown methods and undecodable requests return generated Refusal messages; API misuse crosses as a code plus message and becomes a Go error. Unexpected traps become host errors naming the method. No handwritten message structs or extra operation exports.

`umpire-wasm` includes the production-includable trait Kettle source `umpire/tests/fixtures/kettle_trait/model.rs` by `#[path]`; there is no samples crate. Compare `ExploreResponse.report` byte-for-byte with that fixture's `expected/report.txt` and ordered Rows output with its `expected/rows.txt`. Preserve canonical Spell text and author Rust names. This task precedes the macro surface (R24).

R15 requires more than compilation: a dependency on threads, filesystem or clock in core dsl/checker fails the gate. Host build-script filesystem access is permitted. Use complementary resolved-API and real-artifact checks, with regression controls and documented scope; never weaken core unsafe policy for the narrowly justified ABI export attributes.

Measure real module size, load time and per-call latency over 100,000 Rows calls. Record the stock-toolchain result immediately in `model2/DESIGN.md`: R20 passing means Rust; otherwise record the precise blocker and Scala recommendation. Continue the remaining tasks regardless. No Temporal/Testpilot integration or Scala source changes.

**Feedback loop (R30):** warm inner loop <=3s and `make model2-check` <=30s. Record actual `make model2-time` output; fix breaches here or obtain owner exception. Include CGO-disabled Go parity tests and wasm builds in the default gate within budget; preserve real failure propagation and per-step timing.

**Parallel implementation:** after fixing the schema/ABI contract, use isolated Rust guest and Go host subagent lanes with disjoint ownership. One lead owns schema, shared manifests/lockfiles/Makefile/tooling, integration, final gates, review and task completion. Do not concurrently mutate one checkout.

**Touches:** `model2/umpire/proto/**`, `model2/umpire/wasm/**`, `model2/hosts/go/**`, `model2/Cargo.toml`, `model2/Cargo.lock`, `model2/rust-toolchain.toml`, `model2/clippy.toml`, `model2/deny.toml`, `model2/README.md`, `model2/DESIGN.md`, `model2/scripts/**`, `model2/umpire/dsl/clippy.toml`, `model2/umpire/checker/clippy.toml`, `model2/umpire/dsl/tests/**`, `model2/umpire/checker/tests/**`, `Makefile`

**Quick commands:** from model2, `mise exec -- cargo build -p umpire-wasm --target wasm32-wasip1 --release --locked`; from model2/hosts/go, `CGO_ENABLED=0 mise exec -- go test -count=1 ./...`; from repository root, `make model2-check` and `make model2-time`.
## Acceptance
- The Go host test passes under `CGO_ENABLED=0` with wazero: `ExploreResponse.report` and `RowsResponse` match the trait Kettle's goldens byte for byte.
- Both sides are generated from `oracle.proto`; no hand-written message structs.
- An unknown method returns a `Refusal`; a trap surfaces as a Go error naming the method.
- Module size, load time and per-call latency recorded; `make model2-time` within R30.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
