---
satisfies: [R2]
---
# fn-216-mini-activity-vertical-slice-the-model.2 Model protocol rename and core Any messages with Go encoding

## Description
Land the protocol and host encoding portion of R2. This is the wire compatibility checkpoint;
task .5 adds generic stateful dispatch and .6 supplies the system binding.

**Size:** M
**Files:** core schema/generation, existing Rust generated imports, generic Go module and wrappers.
**Touches:** [model/umpire/proto/**, model/umpire/wasm/build.rs, model/umpire/wasm/src/**, model/umpire/wasm/tests/**, model/umpire/wasm/Cargo.toml, model/umpire/hosts/go/**, model/scripts/proto-go.sh, model/umpire/tests/fixtures/shared/**, model/Cargo.lock]

## Approach
- Before renaming, pin literal existing Explore/Rows requests, successful responses and refusals as exact encoded bytes, independent of regenerated descriptors. Keep existing method numbers, tags, refusal codes and ABI exports unchanged.
- Rename schema/package/modules from oracle to model, including the generic Go module `umpire.local/model` and generated Go import paths. Update every actual consumer found by rg; ask for a precise scope extension if the declaration above misses one. Do not retain symlink aliases.
- Add the new core Reset/Judge/Answer request/response shapes over Any, with zero-step explanations representable and distinct new method IDs. Core Answer carries opaque system reply data, not Temporal-specific completion/failure types.
- Use admitted prost/prost-build to generate well-known types with one shared owner. Do not silently add a prost-types runtime dependency. Update the existing pinned Go generator and freshness check.
- Extend generic Go typed encoding/decoding and method/response-oneof validation. Reject malformed data and mismatched response variants. Until .5, unsupported new operation dispatch remains an honest typed refusal; do not claim stateful behavior is complete here.
- Run existing Kettle native/stock-WASM/Go parity, module contract, allocation/trap/refusal and generator-freshness controls. Only descriptor metadata/import identity changes; old request/response byte controls remain identical.

## Investigation targets
**Required:**
- `model/umpire/proto/umpire/oracle/v1/oracle.proto:3` - current wire owner.
- `model/umpire/wasm/build.rs:4` - Rust generation input.
- `model/scripts/proto-go.sh:18` - pinned Go generation/freshness.
- `model/umpire/wasm/tests/dispatch.rs:70` - encoded dispatch controls.
- `model/umpire/hosts/go/host.go:273` - response validation.
- `model/umpire/hosts/go/host_test.go:117` - module/host controls.

## Quick commands
From model/, run cargo test -p umpire-wasm through mise with the shared CARGO_TARGET_DIR. Run CGO_ENABLED=0 mise exec -- go test ./... in the generic host module. From the root run make model-check and make model-time.

## Acceptance
- [ ] Existing literal wire-byte controls survive the namespace/module rename unchanged; old ABI signatures, method IDs, field tags and refusal codes are pinned.
- [ ] New generic core messages and Go wrappers round-trip valid payloads and reject malformed or method/oneof-mismatched responses; zero-step explanation is representable.
- [ ] Generated Rust and Go are fresh and agree, with one WKT owner and no new runtime library; no Temporal types enter the generic tree.
- [ ] Existing native/WASM/CGO-disabled host parity and full model gate pass; stateful behavior remains explicitly assigned to .5.

## Done summary
Protocol clients now use `umpire.model.v1` and the Go module `umpire.local/model`. Reset, Judge and Answer have distinct core methods, opaque Any inputs/replies, and a present Explanation that represents zero steps independently of Unexplained.

The pre-rename checkpoint `156e0cf241` pins literal Explore/Rows requests, success responses and every legacy refusal value. Both languages still consume those unchanged bytes under `model/umpire/proto/tests/`; named enum assertions also pin codes 0-7 and new Unsupported 8. Shared literal core vectors check Rust/Go agreement. The three application ABI exports and their signatures remain unchanged. Namespace metadata intentionally changes, and Go rejects old-namespace Any identity.

Rust generates the core and one public `umpire_wasm::google::protobuf` owner with admitted prost tooling. The neutral standard WKT inputs cover the later system binding without reading system descriptors or adding a runtime dependency. Generated upstream WKT comments contained non-Rust examples and caused 33 doctest failures on the first attempt; generation suppresses those comments. The pinned Go freshness check compares the entire generated package tree. A stale old package control failed with exit 1, then the clean tree passed.

The Go host adds typed wrappers and rejects malformed responses, all method/outcome mismatches, absent Judge judgments, absent/empty-type Answer replies, missing request Any values and unknown refusal enums. `TestProtocolResponseValidation`, `TestProtocolWrappers`, `TestProtocolWireCompatibility`, `TestCoreProtocolWireCompatibility` and `TestProtocolNamespace` cover this task's R2 surface. Rust's three core protocol tests cover opaque round trips, present-empty explanations and typed unsupported/malformed dispatch. Valid new operations remain Unsupported until the stateful core in .5 and system binding in .6.

baseline: green. Focused Rust and Go commands, `make model-check` and `make model-time` all passed before edits. Baseline timing was 3.339s full and 1.972s checker edit/test. Logs are `.flow/tmp/fn216-2-baseline-{rust,go,model-check,model-time}.log`.

Verification passed. The final focused Rust suite ran 11 integration tests, CGO-disabled Go tests passed, and the post-review `make model-check` passed formatting, Clippy, documentation, native fixtures, stock WASM, ABI/module/allocation/trap/refusal controls, generator freshness and Go parity. `.flow/tmp/fn216-2-post-review-model-check.log` records the final gate; the unittest receipt is `.flow/tmp/green-receipts/9b216134-unittest.json`. The timing run recorded 3.463s full and 1.674s inner against 30s/10s soft targets in `.flow/tmp/fn216-2-verify-model-time.log`.

Test-first evidence is retained in `fn216-2-new-core-red.log`, `fn216-2-go-validation-red.log`, `fn216-2-go-wrapper-red.log` and `fn216-2-unknown-refusal-red.log` under `.flow/tmp/`. The first full gate caught vectors placed in the strict Model fixture tree; they moved into the already-owned proto test tree without changing discovery. Clippy then required the current fixed-chunk and empty-slice assertion styles. The failure records are `fn216-2-full-gate-layout-failure.md`, `fn216-2-verify-model-check.log` and `fn216-2-full-gate-clippy-assert.log`.

Three fresh Codex review contexts used the explicit `codex:gpt-6.1-sol:high` override. Correctness, contracts and integration each returned SHIP with zero findings; the finalizer recorded SHIP. Writer and reviewers share the Codex family. The reviewers' build attempts were sandbox-blocked; worker gate logs supply execution evidence. Receipt `/tmp/impl-review-receipt-8f37faba39e2-fn-216-mini-activity-vertical-slice-the-model.2.json`; fan-out evidence `.flow/review-fanout/b26d600ab01c414bac43dc8975613d9b/`.

`model/DESIGN.md:19` and `MIGRATE.md:243` retain documentation references assigned to .4. Stateful dispatch, system envelopes and live SDK coverage remain assigned to .5/.6/.3. The known legacy generated-Case failure remains recorded separately in `.flow/tmp/fn216-legacy-baseline.md`; this task makes no live server or legacy Case gate claim.

stage: impl-review - ran [2026-10-19T17:20:29Z..2026-10-19T17:22:41Z]

Tier: session (jev-unavailable(no_key)); project implementer routing preserved.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 156e0cf241d61da0345ec03df3197d9274bfd25b, 9b21613488d76d5fb9ae3a8105a0b57d8b100355
- Tests: cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo test -p umpire-wasm, cd model/umpire/hosts/go && CGO_ENABLED=0 mise exec -- go test ./..., make model-check, make model-time, bash model/scripts/proto-go.sh --check
- PRs: