# Gomad: typed boundaries for the Rust host

**Owner acceptance amendment, 2026-10-24 (legacy compatibility retired).** Apply the [current compatibility policy](../artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema, CLI/message and compatibility-only frozen-host requirements below are retired. Current behavior, integrity and same-candidate deterministic replay remain required; compatibility-only proof rows may retire explicitly without replacement proof.

## Conversation Evidence

> user: "where does the actual complexity live in this project? and is code size and complexity coupled? is the complexity worth it or accidental?"
> user: "any tasks we want to add that 242 and 240 don't cover but reduce accidental complexity?"
> user: "yes add"

## Goal & Context

The 2026-10-24 complexity review of `tools/gomad/rust` found the essential difficulty (deterministic models, engine, cluster broker, hardened artifact store) in about a third of the code, and the rest dominated by three repeated patterns that fn-240 (crate layout) and fn-242 (driver and duplicate-check removal) do not change:

- The "read bounded from a pinned handle, digest, compare, decode" sequence is re-implemented per module: 28 files define their own `fn read`, 11 a `fn digest`, 6 a `fn authenticate`, and more than 50 files construct `Sha256` directly (campaign 8, cluster 9, model 6, qualify 4, evidence 4, measurement 3, every `fixture_*` module once).
- Retained JSON records are handled as `serde_json::Value` with string-key indexing: 3860 sites, concentrated in campaign (26 files), measurement (20), qualify (16), cli (12), stress (7). Every reader re-validates shape, and a renamed key fails at run time.
- Typed errors and size lints are now fn-240 R11/R12 and are not repeated here.

Both patterns touch every host module, so they must not race the fn-240 extractions or type records fn-242 is about to delete.

## Architecture & Data Models

**Authenticated reads (R1).** `gomad-evidence` gains one module, `io`, with three pieces: `BoundedRead` (read at most N bytes from an already pinned handle, refusing longer input with the existing boundary message), `Digest` (the one SHA-256 and hex helper, with the canonical-bytes variant), and `Authenticated<T>` (bounded read, digest, compare against an expected identity, then typed decode; the only way to turn retained bytes into a value). Callers in campaign, qualify, minimize, replay, measurement, install, sweep and the fixture modules delete their local copies. The engine side (`wasmhost`) keeps its own identity sites, which bind helper and module identities and already live in one place each.

**Typed receipts (R2).** Each retained JSON record that survives fn-242 gets a serde struct with `#[serde(deny_unknown_fields)]` and the field set documented in CLI.md: qualification result, stress report and recipe, measurement receipt and input catalogue row, scenario, campaign ledger row, minimize checkpoint. Canonical bytes keep flowing through the existing canonical encoder (sorted keys, native integer widths), so on-disk bytes and identities do not change. `serde_json::Value` remains only in the canonical encoder, the `inspect` verb's output assembly, and tests.

## API Contracts

- All public CLI flags, JSON outputs, record formats, identities and goldens are unchanged; the reviewed goldens are the proof.
- `cargo test --workspace`, `make -C tools/gomad test-harness test-rust-engine validate-wasm-determinism lint-rust` pass after every task.
- No new dependency beyond what the workspace already has.

## Edge Cases & Constraints

- Identity bytes are the product. Any change to what is hashed, or in which order, is a finding; `Digest` reproduces the existing byte sequences exactly, and the flip-byte and golden tests from fn-240 stay green.
- `deny_unknown_fields` must not reject records the frozen candidate produced. Each typed record is checked against the retained corpus under `testdata/` and the frozen-candidate artifacts before the loose reader is deleted.
- Workers run in parallel lanes; both tasks are large mechanical diffs. Each lands on its own branch and merges before the next starts.
- No CI run, PR or push is authorized by this spec.

## Acceptance Criteria

- **R1:** `Sha256` is constructed in at most three host-side modules (`gomad-evidence::io` and the two engine identity sites); every deleted `read`/`digest`/`authenticate` helper has a caller on the shared one; identity goldens and the fn-240 flip-byte test are unchanged. Errors: a surviving local digest helper, or a changed identity byte, is a finding.
- **R2:** `serde_json::Value` appears only in the canonical encoder, `inspect` output assembly and tests; every retained record named above has a `deny_unknown_fields` struct that round-trips the retained corpus and the frozen-candidate artifacts byte-for-byte through the canonical encoder. Errors: a `Value` index outside those three places, or a record that fails to round-trip, is a finding.
- **R3:** Before/after counts for the two patterns (Sha256 sites, `Value` sites, helper definitions) are recorded with exact revisions in `.flow/artifacts/fn-245/counts.md`. Errors: a count without revisions is a finding.

## Boundaries

- No behaviour, format or identity change; no new CLI flags.
- No crate moves (fn-240 owns layout) and no removal of checks (fn-242 owns that).
- Typed errors and size lints are fn-240 R11/R12, not this spec.
- No CI run, PR or push.

## Decision Context

Owner, 2026-10-24: accepted the four proposals from the complexity review. Two ride on fn-240 (typed errors per extraction, size lints at .5) and are recorded there as R11/R12; these two touch every host module and are filed separately so they can be sequenced after fn-240's crate graph exists and after fn-242 has deleted what should not be typed. Rejected: a general "un-Go the host code" task, because it has no finish line.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | One authenticated-read abstraction in gomad-evidence. | fn-245-gomad-typed-boundaries-for-the-rust-host.1 | — |
| R2 | Typed receipts replace serde_json::Value. | fn-245-gomad-typed-boundaries-for-the-rust-host.2 | — |
| R3 | Before/after counts recorded. | fn-245-gomad-typed-boundaries-for-the-rust-host.1, fn-245-gomad-typed-boundaries-for-the-rust-host.2 | — |
