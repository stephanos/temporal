# fn-219 identity ledger (2026-10-19) — DRAFT, requalification pending

Base: `e50c6476fb`. Candidate: branch `fn219-promote` (worker fn-219). Raw evidence:
`/home/agent/wt/evidence/fn219/` (`identities-compare.txt`, `pins-before.txt`,
`helper-source-digest.txt`, `helper-binary-sha.txt`, `r2-jq-filtered-corpus-diff.txt`,
`r3-cumulative-header-filtered.diff`).

Status: every identity below is measured or derived, and every rotation has a cause. The
requalification gates have **not run yet**. The host was reserved for a timing-sensitive
qualification, so they are listed at the end with exact commands for the conductor to
schedule on the frozen candidate. This ledger satisfies R5 only after those exits are
recorded.

## Identities

Values come from `go run` of each package's identity function at base and at the candidate.
The choice digest is decoded from the generated literal.

| Identity | Before | After | Status | Cause |
| --- | --- | --- | --- | --- |
| Helper source digest: `wasmhost/src/world/wire_generated.rs` | `b22d7a20…de9a` | `0a0517ac…ae17` | rotated | The generated-file header names `internal/model/cmd/protocolgen` instead of native `gomadtool`. Body is byte-identical. This is the only non-test helper source change; the `PINNED_*` pins are `#[cfg(test)]`. Helper source tree digest `946583cd…ac09` → `b4442d1e…e1b7`. |
| Helper binary SHA (release, default features) | `9027efde…90ce` | `d1dd1faa…4ffe` | **rotated** (measured, reproducible) | Built in one location (`/home/agent/wt/fn219`, target `/home/agent/.cache/fn219-target/default`) with only the header comment of `wire_generated.rs` swapped. Three rebuilds of the candidate gave the same hash. The plan expected "probably unchanged": that is wrong. The official helper must be rebuilt and its `HelperSHA256` re-measured; absolute values differ by build location. |
| `volume.ImplementationSHA256` | `3df0e2bd…e486` | `7c83fb25…d154` | rotated | Projection files renamed (`*_generated.go` → `fs.go`, `handles.go`, `local_handles.go`, `volume.go`, `types.go`, `hashwire/hash.go`); generated and `Source:` headers dropped; `provenance_generated.go` deleted; `hashwire/wire_generated.go` header and choice digest literal changed. Embedded paths and bytes both enter the hash. |
| `network.ImplementationSHA256` | `1e5469e6…38e3` | `d299f192…5f84` | rotated | `generated.go` → `simulation.go` without its header; `generate/main.go` no longer hashed; `stream_status_generated.go` header; adapter package doc. |
| `world.ImplementationSHA256` | `8131a110…399e` | `93724a22…7e29` | rotated | Header of `compressed_wire_generated.go`. |
| Cluster model identity (`internal/model/cluster`) | `64a90676…60bb` | `d62d0fe6…9430` | rotated | Headers of `internal/wire/{process,network,volume}.go` and `process_generated_test.go` (`process.go`, `network.go` and `volume.go` are hashed). |
| WASM cluster identity (`internal/backends/wasm/cluster`) | `0d186559…c261` | `730aabcc…92a2` | rotated | Composes the wasi, world and cluster model identities. |
| `wasi.ImplementationSHA256` = installation `ModelSHA256` | `9b087b96…4c99` | `f9607ca9…4f29` | rotated | Composes the volume, network and world identities. **Every existing WASM installation is refused and must be reinstalled.** |
| WASM overlay `toolchain.ImplementationSHA256` | `6940d421…b079` | `4ac4238f…4ac4` | rotated | Header of `net_stream_status_generated.go.txt`. |
| WASM runtime identity and preparation BuildKey | derived | derived | rotate by construction (not measured) | They bind the overlay identity and the published runtime sources, including the `gomad_choicewire_generated.go.txt` and `net_stream_status_generated.go.txt` headers. All preparation caches are invalid. Measuring them needs a guest preparation, which is pending. |
| Choice `ImplementationSourceSHA256` (shared and native codec literal) | `de8e839b…bc82` | `669bc871…27aa` | rotated | Step 2: template header edits (templates are digest inputs) → `0a4e3995…93a9`. Step 3: `gomad-choice-implementation-source-v3` over parent-module inputs only (schema, three templates, `gomad_semantics.go.txt`, `trace.go`, `tape.go`). v2 also hashed native `runtime/gomad.go` and `go1.27.1.patch`. |
| `choice.ImplementationIdentity(buildKey)` | e.g. `900cbea3…f375` | e.g. `8dd54e14…2706` | rotated | Example values are for a zero build key. Binds the source digest above and the (rotated) BuildKey. Retained artifacts replay only under their own candidate. |
| `choice.InterruptedReplayImplementationSHA256` | `fcf851ca…` | same | unchanged | — |
| `clock.BrokerImplementationSHA256` | `8885de54…88e6` | same | unchanged | `PINNED_GO_CLOCK_SHA256` and the clock corpus are untouched. |
| `strictjson.ImplementationSHA256` | `ae4ebc32…5795` | same | unchanged | — |
| `softassert.ImplementationSHA256` | `586f591f…` | same | unchanged | — |
| Native overlay `gomadchoicewire` digest literal (native toolchain identity) | v2 digest | v3 digest | rotated | Same value as the shared literal. Native execution stays deferred (fn-128/fn-149): neither executed nor requalified. |
| Guest module SHA | — | — | **pending** | Needs a guest preparation (heavy). Header comments only, so it may be unchanged; measure it in the requalification window. |

