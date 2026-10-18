---
satisfies: [R1]
---
# fn-187-gomad-wasm-helper-fixture-tests-and.1 Add the helper transcript corpus and convert the engine scenarios

## Description
Define schema gomad.helper-transcript-corpus/v1 (guest WAT or path, execute request, scripted replies or reply policy, expected reply stream, terminal status, outcome, evidence digests, retained output, transcript hash). Write tests/helper_corpus.rs: load every file under tests/testdata/helper_transcript, run the release binary via CARGO_BIN_EXE_gomad-wasmhost, compare the full expected block, and rewrite expected under GOMAD_HELPER_CORPUS_REGENERATE=1; reject cases that differ across two runs. Add one Go test in wasi that reads the same directory and asserts the same expectations through the Go-side helper client. Early proof: convert fresh-instance execution, fuel bound, unknown import refusal, output routing and module identity mismatch first. Then convert every remaining tests/engine.rs scenario and the single-helper scenarios in tests/cluster.rs, deleting each Rust scenario in the same batch as its corpus case. Land after the fn-180 engine/abi/model edits.

The owner resumed fn-189 and fn-187 on 2026-10-17, prioritizing work whose acceptance is independent of host performance/resource measurements. This task and .2 form the first functional batch. Reuse the preserved source through a selected candidate rooted at `95eed5c83a` in `/home/agent/wt/wasm-fn187-functional`; preserve the paused broader worktrees. Do not include .3 model-unit deletions or .4 module moves in this batch. Correctness, deterministic regeneration, refusal controls, assertion preservation and independent review remain required. No performance/resource claim or native execution qualification is made. Existing measured qualification gates remain open under their owners.

**Size:** L
**Files:** tools/gomad/internal/backends/wasm/wasmhost/tests/helper_corpus.rs, tests/testdata/helper_transcript/**, tests/engine.rs, tests/cluster.rs, tools/gomad/internal/backends/wasm/wasi/helper_corpus_test.go, wasmhost/README or protocol docs
## Acceptance
- [x] Schema, loader and regeneration mode exist; a second regeneration is byte-identical
- [x] Both Rust and Go consume the same corpus directory
- [x] Every tests/engine.rs scenario and single-helper tests/cluster.rs scenario has a corpus case and is deleted, or is listed with the reason it stays
- [x] Non-deterministic cases are refused by the loader
## Done summary
Implemented 215 shared Rust/Go helper transcript cases with strict expectations and nondeterminism refusal. Both regenerations are byte-identical. Converted 61 engine and 13 single-helper cluster scenarios; retained 13 engine and 8 cluster roots with explicit reasons and activated them across feature checks. Minimal automatic cluster result ordering is covered without changing manual control semantics.

Default/feature corpus and retained tests pass on the byte-identical reviewed candidate; primary shared-corpus/identity/control selection passes (46 roots, zero skips), generated validation passes. Independent staged-source review returned SHIP after R2 corrections. [Batch handover](../artifacts/fn187-functional-2026-10-17.md) records commands, source equality, identities, retained assertions and review evidence.

Uncommitted for the owner; no push/PR/CI. No full-spec workload, performance/resource or native execution acceptance is claimed; fn-187.3-.5 and remaining fn-189 gates stay open.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: Rust 1.94.1 release locked helper_corpus: default, diagnostic-profile, diagnostic-fuel-off and guest-profile; retained engine/cluster feature suites; exact commands/logs in batch handover, Two GOMAD_HELPER_CORPUS_REGENERATE=1 runs with byte-identical before/after manifests, go test -p=1 -parallel=1 -tags=test_dep -count=1: cmd/wasmmeasure internal/measurement internal/measure internal/stockgo, Primary 46-root WASI corpus/identity/admission/fixture/transport selection: 46 pass, zero skip/fail, Canonical affected five-package make lint-code plus errortype, fixes=false: zero issues, make -C tools/gomad validate, go list -deps ./internal/backends/wasm/cmd/wasmmeasure: no native or testing dependency, Independent read-only actual staged-patch review: NEEDS_WORK then SHIP; patch 567ab2a5d171cf63c657e9a6d238f323bba429ced5b58f458619fbd85540ca5a, All 261 nondeleted reviewed files SHA-256 match integrated candidate; git diff --check
- PRs: