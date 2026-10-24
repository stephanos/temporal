---
satisfies: [R2, R3]
---
# fn-245-gomad-typed-boundaries-for-the-rust-host.2 Typed receipts replace serde_json::Value

**Owner acceptance amendment, 2026-10-24 (legacy compatibility retired).** Apply the [current compatibility policy](../artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema, CLI/message and compatibility-only frozen-host requirements below are retired. Current behavior, integrity and same-candidate deterministic replay remain required; compatibility-only proof rows may retire explicitly without replacement proof.

## Description
For each retained JSON record surviving fn-242 (qualification result, stress report and recipe, measurement receipt and input catalogue row, scenario, campaign ledger row, minimize checkpoint) define a serde struct with deny_unknown_fields and the documented field set; serialize through the existing canonical encoder so on-disk bytes and identities are unchanged. Before deleting a loose reader, round-trip the retained corpus under testdata/ and the frozen-candidate artifacts through the struct byte-for-byte. Remove Value indexing from campaign, measurement, qualify, cli, stress, replay, sweep, minimize, install, shim; Value stays only in the canonical encoder, inspect output assembly and tests. Ordering: after fn-242.7 (driver deleted, duplicate checks removed) so nothing about to be deleted gets typed; record before/after Value site counts with revisions.

## Acceptance
grep for 'serde_json::Value' and string-key indexing outside the canonical encoder, inspect assembly and tests is empty; every typed record round-trips the retained corpus and frozen-candidate artifacts byte-for-byte; goldens unchanged; cargo test --workspace and the make gates green; counts in .flow/artifacts/fn-245/counts.md.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
