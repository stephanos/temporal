# Gomad: WASM helper fixture tests and module ownership

## Conversation Evidence

> user: "I usually like fixture/golden tests that are end-to-end. they often clearly state the inout and expcted output and don't concern themselves with impl detials"
> user: "file spec for that work"

## Goal & Context

Make fixture corpora the primary contract for the WASM helper, remove the hand-written tests they make redundant, move measurement producers out of test files, and then split the three helper modules whose ownership is unclear, in that order so the structural work lands on tests that survive it. [paraphrase]

Measured on 2026-10-16, the Rust helper is 21.3k production lines, 11.0k inline test lines and 6.3k lines under `tests/`; the Go `wasi` package carries 21.2k test lines, 11.6k of them in one `integration_test.go` with 134 tests and 158 helpers, 27 of which skip unless environment variables name a host, helper receipt, cache and seed. Fixture corpora already exist and already hold the models: network_v2 (114 cases), world conformance (63), clock (55), broker orders (19) under `wasmhost/tests/testdata`, and volume and namespace conformance under `wasi/testdata`. Each case is a JSON file with config, input steps and the expected error, state and snapshot per step, bound to the generating Go implementation by `implementation_sha256` and regenerated through `GOMAD_*_CORPUS_REGENERATE=1` on the Go side. What is missing is a corpus for the helper itself: the 50 scenarios in `tests/engine.rs` drive the real binary with 75 inline WAT guests and scripted JSON replies, but each scenario is Rust code.

## Architecture & Data Models

- **Helper transcript corpus.** A new corpus `wasmhost/tests/testdata/helper_transcript/<name>.json` with schema `gomad.helper-transcript-corpus/v1`: `guest` (WAT text or a path under the corpus directory), `request` (the execute request JSON the Go side sends), `replies` (ordered scripted replies keyed by the request the helper emits, or a reply policy such as `echo`/`deny`), and `expected` (the ordered reply stream, terminal status, classified outcome, evidence digests, retained stdout/stderr and transcript hash). A loader in `tests/helper_corpus.rs` runs every file through `CARGO_BIN_EXE_gomad-wasmhost`, compares the full expected block, and under `GOMAD_HELPER_CORPUS_REGENERATE=1` rewrites `expected` from the reviewed run. The Go side reads the same files in one `wasi` test so both languages assert one artifact. Hand-named cases carry intent; generated sweeps stay generated.
- **Measurement producers.** The env-gated producers in `wasi/integration_test.go` (runtime cost profile, matched release baseline/candidate, warm preparation, namespace sample, diagnostic cluster pair, guest profile build, terminal-exit controls) move to a Go command `internal/backends/wasm/cmd/wasmmeasure` with flags replacing the `GOMAD_WASM_*` variables, one subcommand per producer, sharing the admission, hashing and host-evidence helpers they use today. Retained artifacts keep their schemas and `rerun` fields; the rerun commands change to the new invocation and the fn-172/174/178 artifacts are annotated with the mapping, not rewritten.
- **Redundant unit tests.** For each model with a corpus (network, volume, namespace, world, clock, broker) and for the helper once the transcript corpus exists, run coverage with the inline `mod tests` removed; add a corpus case for every behavior the corpus misses; then delete the inline tests. Keep only invariants a corpus cannot state (overlap and capacity counters, allocation-free probe paths, poison and lock handling in `cache`).
- **Module ownership.** `model.rs` splits into guest state, reply encoding and host-import routing with predictable names; `engine.rs` moves its request and outcome types into their own module so the wire contract has one home; `namespace_call`, `namespace_image` and `namespace_transport` fold under `namespace/`; `network/cluster` and `broker` are reconciled with `cluster/` so one module owns instance handoff; `allow(dead_code)` moves from whole modules to the clearly marked complete-port submodules. Two Cargo bin targets replace the `conformance` feature's `main` swap if the swap is the only thing the feature does.

## API Contracts

- Corpus files are the contract. A behavior change in the helper shows up as a corpus diff reviewed in the commit; `implementation_sha256`-style binding is not used for the helper corpus because the helper is the implementation under test, so the reviewed diff is the approval.
- The helper's wire protocol, reply ordering, terminal statuses, evidence digests and refusal classifications are unchanged by this spec; the corpus pins them before any module moves.
- `make -C internal/backends/wasm test` runs the corpora; `wasmmeasure` is never invoked by a test target and refuses to run without explicit host and identity flags, as the producers do today.
- Module moves keep every public behavior and evidence byte identical; helper, model and runtime identities may rotate because source changed, and the rotation is recorded and the WASM gates rerun on the frozen candidate.

## Edge Cases & Constraints

