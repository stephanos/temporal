---
satisfies: [R1]
---
# fn-248-off-the-shelf-crates-and-consolidation.6 Serde-derived ITF envelope in the Apalache back end

## Description

**Size:** S
**Files:** model/umpire/backends/apalache/src/decode.rs, model/umpire/backends/apalache/tests/controls.rs, model/umpire/backends/apalache/Cargo.toml, model/Cargo.lock
**Touches:** [model/umpire/backends/apalache/src/decode.rs, model/umpire/backends/apalache/tests/controls.rs, model/umpire/backends/apalache/Cargo.toml, model/Cargo.lock]

Isolated source lane after fn247 closure; no whole-task dependency on another fn248 source lane. Coordinate shared interfaces, accept central merge conflicts and retain private source/index/build targets. The final integration task owns snapshot reacceptance, current guest publication and the one R5 full pipeline. No local lane result claims joined validation.
Serde-derived ITF envelope in the Apalache back end.

- ITF envelope (`model/umpire/backends/apalache/src/decode.rs` ~130-160): `#[derive(Deserialize)]` with `#[serde(deny_unknown_fields)]` for the top-level object instead of manual `keys()`/`get()` checks. The strict value visitor, pre-scan budget, `#bigint`/`#tup`/`#set` handling and schema checks stay. Enable serde's `derive` feature only if `serde_derive` is already resolved in `model/Cargo.lock`; otherwise record the admission under fn-210 R18.

The `sha2` replacement of the `shasum` subprocess moved to fn-250.5 (2026-10-24, throughput ordering): it lands once, inside the consolidated `PinnedJar::admit`, instead of being rewritten in two runners first.
### Strict-envelope proof

Retain the pre-scan byte/item/depth budget and strict nested Json visitor. Decode the top-level envelope without silently overwriting duplicate keys. Preserve required #meta/vars/states and optional params semantics: plain `Option<T>` commonly conflates explicit null with absence, whereas current Json rejects null; use a presence-aware field decoder/default and test both. Preserve metadata format/version/type checks, declarations, state bounds/index order, fixed World parameters and nested #bigint/#tup/#set validation. Include unknown/duplicate top-level and nested fields, explicit null, missing fields, floats, budget exhaustion and typed replay-negative controls. serde_derive1.0.229/syn3 is already resolved; feature admission still needs the current host/guest inventory.

## Acceptance
- [ ] The ITF envelope decodes through serde derive; unknown or duplicate top-level keys still refuse.
- [ ] Focused strict-envelope controls pass; final joined validation exercises required Apalache mode.
## Focused verification and integration handoff

Run from the repository root, with a lane-owned target; commands containing `cd model` enter the Rust workspace. These are mandatory affected-scope checks, not another full Model gate:

```bash
cd model && mise exec -- cargo test -p umpire-apalache --test controls --locked
```

Lane-local candidate snapshots may be generated/reviewed to exercise focused controls in the isolated copy; final canonical reacceptance still occurs once after joining. Report actual failures and candidate snapshot diffs. Add controls specified by this task and run the new controls by their actual selectors. After central conflict resolution, rerun affected focused checks; final integration owns reviewed snapshot acceptance/current artifacts and R5. No historical output-byte comparison is required.

## Done summary
Replaced the Apalache ITF top-level envelope checks with serde derive and deny_unknown_fields, retaining strict nested Json decoding, duplicate/required-field refusal, presence-aware params (absence differs from explicit null), object-only roots and prescan byte/item/depth budgets. Metadata/version, declarations, ordered/bounded states, fixed World parameters and tagged composite/type checks remain; independent typed replay is unchanged.

baseline: green (14 controls, live checks disabled), after correcting an inconclusive shell-cwd attempt before source edits. Final worker checks passed28 tests (17 controls+5 decode+6 replay),3 strict-envelope selectors, warnings-denied all-target Clippy and fmt/diff checks. Null-collapse and positional-array weakening mutants each failed their intended assertion and were restored before green verification. The original parser passed the two preservation controls. Worker narrow vector and typed payload/World live replay controls passed with required Apalache; outputs/witnesses and typed tampering negatives are retained under task6/live/. No snapshot output changed.

Root reran the28 focused tests after joining the current traversal/envelope source, then separately ran actual required real_vector_witness_replays_typed_sequence: exit0 in7.18s. Joined/validation.json and logs bind the exact manifest/decoder/control hashes, which still match reviewed source. Optional cases in the default group are not claimed as live execution. Serde_derive1.0.229/syn3.0.9 were already locked; only host Apalache enables derive, with no new packages or lock diff. All6 wasm-target normal/build feature-edge trees match before/after; this is metadata resolution evidence, not guest compilation.

Three fresh read-only codex:gpt-6.1-sol:high draws and backend finalizer returned SHIP, zero findings/unaddressed task6 obligations. Contracts ran existing28 focused binaries; integration ran3 new controls+5 decode+6 replay binaries. Fresh Cargo attempts were blocked by read-only locks; correctness checked committed source/logs. Same-family provenance and overlapping test identities are explicit. No source fix, nontrivial fix-memory capture or extra full/AP-group/Go/guest run occurred. Verify classified FULL for executable configuration; this task has focused checks, so unchanged passing source-bound evidence is reused without fabricated full receipts/skip lines.

Exact review base232292523157176fd484c648aae23bebaedf7870, original worker19a59be036+7fce9cddbd and source integrationc58fda6a1d are retained. Task6 proof.md/validation.json and task6/joined/validation.json retain commands/negatives/live/inventory/source evidence; final backend receipt/provenance are task6/review/receipt.json and provenance.md. Final7 owns covering required validation/current guest publication/snapshot acceptance. Runtime claim was authoritative; the tracked todo definition was not reset or re-claimed. No MILESTONES, parent closure, remote or history rewrite occurred.

stage: impl-review - ran (three first-round SHIP draws; backend finalizer SHIP; task6/review/receipt.json)
Tier: session (jev-unavailable(no_key))

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 19a59be0363cf0e3a8cba2d3510e2b14f1aea8ea, 7fce9cddbd6fd64744b4b631823cd9301937330b, c58fda6a1d4d0fa34c2df82f910cc91bfc68d0b4, 956c017c32e0f009a475c9d0fa664b06f74d9e0a
- Tests: cd model && CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2486-target mise exec -- cargo test -p umpire-apalache --test controls --locked, cd model && CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2486-target mise exec -- cargo test -p umpire-apalache --test controls strict_envelope_ --locked, cd model && CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2486-target mise exec -- cargo test -p umpire-apalache --test controls --test decode --test replay --locked, NEGATIVE_CONTROL expected exit 101: cd model && CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2486-target mise exec -- cargo test -p umpire-apalache --test controls strict_envelope_presence_and_object_controls --locked (null-collapse negative control), NEGATIVE_CONTROL expected exit 101: cd model && CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2486-target mise exec -- cargo test -p umpire-apalache --test controls strict_envelope_presence_and_object_controls --locked (positional-array negative control), cd model && CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2486-target UMPIRE_APALACHE=require mise exec -- cargo test -p umpire-apalache --test controls real_vector_witness_replays_typed_sequence --locked, cd model && CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2486-target UMPIRE_APALACHE=require mise exec -- cargo test -p umpire-apalache --test controls real_typed_payload_rejection_and_after_claims_replay --locked, cd model && CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2486-target mise exec -- cargo clippy -p umpire-apalache --all-targets --locked -- -D warnings, cd model && CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2486-target mise exec -- cargo fmt -p umpire-apalache -- --check, cd model && CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2486-target mise exec -- cargo tree -p umpire-stock-guest -p umpire-control-guest -p umpire-caught-panic-guest -p umpire-overflow-guest -p umpire-kettle-guest -p umpire-temporal-guest --target wasm32-wasip1 --edges normal,build,features --locked, cd model && CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2486-target mise exec -- cargo tree -p umpire-apalache -e features --locked, git diff --check, cargo test -p umpire-apalache --test controls --test decode --test replay --locked (post-merge exit0; task6/joined/validation.json), UMPIRE_APALACHE=require cargo test -p umpire-apalache --test controls real_vector_witness_replays_typed_sequence --locked -- --exact (post-merge exit0; task6/joined/validation.json)
- PRs: