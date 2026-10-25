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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