- Scripted replies must be deterministic across runs; a case that depends on timing or host paths is rejected by the loader, not tolerated.
- Deleting a test is allowed only after the coverage check shows the corpus reaches the same lines or the behavior is listed as an invariant kept in a unit test; the deletion commit names the corpus cases that replaced it.
- Producers that today assert evidence identity across repetitions keep that assertion in the command; moving them must not weaken the measurement admission rules in `QUALIFICATION.md`.
- Coordination: fn-180 edits `abi.rs`, `engine.rs`, `model.rs`; fn-183 edits `cluster/provider.go` and installation; fn-185 later extends `diagnostic.rs`, the current dispatch points and the moved profile collector. Sequence tasks .1 and .2 after fn-180; fn-185 follows this spec and extends the moved measurement command and final dispatch owners. Task .4 follows this spec's corpus and coverage work.
- No semantic change to any model; Go oracles and their corpora stay as they are.

## Acceptance Criteria

- **R1:** A helper transcript corpus exists with a loader, regeneration mode and schema, consumed by Rust and Go tests, covering every scenario in `tests/engine.rs` and the single-helper scenarios in `tests/cluster.rs`; those Rust scenarios are deleted. Errors: a corpus case whose output varies across two runs fails the loader; a scenario without a replacing case is retained.
- **R2:** Every env-gated measurement producer leaves `wasi/*_test.go` into `wasmmeasure` subcommands with flags, with admission, identity and host-evidence rules preserved and retained artifacts annotated with the new rerun commands. Errors: a producer that cannot be rebuilt as a command stays and is listed with the reason; no test target invokes a producer.
- **R3:** Inline unit tests covered by corpora are deleted after a recorded coverage check, with new corpus cases added for every gap, and the remaining unit tests are listed as invariants a corpus cannot state. Errors: coverage loss on a model path blocks the deletion of the test that covered it.
- **R4:** `model.rs`, `engine.rs`, the namespace files, the cluster/broker/network-cluster trio and the `allow(dead_code)` modules are restructured as described, with every corpus and the retained unit tests passing unchanged and identity rotation recorded with rerun WASM gates. Errors: any corpus diff caused by a move is a defect, not an expected update.
- **R5:** Before/after line counts for production, inline test, `tests/` and `wasi` test code, plus `make -C internal/backends/wasm test` wall time on the same host, are retained; README and ARCHITECTURE describe the corpus as the contract, the regeneration workflow and the measurement command. Errors: a claim of simplification without the numbers is a finding.

## Boundaries

- No model semantic, evidence format, protocol or performance change; no change to Go oracles or their corpora beyond adding cases.
- No native code; native stays deferred under fn-128/fn-149 and isolation belongs to fn-184.
- No CI run, PR or push.

## Decision Context

Owner preference (2026-10-16): end-to-end fixture tests that state input and expected output and ignore implementation details. The helper already has that shape for its models; this spec extends it to the helper itself and removes the hand-written layer that duplicates it, then uses the resulting safety net for the structural cleanup the 2026-10-16 structure review identified.

Owner ordering decision (2026-10-16): fn-187 follows fn-184 and precedes fn-185. The corpus and producer work no longer waits for fn-185 task .1; all fixture, coverage, measurement-admission and identity requirements remain unchanged.

## Early proof point

Task .1 converts five existing `tests/engine.rs` scenarios (fresh-instance execution, fuel bound, unknown import refusal, output routing, module identity mismatch) into corpus files and shows the loader passing in both languages with regeneration producing a byte-identical file on a second run. If scripted replies cannot be made deterministic for the remote-callback scenarios, stop and report before converting more.

## Quick commands

```sh
cargo +1.99.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked --test helper_corpus
GOMAD_HELPER_CORPUS_REGENERATE=1 cargo +1.99.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked --test helper_corpus
cd tools/gomad && go test -tags test_dep ./internal/backends/wasm/wasi -run HelperCorpus -count=1
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Helper transcript corpus with loader, regeneration and schema, consumed by Rust and Go, replacing the engine and single-helper cluster scenarios. | fn-187-gomad-wasm-helper-fixture-tests-and.1 | — |
| R2 | Measurement producers move from test files into wasmmeasure subcommands with admission rules preserved and artifacts annotated. | fn-187-gomad-wasm-helper-fixture-tests-and.2 | — |
| R3 | Corpus-covered inline tests deleted after a recorded coverage check; remaining unit tests listed as invariants. | fn-187-gomad-wasm-helper-fixture-tests-and.3 | — |
| R4 | model, engine, namespace, cluster/broker and dead-code modules restructured with zero corpus diffs and recorded identity rotation. | fn-187-gomad-wasm-helper-fixture-tests-and.4 | — |
| R5 | Before/after counts and test wall time retained; docs describe the corpus contract, regeneration and wasmmeasure. | fn-187-gomad-wasm-helper-fixture-tests-and.5 | — |
