---
satisfies: [R1]
---
# fn-187-gomad-wasm-helper-fixture-tests-and.1 Add the helper transcript corpus and convert the engine scenarios

## Description
Define schema gomad3.helper-transcript-corpus/v1 (guest WAT or path, execute request, scripted replies or reply policy, expected reply stream, terminal status, outcome, evidence digests, retained output, transcript hash). Write tests/helper_corpus.rs: load every file under tests/testdata/helper_transcript, run the release binary via CARGO_BIN_EXE_gomad3-wasmhost, compare the full expected block, and rewrite expected under GOMAD_HELPER_CORPUS_REGENERATE=1; reject cases that differ across two runs. Add one Go test in wasi that reads the same directory and asserts the same expectations through the Go-side helper client. Early proof: convert fresh-instance execution, fuel bound, unknown import refusal, output routing and module identity mismatch first. Then convert every remaining tests/engine.rs scenario and the single-helper scenarios in tests/cluster.rs, deleting each Rust scenario in the same commit as its corpus case. Land after the fn-180 engine/abi/model edits.

**Size:** L
**Files:** tools/gomad/internal/backends/wasm/wasmhost/tests/helper_corpus.rs, tests/testdata/helper_transcript/**, tests/engine.rs, tests/cluster.rs, tools/gomad/internal/backends/wasm/wasi/helper_corpus_test.go, wasmhost/README or protocol docs

## Acceptance
- [ ] Schema, loader and regeneration mode exist; a second regeneration is byte-identical
- [ ] Both Rust and Go consume the same corpus directory
- [ ] Every tests/engine.rs scenario and single-helper tests/cluster.rs scenario has a corpus case and is deleted, or is listed with the reason it stays
- [ ] Non-deterministic cases are refused by the loader

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