## Pins and corpora regenerated (unchanged expectations)

Regenerated with `GOMAD_NETWORK_CORPUS_REGENERATE`, `GOMAD_NETWORK_ADAPTER_CORPUS_REGENERATE`,
`GOMAD_NETWORK_CLUSTER_CORPUS_REGENERATE`, `GOMAD_BROKER_LIFECYCLE_CORPUS_REGENERATE` and
`GOMAD_WORLD_CONFORMANCE_REGENERATE`. Two pins have no regenerator and were set by hand to the
new identity: `PINNED_GO_NETWORK_SHA256` and the `wasi/identity_test.go` literal.

- 375 files: `network_v2` 183, `network_cluster_cases` 51, `network_adapter_cases` 62,
  `broker_lifecycle_cases` 10, `world_conformance` 66, three `PINNED_GO_*` files and one test
  literal.
- **jq proof:** removing every key matching `implementation_sha256$`
  (`walk(if type=="object" then with_entries(select(.key|test("implementation_sha256$")|not)) else . end)`)
  leaves all 371 JSON files identical to base. 0 diffs.
- **Byte proof:** substituting each rotated identity (old → new) into the base bytes
  reproduces all 375 files exactly.
- After regeneration, these Go currency tests exit 0 (`t4-pins-after-regeneration.log`):
  `TestNetworkCorpusIsCurrent`, the adapter, cluster, broker-lifecycle, transport, broker-order,
  World and clock currency and drift tests, and the wasi model identity literal.
- Rust differential tests exit 0 with `-j 2`, release (`r2-rust-corpora.log`):
  `network_corpus` (16), `world_conformance` (8), `clock_conformance` (4), and the bin unit
  tests (138 passed, 2 ignored), including the cluster network corpus pins.
- Volume and namespace differential conformance with the conformance helper required: exit 0
  (`t1-test-conformance.log`, run after fn-219.1).

## Requalification gates (deferred; run on the frozen candidate)

Run from the repository root of the frozen candidate with the official stock Go 1.27.1
(`GO=<stock go1.27.1>`), after `make -C tools/gomad/internal/backends/wasm helper`. Record each
numeric exit here.

| Gate | Command | Exit |
| --- | --- | --- |
| Helper corpora (4 builds) | `make -C tools/gomad/internal/backends/wasm test-helper` | pending |
| Conformance (fresh helper) | `make -C tools/gomad/internal/backends/wasm test-conformance GOMAD_WASM_STOCK_GO=$GO` | pending |
| WASM suite | `make -C tools/gomad test-wasm GOMAD_STOCK_GO=$GO` | pending |
| Retained inspection | `cd tools/gomad && GOMAD_STOCK_GO=$GO GOMAD_WASM_TEMPORAL_INSPECTION_ROOT=<retained dir> go test -tags test_dep -count=1 -timeout=120m -run '^TestTemporalInspectionRetainedFailure$' ./internal/backends/wasm/backend` | pending |
| Operator typed scenarios | `cd tools/gomad && GOMAD_STOCK_GO=$GO GOMAD_WASM_TEST_HELPER=<release helper> go test -tags test_dep -count=1 -timeout=60m -run '^TestInstalledOperatorTypedScenario(DetachedReplay\|KillResume)$' ./internal/backends/wasm/command` | pending |
| Runtime qualification | `GOMAD_WASM_QUALIFICATION_ROOT=<root> GOMAD_WASM_QUALIFICATION_MANIFEST=<manifest> make -C tools/gomad/internal/backends/wasm qualify-runtime GOMAD_WASM_STOCK_GO=$GO` | pending |
| Workload qualification (mandatory, after disk cleanup) | `GOMAD_WASM_QUALIFICATION_ROOT=… GOMAD_WASM_INTEGRATED_SOURCE_MANIFEST=… GOMAD_WASM_INTEGRATED_SOURCE_SHA256=… GOMAD_WASM_TEMPORAL_QUALIFICATION_INPUTS=… GOMAD_WASM_TEMPORAL_PREPARATION_SOURCE=… make -C tools/gomad/internal/backends/wasm qualify-workloads GOMAD_WASM_STOCK_GO=$GO` | pending |
| Measurements to add | Official helper `HelperSHA256`, guest module SHA, runtime identity, BuildKey | pending |
