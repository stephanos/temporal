---
satisfies: [R2]
---
# fn-216-mini-activity-vertical-slice-the-model.5 Reusable stateful realization core and ABI library

## Description
Land R2's generic realization, per-run state and reusable ABI library. This consumes
the core protocol checkpoint and leaves all system payload decoding to .6.

**Size:** M
**Files:** reusable WASM library, stock Kettle binding and generic stateful controls.
**Touches:** [model/umpire/wasm/**, model/umpire/tests/fixtures/shared/**, model/umpire/hosts/go/**, model/Cargo.toml, model/Cargo.lock, Makefile, model/scripts/**]

### Approach
- Separate the embedded Kettle binding from system-agnostic ABI/buffer/dispatch mechanics, retaining the current stock guest and its byte-identical reports/rows. A system guest must link the reusable library without duplicate exports or mutable instance state.
- Put the generic Realize interface and stateful engine beside the reusable core unless an existing module provides the same seam. Consume Machine generically; introduce no Temporal vocabulary or imports and no second transition interpreter.
- Represent positive current-state projection separately from transition candidates and unsupported input. Judge applies actual Machine::step results; zero actions explain an observation only after a positive projection match. Expected actions reflect enabled Model actions.
- Make callback handling transactional. Realization validation and response selection must succeed before callback-evidenced transitions commit. Answer cannot claim an unobserved terminal result. Keep each instance's state and first Unexplained marker private; freeze future advancement/Answer until Reset.
- Preserve state on typed refusal, trap and response encoding/allocation failure. Follow existing exploration rollback and buffer ownership behavior rather than weakening it for new operations.
- Use generic test Models/opaque synthetic payloads for reusable engine controls. Pin repeated current observations, unknown projections, impossible transitions, zero-step encoding, reset/frozen behavior, instance isolation and failure rollback. Keep system-policy tests for .6.
- Wire focused native and WASM tests into the existing model gate. Serialize manifest/lockfile/Make edits and keep all prior ABI/refusal/parity controls green.

### Investigation targets
**Required:**
- `model/umpire/wasm/src/lib.rs:12` - embedded binding and generic extraction seam.
- `model/umpire/wasm/src/lib.rs:107` - buffer owner and rollback controls.
- `model/umpire/wasm/tests/dispatch.rs` - native protocol dispatch harness.
- `model/umpire/dsl/src/machine.rs` - canonical Machine transition interface.
- `model/umpire/hosts/go/host_test.go:359` - exhaustion preservation controls.
- `Makefile:1424` - model gate and stock guest build.

### Quick commands
From model/, run the reusable core's focused tests through mise. Run CGO_ENABLED=0 mise exec -- go test ./... in the generic host module, then make model-check and make model-time from the root.

## Acceptance
- [ ] A system guest links the reusable ABI/dispatch library while the stock Kettle guest retains its exact exports and canonical report/rows bytes.
- [ ] Generic Judge/Answer/Reset replay Machine transitions, distinguish positive zero-step matches, and preserve the parent contract's per-run/frozen-state semantics.
- [ ] Native/WASM controls pin malformed/refused/unexplained inputs, reset/isolation and response-allocation failure without partial mutation or ownership leaks.
- [ ] Generic code contains no system vocabulary; focused controls, host parity and the full model gate pass with recorded timing.


## Done summary
System guests can now link the export-free `umpire-wasm` library and install their own three ABI exports with `export_guest!`. The generic Runtime replays Machine transitions for Judge and callback delivery, validates the actual replay endpoint and selected response action, and commits run state or its first Unexplained freeze only after successful response publication.

`Realize` separates positive Current projection, candidate action/choice paths, final-state matching and typed payload refusal. Answer commits delivery evidence while leaving the selected response action for later observation. Frozen runs retain their first expected enabled actions until Reset. Zero or multiple declared starts produce a typed State refusal for stateful operations. Runtime transactions clone State only; the synthetic Model's Outcome and Fact deliberately lack Clone.

The stock binding lives separately from ABI and generic dispatch. `umpire-stock-guest` now builds `umpire_stock_guest.wasm`; `umpire-control-guest` builds `stateful_control.wasm`. A combined WASI build with the library and both guests under `--all-features` passed, and the full workspace documentation gate passed without target collisions. The exact three stock exports/signatures, native/WASM Kettle report and ordered rows, legacy wire vectors and refusal values remain pinned by existing controls. Make and the generic host's artifact paths now build and load both guests.

`wasm/tests/stateful.rs` covers positive zero-step bytes, repeated observations, actual replay labels and choices, empty or mismatched candidates, false Current projections, malformed and unknown inputs, first-Unexplained freeze, closed-state empty expected actions, reset and instance isolation, complete callback validation, disabled responses, repeated delivery, oversized/invalid replies, and traps before or after provisional replay. Native publication controls cover rejected encoding sinks, panicking publishers, rollback of Explore/Judge/Answer/Reset, failed freeze publication and failed Reset while frozen. `wasm/tests/abi_staging.rs` also checks oversized responses and stock exploration rollback. `hosts/go/stateful_test.go` exercises the linked synthetic WASM guest through the real host, including response exhaustion for Judge/Answer/Reset and full-capacity reallocation after cleanup.

baseline: green. Focused Rust tests, CGO-disabled Go tests, `make model-check` and `make model-time` passed before edits. Baseline logs are `.flow/tmp/fn216-5-baseline-{rust,go,model-check,model-time}.log`; timing was 3.429s full and 2.120s inner. The first implementation gate stopped on new test visibility/assertion Clippy rules. The code and tests were corrected without changing the gate.

The missing-API red runs are `.flow/tmp/fn216-5-{abi,runtime}-api-red.log`. Behavioral mutation controls removed the final-projection guard and moved commit before publication separately; each focused test failed for the intended behavior. Those raw failures remain in `.flow/tmp/fn216-5-{projection,publication}-guard-mutation-red.log`. Both mutations were reverted and the complete focused Rust suite passed in `.flow/tmp/fn216-5-post-mutation-rust.log`.

Verification passed. The focused Rust suite ran 22 integration tests, the focused CGO-disabled Go suite passed, and the authoritative post-review `make model-check` passed format, Clippy, documentation, native fixtures, both WASM guests, ABI/refusal/trap/allocation controls, protobuf freshness and host parity. The final logs are `.flow/tmp/fn216-5-post-review-{model-check,model-time}.log`; final timing was 4.147s full and 3.835s inner against 30s/10s soft targets. The green unittest receipt is `.flow/tmp/green-receipts/6f2180a3-unittest.json`.

Three fresh reviewer contexts used explicit `codex:gpt-6.1-sol:high`. Correctness, contracts and integration each returned SHIP with zero findings, and the finalizer recorded SHIP. Writer and reviewers share the Codex family. Reviewers' fresh builds were sandbox-blocked; worker logs supply fresh execution evidence. Review receipt `/tmp/impl-review-receipt-8f37faba39e2-fn-216-mini-activity-vertical-slice-the-model.5.json`; fan-out evidence `.flow/review-fanout/51dd3876544445509e2073f0b0bfe73c/`.

The generic code introduces no system vocabulary, schema change or third-party dependency. Task .6 owns system envelopes and realization; .3 owns the live SDK proof; .4 owns boundary-gate wiring and broader README/migration documentation, including the stock artifact-name update in `model/README.md`. The independent product Model remains unchanged. The known legacy generated-Case baseline remains separate in `.flow/tmp/fn216-legacy-baseline.md`.

stage: impl-review - ran [2026-10-19T17:38:20Z..2026-10-19T17:40:35Z]

Tier: session (jev-unavailable(no_key)); project implementer routing preserved.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 6f2180a362356d7660a50cc4018786808894d6b0
- Tests: cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo test -p umpire-wasm, cd model/umpire/hosts/go && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target CGO_ENABLED=0 mise exec -- go test -tags test_dep ./..., cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo build -p umpire-wasm -p umpire-stock-guest -p umpire-control-guest --all-features --target wasm32-wasip1 --release --locked, make model-check, make model-time
- PRs: